package dib

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/executor"
)

// ImageMetadata contains information about an image and allows to convert it to standard OCI image labels.
type ImageMetadata struct {
	// Fields translated to labels
	Created    time.Time
	Authors    string
	Revision   string
	Source     string
	Title      string
	URL        string
	Version    string
	RefName    string
	BaseName   string
	BaseDigest string

	// Other metadata needed internally
	repositoryRootPath  string
	repositoryRootURL   string
	repositoryCommitURL string
	repositoryTag       string
}

// LoadCommonMetadata returns a ImageMetadata struct filled with metadata from the current build environment.
// It automatically discovers environment variables from different CI vendors (GitHub actions, GitLab CI),
// or fallback to using local git repository as metadata source.
func LoadCommonMetadata(cmd executor.ShellExecutor) ImageMetadata {
	meta := ImageMetadata{}

	metadataLoaded := false

	if os.Getenv("GITHUB_REPOSITORY") != "" {
		loadGitHubMeta(&meta)

		metadataLoaded = true
	}

	if os.Getenv("CI_PROJECT_URL") != "" {
		loadGitLabMeta(&meta)

		metadataLoaded = true
	}

	if !metadataLoaded {
		// Fallback to using metadata form local git repository.
		loadGitMeta(&meta, cmd)
	}

	return meta
}

// WithImage returns a copy of ImageMetadata containing metadata from the given Image.
func (m ImageMetadata) WithImage(image *dag.Image) ImageMetadata {
	absolutePath := path.Join(image.Dockerfile.ContextPath, image.Dockerfile.Filename)
	relativePath := strings.TrimPrefix(absolutePath, m.repositoryRootPath)

	m.Created = time.Now()
	m.Title = image.ShortName
	m.Source = m.repositoryCommitURL + relativePath
	m.URL = m.repositoryRootURL + relativePath
	m.RefName = image.Hash
	m.Version = image.Hash

	if m.repositoryTag != "" {
		m.Version = m.repositoryTag
	}

	if len(image.Dockerfile.From) > 0 {
		// The base image is in the last FROM statement
		m.BaseName = image.Dockerfile.From[len(image.Dockerfile.From)-1].Name
	}

	return m
}

// ToLabels converts the ImageMetadata information to a map of standard OCI labels.
func (m ImageMetadata) ToLabels() map[string]string {
	labels := map[string]string{
		"org.opencontainers.image.created":   m.Created.Format(time.RFC3339),
		"org.opencontainers.image.revision":  m.Revision,
		"org.opencontainers.image.source":    m.Source,
		"org.opencontainers.image.title":     m.Title,
		"org.opencontainers.image.url":       m.URL,
		"org.opencontainers.image.version":   m.Version,
		"org.opencontainers.image.ref.name":  m.RefName,
		"org.opencontainers.image.base.name": m.BaseName,
	}

	// Remove empty labels
	nonEmptyLabels := map[string]string{}

	for k, v := range labels {
		if v == "" {
			continue
		}

		nonEmptyLabels[k] = v
	}

	return nonEmptyLabels
}

func loadGitHubMeta(meta *ImageMetadata) {
	meta.Authors = os.Getenv("GITHUB_ACTOR")
	meta.Revision = os.Getenv("GITHUB_SHA")

	repoURL := fmt.Sprintf("%s/%s/blob", os.Getenv("GITHUB_SERVER_URL"), os.Getenv("GITHUB_REPOSITORY"))
	meta.repositoryRootURL = fmt.Sprintf("%s/%s", repoURL, os.Getenv("GITHUB_BASE_REF"))
	meta.repositoryCommitURL = fmt.Sprintf("%s/%s", repoURL, meta.Revision)
	meta.repositoryRootPath = os.Getenv("GITHUB_WORKSPACE")

	if os.Getenv("GITHUB_REF_TYPE") == "tag" {
		meta.repositoryTag = os.Getenv("GITHUB_REF_NAME")
	}
}

func loadGitLabMeta(meta *ImageMetadata) {
	meta.Authors = os.Getenv("GITLAB_USER_NAME")
	meta.Revision = os.Getenv("CI_COMMIT_SHA")

	repoURL := fmt.Sprintf("%s/-/blob", os.Getenv("CI_PROJECT_URL"))
	meta.repositoryRootURL = fmt.Sprintf("%s/%s", repoURL, os.Getenv("CI_DEFAULT_BRANCH"))
	meta.repositoryCommitURL = fmt.Sprintf("%s/%s", repoURL, meta.Revision)
	meta.repositoryRootPath = os.Getenv("CI_PROJECT_DIR")
	meta.repositoryTag = os.Getenv("CI_COMMIT_TAG")
}

func loadGitMeta(meta *ImageMetadata, cmd executor.ShellExecutor) {
	rev, err := cmd.Execute("git", "rev-parse", "HEAD")
	if err == nil {
		meta.Revision = strings.Trim(rev, "\n")
	}

	root, err := cmd.Execute("git", "rev-parse", "--show-toplevel")
	if err == nil {
		meta.repositoryRootPath = strings.Trim(root, "\n")
	}
}

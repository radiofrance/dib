package dib_test

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/dib"
	"github.com/radiofrance/dib/pkg/dockerfile"
	"github.com/radiofrance/dib/pkg/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_LabelsFromGitHubMetadata(t *testing.T) {
	now := time.Now()
	cwd, err := os.Getwd()
	require.NoError(t, err)

	// Given
	t.Setenv("GITHUB_REPOSITORY", "organization/repository")
	t.Setenv("GITHUB_ACTOR", "John Doe <john.doe@example.org>")
	t.Setenv("GITHUB_SHA", "e6d0536487b24c11ca8675cbf8e1b015f843bd26")
	t.Setenv("GITHUB_REF_TYPE", "tag")
	t.Setenv("GITHUB_REF_NAME", "1.0.0")
	t.Setenv("GITHUB_SERVER_URL", "https://github.com")
	t.Setenv("GITHUB_BASE_REF", "main")
	t.Setenv("GITHUB_WORKSPACE", cwd)

	image := &dag.Image{
		ShortName: "dib",
		Hash:      "silver-august-sierra-washington",
		Dockerfile: &dockerfile.Dockerfile{
			ContextPath: path.Join(cwd, "context/subdir"),
			Filename:    "Dockerfile",
			From: []dockerfile.ImageRef{
				{Name: "debian"},
			},
		},
	}

	// When
	meta := dib.LoadCommonMetadata(&mock.Executor{}).WithImage(image)
	meta.Created = now
	actual := meta.ToLabels()

	// Then
	expected := map[string]string{
		"org.opencontainers.image.created":   now.Format(time.RFC3339),
		"org.opencontainers.image.revision":  "e6d0536487b24c11ca8675cbf8e1b015f843bd26",
		"org.opencontainers.image.title":     "dib",
		"org.opencontainers.image.source":    "https://github.com/organization/repository/blob/e6d0536487b24c11ca8675cbf8e1b015f843bd26/context/subdir/Dockerfile", //nolint:lll
		"org.opencontainers.image.url":       "https://github.com/organization/repository/blob/main/context/subdir/Dockerfile",                                     //nolint:lll
		"org.opencontainers.image.version":   "1.0.0",
		"org.opencontainers.image.ref.name":  "silver-august-sierra-washington",
		"org.opencontainers.image.base.name": "debian",
	}
	assert.Equal(t, expected, actual)
}

func Test_LabelsFromGitLabMetadata(t *testing.T) {
	now := time.Now()
	cwd, err := os.Getwd()
	require.NoError(t, err)

	// Given
	t.Setenv("GITLAB_USER_NAME", "John Doe <john.doe@example.org>")
	t.Setenv("CI_COMMIT_SHA", "e6d0536487b24c11ca8675cbf8e1b015f843bd26")
	t.Setenv("CI_COMMIT_TAG", "1.0.0")
	t.Setenv("CI_PROJECT_URL", "https://gitlab.com/project/repository")
	t.Setenv("CI_DEFAULT_BRANCH", "main")
	t.Setenv("CI_PROJECT_DIR", cwd)

	image := &dag.Image{
		ShortName: "dib",
		Hash:      "silver-august-sierra-washington",
		Dockerfile: &dockerfile.Dockerfile{
			ContextPath: path.Join(cwd, "context/subdir"),
			Filename:    "Dockerfile",
			From: []dockerfile.ImageRef{
				{Name: "debian"},
			},
		},
	}

	// When
	meta := dib.LoadCommonMetadata(&mock.Executor{}).WithImage(image)
	meta.Created = now
	actual := meta.ToLabels()

	// Then
	expected := map[string]string{
		"org.opencontainers.image.created":   now.Format(time.RFC3339),
		"org.opencontainers.image.revision":  "e6d0536487b24c11ca8675cbf8e1b015f843bd26",
		"org.opencontainers.image.title":     "dib",
		"org.opencontainers.image.source":    "https://gitlab.com/project/repository/-/blob/e6d0536487b24c11ca8675cbf8e1b015f843bd26/context/subdir/Dockerfile", //nolint:lll
		"org.opencontainers.image.url":       "https://gitlab.com/project/repository/-/blob/main/context/subdir/Dockerfile",                                     //nolint:lll
		"org.opencontainers.image.version":   "1.0.0",
		"org.opencontainers.image.ref.name":  "silver-august-sierra-washington",
		"org.opencontainers.image.base.name": "debian",
	}
	assert.Equal(t, expected, actual)
}

func Test_LabelsFromGitMetadata(t *testing.T) {
	// Since we run our CI on GitHub, we need to reset this variable to make the test pass.
	t.Setenv("GITHUB_REPOSITORY", "")

	now := time.Now()
	cwd, err := os.Getwd()
	require.NoError(t, err)

	// Given

	image := &dag.Image{
		ShortName: "dib",
		Hash:      "silver-august-sierra-washington",
		Dockerfile: &dockerfile.Dockerfile{
			ContextPath: path.Join(cwd, "../../context/subdir"),
			Filename:    "Dockerfile",
			From: []dockerfile.ImageRef{
				{Name: "debian"},
			},
		},
	}

	cmd := mock.NewExecutor([]mock.ExecutorResult{
		{
			// git rev-parse HEAD
			Output: "e6d0536487b24c11ca8675cbf8e1b015f843bd26\n",
			Error:  nil,
		},
		{
			// git rev-parse --show-toplevel
			Output: path.Join(cwd, "../../") + "\n",
			Error:  nil,
		},
	})

	// When
	meta := dib.LoadCommonMetadata(cmd).WithImage(image)
	meta.Created = now
	actual := meta.ToLabels()

	// Then
	expected := map[string]string{
		"org.opencontainers.image.created":   now.Format(time.RFC3339),
		"org.opencontainers.image.revision":  "e6d0536487b24c11ca8675cbf8e1b015f843bd26",
		"org.opencontainers.image.title":     "dib",
		"org.opencontainers.image.source":    "/context/subdir/Dockerfile",
		"org.opencontainers.image.url":       "/context/subdir/Dockerfile",
		"org.opencontainers.image.version":   "silver-august-sierra-washington",
		"org.opencontainers.image.ref.name":  "silver-august-sierra-washington",
		"org.opencontainers.image.base.name": "debian",
	}
	assert.Equal(t, expected, actual)
}

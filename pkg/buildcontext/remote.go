package buildcontext

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/distribution/reference"
	"github.com/docker/cli/cli/command/image/build"
	"github.com/moby/patternmatcher"
	"github.com/radiofrance/dib/pkg/logger"
	"github.com/radiofrance/dib/pkg/types"
)

// defaultDockerfile is the name of the Dockerfile when none is given.
const defaultDockerfile = "Dockerfile"

// FileUploader is an interface for uploading files to a remote location.
// It basically abstracts storage services such as AWS S3, GCS, etc...
type FileUploader interface {
	UploadFile(ctx context.Context, filePath, targetPath string) error
	PresignedURL(ctx context.Context, targetPath string) (string, error)
}

// RemoteContextProvider allows uploading the build context to a remote location.
type RemoteContextProvider struct {
	uploader FileUploader
	builder  string
}

func NewRemoteContextProvider(uploader FileUploader, builder string) *RemoteContextProvider {
	return &RemoteContextProvider{uploader, builder}
}

// PrepareContext is responsible for creating an archive of the build context files
// and uploading it to the remote location where the build pod can retrieve it later.
func (c *RemoteContextProvider) PrepareContext(ctx context.Context, opts types.ImageBuilderOpts) (string, error) {
	tagParts := strings.Split(opts.Tags[0], ":")
	shortName := path.Base(tagParts[0])
	remoteDir := fmt.Sprintf("%s/%s", c.builder, shortName)
	filename := fmt.Sprintf("context-%s-%s-%s.tar.gz", c.builder, shortName, tagParts[1])
	tarGzPath := path.Join(opts.Context, filename)

	// Parse the first tag to get a normalized reference
	parsedReference, err := reference.ParseNormalizedNamed(opts.Tags[0])
	if err != nil {
		return "", fmt.Errorf("failed to parse image reference: %w", err)
	}

	// Get the familiar name (repository without tag)
	imageName := reference.FamiliarName(parsedReference)

	logger.Infof("Creating the build context archive for image %q", imageName)

	dockerfile := opts.File
	if dockerfile == "" {
		dockerfile = defaultDockerfile
	}

	err = createArchive(opts.Context, dockerfile, tarGzPath)
	if err != nil {
		return "", err
	}

	logger.Infof("Uploading the build context archive for image %q", imageName)

	targetPath := fmt.Sprintf("%s/%s", remoteDir, filename)

	err = uploadBuildContext(ctx, c.uploader, tarGzPath, targetPath)
	if err != nil {
		return "", err
	}

	logger.Infof("Uploading the build context archive for image %q", imageName)

	return c.uploader.PresignedURL(ctx, targetPath)
}

// createArchive builds an archive containing the files in the build context, except the ones excluded by
// its .dockerignore file: BuildKit doesn't apply .dockerignore to a remote context, so it is done here.
// The Dockerfile (a path relative to the build context) is always kept, as BuildKit reads it from the archive.
func createArchive(buildContextDir, dockerfile, tarGzPath string) error {
	// Check if the build context directory exists.
	_, err := os.Stat(buildContextDir)
	if os.IsNotExist(err) {
		return fmt.Errorf("can't access directory %q: it doesn't exist", buildContextDir)
	}

	ignorePatterns, err := build.ReadDockerignore(buildContextDir)
	if err != nil {
		return fmt.Errorf("can't read the .dockerignore file: %w", err)
	}

	ignoreMatcher, err := patternmatcher.New(ignorePatterns)
	if err != nil {
		return fmt.Errorf("can't parse the .dockerignore file: %w", err)
	}

	dockerfile = filepath.Clean(dockerfile)

	// Walk through the build context directory and collect all the files to be archived.
	files := make(map[string]os.FileInfo)

	err = filepath.Walk(buildContextDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing file %s: %w", filePath, err)
		}

		relPath, err := filepath.Rel(buildContextDir, filePath)
		if err != nil {
			return fmt.Errorf("getting relative path for file %q: %w", filePath, err)
		}

		if relPath == "." || relPath == dockerfile {
			if !info.IsDir() {
				files[filePath] = info
			}

			return nil
		}

		ignored, err := ignoreMatcher.MatchesOrParentMatches(relPath)
		if err != nil {
			return fmt.Errorf("matching %q with the .dockerignore patterns: %w", relPath, err)
		}

		if info.IsDir() {
			// Skip an ignored directory, unless an exception ("!pattern") may include some of its files,
			// or it contains the Dockerfile.
			isDockerfileParent := strings.HasPrefix(dockerfile, relPath+string(filepath.Separator))
			if ignored && !ignoreMatcher.Exclusions() && !isDockerfileParent {
				return filepath.SkipDir
			}

			return nil
		}

		if !ignored {
			files[filePath] = info
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error walking the build context directory %s: %w", buildContextDir, err)
	}

	// Create the directory for the .tar.gz file if it doesn't exist.
	err = os.MkdirAll(path.Dir(tarGzPath), 0o750)
	if err != nil {
		return fmt.Errorf("can't create the archive destination directory %q: %w", path.Dir(tarGzPath), err)
	}

	// Create the .tar.gz file.
	tarGzFile, err := os.Create(tarGzPath) //nolint:gosec
	if err != nil {
		return fmt.Errorf("can't create tar.gz file %s: %w", tarGzPath, err)
	}

	defer func() {
		_ = tarGzFile.Close()
	}()

	gzipWriter := gzip.NewWriter(tarGzFile)

	defer func() {
		_ = gzipWriter.Close()
	}()

	tarWriter := tar.NewWriter(gzipWriter)

	defer func() {
		_ = tarWriter.Close()
	}()

	for filePath, info := range files {
		err := writeTarArchive(tarWriter, buildContextDir, filePath, info)
		if err != nil {
			return fmt.Errorf("writing tar archive %q: %w", filePath, err)
		}
	}

	return nil
}

func writeTarArchive(writer *tar.Writer, basePath, path string, info fs.FileInfo) error {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file %q does not exist: %w", path, err)
		}
	}

	header, err := tar.FileInfoHeader(info, path)
	if err != nil {
		return fmt.Errorf("creating header for file %q: %w", path, err)
	}

	relPath, err := filepath.Rel(basePath, path)
	if err != nil {
		return fmt.Errorf("getting relative path for file %q: %w", path, err)
	}

	header.Name = relPath

	err = writer.WriteHeader(header)
	if err != nil {
		return fmt.Errorf("writing header for file %q: %w", header.Name, err)
	}

	file, err := os.Open(path) //nolint:gosec
	if err != nil {
		return fmt.Errorf("opening file %q: %w", path, err)
	}

	_, err = io.Copy(writer, file)
	if err != nil {
		return fmt.Errorf("writing file %q to tar archive: %w", path, err)
	}

	err = file.Close()
	if err != nil {
		logger.Errorf("closing file %q: %s", path, err)
	}

	return nil
}

// uploadBuildContext uploads the file to the remote location.
func uploadBuildContext(ctx context.Context, uploader FileUploader, tarGzPath, targetPath string) error {
	defer func() {
		err := os.Remove(tarGzPath)
		if err != nil {
			logger.Errorf("can't remove file %s: %v", tarGzPath, err)
		}
	}()

	err := uploader.UploadFile(ctx, tarGzPath, targetPath)
	if err != nil {
		return fmt.Errorf("can't upload context archive: %w", err)
	}

	return nil
}

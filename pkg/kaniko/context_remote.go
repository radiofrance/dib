package kaniko

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/radiofrance/dib/internal/logger"
	"github.com/radiofrance/dib/pkg/types"
)

// FileUploader is an interface for uploading files to a remote location.
// It basically abstracts storage services such as AWS S3, GCS, etc...
type FileUploader interface {
	UploadFile(filePath string, targetPath string) error
	URL(targetPath string) string
}

// RemoteContextProvider allows to upload the build context to a remote location.
type RemoteContextProvider struct {
	uploader FileUploader
}

// NewRemoteContextProvider creates a new instance of RemoteContextProvider.
func NewRemoteContextProvider(uploader FileUploader) *RemoteContextProvider {
	return &RemoteContextProvider{uploader}
}

// PrepareContext is responsible for creating an archive of the build context directory
// and uploading it to the remote location where the kaniko build pod can retrieve it later.
func (c RemoteContextProvider) PrepareContext(opts types.ImageBuilderOpts) (string, error) {
	tagParts := strings.Split(opts.Tags[0], ":")
	shortName := path.Base(tagParts[0])
	remoteDir := fmt.Sprintf("kaniko/%s", shortName)
	filename := fmt.Sprintf("context-kaniko-%s-%s.tar.gz", shortName, tagParts[1])

	tarGzPath := path.Join(opts.Context, filename)
	if err := createArchive(opts.Context, tarGzPath); err != nil {
		return "", err
	}

	targetPath := fmt.Sprintf("%s/%s", remoteDir, filename)
	if err := uploadBuildContext(c.uploader, tarGzPath, targetPath); err != nil {
		return "", err
	}

	return c.uploader.URL(targetPath), nil
}

// createArchive builds an archive containing all the files in the build context.
func createArchive(buildContextDir string, tarGzPath string) error {
	logger.Infof("Creating docker build-context for kaniko")

	// Check if the build context directory exists.
	if _, err := os.Stat(buildContextDir); os.IsNotExist(err) {
		return fmt.Errorf("can't access directory %q: it doesn't exist", buildContextDir)
	}

	// Walk through the build context directory, and collect all the files to be archived.
	files := make(map[string]os.FileInfo)
	if err := filepath.Walk(buildContextDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing file %s: %w", filePath, err)
		}

		if !info.IsDir() {
			files[filePath] = info
		}

		return nil
	}); err != nil {
		return fmt.Errorf("error walking the build context directory %s: %w", buildContextDir, err)
	}

	// Create the directory for the .tar.gz file if it doesn't exist.
	if err := os.MkdirAll(path.Dir(tarGzPath), 0o750); err != nil {
		return fmt.Errorf("can't create the archive destination directory %q: %w", path.Dir(tarGzPath), err)
	}

	// Create the .tar.gz file.
	tarGzFile, err := os.Create(tarGzPath)
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
		// Create a tar header for the file.
		header, err := tar.FileInfoHeader(info, filePath)
		if err != nil {
			return fmt.Errorf("error creating tar header for file %s: %w", filePath, err)
		}

		// Update the header name to be relative to the build context directory
		relPath, err := filepath.Rel(buildContextDir, filePath)
		if err != nil {
			return fmt.Errorf("error getting relative path for file %s: %w", filePath, err)
		}
		header.Name = relPath

		// Write the header to the tar archive.
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("error writing tar header for file %s: %w", header.Name, err)
		}

		// Open the file and write its contents to the tar archive.
		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("error opening file %s: %w", filePath, err)
		}

		if _, err := io.Copy(tarWriter, file); err != nil {
			return fmt.Errorf("error writing file %s to tar archive: %w", filePath, err)
		}

		_ = file.Close()
	}

	return nil
}

// uploadBuildContext uploads the file to the remote location.
func uploadBuildContext(uploader FileUploader, tarGzPath string, targetPath string) error {
	logger.Infof("Uploading build-context to S3")

	defer func() {
		if err := os.Remove(tarGzPath); err != nil {
			logger.Errorf("can't remove file %s: %v", tarGzPath, err)
		}
	}()

	err := uploader.UploadFile(tarGzPath, targetPath)
	if err != nil {
		return fmt.Errorf("can't upload context archive: %w", err)
	}

	return nil
}

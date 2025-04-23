package kaniko

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
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
		if err := writeTarArchive(tarWriter, buildContextDir, filePath, info); err != nil {
			return fmt.Errorf("writing tar archive %q: %w", filePath, err)
		}
	}

	return nil
}

func writeTarArchive(writer *tar.Writer, basePath, path string, info fs.FileInfo) error {
	header, err := tar.FileInfoHeader(info, path)
	if err != nil {
		return fmt.Errorf("creating header for file %q: %w", path, err)
	}

	relPath, err := filepath.Rel(basePath, path)
	if err != nil {
		return fmt.Errorf("getting relative path for file %q: %w", path, err)
	}
	header.Name = relPath

	if err := writer.WriteHeader(header); err != nil {
		return fmt.Errorf("writing header for file %q: %w", header.Name, err)
	}

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening file %q: %w", path, err)
	}

	if _, err := io.Copy(writer, file); err != nil {
		return fmt.Errorf("writing file %q to tar archive: %w", path, err)
	}

	if err := file.Close(); err != nil {
		logger.Errorf("closing file %q: %s", path, err)
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

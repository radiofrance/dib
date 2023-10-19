package kaniko

import (
	"compress/gzip"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/mholt/archiver/v3"
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

	tarGzArchiver := archiver.TarGz{
		Tar: &archiver.Tar{
			OverwriteExisting:      true,
			MkdirAll:               true,
			ImplicitTopLevelFolder: false,
			ContinueOnError:        false,
		},
		CompressionLevel: gzip.BestCompression,
	}

	var filesToArchive []string

	fileOrDirInfos, err := os.ReadDir(buildContextDir)
	if err != nil {
		return fmt.Errorf("can't access directory %s, err is : %w", buildContextDir, err)
	}

	for _, fileOrDir := range fileOrDirInfos {
		filesToArchive = append(filesToArchive, path.Join(buildContextDir, fileOrDir.Name()))
	}

	if err := tarGzArchiver.Archive(filesToArchive, tarGzPath); err != nil {
		return fmt.Errorf("can't create tar archive %s: %w", tarGzPath, err)
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

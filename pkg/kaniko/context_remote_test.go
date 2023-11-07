package kaniko_test

import (
	"os"
	"testing"

	"github.com/radiofrance/dib/pkg/kaniko"
	"github.com/radiofrance/dib/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeUploader struct {
	Src  string
	Dest string
	Err  error
}

func (f *fakeUploader) UploadFile(filePath string, targetPath string) error {
	f.Src = filePath
	f.Dest = targetPath

	return f.Err
}

func (f *fakeUploader) URL(targetPath string) string {
	return "fakes3://bucket/" + targetPath
}

func provideDefaultBuildOptions() types.ImageBuilderOpts {
	return types.ImageBuilderOpts{
		Context: "/tmp/kaniko-context",
		Tags:    []string{"gcr.io/project-id/image:version"},
		BuildArgs: map[string]string{
			"someArg": "someValue",
		},
		Labels: nil,
		Push:   true,
	}
}

//nolint:paralleltest
func Test_RemoteContextProvider_FailsWhenContextDirectoryDoesNotExist(t *testing.T) {
	fakeUploader := &fakeUploader{}
	contextProvider := kaniko.NewRemoteContextProvider(fakeUploader)

	opts := provideDefaultBuildOptions()

	_, err := contextProvider.PrepareContext(opts)

	require.Error(t, err, "can't access directory %s, err is : open %s: no such file or directory",
		opts.Context, opts.Context)
}

//nolint:paralleltest
func Test_RemoteContextProvider_UploadsBuildContext(t *testing.T) {
	fakeUploader := &fakeUploader{}
	contextProvider := kaniko.NewRemoteContextProvider(fakeUploader)

	opts := provideDefaultBuildOptions()

	// Create the build context directory
	err := os.Mkdir(opts.Context, 0o755)
	require.NoErrorf(t, err, "cannot create directory %s", opts.Context)
	if err != nil {
		t.Errorf("cannot create directory %s: %v", opts.Context, err)
	}

	defer os.Remove(opts.Context)

	URL, err := contextProvider.PrepareContext(opts)

	require.NoError(t, err)

	expectedSrc := "/tmp/kaniko-context/context-kaniko-image-version.tar.gz"
	if fakeUploader.Src != expectedSrc {
		t.Errorf("Expected S3 client to have been called with src file %s, got %s instead.", expectedSrc, fakeUploader.Src)
	}

	expectedDst := "kaniko/image/context-kaniko-image-version.tar.gz"
	if fakeUploader.Dest != expectedDst {
		t.Errorf("Expected S3 client to have been called with dest file %s, got %s instead.", expectedDst, fakeUploader.Dest)
	}

	_, err = os.Stat(expectedSrc)
	if err == nil {
		t.Errorf("Expected context archive to be deleted after upload, but is still present on disk.")
	}

	assert.Equal(t, "fakes3://bucket/kaniko/image/context-kaniko-image-version.tar.gz", URL)
}

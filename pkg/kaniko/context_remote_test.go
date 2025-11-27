//nolint:paralleltest
package kaniko_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/radiofrance/dib/pkg/kaniko"
	"github.com/radiofrance/dib/pkg/types"
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

func Test_RemoteContextProvider_FailsWhenContextDirectoryDoesNotExist(t *testing.T) {
	fu := &fakeUploader{}
	contextProvider := kaniko.NewRemoteContextProvider(fu)

	opts := provideDefaultBuildOptions()
	_, err := contextProvider.PrepareContext(opts)
	require.Error(t, err)
}

func Test_RemoteContextProvider_UploadsBuildContext(t *testing.T) {
	fu := &fakeUploader{}
	contextProvider := kaniko.NewRemoteContextProvider(fu)

	// Create the build context directory
	opts := provideDefaultBuildOptions()
	require.NoError(t, os.Mkdir(opts.Context, 0o750))
	t.Cleanup(func() {
		_ = os.Remove(opts.Context)
	})

	url, err := contextProvider.PrepareContext(opts)
	require.NoError(t, err)

	expectedSrc := "/tmp/kaniko-context/context-kaniko-image-version.tar.gz"
	assert.Equal(t, expectedSrc, fu.Src)

	expectedDst := "kaniko/image/context-kaniko-image-version.tar.gz"
	assert.Equal(t, expectedDst, fu.Dest)

	_, err = os.Stat(expectedSrc)
	require.Error(t, err, "Expected context archive to be deleted after upload, but is still present on disk.")

	assert.Equal(t, "fakes3://bucket/kaniko/image/context-kaniko-image-version.tar.gz", url)
}

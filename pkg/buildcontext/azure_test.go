package buildcontext_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/radiofrance/dib/pkg/buildcontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAzureUploader(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()

		uploader, err := buildcontext.NewAzureUploader("account", "container")
		require.NoError(t, err)
		assert.NotNil(t, uploader)
	})

	t.Run("no container", func(t *testing.T) {
		t.Parallel()

		_, err := buildcontext.NewAzureUploader("account", "")
		require.Error(t, err)
		assert.ErrorContains(t, err, "azure container name is required")
	})

	t.Run("no account", func(t *testing.T) {
		t.Parallel()

		_, err := buildcontext.NewAzureUploader("", "container")
		require.Error(t, err)
		assert.ErrorContains(t, err, "invalid azure storage service URL")
	})

	t.Run("invalid account", func(t *testing.T) {
		t.Parallel()

		_, err := buildcontext.NewAzureUploader("rm -rf *", "container")
		require.Error(t, err)
		assert.ErrorContains(t, err, "invalid azure storage service URL")
	})
}

func TestUploadFileAzure(t *testing.T) {
	t.Parallel()

	uploader, err := buildcontext.NewAzureUploader("account", "container")
	require.NoError(t, err)

	t.Run("existing file", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		existingFile := filepath.Join(dir, "file.txt")
		require.NoError(t, os.WriteFile(existingFile, []byte("content"), 0o644))
		err = uploader.UploadFile(context.Background(), existingFile, "target/path")
		require.Error(t, err)
		assert.ErrorContains(t, err,
			"failed to upload file to Azure Blob Storage: DefaultAzureCredential: failed to acquire a token.")
	})

	t.Run("missing file", func(t *testing.T) {
		t.Parallel()

		missingFile := filepath.Join(t.TempDir(), "does-not-exist")
		err = uploader.UploadFile(context.Background(), missingFile, "target/path")
		require.Error(t, err)
		assert.ErrorContains(t, err, "no such file or directory")
	})
}

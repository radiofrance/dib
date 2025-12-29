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

func TestNewS3Uploader(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()

		uploader, err := buildcontext.NewS3Uploader(context.Background(), "eu-west-3", "bucket")
		require.NoError(t, err)
		assert.NotNil(t, uploader)
	})

	t.Run("no bucket", func(t *testing.T) {
		t.Parallel()

		_, err := buildcontext.NewS3Uploader(context.Background(), "eu-west-3", "")
		require.Error(t, err)
		assert.ErrorContains(t, err, "bucket name is required for S3 upload")
	})

	t.Run("no region", func(t *testing.T) {
		t.Parallel()

		_, err := buildcontext.NewS3Uploader(context.Background(), "", "bucket")
		require.NoError(t, err)
	})
}

func TestUploadFileS3(t *testing.T) {
	t.Parallel()

	uploader, err := buildcontext.NewS3Uploader(context.Background(), "eu-west-3", "bucket")
	require.NoError(t, err)

	t.Run("existing file", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		existingFile := filepath.Join(dir, "file.txt")
		require.NoError(t, os.WriteFile(existingFile, []byte("content"), 0o644))
		err = uploader.UploadFile(context.Background(), existingFile, "target/path")
		require.Error(t, err)
		assert.ErrorContains(t, err, "can't send S3 PUT request")
	})

	t.Run("missing file", func(t *testing.T) {
		t.Parallel()

		missingFile := filepath.Join(t.TempDir(), "does-not-exist")
		err = uploader.UploadFile(context.Background(), missingFile, "target/path")
		require.Error(t, err)
		assert.ErrorContains(t, err, "no such file or directory")
	})
}

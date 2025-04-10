package kaniko

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_createArchive(t *testing.T) {
	t.Parallel()

	t.Run("successful archive creation", func(t *testing.T) {
		t.Parallel()

		srcDir := t.TempDir()
		err := os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(srcDir, "file2.txt"), []byte("content2"), 0o644)
		require.NoError(t, err)

		archivePath := filepath.Join(srcDir, "test.tar.gz")
		err = createArchive(srcDir, archivePath)
		require.NoError(t, err)

		verifyArchive(t, archivePath, []string{"file1.txt", "file2.txt"})
	})

	t.Run("non-existent destination path", func(t *testing.T) {
		t.Parallel()

		srcDir := t.TempDir()
		err := os.WriteFile(filepath.Join(srcDir, "file.txt"), []byte("content"), 0o644)
		require.NoError(t, err)

		archivePath := filepath.Join(srcDir, "non-existent", "test.tar.gz")
		err = createArchive(srcDir, archivePath)
		require.NoError(t, err)

		verifyArchive(t, archivePath, []string{"file.txt"})
	})

	t.Run("empty directory", func(t *testing.T) {
		t.Parallel()

		srcDir := t.TempDir()
		archivePath := filepath.Join(srcDir, "test.tar.gz")

		err := createArchive(srcDir, archivePath)
		require.NoError(t, err)

		verifyArchive(t, archivePath, []string{})
	})

	t.Run("non-existent source directory", func(t *testing.T) {
		t.Parallel()

		srcDir := "/non/existent/directory"
		archivePath := "/tmp/test.tar.gz"

		err := createArchive(srcDir, archivePath)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "can't access directory")
	})
}

// Helper function to verify the contents of a .tar.gz archive.
func verifyArchive(t *testing.T, archivePath string, expectedFiles []string) {
	t.Helper()

	file, err := os.Open(archivePath)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = file.Close()
	})

	gzipReader, err := gzip.NewReader(file)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = gzipReader.Close()
	})

	tarReader := tar.NewReader(gzipReader)

	var files []string
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		files = append(files, header.Name)
	}

	assert.ElementsMatch(t, expectedFiles, files)
}

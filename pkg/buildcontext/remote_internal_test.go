//nolint:gosec
package buildcontext

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
		err = createArchive(srcDir, "Dockerfile", archivePath)
		require.NoError(t, err)

		verifyArchive(t, archivePath, []string{"file1.txt", "file2.txt"})
	})

	t.Run("non-existent destination path", func(t *testing.T) {
		t.Parallel()

		srcDir := t.TempDir()
		err := os.WriteFile(filepath.Join(srcDir, "file.txt"), []byte("content"), 0o644)
		require.NoError(t, err)

		archivePath := filepath.Join(srcDir, "non-existent", "test.tar.gz")
		err = createArchive(srcDir, "Dockerfile", archivePath)
		require.NoError(t, err)

		verifyArchive(t, archivePath, []string{"file.txt"})
	})

	t.Run("empty directory", func(t *testing.T) {
		t.Parallel()

		srcDir := t.TempDir()
		archivePath := filepath.Join(srcDir, "test.tar.gz")

		err := createArchive(srcDir, "Dockerfile", archivePath)
		require.NoError(t, err)

		verifyArchive(t, archivePath, []string{})
	})

	t.Run("files excluded by .dockerignore", func(t *testing.T) {
		t.Parallel()

		srcDir := t.TempDir()
		writeFiles(t, srcDir, map[string]string{
			".dockerignore":            ".git\nnode_modules\n.env\n**/*.md\n",
			"Dockerfile":               "FROM scratch",
			"app.go":                   "package main",
			".env":                     "SECRET=value",
			".git/HEAD":                "ref: refs/heads/main",
			"node_modules/pkg/main.js": "",
			"docs/readme.md":           "",
		})

		archivePath := filepath.Join(t.TempDir(), "test.tar.gz")
		err := createArchive(srcDir, "Dockerfile", archivePath)
		require.NoError(t, err)

		verifyArchive(t, archivePath, []string{".dockerignore", "Dockerfile", "app.go"})
	})

	t.Run("exceptions in .dockerignore", func(t *testing.T) {
		t.Parallel()

		srcDir := t.TempDir()
		writeFiles(t, srcDir, map[string]string{
			".dockerignore": "logs\n!logs/keep.log\n",
			"Dockerfile":    "FROM scratch",
			"logs/app.log":  "",
			"logs/keep.log": "",
		})

		archivePath := filepath.Join(t.TempDir(), "test.tar.gz")
		err := createArchive(srcDir, "Dockerfile", archivePath)
		require.NoError(t, err)

		verifyArchive(t, archivePath, []string{".dockerignore", "Dockerfile", "logs/keep.log"})
	})

	t.Run("Dockerfile excluded by .dockerignore is kept", func(t *testing.T) {
		t.Parallel()

		srcDir := t.TempDir()
		writeFiles(t, srcDir, map[string]string{
			".dockerignore":     "docker\n.dockerignore\n",
			"docker/Dockerfile": "FROM scratch",
			"docker/other.txt":  "",
			"app.go":            "package main",
		})

		archivePath := filepath.Join(t.TempDir(), "test.tar.gz")
		err := createArchive(srcDir, "docker/Dockerfile", archivePath)
		require.NoError(t, err)

		verifyArchive(t, archivePath, []string{"docker/Dockerfile", "app.go"})
	})

	t.Run("non-existent source directory", func(t *testing.T) {
		t.Parallel()

		srcDir := "/non/existent/directory"
		archivePath := "/tmp/test.tar.gz"

		err := createArchive(srcDir, "Dockerfile", archivePath)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "can't access directory")
	})
}

// writeFiles creates the given files (path relative to dir: content), and their parent directories.
func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()

	for name, content := range files {
		filePath := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0o750))
		require.NoError(t, os.WriteFile(filePath, []byte(content), 0o644))
	}
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

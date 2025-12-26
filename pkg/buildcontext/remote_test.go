//nolint:testpackage
package buildcontext

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/radiofrance/dib/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementation for FileUploader.
type mockUploader struct {
	mockUploadResponseVal       error
	mockPresignedURLResponseVal string
	mockPresignedURLErrorVal    error
}

func newMockUploader() *mockUploader {
	return &mockUploader{}
}

func (m *mockUploader) UploadFile(_ context.Context, _, _ string) error {
	return m.mockUploadResponseVal
}

func (m *mockUploader) PresignedURL(_ context.Context, _ string) (string, error) {
	return m.mockPresignedURLResponseVal, m.mockPresignedURLErrorVal
}

func (m *mockUploader) mockUploadResponse(err error) {
	m.mockUploadResponseVal = err
}

func (m *mockUploader) mockPresignedURLResponse(url string, err error) {
	m.mockPresignedURLResponseVal = url
	m.mockPresignedURLErrorVal = err
}

func TestUploadBuildContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setup         func(mup *mockUploader)
		tarGzPath     string
		targetPath    string
		expectedError bool
	}{
		{
			name: "successful_upload",
			setup: func(mup *mockUploader) {
				mup.mockUploadResponse(nil)
			},
			tarGzPath:     "test.tar.gz",
			targetPath:    "remote/test.tar.gz",
			expectedError: false,
		},
		{
			name: "upload_error",
			setup: func(mup *mockUploader) {
				mup.mockUploadResponse(fmt.Errorf("upload failed"))
			},
			tarGzPath:     "test.tar.gz",
			targetPath:    "remote/test.tar.gz",
			expectedError: true,
		},
		{
			name: "file_deletion_error",
			setup: func(mup *mockUploader) {
				mup.mockUploadResponse(nil)
			},
			tarGzPath:     "missing.tar.gz",
			targetPath:    "remote/missing.tar.gz",
			expectedError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			mockUploader := newMockUploader()
			if test.setup != nil {
				test.setup(mockUploader)
			}

			test.tarGzPath = filepath.Join(t.TempDir(), test.tarGzPath)

			f, err := os.Create(test.tarGzPath)
			if err == nil {
				err = f.Close()
				require.NoError(t, err)
			}

			err = uploadBuildContext(context.Background(), mockUploader, test.tarGzPath, test.targetPath)
			assert.Equal(t, test.expectedError, err != nil)

			if !test.expectedError {
				_, err := os.Stat(test.tarGzPath)
				assert.True(t, os.IsNotExist(err), "file should be removed after upload")
			}
		})
	}
}

func TestPrepareContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		imageOpts     types.ImageBuilderOpts
		mockResponses func(mup *mockUploader)
		expectedURL   string
		expectedError bool
	}{
		{
			name: "success_setup",
			imageOpts: types.ImageBuilderOpts{
				Context: "test_context_success",
				Tags:    []string{"sample/image:tag"},
			},
			mockResponses: func(mup *mockUploader) {
				mup.mockUploadResponse(nil)
				mup.mockPresignedURLResponse("https://example.com/buildkit/sample/image/context.tar.gz", nil)
			},
			expectedURL:   "https://example.com/buildkit/sample/image/context.tar.gz",
			expectedError: false,
		},
		{
			name: "missing_context_dir",
			imageOpts: types.ImageBuilderOpts{
				Context: "invalid_context",
				Tags:    []string{"sample/image:tag"},
			},
			mockResponses: nil,
			expectedError: true,
		},
		{
			name: "upload_fail",
			imageOpts: types.ImageBuilderOpts{
				Context: "test_context_upload_fail",
				Tags:    []string{"sample/image:tag"},
			},
			mockResponses: func(mup *mockUploader) {
				mup.mockUploadResponse(fmt.Errorf("upload failed"))
			},
			expectedError: true,
		},
		{
			name: "presigned_url_fail",
			imageOpts: types.ImageBuilderOpts{
				Context: "test_context_presigned_url_fail",
				Tags:    []string{"sample/image:tag"},
			},
			mockResponses: func(mup *mockUploader) {
				mup.mockUploadResponse(nil)
				mup.mockPresignedURLResponse("", fmt.Errorf("failed to get presigned url"))
			},
			expectedError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			mockUploader := newMockUploader()
			provider := NewRemoteContextProvider(mockUploader, "mockBuilder")

			if test.mockResponses != nil {
				test.mockResponses(mockUploader)
			}

			if test.imageOpts.Context != "invalid_context" {
				test.imageOpts.Context = filepath.Join(t.TempDir(), test.imageOpts.Context)
				err := os.Mkdir(test.imageOpts.Context, 0o750)
				require.NoError(t, err)

				defer func() {
					err = os.RemoveAll(test.imageOpts.Context)
					require.NoError(t, err)
				}()
			}

			url, err := provider.PrepareContext(context.Background(), test.imageOpts)

			assert.Equal(t, test.expectedError, err != nil)

			if !test.expectedError {
				assert.Equal(t, test.expectedURL, url)
			}
		})
	}
}

func TestWriteTarArchive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		files         map[string]string
		expectedFiles map[string]string
		expectedError bool
	}{
		{
			name: "success_single_file",
			files: map[string]string{
				"file1.txt": "content1",
			},
			expectedFiles: map[string]string{
				"file1.txt": "content1",
			},
			expectedError: false,
		},
		{
			name: "success_multiple_files",
			files: map[string]string{
				"file1.txt": "content1",
				"file2.txt": "content2",
			},
			expectedFiles: map[string]string{
				"file1.txt": "content1",
				"file2.txt": "content2",
			},
			expectedError: false,
		},
		{
			name:          "empty_directory",
			files:         nil,
			expectedFiles: map[string]string{},
			expectedError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			// Create a temporary directory for the test
			tempDir := t.TempDir()

			// Create test files if specified
			for fileName, content := range test.files {
				filePath := filepath.Join(tempDir, fileName)
				err := os.MkdirAll(filepath.Dir(filePath), 0o750)
				require.NoError(t, err, "failed to create directory structure")
				err = os.WriteFile(filePath, []byte(content), 0o644)
				require.NoError(t, err, "unable to create file", filePath)
			}

			// Create a buffer to store the tar archive
			var buffer io.ReadWriter = new(bytes.Buffer)

			tarWriter := tar.NewWriter(buffer)

			defer func() {
				err := tarWriter.Close()
				require.NoError(t, err)
			}()

			// Test writing files to a tar archive
			for fileName := range test.files {
				filePath := filepath.Join(tempDir, fileName)

				fileInfo, err := os.Stat(filePath)
				if os.IsNotExist(err) {
					assert.True(t, test.expectedError, "expected an error but didn't get one")
					return
				} else if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				err = writeTarArchive(tarWriter, tempDir, filePath, fileInfo)
				assert.Equal(t, test.expectedError, err != nil)
			}

			// Verify archive content if no error was expected
			if !test.expectedError {
				tarReader := tar.NewReader(buffer)
				extractedFiles := map[string]string{}

				for {
					header, err := tarReader.Next()
					if errors.Is(err, io.EOF) {
						break
					}

					require.NoError(t, err, "failed to read tar archive")
					content, err := io.ReadAll(tarReader)
					require.NoError(t, err, "failed to read file content from tar archive")

					extractedFiles[header.Name] = string(content)
				}

				assert.Equal(t, test.expectedFiles, extractedFiles)
			}
		})
	}
}

func TestCreateArchive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		files         map[string]string
		expectedError bool
	}{
		{
			name: "success",
			files: map[string]string{
				"file1.txt": "content1",
				"file2.txt": "content2",
			},
			expectedError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			tempDir := t.TempDir()

			if test.files != nil {
				for fileName, content := range test.files {
					filePath := filepath.Join(tempDir, fileName)

					err := os.WriteFile(filePath, []byte(content), 0o644)
					if err != nil {
						t.Fatalf("unable to create test file: %v", err)
					}
				}
			}

			archiveFilePath := filepath.Join(tempDir, "test.tar.gz")
			err := createArchive(tempDir, archiveFilePath)
			assert.Equal(t, test.expectedError, err != nil)

			if !test.expectedError {
				_, err := os.Stat(archiveFilePath)
				assert.NoError(t, err, "expected archive file to exist")
			}
		})
	}
}

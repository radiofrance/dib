//nolint:testpackage
package buildkit

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/radiofrance/dib/pkg/mock"
	"github.com/radiofrance/dib/pkg/testutil"
)

func TestBuildctlBaseArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		buildkitHost string
		expectedArgs []string
	}{
		{
			name:         "valid host address",
			buildkitHost: "tcp://127.0.0.1:1234",
			expectedArgs: []string{"--addr=tcp://127.0.0.1:1234"},
		},
		{
			name:         "empty host address",
			buildkitHost: "",
			expectedArgs: []string{"--addr="},
		},
		{
			name:         "unix socket address",
			buildkitHost: "unix:///var/run/buildkit/buildkitd.sock",
			expectedArgs: []string{"--addr=unix:///var/run/buildkit/buildkitd.sock"},
		},
		{
			name:         "weird formatting in address",
			buildkitHost: "   tcp://host:1234   ",
			expectedArgs: []string{"--addr=   tcp://host:1234   "},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			args := buildctlBaseArgs(tt.buildkitHost)
			assert.Equal(t, tt.expectedArgs, args)
		})
	}
}

func TestBuildKitFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		missingDir    bool
		missingFile   bool
		expectedError bool
	}{
		{
			name:          "valid directory and file",
			missingDir:    false,
			missingFile:   false,
			expectedError: false,
		},
		{
			name:          "file missing",
			missingFile:   true,
			missingDir:    false,
			expectedError: true,
		},
		{
			name:          "invalid directory",
			missingFile:   false,
			missingDir:    true,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				dir  string
				file string
			)
			if !tt.missingDir {
				dir = t.TempDir()
			} else {
				dir = "missing-dir"
			}

			if !tt.missingFile && !tt.missingDir {
				file = "Dockerfile"
				dockerfile := fmt.Sprintf(`FROM %s
	CMD ["echo", "dib-build-test-string"]`, testutil.CommonImage)
				err := os.WriteFile(filepath.Join(dir, file), []byte(dockerfile), 0o600)
				require.NoError(t, err)
			}

			absDir, bkFile, err := buildKitFile(dir, file)
			if tt.expectedError {
				assert.Empty(t, absDir)
				assert.Empty(t, bkFile)
				assert.Error(t, err)
			} else {
				assert.Equal(t, dir, absDir)
				assert.Equal(t, defaultDockerfileName, file)
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetBuildkitWorkerType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		buildctlBinary string
		buildkitHost   string
		mockOutput     string
		mockError      error
		expectedType   string
		expectedError  bool
		errorContains  string
	}{
		{
			name:           "success with oci executor",
			buildctlBinary: "buildctl",
			buildkitHost:   "tcp://127.0.0.1:1234",
			mockOutput:     `[{"labels":{"org.mobyproject.buildkit.worker.executor":"oci"}}]`,
			mockError:      nil,
			expectedType:   OciExecutorType,
			expectedError:  false,
		},
		{
			name:           "success with containerd executor",
			buildctlBinary: "buildctl",
			buildkitHost:   "tcp://127.0.0.1:1234",
			mockOutput:     `[{"labels":{"org.mobyproject.buildkit.worker.executor":"containerd"}}]`,
			mockError:      nil,
			expectedType:   ContainerdExecutorType,
			expectedError:  false,
		},
		{
			name:           "execution error",
			buildctlBinary: "buildctl",
			buildkitHost:   "tcp://127.0.0.1:1234",
			mockOutput:     "",
			mockError:      fmt.Errorf("execution failed"),
			expectedType:   "",
			expectedError:  true,
			errorContains:  "execution failed",
		},
		{
			name:           "invalid JSON output",
			buildctlBinary: "buildctl",
			buildkitHost:   "tcp://127.0.0.1:1234",
			mockOutput:     "invalid json",
			mockError:      nil,
			expectedType:   "",
			expectedError:  true,
			errorContains:  "failed to parse buildkit workers output",
		},
		{
			name:           "no workers found",
			buildctlBinary: "buildctl",
			buildkitHost:   "tcp://127.0.0.1:1234",
			mockOutput:     "[]",
			mockError:      nil,
			expectedType:   "",
			expectedError:  true,
			errorContains:  "no buildkit workers found",
		},
		{
			name:           "missing labels",
			buildctlBinary: "buildctl",
			buildkitHost:   "tcp://127.0.0.1:1234",
			mockOutput:     `[{"no_labels":true}]`,
			mockError:      nil,
			expectedType:   "",
			expectedError:  true,
			errorContains:  "worker labels not found or invalid format",
		},
		{
			name:           "missing executor type",
			buildctlBinary: "buildctl",
			buildkitHost:   "tcp://127.0.0.1:1234",
			mockOutput:     `[{"labels":{}}]`,
			mockError:      nil,
			expectedType:   "",
			expectedError:  true,
			errorContains:  "executor type not found or invalid format",
		},
		{
			name:           "unknown executor type",
			buildctlBinary: "buildctl",
			buildkitHost:   "tcp://127.0.0.1:1234",
			mockOutput:     `[{"labels":{"org.mobyproject.buildkit.worker.executor":"unknown"}}]`,
			mockError:      nil,
			expectedType:   "",
			expectedError:  true,
			errorContains:  "unknown buildkit worker type: unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a mock shell executor with the expected output and error
			mockExecutor := mock.NewShellExecutor([]mock.ExecutorResult{
				{
					Output: tt.mockOutput,
					Error:  tt.mockError,
				},
			})

			// Call the function under test
			workerType, err := GetBuildkitWorkerType(tt.buildctlBinary, tt.buildkitHost, mockExecutor)

			// Verify the results
			if tt.expectedError {
				require.Error(t, err)

				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}

				assert.Empty(t, workerType)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedType, workerType)
			}

			// Verify the command was executed correctly
			require.Len(t, mockExecutor.Executed, 1)
			assert.Equal(t, tt.buildctlBinary, mockExecutor.Executed[0].Command)

			expectedArgs := append(buildctlBaseArgs(tt.buildkitHost), "debug", "workers", "--format={{json .}}")
			assert.Equal(t, expectedArgs, mockExecutor.Executed[0].Args)
		})
	}
}

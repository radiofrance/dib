//nolint:testpackage
package buildkit

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/radiofrance/dib/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

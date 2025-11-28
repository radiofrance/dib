//nolint:paralleltest
package cmd_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/containerd/nerdctl/mod/tigron/expect"
	"github.com/containerd/nerdctl/mod/tigron/test"
	"github.com/radiofrance/dib/pkg/testutil/dibtest"
)

func TestIntegBuild(t *testing.T) {
	buildctlCmd := exec.CommandContext(t.Context(), "buildctl", "--version")

	err := buildctlCmd.Run()
	if err != nil {
		t.Skip("Skipping test because buildctl is not available")
	}

	pingCmd := exec.CommandContext(t.Context(), "buildctl", "debug", "workers")

	err = pingCmd.Run()
	if err != nil {
		t.Skip("Skipping test because buildkitd is not running")
	}

	tempDir := t.TempDir()

	absPath, err := filepath.Abs(tempDir)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	tempDir = absPath

	t.Logf("Using absolute path for build: %s", tempDir)

	dockerfileContent := `FROM alpine:latest
LABEL name="test-image"
RUN echo "Hello, World!"
`
	dockerfilePath := filepath.Join(tempDir, "Dockerfile")

	err = os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create Dockerfile: %v", err)
	}

	dibYamlContent := `images:
  test-image:
    dockerfile: Dockerfile
    context: .
`
	dibYamlPath := filepath.Join(tempDir, "dib.yaml")

	err = os.WriteFile(dibYamlPath, []byte(dibYamlContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create dib.yaml: %v", err)
	}

	testCase := dibtest.Setup()
	testCase.Command = test.Command("build",
		"--backend", "buildkit",
		"--local-only",
		"--dry-run",             // Use dry-run to avoid actually pushing images
		"--no-tests",            // Skip tests to simplify the build process
		"--no-retag",            // Skip retagging to simplify the build process
		"--build-path", tempDir) // Use the temporary directory as the build path

	// We expect a successful build with exit code 0
	// The output should contain information about using the buildkit backend
	testCase.Expected = test.Expects(0, nil, expect.Contains("buildkit"))
	testCase.Run(t)
}

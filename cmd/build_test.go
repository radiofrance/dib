package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/containerd/nerdctl/mod/tigron/expect"
	"github.com/containerd/nerdctl/mod/tigron/test"
	"github.com/radiofrance/dib/pkg/testutil/dibtest"
)

func TestBuildWithBuildkitBackendLocalOnly(t *testing.T) {
	buildctlCmd := exec.Command("buildctl", "--version")
	if err := buildctlCmd.Run(); err != nil {
		t.Skip("Skipping test because buildctl is not available")
	}

	pingCmd := exec.Command("buildctl", "debug", "workers")
	if err := pingCmd.Run(); err != nil {
		t.Skip("Skipping test because buildkitd is not running")
	}

	tempDir, err := os.MkdirTemp("", "dib-build-test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

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
	err = os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create Dockerfile: %v", err)
	}

	dibYamlContent := `images:
  test-image:
    dockerfile: Dockerfile
    context: .
`
	dibYamlPath := filepath.Join(tempDir, "dib.yaml")
	err = os.WriteFile(dibYamlPath, []byte(dibYamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create dib.yaml: %v", err)
	}

	testCase := dibtest.Setup()
	testCase.Command = test.Command("build",
		"--backend", "buildkit",
		"--local-only",
		"--dry-run",  // Use dry-run to avoid actually pushing images
		"--no-tests", // Skip tests to simplify the build process
		"--no-retag", // Skip retagging to simplify the build process
		"--build-path", tempDir) // Use the temporary directory as the build path

	// We expect a successful build with exit code 0
	// The output should contain information about using the buildkit backend
	testCase.Expected = test.Expects(0, nil, expect.Contains("buildkit"))
	testCase.Run(t)
}

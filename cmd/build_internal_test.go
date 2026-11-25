package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/radiofrance/dib/pkg/dib"
	"github.com/radiofrance/dib/pkg/exec"
	"github.com/radiofrance/dib/pkg/types"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCommand(t *testing.T) {
	t.Parallel()

	cmd := buildCommand()

	assert.NotNil(t, cmd)
	assert.IsType(t, &cobra.Command{}, cmd)
	assert.Equal(t, "build", cmd.Use)
	assert.Equal(t, "Run oci images builds", cmd.Short)
	assert.Contains(t, cmd.Long, "dib build will compute the graph of images")
	assert.NotNil(t, cmd.RunE)
}

// Not parallel: the test changes PATH and the Kubernetes environment.
func TestNewBuildkitBuilder(t *testing.T) {
	noBuildctlDir := t.TempDir()

	buildctlDir := t.TempDir()
	//nolint:gosec // buildctl must be executable to be found in PATH.
	err := os.WriteFile(filepath.Join(buildctlDir, "buildctl"), []byte("#!/bin/sh\n"), 0o755)
	require.NoError(t, err)

	tests := []struct {
		name      string
		localOnly bool
		path      string
		wantErr   string
	}{
		{
			name:      "local build without buildctl",
			localOnly: true,
			path:      noBuildctlDir,
			wantErr:   "cannot find buildctl binary",
		},
		{
			name:      "local build with buildctl",
			localOnly: true,
			path:      buildctlDir,
		},
		{
			// buildctl runs inside the BuildKit pods, so it must not be looked up locally.
			// Without a Kubernetes cluster, creating the builder fails after that step.
			name:      "kubernetes build without buildctl",
			localOnly: false,
			path:      noBuildctlDir,
			wantErr:   "creating buildkit builder",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("PATH", test.path)
			t.Setenv("KUBECONFIG", filepath.Join(noBuildctlDir, "kubeconfig"))
			t.Setenv("KUBERNETES_SERVICE_HOST", "")

			opts := dib.BuildOpts{LocalOnly: test.localOnly}

			builder, err := newBuildkitBuilder(t.Context(), opts, exec.NewShellExecutor(t.TempDir(), nil))
			if test.wantErr == "" {
				require.NoError(t, err)
				assert.NotNil(t, builder)

				return
			}

			require.ErrorContains(t, err, test.wantErr)
			assert.Nil(t, builder)

			if !test.localOnly {
				assert.NotContains(t, err.Error(), "buildctl", "buildctl must not be looked up")
			}
		})
	}
}

// Not parallel: the test changes PATH and the Kubernetes environment.
func TestDoBuild_KubernetesWithoutBuildctl(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("KUBECONFIG", filepath.Join(t.TempDir(), "kubeconfig"))
	t.Setenv("KUBERNETES_SERVICE_HOST", "")

	// doBuild looks for Dockerfiles in the working directory: use an empty one.
	previousWorkingDir := workingDir
	workingDir = t.TempDir()

	t.Cleanup(func() { workingDir = previousWorkingDir })

	opts := dib.BuildOpts{
		RegistryURL:  "registry.example.org/dib",
		Backend:      types.BuildKitBackend,
		ForceRebuild: true,
		NoTests:      true,
	}

	// Without a Kubernetes cluster, the build fails when creating the BuildKit builder,
	// but never because buildctl is missing.
	err := doBuild(t.Context(), opts, map[string]string{})
	require.ErrorContains(t, err, "creating buildkit builder")
	assert.NotContains(t, err.Error(), "buildctl")
}

// Not parallel: the test sets environment variables.
func TestParseBuildArgs(t *testing.T) {
	t.Setenv("FROM_ENV", "env-value")
	t.Setenv("REGISTRY", "registry.example.org")

	buildArgs := parseBuildArgs([]string{
		"VERSION=1.0.0",
		"IMAGE=$REGISTRY/app",
		"FROM_ENV",
		"UNSET_VARIABLE_FOR_TEST",
	})

	assert.Equal(t, map[string]string{
		"VERSION":  "1.0.0",
		"IMAGE":    "registry.example.org/app",
		"FROM_ENV": "env-value",
	}, buildArgs)
}

// Not parallel: the test changes PATH, the Kubernetes environment and the working directory.
func TestBuildAction_Kubernetes(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("KUBECONFIG", filepath.Join(t.TempDir(), "kubeconfig"))
	t.Setenv("KUBERNETES_SERVICE_HOST", "")

	// "dib build" looks for Dockerfiles in the "docker" directory of the working directory: use an empty one.
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "docker"), 0o750))
	t.Chdir(dir)

	cmd := buildCommand()
	cmd.SetArgs([]string{"--force-rebuild", "--no-tests"})
	cmd.SetContext(t.Context())
	cmd.SetOut(&strings.Builder{})
	cmd.SetErr(&strings.Builder{})

	// Without a Kubernetes cluster, the build fails when creating the BuildKit builder.
	err := cmd.Execute()
	require.ErrorContains(t, err, "creating buildkit builder")
}

// Not parallel: the test changes PATH.
func TestBuildAction_UnreachableBuildkitHost(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Chdir(t.TempDir())

	cmd := buildCommand()
	cmd.SetArgs([]string{"--local-only", "--buildkit-host", "unix:///run/missing/buildkitd.sock"})
	cmd.SetContext(t.Context())
	cmd.SetOut(&strings.Builder{})
	cmd.SetErr(&strings.Builder{})

	err := cmd.Execute()
	require.ErrorContains(t, err, "`buildctl` needs to be installed")
}

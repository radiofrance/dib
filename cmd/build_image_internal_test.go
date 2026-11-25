package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/radiofrance/dib/pkg/dib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildImageCommand(t *testing.T) {
	t.Parallel()

	cmd := buildImageCommand()

	assert.Equal(t, "build-image [CONTEXT]", cmd.Use)
	assert.NotNil(t, cmd.RunE)

	require.NoError(t, cmd.ParseFlags([]string{}))

	err := cmd.ValidateRequiredFlags()
	require.ErrorContains(t, err, `required flag(s) "tag" not set`)
}

func TestNewImageBuilderOpts(t *testing.T) {
	t.Parallel()

	contextDir := t.TempDir()
	writeTestFile(t, filepath.Join(contextDir, "Dockerfile"), "FROM scratch\n")
	writeTestFile(t, filepath.Join(contextDir, "docker", "Dockerfile"), "FROM scratch\n")
	writeTestFile(t, filepath.Join(contextDir, "file.txt"), "")

	tests := []struct {
		name         string
		opts         dib.BuildOpts
		buildContext string
		tags         []string
		wantTags     []string
		wantFile     string
		wantErr      string
	}{
		{
			name:         "default Dockerfile",
			buildContext: contextDir,
			tags:         []string{"registry.example.org/team/app:1.0.0"},
			wantFile:     "Dockerfile",
		},
		{
			name:         "Dockerfile in a sub-directory",
			opts:         dib.BuildOpts{File: "docker/Dockerfile"},
			buildContext: contextDir,
			tags:         []string{"registry.example.org/team/app:1.0.0"},
			wantFile:     "docker/Dockerfile",
		},
		{
			name:         "absolute Dockerfile path inside the build context",
			opts:         dib.BuildOpts{File: filepath.Join(contextDir, "docker", "Dockerfile")},
			buildContext: contextDir,
			tags:         []string{"registry.example.org/team/app:1.0.0"},
			wantFile:     "docker/Dockerfile",
		},
		{
			// buildctl reads the Dockerfile from the local filesystem.
			name:         "local build",
			opts:         dib.BuildOpts{File: "docker/Dockerfile", LocalOnly: true},
			buildContext: contextDir,
			tags:         []string{"registry.example.org/team/app:1.0.0"},
			wantFile:     filepath.Join(contextDir, "docker", "Dockerfile"),
		},
		{
			name:         "several tags, one without version",
			buildContext: contextDir,
			tags:         []string{"registry.example.org/team/app:1.0.0", "registry.example.org/team/app"},
			wantTags:     []string{"registry.example.org/team/app:1.0.0", "registry.example.org/team/app:latest"},
			wantFile:     "Dockerfile",
		},
		{
			name:         "no tag",
			buildContext: contextDir,
			wantErr:      "at least one tag is required",
		},
		{
			name:         "invalid tag",
			buildContext: contextDir,
			tags:         []string{"registry.example.org/App:1.0.0"},
			wantErr:      `invalid tag "registry.example.org/App:1.0.0"`,
		},
		{
			name:         "missing build context",
			buildContext: filepath.Join(contextDir, "missing"),
			tags:         []string{"registry.example.org/team/app:1.0.0"},
			wantErr:      "invalid build context",
		},
		{
			name:         "build context is a file",
			buildContext: filepath.Join(contextDir, "file.txt"),
			tags:         []string{"registry.example.org/team/app:1.0.0"},
			wantErr:      "not a directory",
		},
		{
			name:         "Dockerfile outside the build context",
			opts:         dib.BuildOpts{File: "../Dockerfile"},
			buildContext: filepath.Join(contextDir, "docker"),
			tags:         []string{"registry.example.org/team/app:1.0.0"},
			wantErr:      "must be inside the build context",
		},
		{
			name:         "missing Dockerfile",
			opts:         dib.BuildOpts{File: "Dockerfile.missing"},
			buildContext: contextDir,
			tags:         []string{"registry.example.org/team/app:1.0.0"},
			wantErr:      "cannot read Dockerfile",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			opts := test.opts
			opts.BuildArg = []string{"VERSION=1.0.0"}
			opts.Secret = []string{"id=npmrc,env=NPMRC"}
			opts.SSH = []string{"default"}

			imageOpts, err := newImageBuilderOpts(opts, test.buildContext, test.tags)
			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.buildContext, imageOpts.Context)
			assert.Equal(t, test.wantFile, imageOpts.File)

			wantTags := test.wantTags
			if wantTags == nil {
				wantTags = test.tags
			}

			assert.Equal(t, wantTags, imageOpts.Tags)
			assert.Equal(t, opts.LocalOnly, imageOpts.LocalOnly)
			assert.Equal(t, map[string]string{"VERSION": "1.0.0"}, imageOpts.BuildArgs)
			assert.Equal(t, []string{"id=npmrc,env=NPMRC"}, imageOpts.Secrets)
			assert.Equal(t, []string{"default"}, imageOpts.SSH)
			assert.Equal(t, "app", imageOpts.Labels["org.opencontainers.image.title"])
		})
	}
}

// Not parallel: the test changes PATH.
func TestBuildImageAction_Local(t *testing.T) {
	// A fake buildctl that records its arguments.
	argsFile := filepath.Join(t.TempDir(), "args")
	binDir := t.TempDir()
	//nolint:gosec // buildctl must be executable to be found in PATH.
	err := os.WriteFile(filepath.Join(binDir, "buildctl"),
		[]byte("#!/bin/sh\nfor arg in \"$@\"; do echo \"$arg\"; done >> \""+argsFile+"\"\n"), 0o755)
	require.NoError(t, err)
	t.Setenv("PATH", binDir)

	contextDir := t.TempDir()
	writeTestFile(t, filepath.Join(contextDir, "docker", "Dockerfile"), "FROM scratch AS final\n")

	// The build context is given relative to the working directory.
	t.Chdir(filepath.Dir(contextDir))

	cmd := buildImageCommand()
	cmd.SetArgs([]string{
		"--local-only", "--buildkit-host", "unix:///run/buildkit.sock",
		"--tag", "registry.example.org/team/app:0123456789-main",
		"--file", "docker/Dockerfile", "--target", "final", "--build-arg", "VERSION=1.0.0",
		"--secret", "id=npmrc,env=NPMRC", "--ssh", "default",
		filepath.Base(contextDir),
	})
	cmd.SetContext(t.Context())
	cmd.SetOut(&strings.Builder{})
	cmd.SetErr(&strings.Builder{})

	err = cmd.Execute()
	require.NoError(t, err)

	content, err := os.ReadFile(argsFile) //nolint:gosec // The file is created by this test.
	require.NoError(t, err)

	args := strings.Split(strings.TrimSpace(string(content)), "\n")
	// The first call pings the BuildKit host, the second one builds the image.
	assert.Equal(t, []string{"--addr=unix:///run/buildkit.sock", "debug", "workers"}, args[:3])
	assert.Subset(t, args[3:], []string{
		"--addr=unix:///run/buildkit.sock",
		"build",
		"--local=context=" + contextDir,
		"--output=type=image,unpack=true,name=registry.example.org/team/app:0123456789-main",
		"--local=dockerfile=" + filepath.Join(contextDir, "docker"),
		"--opt=filename=Dockerfile",
		"--opt=target=final",
		"--opt=build-arg:VERSION=1.0.0",
		"--opt=label:org.opencontainers.image.title=app",
		"--secret=id=npmrc,env=NPMRC",
		"--ssh=default",
	})
}

// Not parallel: the test changes PATH and the Kubernetes environment.
func TestBuildImageAction_KubernetesWithoutBuildctl(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("KUBECONFIG", filepath.Join(t.TempDir(), "kubeconfig"))
	t.Setenv("KUBERNETES_SERVICE_HOST", "")

	contextDir := t.TempDir()
	writeTestFile(t, filepath.Join(contextDir, "Dockerfile"), "FROM scratch\n")

	cmd := buildImageCommand()
	cmd.SetArgs([]string{"--tag", "registry.example.org/team/app:1.0.0", contextDir})
	cmd.SetContext(t.Context())
	cmd.SetOut(&strings.Builder{})
	cmd.SetErr(&strings.Builder{})

	// Without a Kubernetes cluster, the build fails when creating the BuildKit builder,
	// but never because buildctl is missing.
	err := cmd.Execute()
	require.ErrorContains(t, err, "creating buildkit builder")
	assert.NotContains(t, err.Error(), "buildctl")
}

func writeTestFile(t *testing.T, filePath, content string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0o750))
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0o600))
}

// Not parallel: the test changes PATH.
func TestBuildImageAction_MissingBuildContext(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Chdir(t.TempDir())

	cmd := buildImageCommand()
	cmd.SetArgs([]string{"--tag", "registry.example.org/app:1.0.0", "missing"})
	cmd.SetContext(t.Context())
	cmd.SetOut(&strings.Builder{})
	cmd.SetErr(&strings.Builder{})

	err := cmd.Execute()
	require.ErrorContains(t, err, "invalid build context")
}

// Not parallel: the test changes PATH.
func TestBuildImageAction_UnreachableBuildkitHost(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	contextDir := t.TempDir()
	writeTestFile(t, filepath.Join(contextDir, "Dockerfile"), "FROM scratch\n")
	t.Chdir(contextDir)

	cmd := buildImageCommand()
	cmd.SetArgs([]string{
		"--local-only", "--buildkit-host", "unix:///run/missing/buildkitd.sock",
		"--tag", "registry.example.org/app:1.0.0",
	})
	cmd.SetContext(t.Context())
	cmd.SetOut(&strings.Builder{})
	cmd.SetErr(&strings.Builder{})

	err := cmd.Execute()
	require.ErrorContains(t, err, "`buildctl` needs to be installed")
}

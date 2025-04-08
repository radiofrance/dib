//nolint:testpackage
package buildkit

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/radiofrance/dib/pkg/testutil"

	"github.com/radiofrance/dib/pkg/mock"
	"github.com/radiofrance/dib/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func provideDefaultOptions(t *testing.T) types.ImageBuilderOpts {
	t.Helper()
	// generate Dockerfile
	buildCtx := t.TempDir()
	dockerfile := fmt.Sprintf(`FROM %s
	CMD ["echo", "dib-build-test-string"]`, testutil.CommonImage)

	err := os.WriteFile(filepath.Join(buildCtx, defaultDockerfileName), []byte(dockerfile), 0o600)
	require.NoError(t, err)

	return types.ImageBuilderOpts{
		BuildkitHost: getBuildkitHostAddress(),
		Context:      buildCtx,
		Tags: []string{
			"gcr.io/project-id/image:version",
			"gcr.io/project-id/image:latest",
		},
		BuildArgs: map[string]string{
			"someArg": "someValue",
		},
		Labels: map[string]string{
			"someLabel": "someValue",
		},
		Push:      true,
		LogOutput: &bytes.Buffer{},
		Progress:  "auto",
	}
}

func Test_Build(t *testing.T) {
	t.Parallel()

	const buildctlBinary = "buildctl"
	testCases := []struct {
		name                  string
		modifyOpts            func(opts *types.ImageBuilderOpts)
		expectedBuildArgsFunc func(context string) []string
		expectedError         error
	}{
		{
			name:       "Executes",
			modifyOpts: func(opts *types.ImageBuilderOpts) {},
			expectedBuildArgsFunc: func(context string) []string {
				return []string{
					fmt.Sprintf("--addr=%s", fmt.Sprintf("unix://%s/buildkit/buildkitd.sock", runtimeVariableDataDir)),
					"build",
					"--progress=auto",
					"--frontend=dockerfile.v0",
					fmt.Sprintf("--local=context=%s", context),
					"--output=type=image,name=gcr.io/project-id/image:version,name=gcr.io/project-id/image:latest,push=true",
					fmt.Sprintf("--local=dockerfile=%s", context),
					"--opt=filename=Dockerfile",
					"--opt=build-arg:someArg=someValue",
					"--opt=label=someValue",
				}
			},
			expectedError: nil,
		},
		{
			name: "ExecutesDisablePush",
			modifyOpts: func(opts *types.ImageBuilderOpts) {
				opts.Push = false
			},
			expectedBuildArgsFunc: func(context string) []string {
				return []string{
					fmt.Sprintf("--addr=%s", fmt.Sprintf("unix://%s/buildkit/buildkitd.sock", runtimeVariableDataDir)),
					"build",
					"--progress=auto",
					"--frontend=dockerfile.v0",
					fmt.Sprintf("--local=context=%s", context),
					"--output=type=image,name=gcr.io/project-id/image:version,name=gcr.io/project-id/image:latest",
					fmt.Sprintf("--local=dockerfile=%s", context),
					"--opt=filename=Dockerfile",
					"--opt=build-arg:someArg=someValue",
					"--opt=label=someValue",
				}
			},
			expectedError: nil,
		},
		{
			name: "ExecutesWithoutTags",
			modifyOpts: func(opts *types.ImageBuilderOpts) {
				opts.Tags = nil
			},
			expectedBuildArgsFunc: func(context string) []string {
				return []string{
					fmt.Sprintf("--addr=%s", fmt.Sprintf("unix://%s/buildkit/buildkitd.sock", runtimeVariableDataDir)),
					"build",
					"--progress=auto",
					"--frontend=dockerfile.v0",
					fmt.Sprintf("--local=context=%s", context),
					"--output=type=image,dangling-name-prefix=<none>,push=true",
					fmt.Sprintf("--local=dockerfile=%s", context),
					"--opt=filename=Dockerfile",
					"--opt=build-arg:someArg=someValue",
					"--opt=label=someValue",
				}
			},
			expectedError: nil,
		},
		{
			name: "ExecutesWithFile",
			modifyOpts: func(opts *types.ImageBuilderOpts) {
				opts.File = filepath.Join(opts.Context, defaultDockerfileName)
			},
			expectedBuildArgsFunc: func(context string) []string {
				return []string{
					fmt.Sprintf("--addr=%s", fmt.Sprintf("unix://%s/buildkit/buildkitd.sock", runtimeVariableDataDir)),
					"build",
					"--progress=auto",
					"--frontend=dockerfile.v0",
					fmt.Sprintf("--local=context=%s", context),
					"--output=type=image,name=gcr.io/project-id/image:version,name=gcr.io/project-id/image:latest,push=true",
					fmt.Sprintf("--local=dockerfile=%s", context),
					"--opt=filename=Dockerfile",
					"--opt=build-arg:someArg=someValue",
					"--opt=label=someValue",
				}
			},
			expectedError: nil,
		},
		{
			name: "ExecutesWithTarget",
			modifyOpts: func(opts *types.ImageBuilderOpts) {
				opts.Target = "prod"
			},
			expectedBuildArgsFunc: func(context string) []string {
				return []string{
					fmt.Sprintf("--addr=%s", fmt.Sprintf("unix://%s/buildkit/buildkitd.sock", runtimeVariableDataDir)),
					"build",
					"--progress=auto",
					"--frontend=dockerfile.v0",
					fmt.Sprintf("--local=context=%s", context),
					"--output=type=image,name=gcr.io/project-id/image:version,name=gcr.io/project-id/image:latest,push=true",
					fmt.Sprintf("--local=dockerfile=%s", context),
					"--opt=filename=Dockerfile",
					"--opt=build-arg:someArg=someValue",
					"--opt=label=someValue",
					fmt.Sprintf("--opt=target=%s", "prod"),
				}
			},
			expectedError: nil,
		},
		{
			name:                  "FailsOnExecutorError",
			modifyOpts:            func(opts *types.ImageBuilderOpts) {},
			expectedBuildArgsFunc: nil,
			expectedError:         errors.New("something wrong happened"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fakeExecutor := mock.NewExecutor(nil)
			if tc.name == "FailsOnExecutorError" {
				fakeExecutor = mock.NewExecutor([]mock.ExecutorResult{
					{
						Output: "",
						Error:  errors.New("something wrong happened"),
					},
				})
			}
			builder, err := NewBKBuilder(fakeExecutor, buildctlBinary)
			require.NoError(t, err)

			opts := provideDefaultOptions(t)
			tc.modifyOpts(&opts)

			err = builder.Build(opts)
			if tc.expectedError != nil {
				require.EqualError(t, err, tc.expectedError.Error())
			} else {
				require.NoError(t, err)
				assert.Len(t, fakeExecutor.Executed, 1)
				assert.Equal(t, buildctlBinary, fakeExecutor.Executed[0].Command)
				expectedBuildArgs := tc.expectedBuildArgsFunc(opts.Context)
				assert.ElementsMatch(t, expectedBuildArgs, fakeExecutor.Executed[0].Args)
			}
		})
	}
}

func Test_Build_FailsOnExecutorError(t *testing.T) {
	t.Parallel()

	fakeExecutor := mock.NewExecutor([]mock.ExecutorResult{
		{
			Output: "",
			Error:  errors.New("something wrong happened"),
		},
	})
	builder, err := NewBKBuilder(fakeExecutor, "")
	require.NoError(t, err)

	err = builder.Build(provideDefaultOptions(t))
	require.EqualError(t, err, "something wrong happened")
}

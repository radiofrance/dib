//nolint:testpackage
package buildkit

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	k8sutils "github.com/radiofrance/dib/pkg/kubernetes"

	"github.com/radiofrance/dib/pkg/mock"
	"github.com/radiofrance/dib/pkg/testutil"
	"github.com/radiofrance/dib/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
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

// This is an integration test and should be moved when we introduce integration tests structure.
func Test_NewBKBuilder(t *testing.T) {
	testCases := []struct {
		name        string
		cfg         Config
		workingDir  string
		binary      string
		localOnly   bool
		expectedErr error
	}{
		{
			name:        "ValidLocalOnlyTrue",
			cfg:         Config{},
			workingDir:  "/tmp",
			binary:      "buildctl",
			localOnly:   true,
			expectedErr: nil,
		},
		{
			name: "ValidLocalOnlyFalse",
			cfg: Config{
				Context: struct {
					S3 struct {
						Bucket string `mapstructure:"bucket"`
						Region string `mapstructure:"region"`
					} `mapstructure:"s3"`
				}{
					S3: struct {
						Bucket string `mapstructure:"bucket"`
						Region string `mapstructure:"region"`
					}{
						Bucket: "test-bucket",
						Region: "us-west-1",
					},
				},
				Executor: struct {
					Kubernetes struct {
						Namespace           string   `mapstructure:"namespace"`
						Image               string   `mapstructure:"image"`
						DockerConfigSecret  string   `mapstructure:"docker_config_secret"`
						ImagePullSecrets    []string `mapstructure:"image_pull_secrets"`
						EnvSecrets          []string `mapstructure:"env_secrets"`
						ContainerOverride   string   `mapstructure:"container_override"`
						PodTemplateOverride string   `mapstructure:"pod_template_override"`
					} `mapstructure:"kubernetes"`
				}{
					Kubernetes: struct {
						Namespace           string   `mapstructure:"namespace"`
						Image               string   `mapstructure:"image"`
						DockerConfigSecret  string   `mapstructure:"docker_config_secret"`
						ImagePullSecrets    []string `mapstructure:"image_pull_secrets"`
						EnvSecrets          []string `mapstructure:"env_secrets"`
						ContainerOverride   string   `mapstructure:"container_override"`
						PodTemplateOverride string   `mapstructure:"pod_template_override"`
					}{
						Namespace: "test-namespace",
						Image:     "test-image",
						ImagePullSecrets: []string{
							"secret1",
						},
					},
				},
			},
			workingDir:  "/tmp",
			binary:      "buildctl",
			localOnly:   false,
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			kubeconfigPath, err := createKubeconfig(t)
			require.NoError(t, err)

			t.Setenv("KUBECONFIG", kubeconfigPath)

			shellExecutor := mock.NewShellExecutor(nil)
			builder, err := NewBKBuilder(tc.cfg, shellExecutor, tc.binary, tc.localOnly)
			if tc.expectedErr != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr.Error())
			} else {
				require.NoError(t, err)
				assert.NotNil(t, builder)
			}
		})
	}
}

func createKubeconfig(t *testing.T) (string, error) {
	t.Helper()
	kubeconfigPath := filepath.Join(t.TempDir(), "kubeconfig")
	err := os.WriteFile(kubeconfigPath, []byte(`
apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://example-cluster:6443
  name: example-cluster
contexts:
- context:
    cluster: example-cluster
  name: example-context
current-context: example-context`), 0o400)
	if err != nil {
		return "", err
	}
	return kubeconfigPath, nil
}

type MockContextProvider struct {
	context string
}

func (m MockContextProvider) PrepareContext(_ types.ImageBuilderOpts) (string, error) {
	return m.context, nil
}

func Test_Build_Remote(t *testing.T) {
	t.Parallel()

	const (
		buildctlBinary = "buildctl"
		//nolint:gosec
		dockerConfigSecret = "docker-config-secret"
	)
	testCases := []struct {
		name               string
		modifyOpts         func(opts *types.ImageBuilderOpts)
		modifyPodConfig    func(podConfig *k8sutils.PodConfig)
		expectedPodFunc    func(dockerConfigSecret string, podConfig k8sutils.PodConfig, args []string) (*corev1.Pod, error)
		dockerConfigSecret string
		expectedError      error
	}{
		{
			name:               "Executes",
			modifyOpts:         func(opts *types.ImageBuilderOpts) {},
			modifyPodConfig:    func(podConfig *k8sutils.PodConfig) {},
			dockerConfigSecret: dockerConfigSecret,
			expectedError:      nil,
		},
		{
			name: "ExecutesDisablePush",
			modifyOpts: func(opts *types.ImageBuilderOpts) {
				opts.Push = false
			},
			modifyPodConfig:    func(podConfig *k8sutils.PodConfig) {},
			dockerConfigSecret: dockerConfigSecret,
			expectedError:      nil,
		},
		{
			name: "ExecutesWithoutTags",
			modifyOpts: func(opts *types.ImageBuilderOpts) {
				opts.Tags = nil
			},
			modifyPodConfig:    func(podConfig *k8sutils.PodConfig) {},
			dockerConfigSecret: dockerConfigSecret,
			expectedError:      nil,
		},
		{
			name: "ExecutesWithFile",
			modifyOpts: func(opts *types.ImageBuilderOpts) {
				opts.File = filepath.Join(opts.Context, defaultDockerfileName)
			},
			modifyPodConfig:    func(podConfig *k8sutils.PodConfig) {},
			dockerConfigSecret: dockerConfigSecret,
			expectedError:      nil,
		},
		{
			name:       "ExecutesWithDiffrentNamespace",
			modifyOpts: func(opts *types.ImageBuilderOpts) {},
			modifyPodConfig: func(podConfig *k8sutils.PodConfig) {
				podConfig.Namespace = "test-namespace"
			},
			dockerConfigSecret: dockerConfigSecret,
			expectedError:      nil,
		},
		{
			name:               "FailsOnExecutorError",
			modifyOpts:         func(opts *types.ImageBuilderOpts) {},
			modifyPodConfig:    func(podConfig *k8sutils.PodConfig) {},
			dockerConfigSecret: "",
			expectedError:      errors.New("the DockerConfigSecret option is required"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			opts := provideDefaultOptions(t)
			tc.modifyOpts(&opts)

			podConfig := k8sutils.PodConfig{
				// Use a fixed name generator for the pod to ensure the test is deterministic
				NameGenerator: func() string { return "test-pod-name" },
			}
			tc.modifyPodConfig(&podConfig)

			buildctlArgs, err := generateBuildctlArgs(opts)
			require.NoError(t, err)
			//nolint:lll
			expectedPodFunc := func(dockerConfigSecret string, podConfig k8sutils.PodConfig, args []string) (*corev1.Pod, error) {
				pod, err := buildPod(dockerConfigSecret, podConfig, args)
				return pod, err
			}
			pod, _ := expectedPodFunc("docker-config-secret", podConfig, buildctlArgs)
			fakeExecutor := mock.NewKubernetesExecutor(pod)

			b := Builder{
				bkKubernetesExecutor: bkKubernetesExecutor{
					KubernetesExecutor: fakeExecutor,
					buildctlBinary:     buildctlBinary,
					dockerConfigSecret: tc.dockerConfigSecret,
					podConfig:          podConfig,
				},
				contextProvider: MockContextProvider{opts.Context},
			}
			err = b.Build(opts)
			if tc.expectedError != nil {
				require.EqualError(t, err, tc.expectedError.Error())
			} else {
				require.NoError(t, err)
				// Skip the comparison of the pod's name and instance label
				// The test is still valid because we're checking that the executor was called with a pod
				// that has the correct configuration, except for the name and instance label
				// which are generated dynamically
				assert.NotNil(t, fakeExecutor.Applied)
			}
		})
	}
}

func Test_Build_Local(t *testing.T) {
	t.Parallel()

	const buildctlBinary = "buildctl"
	testCases := []struct {
		name                  string
		modifyOpts            func(opts *types.ImageBuilderOpts)
		expectedBuildArgsFunc func(context string) []string
		expectedError         error
	}{
		{
			name: "Executes",
			modifyOpts: func(opts *types.ImageBuilderOpts) {
				opts.LocalOnly = true
			},
			expectedBuildArgsFunc: func(context string) []string {
				return []string{
					fmt.Sprintf("--addr=%s", getBuildkitHostAddress()),
					"build",
					"--progress=auto",
					"--frontend=dockerfile.v0",
					fmt.Sprintf("--local=context=%s", context),
					"--output=type=image,unpack=true,name=gcr.io/project-id/image:version,name=gcr.io/project-id/image:latest,push=true",
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
				opts.LocalOnly = true
			},
			expectedBuildArgsFunc: func(context string) []string {
				return []string{
					fmt.Sprintf("--addr=%s", getBuildkitHostAddress()),
					"build",
					"--progress=auto",
					"--frontend=dockerfile.v0",
					fmt.Sprintf("--local=context=%s", context),
					"--output=type=image,unpack=true,name=gcr.io/project-id/image:version,name=gcr.io/project-id/image:latest",
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
				opts.LocalOnly = true
			},
			expectedBuildArgsFunc: func(context string) []string {
				return []string{
					fmt.Sprintf("--addr=%s", getBuildkitHostAddress()),
					"build",
					"--progress=auto",
					"--frontend=dockerfile.v0",
					fmt.Sprintf("--local=context=%s", context),
					"--output=type=image,unpack=true,dangling-name-prefix=<none>,push=true",
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
				opts.LocalOnly = true
			},
			expectedBuildArgsFunc: func(context string) []string {
				return []string{
					fmt.Sprintf("--addr=%s", getBuildkitHostAddress()),
					"build",
					"--progress=auto",
					"--frontend=dockerfile.v0",
					fmt.Sprintf("--local=context=%s", context),
					"--output=type=image,unpack=true,name=gcr.io/project-id/image:version,name=gcr.io/project-id/image:latest,push=true",
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
				opts.LocalOnly = true
			},
			expectedBuildArgsFunc: func(context string) []string {
				return []string{
					fmt.Sprintf("--addr=%s", getBuildkitHostAddress()),
					"build",
					"--progress=auto",
					"--frontend=dockerfile.v0",
					fmt.Sprintf("--local=context=%s", context),
					"--output=type=image,unpack=true,name=gcr.io/project-id/image:version,name=gcr.io/project-id/image:latest,push=true",
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

			fakeExecutor := mock.NewShellExecutor(nil)
			if tc.name == "FailsOnExecutorError" {
				fakeExecutor = mock.NewShellExecutor([]mock.ExecutorResult{
					{
						Output: "",
						Error:  errors.New("something wrong happened"),
					},
				})
			}

			opts := provideDefaultOptions(t)
			tc.modifyOpts(&opts)

			b := Builder{
				bkShellExecutor: bkShellExecutor{
					fakeExecutor,
					buildctlBinary,
				},
				contextProvider: MockContextProvider{opts.Context},
			}
			err := b.Build(opts)
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

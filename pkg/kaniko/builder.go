package kaniko

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/radiofrance/dib/pkg/executor"
	"github.com/radiofrance/dib/pkg/logger"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/radiofrance/dib/pkg/kubernetes"
	"github.com/radiofrance/dib/pkg/types"
	"github.com/radiofrance/kubecli"
)

// ContextProvider provides a layer of abstraction for different build context sources.
type ContextProvider interface {
	// PrepareContext allows to do some operations on the build context before the executor runs,
	// like moving it to a remote location in order to be accessible by remote executors.
	// It must return a URL compatible with Kaniko's `--context` flag.
	PrepareContext(ctx context.Context, opts types.ImageBuilderOpts) (string, error)
}

// Executor executes the Kaniko build.
type Executor interface {
	// Execute the kaniko build, passing a slice of arguments to the kaniko command.
	Execute(ctx context.Context, output io.Writer, args []string) error
}

// Builder uses Kaniko as build backend.
type Builder struct {
	executor        Executor
	contextProvider ContextProvider
	DryRun          bool // When dry-run mode is enabled, the executor won't be called for real.
}

// Config holds the configuration for the Kaniko build backend.
type Config struct {
	Context struct {
		S3 struct {
			Bucket string `mapstructure:"bucket"`
			Region string `mapstructure:"region"`
		} `mapstructure:"s3"`
	} `mapstructure:"context"`
	Executor struct {
		Docker struct {
			Image string `mapstructure:"image"`
		} `mapstructure:"docker"`
		Kubernetes struct {
			Namespace           string   `mapstructure:"namespace"`
			Image               string   `mapstructure:"image"`
			DockerConfigSecret  string   `mapstructure:"docker_config_secret"`
			ImagePullSecrets    []string `mapstructure:"image_pull_secrets"`
			EnvSecrets          []string `mapstructure:"env_secrets"`
			ContainerOverride   string   `mapstructure:"container_override"`
			PodTemplateOverride string   `mapstructure:"pod_template_override"`
		} `mapstructure:"kubernetes"`
	} `mapstructure:"executor"`
}

// NewBuilder creates a new instance of Builder.
func NewBuilder(exec Executor, contextProvider ContextProvider) *Builder {
	return &Builder{
		executor:        exec,
		contextProvider: contextProvider,
	}
}

// Build the image using the Kaniko backend.
func (b *Builder) Build(ctx context.Context, opts types.ImageBuilderOpts) error {
	contextPath, err := b.contextProvider.PrepareContext(ctx, opts)
	if err != nil {
		return fmt.Errorf("cannot prepare kaniko build context: %w", err)
	}

	// More infos, on Kaniko args here: https://github.com/GoogleContainerTools/kaniko#additional-flags
	kanikoArgs := []string{
		"--context=" + contextPath,
		"--log-format=text",
		"--snapshot-mode=redo",
		"--single-snapshot",
	}

	for _, tag := range opts.Tags {
		kanikoArgs = append(kanikoArgs, fmt.Sprintf("--destination=%s", tag))
	}

	for k, v := range opts.BuildArgs {
		kanikoArgs = append(kanikoArgs, fmt.Sprintf("--build-arg=%s=%s", k, v))
	}

	for k, v := range opts.Labels {
		kanikoArgs = append(kanikoArgs, fmt.Sprintf("--label=%s=%s", k, v))
	}

	if !opts.Push {
		kanikoArgs = append(kanikoArgs, "--no-push")
	}

	if b.DryRun {
		logger.Infof("[DRY-RUN] kaniko %s", strings.Join(kanikoArgs, " "))
		return nil
	}

	return b.executor.Execute(ctx, opts.LogOutput, kanikoArgs)
}

func CreateBuilder(ctx context.Context, cfg Config, shell executor.ShellExecutor,
	workingDir string, localOnly, dryRun bool,
) *Builder {
	var (
		err             error
		executor        Executor
		contextProvider ContextProvider
	)

	if localOnly {
		executor = createKanikoDockerExecutor(shell, workingDir, cfg)
		contextProvider = NewLocalContextProvider()
	} else {
		executor, err = createKanikoKubernetesExecutor(cfg)
		if err != nil {
			logger.Fatalf("cannot create kaniko kubernetes executor: %v", err)
		}

		awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.Context.S3.Region))
		if err != nil {
			logger.Fatalf("cannot load AWS config: %v", err)
		}

		s3 := NewS3Uploader(awsCfg, cfg.Context.S3.Bucket)
		contextProvider = NewRemoteContextProvider(s3)
	}

	kanikoBuilder := NewBuilder(executor, contextProvider)
	kanikoBuilder.DryRun = dryRun

	return kanikoBuilder
}

func createKanikoDockerExecutor(shell executor.ShellExecutor, contextRootDir string, cfg Config) *DockerExecutor {
	dockerCfg := ContainerConfig{
		Image: cfg.Executor.Docker.Image,
		Env:   map[string]string{},
		Volumes: map[string]string{
			contextRootDir: contextRootDir,
		},
	}

	return NewDockerExecutor(shell, dockerCfg)
}

func createKanikoKubernetesExecutor(cfg Config) (*KubernetesExecutor, error) {
	k8sClient, err := kubecli.New("")
	if err != nil {
		return nil, fmt.Errorf("could not get kube client from context: %w", err)
	}

	executor := NewKubernetesExecutor(k8sClient.ClientSet, kubernetes.PodConfig{
		Namespace:     cfg.Executor.Kubernetes.Namespace,
		NameGenerator: kubernetes.UniquePodName("kaniko-dib"),
		Labels: map[string]string{
			"app.kubernetes.io/managed-by": "dib",
		},
		Image:            cfg.Executor.Kubernetes.Image,
		ImagePullSecrets: cfg.Executor.Kubernetes.ImagePullSecrets,
		EnvSecrets:       cfg.Executor.Kubernetes.EnvSecrets,
		Env: map[string]string{
			"AWS_REGION": cfg.Context.S3.Region,
			"container":  "kube", // Fix for https://github.com/GoogleContainerTools/kaniko/issues/1542
		},
		PodOverride:       cfg.Executor.Kubernetes.PodTemplateOverride,
		ContainerOverride: cfg.Executor.Kubernetes.ContainerOverride,
	})
	executor.DockerConfigSecret = cfg.Executor.Kubernetes.DockerConfigSecret

	return executor, nil
}

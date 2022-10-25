package main

import (
	"context"
	"fmt"
	"path"

	k8sutils "github.com/radiofrance/dib/pkg/kubernetes"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/radiofrance/dib/pkg/dib"
	"github.com/radiofrance/dib/pkg/docker"
	"github.com/radiofrance/dib/pkg/exec"
	"github.com/radiofrance/dib/pkg/goss"
	"github.com/radiofrance/dib/pkg/kaniko"
	"github.com/radiofrance/dib/pkg/preflight"
	"github.com/radiofrance/dib/pkg/ratelimit"
	"github.com/radiofrance/dib/pkg/registry"
	"github.com/radiofrance/dib/pkg/types"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	kube "gitlab.com/radiofrance/kubecli"
)

const (
	backendDocker = "docker"
	backendKaniko = "kaniko"
)

type buildOpts struct {
	// Root options
	BuildPath      string `mapstructure:"build_path"`
	RegistryURL    string `mapstructure:"registry_url"`
	PlaceholderTag string `mapstructure:"placeholder_tag"`

	// Build specific options
	DisableGenerateGraph bool         `mapstructure:"no_graph"`
	DisableJunitReports  bool         `mapstructure:"no_junit_reports"`
	DisableRunTests      bool         `mapstructure:"no_tests"`
	DryRun               bool         `mapstructure:"dry_run"`
	ForceRebuild         bool         `mapstructure:"force_rebuild"`
	NoRetag              bool         `mapstructure:"no_retag"`
	LocalOnly            bool         `mapstructure:"local_only"`
	Release              bool         `mapstructure:"release"`
	Backend              string       `mapstructure:"backend"`
	Goss                 gossConfig   `mapstructure:"goss"`
	Kaniko               kanikoConfig `mapstructure:"kaniko"`
	RateLimit            int          `mapstructure:"rate_limit"`
}

// gossConfig holds the configuration for the Goss test runner.
type gossConfig struct {
	Executor struct {
		Kubernetes struct {
			Enabled           bool     `mapstructure:"enabled"`
			Namespace         string   `mapstructure:"namespace"`
			Image             string   `mapstructure:"image"`
			ImagePullSecrets  []string `mapstructure:"image_pull_secrets"`
			ContainerOverride string   `mapstructure:"container_override"`
			PodOverride       string   `mapstructure:"pod_override"`
		} `mapstructure:"kubernetes"`
	} `mapstructure:"executor"`
}

// kanikoConfig holds the configuration for the Kaniko build backend.
type kanikoConfig struct {
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

var supportedBackends = []string{
	backendDocker,
	backendKaniko,
}

// buildCmd represents the build command.
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Run docker images builds",
	Long: `dib build will compute the graph of images, and compare it to the last built state

For each image, if any file part of its docker context has changed, the image will be rebuilt.
Otherwise, dib will create a new tag based on the previous tag`,
	Run: func(cmd *cobra.Command, args []string) {
		bindPFlagsSnakeCase(cmd.Flags())

		opts := buildOpts{}
		hydrateOptsFromViper(&opts)

		if opts.Backend == backendKaniko && opts.LocalOnly {
			logrus.Warnf("Using Backend \"kaniko\" with the --local-only flag is partially supported.")
		}

		if opts.Backend == backendDocker {
			preflight.RunPreflightChecks([]string{"docker"})
		}

		err := doBuild(opts)
		if err != nil {
			logrus.Fatalf("Build failed: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().Bool("dry-run", false,
		"Simulate what would happen without actually doing anything dangerous.")
	buildCmd.Flags().Bool("force-rebuild", false,
		"Forces rebuilding the entire image graph, without regarding if the target version already exists.")
	buildCmd.Flags().Bool("no-retag", false,
		"Disable re-tagging images after build. "+
			"Note that temporary tags with the \"dev-\" prefix may still be pushed to the registry.")
	buildCmd.Flags().Bool("no-graph", false,
		"Disable generation of graph during the build process.")
	buildCmd.Flags().Bool("no-tests", false,
		"Disable execution of tests during the build process.")
	buildCmd.Flags().Bool("no-junit", false,
		"Disable generation of junit reports when running tests")
	buildCmd.Flags().Bool("release", false,
		"Enable release mode to tag all images with extra tags found in the `dib.extra-tags` Dockerfile labels.")
	buildCmd.Flags().Bool("local-only", false,
		"Build docker images locally, do not push on remote registry")
	buildCmd.Flags().StringP("backend", "b", backendDocker,
		fmt.Sprintf("Build Backend used to run image builds. Supported backends: %v", supportedBackends))
	buildCmd.Flags().Int("rate-limit", 1,
		"Concurrent number of builds that can run simultaneously")
}

func doBuild(opts buildOpts) error {
	workingDir, err := getWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	gcrRegistry, err := registry.NewRegistry(opts.RegistryURL, opts.DryRun)
	if err != nil {
		return err
	}

	var testRunners []types.TestRunner
	if !opts.DisableRunTests {
		gossRunner, err := createGossTestRunner(opts, workingDir)
		if err != nil {
			return fmt.Errorf("cannot create goss test runner: %w", err)
		}
		testRunners = []types.TestRunner{
			gossRunner,
		}
	}

	shell := &exec.ShellExecutor{
		Dir: workingDir,
	}
	dockerBuilderTagger := docker.NewImageBuilderTagger(shell, opts.DryRun)

	var builder types.ImageBuilder
	switch opts.Backend {
	case backendDocker:
		builder = dockerBuilderTagger
	case backendKaniko:
		builder = createKanikoBuilder(opts, shell, workingDir)
	default:
		logrus.Fatalf("Invalid backend \"%s\": not supported", opts.Backend)
	}

	var tagger types.ImageTagger
	if opts.LocalOnly {
		tagger = dockerBuilderTagger
	} else {
		tagger = gcrRegistry
	}

	logrus.Infof("Building images in directory \"%s\"", path.Join(workingDir, opts.BuildPath))

	logrus.Debug("Generate DAG")
	DAG := dib.GenerateDAG(path.Join(workingDir, opts.BuildPath), opts.RegistryURL)
	logrus.Debug("Generate DAG -- Done")

	err = dib.Plan(DAG, gcrRegistry, opts.ForceRebuild, !opts.DisableRunTests)
	if err != nil {
		return err
	}

	rateLimiter := ratelimit.NewChannelRateLimiter(opts.RateLimit)
	if err := dib.Rebuild(DAG, builder, testRunners, rateLimiter, opts.PlaceholderTag, opts.LocalOnly); err != nil {
		return err
	}

	if !opts.NoRetag {
		err = dib.Retag(DAG, tagger, opts.PlaceholderTag, opts.Release)
		if err != nil {
			return err
		}
	}

	logrus.Info("Build process completed")
	return nil
}

func createKanikoBuilder(opts buildOpts, shell exec.Executor, workingDir string) *kaniko.Builder {
	var (
		err             error
		executor        kaniko.Executor
		contextProvider kaniko.ContextProvider
	)

	if opts.LocalOnly {
		executor = createKanikoDockerExecutor(shell, workingDir, opts.Kaniko)
		contextProvider = kaniko.NewLocalContextProvider()
	} else {
		executor, err = createKanikoKubernetesExecutor(opts.Kaniko)
		if err != nil {
			logrus.Fatalf("cannot create kaniko kubernetes executor: %v", err)
		}

		awsCfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(opts.Kaniko.Context.S3.Region))
		if err != nil {
			logrus.Fatalf("cannot load AWS config: %v", err)
		}
		s3 := kaniko.NewS3Uploader(awsCfg, opts.Kaniko.Context.S3.Bucket)
		contextProvider = kaniko.NewRemoteContextProvider(s3)
	}

	kanikoBuilder := kaniko.NewBuilder(executor, contextProvider)
	kanikoBuilder.DryRun = opts.DryRun

	return kanikoBuilder
}

func createKanikoDockerExecutor(shell exec.Executor, contextRootDir string, cfg kanikoConfig) *kaniko.DockerExecutor {
	dockerCfg := kaniko.ContainerConfig{
		Image: cfg.Executor.Docker.Image,
		Env:   map[string]string{},
		Volumes: map[string]string{
			contextRootDir: contextRootDir,
		},
	}

	return kaniko.NewDockerExecutor(shell, dockerCfg)
}

func createKanikoKubernetesExecutor(cfg kanikoConfig) (*kaniko.KubernetesExecutor, error) {
	k8sClient, err := kube.New("")
	if err != nil {
		return nil, fmt.Errorf("could not get kube client from context: %w", err)
	}

	executor := kaniko.NewKubernetesExecutor(k8sClient.ClientSet, k8sutils.PodConfig{
		Namespace:     cfg.Executor.Kubernetes.Namespace,
		NameGenerator: k8sutils.UniquePodName("kaniko-dib"),
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

func createGossTestRunner(opts buildOpts, workingDir string) (*goss.TestRunner, error) {
	runnerOpts := goss.TestRunnerOptions{
		WorkingDirectory: workingDir,
		JUnitReports:     !opts.DisableJunitReports,
	}

	if opts.Goss.Executor.Kubernetes.Enabled && !opts.LocalOnly {
		executor, err := createGossKubernetesExecutor(opts.Goss)
		if err != nil {
			return nil, err
		}
		return goss.NewTestRunner(executor, runnerOpts), nil
	}

	return goss.NewTestRunner(goss.NewDGossExecutor(), runnerOpts), nil
}

func createGossKubernetesExecutor(cfg gossConfig) (*goss.KubernetesExecutor, error) {
	k8sClient, err := kube.New("")
	if err != nil {
		return nil, fmt.Errorf("could not get kube client from context: %w", err)
	}
	executor := goss.NewKubernetesExecutor(*k8sClient.Config, k8sClient.ClientSet, k8sutils.PodConfig{
		NameGenerator:     k8sutils.UniquePodName("goss"),
		Namespace:         cfg.Executor.Kubernetes.Namespace,
		Image:             cfg.Executor.Kubernetes.Image,
		ImagePullSecrets:  cfg.Executor.Kubernetes.ImagePullSecrets,
		PodOverride:       cfg.Executor.Kubernetes.PodOverride,
		ContainerOverride: cfg.Executor.Kubernetes.ContainerOverride,
	})

	return executor, nil
}

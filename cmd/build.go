package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/radiofrance/dib/ratelimit"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/radiofrance/dib/dag"
	"github.com/radiofrance/dib/dgoss"
	"github.com/radiofrance/dib/docker"
	"github.com/radiofrance/dib/exec"
	"github.com/radiofrance/dib/graphviz"
	"github.com/radiofrance/dib/kaniko"
	"github.com/radiofrance/dib/preflight"
	"github.com/radiofrance/dib/registry"
	"github.com/radiofrance/dib/types"
	versn "github.com/radiofrance/dib/version"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	kube "gitlab.com/radiofrance/kubecli"
)

const (
	backendDocker = "docker"
	backendKaniko = "kaniko"

	placeholderNonExistent = "non-existent"
	junitReportsDirectory  = "dist/testresults/goss"
)

type buildOpts struct {
	// Root options
	BuildPath        string `mapstructure:"build_path"`
	RegistryURL      string `mapstructure:"registry_url"`
	ReferentialImage string `mapstructure:"referential_image"`

	// Build specific options
	DisableGenerateGraph bool         `mapstructure:"no_graph"`
	DisableJunitReports  bool         `mapstructure:"no_junit_reports"`
	DisableRunTests      bool         `mapstructure:"no_tests"`
	DryRun               bool         `mapstructure:"dry_run"`
	ForceRebuild         bool         `mapstructure:"force_rebuild"`
	LocalOnly            bool         `mapstructure:"local_only"`
	RetagLatest          bool         `mapstructure:"retag_latest"`
	Backend              string       `mapstructure:"backend"`
	Kaniko               kanikoConfig `mapstructure:"kaniko"`
	RateLimit            int          `mapstructure:"rate_limit"`
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
		opts := buildOpts{}
		hydrateOptsFromViper(&opts)

		if opts.Backend == backendKaniko && opts.LocalOnly {
			logrus.Warnf("Using Backend \"kaniko\" with the --local-only flag is partially supported.")
		}

		if opts.Backend == backendDocker {
			preflight.RunPreflightChecks([]string{"docker"})
		}

		DAG, err := doBuild(opts)
		if err != nil {
			logrus.Fatalf("Build failed: %v", err)
		}

		if !opts.DisableGenerateGraph {
			if err := graphviz.GenerateGraph(DAG); err != nil {
				logrus.Fatalf("Generating graph failed: %v", err)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().Bool("dry-run", false, "Simulate what would happen without actually doing anything dangerous.")
	buildCmd.Flags().Bool("force-rebuild", false, "Forces rebuilding the entire image graph, without regarding if the target version already exists.") //nolint:lll
	buildCmd.Flags().Bool("no-graph", false, "Disable generation of graph during the build process.")                                                  //nolint:lll
	buildCmd.Flags().Bool("no-tests", false, "Disable execution of tests during the build process.")                                                   //nolint:lll
	buildCmd.Flags().Bool("no-junit", false, "Disable generation of junit reports when running tests")
	buildCmd.Flags().Bool("retag-latest", false, "Should images be retagged with the 'latest' tag for this build") //nolint:lll
	buildCmd.Flags().Bool("local-only", false, "Build docker images locally, do not push on remote registry")
	buildCmd.Flags().StringP("backend", "b", backendDocker, fmt.Sprintf("Build Backend used to run image builds. Supported backends: %v", supportedBackends)) //nolint:lll
	buildCmd.Flags().Int("rate-limit", 1, "Concurrent number of build that can run simultaneously")                                                           //nolint:lll

	bindPFlagsSnakeCase(buildCmd.Flags())
}

func doBuild(opts buildOpts) (*dag.DAG, error) {
	workingDir, err := getWorkingDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get current working directory: %w", err)
	}
	dockerDir, err := findDockerRootDir(workingDir, opts.BuildPath)
	if err != nil {
		return nil, err
	}

	gcrRegistry, err := registry.NewRegistry(opts.RegistryURL, opts.DryRun)
	if err != nil {
		return nil, err
	}

	DAG := &dag.DAG{
		Registry: gcrRegistry,
		TestRunners: []types.TestRunner{
			dgoss.NewTestRunner(dgoss.TestRunnerOptions{
				ReportsDirectory: junitReportsDirectory,
				WorkingDirectory: workingDir,
				JUnitReports:     !opts.DisableJunitReports,
			}),
		},
		RateLimiter: ratelimit.NewChannelRateLimiter(opts.RateLimit),
	}

	shell := &exec.ShellExecutor{
		Dir: workingDir,
	}
	dockerBuilderTagger := docker.NewImageBuilderTagger(shell, opts.DryRun)

	switch opts.Backend {
	case backendDocker:
		DAG.Builder = dockerBuilderTagger
	case backendKaniko:
		DAG.Builder = createKanikoBuilder(opts, shell, workingDir, dockerDir)
	default:
		logrus.Fatalf("Invalid backend \"%s\": not supported", opts.Backend)
	}

	if opts.LocalOnly {
		DAG.Tagger = dockerBuilderTagger
	} else {
		DAG.Tagger = gcrRegistry
	}

	logrus.Infof("Building images in directory \"%s\"", path.Join(workingDir, opts.BuildPath))

	logrus.Debug("Generate DAG")
	DAG.GenerateDAG(workingDir, opts.BuildPath, opts.RegistryURL)
	logrus.Debug("Generate DAG -- Done")

	currentVersion, err := versn.CheckDockerVersionIntegrity(path.Join(workingDir, dockerDir))
	if err != nil {
		return nil, fmt.Errorf("cannot find current version: %w", err)
	}

	previousVersion, diffs, err := versn.GetDiffSinceLastDockerVersionChange(
		workingDir, shell, gcrRegistry, path.Join(dockerDir, versn.DockerVersionFilename),
		path.Join(opts.RegistryURL, opts.ReferentialImage))
	if err != nil {
		if errors.Is(err, versn.ErrNoPreviousBuild) {
			previousVersion = placeholderNonExistent
		} else {
			return nil, fmt.Errorf("cannot find previous version: %w", err)
		}
	}

	if opts.ForceRebuild {
		logrus.Info("--force-rebuild is set to true, all images will be rebuild regardless of their changes ")
		DAG.TagForRebuild()
	} else {
		DAG.CheckForDiff(diffs)
	}

	err = DAG.Retag(currentVersion, previousVersion)
	if err != nil {
		return nil, err
	}

	if err := DAG.Rebuild(currentVersion, opts.ForceRebuild, opts.DisableRunTests, opts.LocalOnly); err != nil {
		return nil, err
	}

	if opts.RetagLatest {
		logrus.Info("--retag-latest is set to true, latest tag will now use current image versions")
		if err := DAG.RetagLatest(currentVersion); err != nil {
			return nil, err
		}
	}

	if !opts.LocalOnly {
		// We retag the referential image to explicit this commit was build using dib
		if err := DAG.Tagger.Tag(fmt.Sprintf("%s:%s", path.Join(opts.RegistryURL, opts.ReferentialImage), "latest"),
			fmt.Sprintf("%s:%s", path.Join(opts.RegistryURL, opts.ReferentialImage), currentVersion)); err != nil {
			return nil, err
		}
	}

	logrus.Info("Build process completed")
	return DAG, nil
}

// findDockerRootDir iterates over the BuildPath to find the first matching directory containing
// a .docker-version file. We consider this directory as the root docker directory containing all the dockerfiles.
func findDockerRootDir(workingDir, buildPath string) (string, error) {
	searchPath := buildPath
	for {
		if _, err := os.Stat(path.Join(workingDir, searchPath, versn.DockerVersionFilename)); err == nil {
			return searchPath, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}

		dir, _ := path.Split(buildPath)
		if dir == "" {
			return "", fmt.Errorf("searching for docker root dir failed, no directory in %s "+
				"contains a %s file", buildPath, versn.DockerVersionFilename)
		}
		searchPath = dir
	}
}

func createKanikoBuilder(opts buildOpts, shell exec.Executor, workingDir, dockerDir string) *kaniko.Builder {
	var (
		err             error
		executor        kaniko.Executor
		contextProvider kaniko.ContextProvider
	)

	if opts.LocalOnly {
		executor = createDockerExecutor(shell, path.Join(workingDir, dockerDir), opts.Kaniko)
		contextProvider = kaniko.NewLocalContextProvider()
	} else {
		executor, err = createKubernetesExecutor(opts.Kaniko)
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

func createDockerExecutor(shell exec.Executor, contextRootDir string, cfg kanikoConfig) *kaniko.DockerExecutor {
	dockerCfg := kaniko.ContainerConfig{
		Image: cfg.Executor.Docker.Image,
		Env:   map[string]string{},
		Volumes: map[string]string{
			contextRootDir: contextRootDir,
		},
	}

	return kaniko.NewDockerExecutor(shell, dockerCfg)
}

func createKubernetesExecutor(cfg kanikoConfig) (*kaniko.KubernetesExecutor, error) {
	k8sClient, err := kube.New("")
	if err != nil {
		return nil, fmt.Errorf("could not get kube client from context: %w", err)
	}

	executor := kaniko.NewKubernetesExecutor(k8sClient.ClientSet, kaniko.PodConfig{
		Namespace:     cfg.Executor.Kubernetes.Namespace,
		NameGenerator: kaniko.UniquePodName("dib"),
		Labels: map[string]string{
			"app.kubernetes.io/managed-by": "dib",
		},
		Image:            cfg.Executor.Kubernetes.Image,
		ImagePullSecrets: cfg.Executor.Kubernetes.ImagePullSecrets,
		EnvSecrets:       cfg.Executor.Kubernetes.EnvSecrets,
		Env: map[string]string{
			"AWS_REGION": cfg.Context.S3.Region,
		},
		PodOverride:       cfg.Executor.Kubernetes.PodTemplateOverride,
		ContainerOverride: cfg.Executor.Kubernetes.ContainerOverride,
	})
	executor.DockerConfigSecret = cfg.Executor.Kubernetes.DockerConfigSecret

	return executor, nil
}

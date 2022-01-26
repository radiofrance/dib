package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/spf13/viper"

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
		opts := buildOptsFromViper()

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

	buildCmd.Flags().Bool(keyDryRun, false, "Simulate what would happen without actually doing anything dangerous.")
	buildCmd.Flags().Bool(keyForceRebuild, false, "Forces rebuilding the entire image graph, without regarding if the target version already exists.") //nolint:lll
	buildCmd.Flags().Bool(keyDisableGraph, false, "Disable generation of graph during the build process.")                                             //nolint:lll
	buildCmd.Flags().Bool(keyDisableTests, false, "Disable execution of tests during the build process.")                                              //nolint:lll
	buildCmd.Flags().Bool(keyDisableJUnit, false, "Disable generation of junit reports when running tests")
	buildCmd.Flags().Bool(keyRetagLatest, false, "Should images be retagged with the 'latest' tag for this build") //nolint:lll
	buildCmd.Flags().Bool(keyLocalOnly, false, "Build docker images locally, do not push on remote registry")
	buildCmd.Flags().StringP(keyBackend, "b", backendDocker, fmt.Sprintf("Build Backend used to run image builds. Supported backends: %v", supportedBackends)) //nolint:lll

	_ = viper.BindPFlags(buildCmd.Flags())
	_ = viper.BindPFlags(rootCmd.PersistentFlags())
}

func doBuild(opts BuildOpts) (*dag.DAG, error) {
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

func createKanikoBuilder(opts BuildOpts, shell exec.Executor, workingDir, dockerDir string) *kaniko.Builder {
	var (
		err             error
		executor        kaniko.Executor
		contextProvider kaniko.ContextProvider
	)

	if opts.LocalOnly {
		executor = createDockerExecutor(shell, path.Join(workingDir, dockerDir))
		contextProvider = kaniko.NewLocalContextProvider()
	} else {
		executor, err = createKubernetesExecutor()
		if err != nil {
			logrus.Fatalf("cannot create kaniko kubernetes executor: %v", err)
		}

		awsCfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion("eu-west-3"))
		if err != nil {
			logrus.Fatalf("cannot load AWS config: %v", err)
		}
		s3 := kaniko.NewS3Uploader(awsCfg, "rf-kaniko-build-context-preprod")
		contextProvider = kaniko.NewRemoteContextProvider(s3)
	}

	kanikoBuilder := kaniko.NewBuilder(executor, contextProvider)
	kanikoBuilder.DryRun = opts.DryRun

	return kanikoBuilder
}

func createDockerExecutor(shell exec.Executor, contextRootDir string) *kaniko.DockerExecutor {
	dockerCfg := kaniko.ContainerConfig{
		Image: "gcr.io/kaniko-project/executor:v1.7.0",
		Env:   map[string]string{},
		Volumes: map[string]string{
			contextRootDir: contextRootDir,
		},
	}

	return kaniko.NewDockerExecutor(shell, dockerCfg)
}

func createKubernetesExecutor() (*kaniko.KubernetesExecutor, error) {
	k8sClient, err := kube.New("")
	if err != nil {
		return nil, fmt.Errorf("could not get kube client from context: %w", err)
	}

	executor := kaniko.NewKubernetesExecutor(k8sClient.ClientSet, kaniko.JobConfig{
		Namespace: "ppkaniko",
		Name:      kaniko.UniqueJobName("dib"),
		Labels: map[string]string{
			"app.kubernetes.io/managed-by": "dib",
		},
		Image:            "eu.gcr.io/radio-france-k8s/kaniko:v1.7.0",
		ImagePullSecrets: []string{"gcr-json-key"},
		EnvSecrets:       []string{"kanikoroawssecret"},
		Env: map[string]string{
			"AWS_REGION": "eu-west-3",
		},
		PodTemplateOverride: `
spec:
  affinity:
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchExpressions:
          - key: kops.k8s.io/instancegroup
            operator: In
            values:
            - nodes_spot
  tolerations:
  - effect: NoSchedule
    key: dedicated
    operator: Equal
    value: spot
`,
		ContainerOverride: `
resources:
  limits:
    cpu: 2
    memory: 2Gi
  requests:
    cpu: 1
    memory: 1Gi
`,
	})
	executor.DockerConfigSecret = "kanikodockersecret"

	return executor, nil
}

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/aws/aws-sdk-go-v2/config"
	cli "github.com/jawher/mow.cli"
	"github.com/sirupsen/logrus"
	kube "gitlab.com/radiofrance/kubecli"

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

func cmdBuild(cmd *cli.Cmd) {
	var opts buildOpts

	defaultOpts(&opts, cmd)
	cmd.BoolOptPtr(&opts.dryRun, "dry-run", false, "Simulate what would happen without actually doing anything dangerous.")
	cmd.BoolOptPtr(&opts.forceRebuild, "force-rebuild", false, "Forces rebuilding the entire image graph, without regarding if the target version already exists.") //nolint:lll
	cmd.BoolOptPtr(&opts.disableGenerateGraph, "no-graph", false, "Disable generation of graph during the build process.")                                          //nolint:lll
	cmd.BoolOptPtr(&opts.disableRunTests, "no-tests", false, "Disable execution of tests during the build process.")                                                //nolint:lll
	cmd.BoolOptPtr(&opts.disableJunitReports, "no-junit", false, "Disable generation of junit reports when running tests")
	cmd.BoolOptPtr(&opts.retagLatest, "retag-latest", false, "Should images be retagged with the 'latest' tag for this build") //nolint:lll
	cmd.BoolOptPtr(&opts.localOnly, "local-only", false, "Build docker images locally, do not push on remote registry")
	cmd.StringOptPtr(&opts.backend, "b backend", backendDocker, fmt.Sprintf("Build backend used to run image builds. Supported backends: %v", supportedBackends)) //nolint:lll

	cmd.Action = func() {
		backendIsSupported := false
		for _, b := range supportedBackends {
			if opts.backend == b {
				backendIsSupported = true
				break
			}
		}
		if !backendIsSupported {
			logrus.Fatalf("Invalid backend \"%s\": not supported", opts.backend)
		}

		if opts.backend == backendKaniko && opts.localOnly {
			logrus.Warnf("Using backend \"kaniko\" with the --local-only flag is partially supported.")
		}

		if opts.backend == backendDocker {
			preflight.RunPreflightChecks([]string{"docker"})
		}

		DAG, err := doBuild(opts)
		if err != nil {
			logrus.Fatalf("Build failed: %v", err)
		}

		if !opts.disableGenerateGraph {
			workingDir, err := getWorkingDir()
			if err != nil {
				logrus.Fatalf("failed to get current working directory: %v", err)
			}
			if err := graphviz.GenerateGraph(DAG, workingDir); err != nil {
				logrus.Fatalf("Generating graph failed: %v", err)
			}
		}
	}
}

func getWorkingDir() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}
	return currentDir, nil
}

func doBuild(opts buildOpts) (*dag.DAG, error) {
	workingDir, err := getWorkingDir()
	if err != nil {
		return nil, err
	}
	dockerDir, err := findDockerRootDir(workingDir, opts.buildPath)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get current working directory: %w", err)
	}
	shell := &exec.ShellExecutor{
		Dir: workingDir,
	}

	gcrRegistry, err := registry.NewRegistry(opts.registryURL, opts.dryRun)
	if err != nil {
		return nil, err
	}
	dockerBuilderTagger := docker.NewImageBuilderTagger(shell, opts.dryRun)
	DAG := &dag.DAG{
		Registry: gcrRegistry,
		Builder:  dockerBuilderTagger,
		TestRunners: []types.TestRunner{
			dgoss.NewTestRunner(dgoss.TestRunnerOptions{
				ReportsDirectory: junitReportsDirectory,
				WorkingDirectory: workingDir,
				JUnitReports:     !opts.disableJunitReports,
			}),
		},
	}

	if opts.backend == backendKaniko {
		var executor kaniko.Executor
		var contextProvider kaniko.ContextProvider
		if opts.localOnly {
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
		kanikoBuilder.DryRun = opts.dryRun

		DAG.Builder = kanikoBuilder
	}

	if opts.localOnly {
		DAG.Tagger = dockerBuilderTagger
	} else {
		DAG.Tagger = gcrRegistry
	}

	logrus.Infof("Building images in directory \"%s\"", path.Join(workingDir, opts.buildPath))

	logrus.Debug("Generate DAG")
	DAG.GenerateDAG(workingDir, opts.buildPath, opts.registryURL)
	logrus.Debug("Generate DAG -- Done")

	currentVersion, err := versn.CheckDockerVersionIntegrity(path.Join(workingDir, dockerDir))
	if err != nil {
		return nil, fmt.Errorf("cannot find current version: %w", err)
	}

	previousVersion, diffs, err := versn.GetDiffSinceLastDockerVersionChange(
		workingDir, shell, gcrRegistry, path.Join(dockerDir, versn.DockerVersionFilename),
		path.Join(opts.registryURL, opts.referentialImage))
	if err != nil {
		if errors.Is(err, versn.ErrNoPreviousBuild) {
			previousVersion = placeholderNonExistent
		} else {
			return nil, fmt.Errorf("cannot find previous version: %w", err)
		}
	}

	if opts.forceRebuild {
		logrus.Info("--force-rebuild is set to true, all images will be rebuild regarless of their changes ")
		DAG.TagForRebuild()
	} else {
		DAG.CheckForDiff(diffs)
	}

	err = DAG.Retag(currentVersion, previousVersion)
	if err != nil {
		return nil, err
	}

	if err := DAG.Rebuild(currentVersion, opts.forceRebuild, opts.disableRunTests, opts.localOnly); err != nil {
		return nil, err
	}

	if opts.retagLatest {
		logrus.Info("--retag-latest is set to true, latest tag will now use current image versions")
		if err := DAG.RetagLatest(currentVersion); err != nil {
			return nil, err
		}
	}

	if !opts.localOnly {
		// We retag the referential image to explicit this commit was build using dib
		if err := DAG.Tagger.Tag(fmt.Sprintf("%s:%s", path.Join(opts.registryURL, opts.referentialImage), "latest"),
			fmt.Sprintf("%s:%s", path.Join(opts.registryURL, opts.referentialImage), currentVersion)); err != nil {
			return nil, err
		}
	}

	logrus.Info("Build process completed")
	return DAG, nil
}

// findDockerRootDir iterates over the buildPath to find the first matching directory containing
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

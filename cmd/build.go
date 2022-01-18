package main

import (
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/radiofrance/dib/docker"

	"github.com/radiofrance/dib/registry"

	"github.com/radiofrance/dib/types"

	"github.com/radiofrance/dib/dgoss"

	"github.com/radiofrance/dib/graphviz"

	cli "github.com/jawher/mow.cli"
	"github.com/sirupsen/logrus"

	"github.com/radiofrance/dib/dag"
	"github.com/radiofrance/dib/exec"
	"github.com/radiofrance/dib/preflight"
	versn "github.com/radiofrance/dib/version"
)

const placeholderNonExistent = "non-existent"

func cmdBuild(cmd *cli.Cmd) {
	var opts buildOpts

	defaultOpts(&opts, cmd)
	cmd.BoolOptPtr(&opts.dryRun, "dry-run", false, "Simulate what would happen without actually doing anything dangerous.")
	cmd.BoolOptPtr(&opts.forceRebuild, "force-rebuild", false, "Forces rebuilding the entire image graph, without regarding if the target version already exists.") //nolint:lll
	cmd.BoolOptPtr(&opts.disableGenerateGraph, "no-graph", false, "Disable generation of graph during the build process.")                                          //nolint:lll
	cmd.BoolOptPtr(&opts.disableRunTests, "no-tests", false, "Disable execution of tests during the build process.")                                                //nolint:lll
	cmd.BoolOptPtr(&opts.retagLatest, "retag-latest", false, "Should images be retagged with the 'latest' tag for this build")                                      //nolint:lll
	cmd.BoolOptPtr(&opts.localOnly, "local-only", false, "Build docker images locally, do not push on remote registry")

	cmd.Action = func() {
		preflight.RunPreflightChecks([]string{"docker"})

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
			dgoss.TestRunner{},
		},
	}

	if opts.localOnly {
		DAG.Tagger = dockerBuilderTagger
	} else {
		DAG.Tagger = gcrRegistry
	}

	buildPath := path.Join(workingDir, opts.buildPath)
	logrus.Infof("Building images in directory \"%s\"", buildPath)

	logrus.Debug("Generate DAG")
	DAG.GenerateDAG(buildPath, opts.registryURL)
	logrus.Debug("Generate DAG -- Done")

	dockerDir, err := findDockerRootDir(workingDir, opts.buildPath)
	if err != nil {
		return nil, err
	}

	currentVersion, err := versn.CheckDockerVersionIntegrity(path.Join(workingDir, dockerDir))
	if err != nil {
		return nil, err
	}

	previousVersion, diffs, err := versn.GetDiffSinceLastDockerVersionChange(
		workingDir, shell, gcrRegistry, path.Join(opts.buildPath, versn.DockerVersionFilename),
		path.Join(opts.registryURL, opts.referentialImage))
	if err != nil {
		if errors.Is(err, versn.ErrNoPreviousBuild) {
			previousVersion = placeholderNonExistent
		} else {
			return nil, err
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

	// We retag the referential image to explicit this commit was build using dib
	if err := DAG.Tagger.Tag(fmt.Sprintf("%s:%s", path.Join(opts.registryURL, opts.referentialImage), "latest"),
		fmt.Sprintf("%s:%s", path.Join(opts.registryURL, opts.referentialImage), currentVersion)); err != nil {
		return nil, err
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

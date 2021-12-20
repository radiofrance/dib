package main

import (
	"log"
	"os"
	"path"

	"github.com/radiofrance/dib/dgoss"

	"github.com/radiofrance/dib/graphviz"

	cli "github.com/jawher/mow.cli"
	"github.com/sirupsen/logrus"

	"github.com/radiofrance/dib/dag"
	"github.com/radiofrance/dib/docker"
	"github.com/radiofrance/dib/exec"
	"github.com/radiofrance/dib/preflight"
	versn "github.com/radiofrance/dib/version"
)

func cmdBuild(cmd *cli.Cmd) {
	buildDir := getBuildDirectoryArg(cmd)

	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	dryRun := cmd.BoolOpt("dry-run", false, "Simulate what would happen without actually doing anything dangerous.")
	forceRebuild := cmd.BoolOpt("force-rebuild", false, "Forces rebuilding the entire image graph, without regarding if the target version already exists.") //nolint:lll
	outputDir := cmd.StringOpt("o output", pwd, "Output directory where .dot and .png files will be generated.")
	inputDir := cmd.StringOpt("i input", pwd, "Root directory where docker directory and .dockerversion files are stored.")
	registryURL := cmd.StringOpt("registry-url", defaultRegistryURL, "Docker registry URL where images are stored.")
	graph := cmd.BoolOpt("g graph", false, "Instruct dib to generate graphviz during the build process.")
	test := cmd.BoolOpt("t test", false, "Instruct dib to run goss tests during the build process.")

	cmd.Action = func() {
		preflight.RunPreflightChecks([]string{"docker"})
		DAG, err := doBuild(*dryRun, *forceRebuild, *test, *buildDir, *inputDir, *registryURL)
		if err != nil {
			logrus.Fatalf("Build failed: %v", err)
		}

		if *graph {
			if err := graphviz.GenerateGraph(DAG, outputDir); err != nil {
				logrus.Fatalf("Generating graph failed: %v", err)
			}
		}
	}
}

func doBuild(dryRun, forceRebuild, runTests bool, buildDir, inputDir, registryURL string) (*dag.DAG, error) {
	shell := &exec.ShellExecutor{
		Dir: inputDir,
	}

	var err error
	registry, err := docker.NewRegistry(registryURL, dryRun)
	if err != nil {
		return nil, err
	}
	DAG := &dag.DAG{
		Registry: registry,
		Builder:  docker.NewImageBuilder(shell, dryRun),
		TestRunners: []dag.TestRunner{
			dgoss.TestRunner{},
		},
	}

	buildPath := path.Join(inputDir, buildDir)
	logrus.Infof("Building images in directory \"%s\"", buildPath)

	logrus.Debug("Generate DAG")
	DAG.GenerateDAG(buildPath, registryURL)
	logrus.Debug("Generate DAG -- Done")

	currentVersion, err := versn.CheckDockerVersionIntegrity(inputDir, shell)
	if err != nil {
		return nil, err
	}

	previousVersion, diffs, err := versn.GetDiffSinceLastDockerVersionChange(inputDir, shell)
	if err != nil {
		return nil, err
	}

	if forceRebuild {
		logrus.Info("--force-rebuild is set to true, all images will be rebuild regarless of their changes ")
		DAG.TagForRebuild()
	} else {
		DAG.CheckForDiff(diffs)
	}

	if err = DAG.Retag(currentVersion, previousVersion); err != nil {
		return nil, err
	}
	DAG.Rebuild(currentVersion, forceRebuild, runTests)
	logrus.Info("Build process completed")
	return DAG, nil
}

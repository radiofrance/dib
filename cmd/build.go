package main

import (
	"path"

	"github.com/radiofrance/dib/builder/docker"
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

func cmdBuild(cmd *cli.Cmd) {
	var opts buildOpts

	defaultOpts(&opts, cmd)
	cmd.BoolOptPtr(&opts.dryRun, "dry-run", false, "Simulate what would happen without actually doing anything dangerous.")
	cmd.BoolOptPtr(&opts.forceRebuild, "force-rebuild", false, "Forces rebuilding the entire image graph, without regarding if the target version already exists.") //nolint:lll
	cmd.BoolOptPtr(&opts.generateGraph, "g graph", false, "Instruct dib to generate graphviz during the build process.")
	cmd.BoolOptPtr(&opts.runTests, "t test", false, "Instruct dib to run goss tests during the build process.")
	cmd.BoolOptPtr(&opts.retagLatest, "retag-latest", false,
		"Should images be retagued with the 'latest' tag for this build")

	cmd.Action = func() {
		preflight.RunPreflightChecks([]string{"docker"})
		DAG, err := doBuild(opts)
		if err != nil {
			logrus.Fatalf("Build failed: %v", err)
		}

		if opts.generateGraph {
			if err := graphviz.GenerateGraph(DAG, opts.outputDir); err != nil {
				logrus.Fatalf("Generating graph failed: %v", err)
			}
		}
	}
}

func doBuild(opts buildOpts) (*dag.DAG, error) {
	shell := &exec.ShellExecutor{
		Dir: opts.inputDir,
	}

	var err error
	reg, err := registry.NewRegistry(opts.registryURL, opts.dryRun)
	if err != nil {
		return nil, err
	}
	DAG := &dag.DAG{
		Registry: reg,
		Builder:  docker.NewImageBuilder(shell, opts.dryRun),
		TestRunners: []types.TestRunner{
			dgoss.TestRunner{},
		},
	}

	buildPath := path.Join(opts.inputDir, opts.buildDir)
	logrus.Infof("Building images in directory \"%s\"", buildPath)

	logrus.Debug("Generate DAG")
	DAG.GenerateDAG(buildPath, opts.registryURL)
	logrus.Debug("Generate DAG -- Done")

	currentVersion, err := versn.CheckDockerVersionIntegrity(opts.inputDir, shell)
	if err != nil {
		return nil, err
	}

	previousVersion, diffs, err := versn.GetDiffSinceLastDockerVersionChange(opts.inputDir, shell)
	if err != nil {
		return nil, err
	}

	if opts.forceRebuild {
		logrus.Info("--force-rebuild is set to true, all images will be rebuild regarless of their changes ")
		DAG.TagForRebuild()
	} else {
		DAG.CheckForDiff(diffs)
	}

	if err = DAG.Retag(currentVersion, previousVersion); err != nil {
		return nil, err
	}
	DAG.Rebuild(currentVersion, opts.forceRebuild, opts.runTests)
	if opts.retagLatest {
		if err := DAG.RetagLatest(currentVersion); err != nil {
			return nil, err
		}
	}
	logrus.Info("Build process completed")
	return DAG, nil
}

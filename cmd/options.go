package main

import (
	"log"
	"os"

	cli "github.com/jawher/mow.cli"
)

type buildOpts struct {
	dryRun        bool
	forceRebuild  bool
	runTests      bool
	retagLatest   bool
	generateGraph bool
	buildDir      string
	inputDir      string
	outputDir     string
	registryURL   string
}

func defaultOpts(opts *buildOpts, cmd *cli.Cmd) {
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	//nolint:lll
	desc := `Path to the directory containing the Dockerfiles, relative to the input directory specified by --input.
All Dockerfiles within this directory will be recursively found and added to the build graph.
You can provide any subdirectory if you want to focus on a reduced set of images, as long as it has at least one Dockerfile in it.`

	cmd.Spec = "[OPTIONS] [BUILD_PATH]"
	cmd.StringArgPtr(&opts.buildDir, "BUILD_PATH", "docker", desc)

	cmd.StringOptPtr(&opts.registryURL, "registry-url", defaultRegistryURL, "Docker registry URL where images are stored.")
	cmd.StringOptPtr(&opts.outputDir, "o output", pwd,
		"Output directory where .dot and .png files will be generated")
	cmd.StringOptPtr(&opts.inputDir, "i input", pwd,
		"Root directory where docker directory and .dockerversion files are stored")
}

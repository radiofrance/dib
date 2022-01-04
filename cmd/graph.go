package main

import (
	"log"
	"os"

	cli "github.com/jawher/mow.cli"
	"github.com/radiofrance/dib/graphviz"
	"github.com/radiofrance/dib/preflight"
)

func cmdGraph(cmd *cli.Cmd) {
	buildDir := getBuildDirectoryArg(cmd)
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	registryURL := cmd.StringOpt("registry-url", defaultRegistryURL, "Docker registry URL where images are stored.")
	outputDir := cmd.StringOpt("o output", pwd, "Output directory where .dot and .png files will be generated")
	inputDir := cmd.StringOpt("i input", pwd, "Root directory where docker directory and .dockerversion files are stored")

	cmd.Action = func() {
		preflight.RunPreflightChecks([]string{"dot"})
		DAG, err := doBuild(true, false, false, false, *buildDir, *inputDir, *registryURL)
		if err != nil {
			log.Fatal(err)
		}
		if err := graphviz.GenerateGraph(DAG, outputDir); err != nil {
			log.Fatal(err)
		}
	}
}

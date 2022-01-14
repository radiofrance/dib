package main

import (
	"log"

	"github.com/sirupsen/logrus"

	cli "github.com/jawher/mow.cli"
	"github.com/radiofrance/dib/graphviz"
	"github.com/radiofrance/dib/preflight"
)

func cmdGraph(cmd *cli.Cmd) {
	var opts buildOpts
	defaultOpts(&opts, cmd)

	opts.dryRun = true
	opts.generateGraph = true

	cmd.Action = func() {
		preflight.RunPreflightChecks([]string{"dot"})
		DAG, err := doBuild(opts)
		if err != nil {
			log.Fatal(err)
		}
		workingDir, err := getWorkingDir()
		if err != nil {
			logrus.Fatalf("failed to get current working directory: %v", err)
		}
		if err := graphviz.GenerateGraph(DAG, workingDir); err != nil {
			log.Fatal(err)
		}
	}
}

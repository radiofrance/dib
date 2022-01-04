package main

import (
	"log"
	"os"

	cli "github.com/jawher/mow.cli"
	"github.com/radiofrance/dib/preflight"
)

func cmdTest(cmd *cli.Cmd) {
	buildDir := getBuildDirectoryArg(cmd)
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	registryURL := cmd.StringOpt("registry-url", defaultRegistryURL, "Docker registry URL where images are stored.")

	cmd.Action = func() {
		preflight.RunPreflightChecks([]string{"dgoss"})
		if _, err := doBuild(true, true, true, false, *buildDir, pwd, *registryURL); err != nil {
			log.Fatal(err)
		}
	}
}

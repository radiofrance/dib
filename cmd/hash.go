package main

import (
	"fmt"
	"log"

	versionpkg "github.com/radiofrance/dib/version"

	cli "github.com/jawher/mow.cli"
)

func cmdHash(cmd *cli.Cmd) {
	cmd.Spec = "[BUILD_PATH]"
	buildPath := cmd.StringArg("BUILD_PATH", "docker", "Path to the directory containing the Dockerfiles,")

	cmd.Action = func() {
		hash, err := versionpkg.GetDockerVersionHash(*buildPath)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print(hash) // nolint: forbidigo
	}
}

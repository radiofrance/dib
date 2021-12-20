package main

import (
	"fmt"

	"github.com/radiofrance/dib/preflight"

	cli "github.com/jawher/mow.cli"
)

func cmdHash(cmd *cli.Cmd) {
	_ = getBuildDirectoryArg(cmd)

	cmd.Action = func() {
		preflight.RunPreflightChecks([]string{})
		fmt.Println("Not implemented yet.") //nolint:forbidigo
	}
}

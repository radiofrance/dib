package main

import (
	"fmt"

	cli "github.com/jawher/mow.cli"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"
)

func cmdVersion(cmd *cli.Cmd) {
	cmd.Action = func() {
		fmt.Printf("dib v%s\n", version)                 //nolint:forbidigo
		fmt.Printf("commit %s\n", commit)                //nolint:forbidigo
		fmt.Printf("built at %s by %s\n", date, builtBy) //nolint:forbidigo
	}
}

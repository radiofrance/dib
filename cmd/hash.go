package main

import (
	"fmt"

	cli "github.com/jawher/mow.cli"
)

func cmdHash(cmd *cli.Cmd) {
	cmd.Action = func() {
		fmt.Println("Not implemented yet.") //nolint:forbidigo
	}
}

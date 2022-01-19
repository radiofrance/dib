package main

import (
	"log"

	cli "github.com/jawher/mow.cli"
	"github.com/radiofrance/dib/preflight"
)

func cmdTest(cmd *cli.Cmd) {
	var opts buildOpts
	defaultOpts(&opts, cmd)

	cmd.BoolOptPtr(&opts.disableJunitReports, "no-junit", false, "Disable generation of junit reports when running tests")

	opts.dryRun = true
	opts.forceRebuild = true

	cmd.Action = func() {
		preflight.RunPreflightChecks([]string{"dgoss"})

		if _, err := doBuild(opts); err != nil {
			log.Fatal(err)
		}
	}
}

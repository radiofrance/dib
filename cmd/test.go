package cmd

import (
	"log"

	"github.com/radiofrance/dib/preflight"
	"github.com/spf13/cobra"
)

// testCmd represents the test command.
var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run tests on docker images. This command expects referenced images to exist\"",
	Run: func(cmd *cobra.Command, args []string) {
		preflight.RunPreflightChecks([]string{"dgoss"})

		opts := buildOptsFromViper()
		opts.DryRun = true
		opts.ForceRebuild = true

		if _, err := doBuild(opts); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(testCmd)
}

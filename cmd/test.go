package cmd

import (
	"log"

	"github.com/radiofrance/dib/preflight"
	"github.com/spf13/cobra"
)

// testCmd represents the test command.
var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run tests only on docker images. This command expects images references to exist\"",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
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

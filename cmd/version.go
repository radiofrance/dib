package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"
)

// versionCmd represents the version command.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print current dib version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("dib v%s\n", version)                 //nolint:forbidigo
		fmt.Printf("commit %s\n", commit)                //nolint:forbidigo
		fmt.Printf("built at %s by %s\n", date, builtBy) //nolint:forbidigo
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

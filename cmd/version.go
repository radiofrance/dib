package main

import (
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var (
	// Automatically filled by GoReleaser during build process
	// @see: https://goreleaser.com/cookbooks/using-main.version/
	version = "unreleased-dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"
)

// versionCmd represents the version command.
//
//nolint:forbidigo
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "print current dib version",
	Run: func(cmd *cobra.Command, args []string) {
		goVersion := "unknown"
		buildInfo, available := debug.ReadBuildInfo()
		if available {
			goVersion = buildInfo.GoVersion
		}

		fmt.Printf("version: v%s\n", version)
		fmt.Printf("build on %s/%s by %s at %s with %s (from commit %s)\n",
			runtime.GOOS,
			runtime.GOARCH,
			builtBy,
			date,
			goVersion,
			commit,
		)
		if available && buildInfo.Main.Sum != "" {
			fmt.Printf("module version: %s, checksum: %q\n", buildInfo.Main.Version, buildInfo.Main.Sum)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

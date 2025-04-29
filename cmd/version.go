//nolint:forbidigo
package cmd

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

func versionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "print current dib version",
		Run:   versionAction,
	}
}

func versionAction(*cobra.Command, []string) {
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
}

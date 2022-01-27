package cmd

import (
	"fmt"
	"log"

	versionpkg "github.com/radiofrance/dib/version"
	"github.com/spf13/cobra"
)

// hashCmd represents the hash command.
var hashCmd = &cobra.Command{
	Use:   "hash [build path]",
	Short: "Generates a version hash of the docker directory",
	Long: `dib hash will calculate a unique human readable hash of the "docker" directory, which
contains all Dockerfiles. If no argument is passed to dib hash, it will use 'docker' as the directory name`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		opts := rootOpts{}
		hydrateOptsFromViper(&opts)
		hash, err := versionpkg.GetDockerVersionHash(opts.BuildPath)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print(hash) // nolint: forbidigo
	},
}

func init() {
	rootCmd.AddCommand(hashCmd)
}

package cmd

import (
	"log"

	"github.com/radiofrance/dib/graphviz"
	"github.com/radiofrance/dib/preflight"
	"github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

// graphCmd represents the graph command.
var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Create a visual representation of the build graph",
	Long: `Create a visual representation of the build graph using graphviz

In the generated graph, images are represented with color status
Red means the image will be rebuilt
Yellow means the image will be re-taged from its previous built version
Transparent means no action on the image`,
	Run: func(cmd *cobra.Command, args []string) {
		preflight.RunPreflightChecks([]string{"dot"})
		opts := buildOptsFromViper()
		opts.DryRun = true
		opts.DisableRunTests = true
		opts.DisableJunitReports = true
		DAG, err := doBuild(opts)
		if err != nil {
			log.Fatal(err)
		}
		workingDir, err := getWorkingDir()
		if err != nil {
			logrus.Fatalf("failed to get current working directory: %v", err)
		}
		if err := graphviz.GenerateGraph(DAG, workingDir); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(graphCmd)
}

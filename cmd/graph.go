package cmd

import (
	"fmt"
	"log"
	"path"

	"github.com/radiofrance/dib/dib"
	"github.com/radiofrance/dib/graphviz"
	"github.com/radiofrance/dib/preflight"
	"github.com/radiofrance/dib/registry"

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

		opts := rootOpts{}
		hydrateOptsFromViper(&opts)
		err := doGraph(opts)
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(graphCmd)
}

func doGraph(opts rootOpts) error {
	workingDir, err := getWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	gcrRegistry, err := registry.NewRegistry(opts.RegistryURL, true)
	if err != nil {
		return err
	}

	logrus.Debug("Generate DAG")
	DAG := dib.GenerateDAG(path.Join(workingDir, opts.BuildPath), opts.RegistryURL)
	logrus.Debug("Generate DAG -- Done")

	err = dib.Plan(DAG, gcrRegistry, false, false)
	if err != nil {
		return err
	}

	if err := graphviz.GenerateGraph(DAG); err != nil {
		log.Fatal(err)
	}

	return nil
}

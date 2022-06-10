package main

import (
	"fmt"
	"log"
	"path"

	"github.com/radiofrance/dib/pkg/dib"
	"github.com/radiofrance/dib/pkg/graphviz"
	"github.com/radiofrance/dib/pkg/preflight"
	"github.com/radiofrance/dib/pkg/registry"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// graphCmd represents the graph command.
var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Create a visual representation of the build graph",
	Long: `Create a visual representation of the build graph using graphviz.

In the generated graph, images are represented with red color when it needs to be rebuilt.
It remains colorless when no action is required for the image.`,
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

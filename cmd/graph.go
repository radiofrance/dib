package main

import (
	"fmt"
	"path"

	"github.com/radiofrance/dib/internal/logger"
	"github.com/radiofrance/dib/pkg/dib"
	"github.com/spf13/cobra"
)

type GraphOpts struct {
	BuildPath        string `mapstructure:"build_path"`
	RegistryURL      string `mapstructure:"registry_url"`
	PlaceholderTag   string `mapstructure:"placeholder_tag"`
	HashListFilePath string `mapstructure:"hash_list_file_path"`
}

// buildCmd represents the build command.
var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Compute the graph of images, and print it.",
	Long:  "Compute the graph of images, and print it.",
	Run: func(cmd *cobra.Command, _ []string) {
		bindPFlagsSnakeCase(cmd.Flags())

		opts := GraphOpts{}
		hydrateOptsFromViper(&opts)

		if err := doGraph(opts); err != nil {
			logger.Fatalf("graph command failed: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(graphCmd)
}

func doGraph(opts GraphOpts) error {
	workingDir, err := getWorkingDir()
	if err != nil {
		logger.Fatalf("failed to get current working directory: %v", err)
	}

	buildPath := path.Join(workingDir, opts.BuildPath)
	logger.Infof("Building images in directory \"%s\"", buildPath)

	logger.Debugf("Generate DAG")
	graph, err := dib.GenerateDAG(buildPath, opts.RegistryURL, opts.HashListFilePath, map[string]string{})
	if err != nil {
		return fmt.Errorf("generating DAG: %w", err)
	}
	logger.Debugf("Generate DAG -- Done")

	logger.Debugf("Print DAG")
	graph.Print(opts.BuildPath)
	logger.Debugf("Print DAG -- Done")
	return nil
}

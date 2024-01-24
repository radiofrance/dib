package main

import (
	"fmt"
	"path"

	"github.com/radiofrance/dib/internal/logger"
	"github.com/radiofrance/dib/pkg/dib"
	"github.com/spf13/cobra"
)

// listCmd represents the output command.
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Print list of images managed by DIB",
	Long:  `dib list will print a list of all Docker images managed by DIB`,
	Run: func(cmd *cobra.Command, _ []string) {
		bindPFlagsSnakeCase(cmd.Flags())

		opts := dib.ListOpts{}
		hydrateOptsFromViper(&opts)

		if err := doList(opts); err != nil {
			logger.Fatalf("List failed: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringP("output", "o", "", ""+
		"Output format (console|go-template-file)\n"+
		"You can provide a custom format using go-template: like this: \"-o go-template-file=...\".")
}

func doList(opts dib.ListOpts) error {
	formatOpts, err := dib.ParseOutputOptions(opts.Output)
	if err != nil {
		return fmt.Errorf("error while parsing output options: %w", err)
	}

	workingDir, err := getWorkingDir()
	if err != nil {
		return err
	}

	buildPath := path.Join(workingDir, opts.BuildPath)
	graph := dib.GenerateDAG(buildPath, opts.RegistryURL, opts.HashListFilePath)
	return dib.GenerateList(graph, formatOpts)
}

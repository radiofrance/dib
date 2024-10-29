package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/radiofrance/dib/internal/logger"
	"github.com/radiofrance/dib/pkg/dib"
	"github.com/spf13/cobra"
)

// listCmd represents the output command.
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Print list of images managed by dib",
	Long:  `dib list will print a list of all Docker images managed by dib`,
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
	listCmd.Flags().StringArray("build-arg", []string{},
		"`argument=value` to supply to the builder")
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

	buildArgs := map[string]string{}
	for _, arg := range opts.BuildArg {
		key, val, hasVal := strings.Cut(arg, "=")
		if hasVal {
			buildArgs[key] = os.ExpandEnv(val)
		} else {
			// check if the env is set in the local environment and use that value if it is
			if val, present := os.LookupEnv(key); present {
				buildArgs[key] = os.ExpandEnv(val)
			} else {
				delete(buildArgs, key)
			}
		}
	}

	buildPath := path.Join(workingDir, opts.BuildPath)
	graph, err := dib.GenerateDAG(buildPath, opts.RegistryURL, opts.HashListFilePath, buildArgs)
	if err != nil {
		return fmt.Errorf("cannot generate DAG: %w", err)
	}

	return dib.GenerateList(graph, formatOpts)
}

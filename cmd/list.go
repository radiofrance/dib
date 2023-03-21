package main

import (
	"path"

	"github.com/radiofrance/dib/pkg/dib"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type listOpts struct {
	// Root options
	BuildPath      string `mapstructure:"build_path"`
	RegistryURL    string `mapstructure:"registry_url"`
	PlaceholderTag string `mapstructure:"placeholder_tag"`

	// List specific options
	Output string `mapstructure:"output"`
}

// listCmd represents the output command.
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Print list of images managed by DIB",
	Long:  `dib list will print a list of all Docker images managed by DIB`,
	Run: func(cmd *cobra.Command, args []string) {
		bindPFlagsSnakeCase(cmd.Flags())
		opts := listOpts{}
		hydrateOptsFromViper(&opts)

		err := doList(opts)
		if err != nil {
			logrus.Fatalf("command \"dib list\" failed: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringP("output", "o", "", ""+
		"Output format (console|go-template-file)\n"+
		"You can provide a custom format using go-template: like this: \"-o go-template-file=...\".")
}

func doList(opts listOpts) error {
	formatOpts, err := dib.ParseOutputOptions(opts.Output)
	if err != nil {
		return err
	}

	workingDir, err := getWorkingDir()
	if err != nil {
		return err
	}

	DAG := dib.GenerateDAG(path.Join(workingDir, opts.BuildPath), opts.RegistryURL)
	return dib.GenerateList(DAG, formatOpts)
}

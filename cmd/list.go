package cmd

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/radiofrance/dib/pkg/dib"
	"github.com/spf13/cobra"
)

func listCommand() *cobra.Command {
	const longHelp = `Command list provide different ways to print the list of all Docker images managed by dib.

The output can be customized with the --output flag :
• console (default output)
  ex : dib list

• go-template-file (render output using a Go template)
  ex : dib list -o go-template-file=dib_list.tmpl

• graphviz (dot language output)
  ex : dib list -o graphviz

  You can also generate a PNG image from the graphviz output using the following command :
  dib list -o graphviz | dot -Tpng > dib.png
`

	cmd := &cobra.Command{
		Use:           "list",
		Short:         "List all images managed by dib",
		Long:          longHelp,
		RunE:          listAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.Flags().StringP("output", "o", dib.ConsoleFormat,
		"Output format : console|graphviz|go-template-file")
	cmd.Flags().StringArray("build-arg", []string{},
		"`argument=value` to supply to the builder")

	return cmd
}

func listAction(cmd *cobra.Command, _ []string) error {
	// Bind command flags to viper configuration using snake_case
	bindPFlagsSnakeCase(cmd.Flags())

	opts := dib.ListOpts{}
	hydrateOptsFromViper(&opts)

	formatOpts, err := dib.ParseOutputOptions(opts.Output)
	if err != nil {
		return fmt.Errorf("error while parsing output options: %w", err)
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

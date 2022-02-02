package cmd

import (
	"errors"
	"fmt"
	"log"
	"path"

	"github.com/radiofrance/dib/dib"
	"github.com/radiofrance/dib/exec"
	"github.com/radiofrance/dib/graphviz"
	"github.com/radiofrance/dib/preflight"
	"github.com/radiofrance/dib/registry"
	versn "github.com/radiofrance/dib/version"
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
	dockerDir, err := findDockerRootDir(workingDir, opts.BuildPath)
	if err != nil {
		return err
	}

	gcrRegistry, err := registry.NewRegistry(opts.RegistryURL, true)
	if err != nil {
		return err
	}

	shell := &exec.ShellExecutor{
		Dir: workingDir,
	}

	currentVersion, err := versn.CheckDockerVersionIntegrity(path.Join(workingDir, dockerDir))
	if err != nil {
		return fmt.Errorf("cannot find current version: %w", err)
	}

	previousVersion, diffs, err := versn.GetDiffSinceLastDockerVersionChange(
		workingDir, shell, gcrRegistry, path.Join(dockerDir, versn.DockerVersionFilename),
		path.Join(opts.RegistryURL, opts.ReferentialImage))
	if err != nil {
		if errors.Is(err, versn.ErrNoPreviousBuild) {
			previousVersion = placeholderNonExistent
		} else {
			return fmt.Errorf("cannot find previous version: %w", err)
		}
	}

	logrus.Debug("Generate DAG")
	DAG := dib.GenerateDAG(path.Join(workingDir, opts.BuildPath), opts.RegistryURL)
	logrus.Debug("Generate DAG -- Done")

	err = dib.Plan(DAG, gcrRegistry, diffs, previousVersion, currentVersion, false, false)
	if err != nil {
		return err
	}

	if err := graphviz.GenerateGraph(DAG); err != nil {
		log.Fatal(err)
	}

	return nil
}

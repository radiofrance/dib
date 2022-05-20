package cmd

import (
	"fmt"
	"path"

	"github.com/radiofrance/dib/dag"
	"github.com/radiofrance/dib/dib"
	"github.com/radiofrance/dib/registry"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type migrateOpts struct {
	// Root options
	BuildPath   string `mapstructure:"build_path"`
	RegistryURL string `mapstructure:"registry_url"`

	// Migrate specific options
	DryRun        bool   `mapstructure:"dry_run"`
	DockerVersion string `mapstructure:"docker_version"`
}

// migrateCmd represents the migrate command.
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate from old version system (<0.9.0)",
	Long:  "This command will tag every image from the given docker-version tag with their new tag computed from the current filesystem state.", //nolint:lll
	Run: func(cmd *cobra.Command, args []string) {
		bindPFlagsSnakeCase(cmd.Flags())

		opts := migrateOpts{}
		hydrateOptsFromViper(&opts)

		err := doMigrate(opts)
		if err != nil {
			logrus.Fatalf("Migration failed: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)

	migrateCmd.Flags().String("docker-version", "",
		"The tag present in the .docker-version file")
	migrateCmd.Flags().Bool("dry-run", false,
		"Dry run mode, see what would happen without actually changing anything")
}

func doMigrate(opts migrateOpts) error {
	workingDir, err := getWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	gcrRegistry, err := registry.NewRegistry(opts.RegistryURL, opts.DryRun)
	if err != nil {
		return err
	}
	logrus.Debug("Generate DAG")
	DAG := dib.GenerateDAG(path.Join(workingDir, opts.BuildPath), opts.RegistryURL)
	logrus.Debug("Generate DAG -- Done")

	DAG.Walk(func(node *dag.Node) {
		oldTag := node.Image.DockerRef(opts.DockerVersion)
		newTag := node.Image.DockerRef(node.Image.Hash)
		if err := gcrRegistry.Tag(oldTag, newTag); err != nil {
			logrus.Errorf("Failed to tag \"%s\" from \"%s\"", oldTag, newTag)
		}
	})

	logrus.Info("Migration process completed")
	return nil
}

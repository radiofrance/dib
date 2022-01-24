package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/spf13/viper"
)

var cfgFile string

const (
	defaultRegistryURL      = "eu.gcr.io/my-test-repository"
	defaultReferentialImage = "dib-referential"
	defaultLogLevel         = "info"
	defaultBuildPath        = "docker"

	keyBackend          = "Backend"
	keyLocalOnly        = "local-only"
	keyDisableGraph     = "no-graph"
	keyDisableJUnit     = "no-junit"
	keyDisableTests     = "no-tests"
	keyBuildPath        = "build-path"
	keyDryRun           = "dry-run"
	keyRegistryURL      = "registry-url"
	keyForceRebuild     = "force-rebuild"
	keyRetagLatest      = "retag-latest"
	keyReferentialImage = "referential-image"
	keyLogLevel         = "log-level"
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "dib",
	Short: "An Opinionated Docker Image Builder",
	Long: `Docker Image Builder helps building a complex image dependency graph

Run dib --help for mor information`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig, initLogLevel)

	desc := `Path to the directory you want to build All Dockerfiles within this directory will be recursively 
found and added to the build graph. You can provide any subdirectory if you want to focus on a reduced set of images, 
as long as it has at least one Dockerfile in it.

It is also expected that one of the director in this path contains a .docker-version file. This directory will 
be considered as the root directory for the hash generation and comparison`

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.dib.yaml)")
	rootCmd.PersistentFlags().String(keyBuildPath, defaultBuildPath, desc)
	rootCmd.PersistentFlags().String(keyRegistryURL, defaultRegistryURL, "Docker registry URL where images are stored.")
	rootCmd.PersistentFlags().String(keyReferentialImage, defaultReferentialImage, "Name of an image on "+
		"the registry. This image will serve as a reference for checking build completion of previous dib runs. Tags will be"+
		"added to this image but it has no other purpose.")
	rootCmd.PersistentFlags().StringP(keyLogLevel, "l", defaultLogLevel, "Log level. Can be any level "+
		"supported by logrus (\"info\", \"debug\", etc...)")

	_ = viper.BindPFlags(rootCmd.PersistentFlags())
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		workingDir, err := getWorkingDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".dib" (without extension).
		viper.AddConfigPath(home)
		viper.AddConfigPath(workingDir)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".dib")
	}

	viper.SetEnvPrefix("dib")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.SetDefault(keyBuildPath, "docker")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func getWorkingDir() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}
	return currentDir, nil
}

type BuildOpts struct {
	BuildPath            string
	DisableGenerateGraph bool
	DisableJunitReports  bool
	DisableRunTests      bool
	DryRun               bool
	ForceRebuild         bool
	LocalOnly            bool
	ReferentialImage     string
	RegistryURL          string
	RetagLatest          bool
	Backend              string
}

func buildOptsFromViper() BuildOpts {
	return BuildOpts{
		BuildPath:            viper.GetString(keyBuildPath),
		DisableGenerateGraph: viper.GetBool(keyDisableGraph),
		DisableJunitReports:  viper.GetBool(keyDisableJUnit),
		DisableRunTests:      viper.GetBool(keyDisableTests),
		DryRun:               viper.GetBool(keyDryRun),
		ForceRebuild:         viper.GetBool(keyForceRebuild),
		LocalOnly:            viper.GetBool(keyLocalOnly),
		ReferentialImage:     viper.GetString(keyReferentialImage),
		RegistryURL:          viper.GetString(keyRegistryURL),
		RetagLatest:          viper.GetBool(keyRetagLatest),
		Backend:              viper.GetString(keyBackend),
	}
}

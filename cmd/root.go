package cmd

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	defaultRegistryURL         = "eu.gcr.io/my-test-repository"
	defaultLogLevel            = "info"
	defaultBuildPath           = "docker"
	defaultGossImage           = "aelsabbahy/goss:latest"
	defaultKanikoImage         = "gcr.io/kaniko-project/executor:v1.8.1"
	defaultKubernetesNamespace = "default"
	distDirectory              = "dist"
)

type rootOpts struct {
	BuildPath        string `mapstructure:"build_path"`
	RegistryURL      string `mapstructure:"registry_url"`
	ReferentialImage string `mapstructure:"referential_image"`
}

var cfgFile string

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "dib",
	Short: "An Opinionated Docker Image Builder",
	Long: `Docker Image Builder helps building a complex image dependency graph

Run dib --help for more information`,
}

// Execute runs the root cobra command.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig, initLogLevel, cleanupDist)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
		"config file (default is $HOME/.config/.dib.yaml)")
	rootCmd.PersistentFlags().String("build-path", defaultBuildPath,
		`Path to the directory containing all Dockerfiles to be built by dib. Every Dockerfile will be recursively 
found and added to the build graph. You can provide any subdirectory if you want to focus on a reduced set of images, 
as long as it has at least one Dockerfile in it.`)
	rootCmd.PersistentFlags().String("registry-url", defaultRegistryURL,
		"Docker registry URL where images are stored.")
	rootCmd.PersistentFlags().StringP("log-level", "l", defaultLogLevel,
		"Log level. Can be any level supported by logrus (\"info\", \"debug\", etc...)")
}

func cleanupDist() {
	logrus.Debugf("Cleaning up %s directory before startup", distDirectory)
	// Ignore any error
	_ = os.RemoveAll(distDirectory)
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
		viper.SetConfigType("yaml")
		viper.SetConfigName(".dib")
		viper.AddConfigPath(path.Join(home, ".config"))
		viper.AddConfigPath(workingDir)
	}

	// Set defaults for config values that have no flag bound to them.
	viper.SetDefault("kaniko.executor.docker.image", defaultKanikoImage)
	viper.SetDefault("kaniko.executor.kubernetes.image", defaultKanikoImage)
	viper.SetDefault("kaniko.executor.kubernetes.namespace", defaultKubernetesNamespace)
	viper.SetDefault("goss.executor.kubernetes.image", defaultGossImage)
	viper.SetDefault("goss.executor.kubernetes.namespace", defaultKubernetesNamespace)

	// Env vars starting with the DIB_ prefix can override any configuration.
	// e.g. DIB_LOG_LEVEL, DIB_KANIKO_CONTEXT_S3_BUCKET, etc...
	viper.SetEnvPrefix("dib")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // Allows to override any sub-level in file config.
	viper.AutomaticEnv()                                   // read in environment variables that match

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

// hydrateOptsFromViper copies all the viper values into our config struct.
// The mapping between viper identifiers and struct field names
// is ensured by `mapstructure` struct tags.
func hydrateOptsFromViper(opts interface{}) {
	_ = viper.Unmarshal(opts)
}

// bindPFlagsSnakeCase binds the flags with viper values. The identifier of the viper value
// is the name of the flag with dashes replaced by underscores. This is required so we can
// retrieve values from viper with the same behaviour with config coming from files
// (my_config: "value") or from flags (--my-config=value).
func bindPFlagsSnakeCase(flags *pflag.FlagSet) {
	flags.VisitAll(func(flag *pflag.Flag) {
		_ = viper.BindPFlag(strings.ReplaceAll(flag.Name, "-", "_"), flag)
	})
}

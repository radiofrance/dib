package cmd

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/radiofrance/dib/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	defaultRegistryURL         = "eu.gcr.io/my-test-repository"
	defaultPlaceholderTag      = "latest"
	defaultLogLevel            = "info"
	defaultBuildPath           = "docker"
	defaultGossImage           = "aelsabbahy/goss:latest"
	defaultKanikoImage         = "gcr.io/kaniko-project/executor:v1.9.1"
	defaultKubernetesNamespace = "default"
)

var (
	workingDir string
	cfgFile    string
)

var rootCmd = &cobra.Command{
	Use: "dib",
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd: true,
	},
	Short: "An Opinionated DAG Image Builder",
	Long: `DAG Image Builder helps building a complex image dependency graph

Run dib --help for more information`,
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig, initLogLevel)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
		"config file (default is $HOME/.config/.dib.yaml)")
	rootCmd.PersistentFlags().String("build-path", defaultBuildPath,
		`Path to the directory containing all Dockerfiles to be built by dib. Every Dockerfile will be recursively 
found and added to the build graph. You can provide any subdirectory if you want to focus on a reduced set of images, 
as long as it has at least one Dockerfile in it.`)
	rootCmd.PersistentFlags().String("registry-url", defaultRegistryURL,
		"Docker registry URL where images are stored.")
	rootCmd.PersistentFlags().String("placeholder-tag", defaultPlaceholderTag,
		`Tag used as placeholder in Dockerfile "from" statements, and replaced internally by dib during builds 
to use the latest tags from parent images. In release mode, all images will be tagged with the placeholder tag, so 
Dockerfiles are always valid (images can still be built even without using dib).`)
	rootCmd.PersistentFlags().StringP("log-level", "l", defaultLogLevel,
		`Log level. Can be any standard log-level ("info", "debug", etc...)`)
	rootCmd.PersistentFlags().String("hash-list-file-path", "",
		"Path to custom hash list file that will be used to humanize hash")

	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		cobra.CheckErr(err)
	}

	rootCmd.AddCommand(versionCommand())
	rootCmd.AddCommand(listCommand())
	rootCmd.AddCommand(buildCommand())
	rootCmd.AddCommand(docgenCommand())
}

func initConfig() {
	var err error
	workingDir, err = os.Getwd()
	cobra.CheckErr(err)

	viper.SetConfigType("yaml")

	if cfgFile != "" {
		// Use config file from the flag.
		setConfigFile(cfgFile)
	} else if val := os.Getenv("DIB_CONFIG"); val != "" {
		// Use config file from the env variable.
		setConfigFile(val)
	} else {
		// Add $HOME/.config and current directory as paths for Viper to search for the config file in.
		homeDir, err := os.UserHomeDir()
		cobra.CheckErr(err)
		viper.AddConfigPath(path.Join(homeDir, ".config"))
		viper.AddConfigPath(workingDir)

		// Search config file with name ".dib.yaml".
		viper.SetConfigName(".dib")
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
	// Allows to override any sub-level in file config.
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	// Read in environment variables that match.
	viper.AutomaticEnv()

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		// Non-blocking, because some command does not require config file, ie: docgen.
		logger.Warnf("%s", err)
	} else {
		logger.Infof("Using config file: %s", viper.ConfigFileUsed())
	}
}

func initLogLevel() {
	logLevel := viper.GetString("log-level")
	logger.SetLevel(&logLevel)
}

func setConfigFile(name string) {
	if _, err := os.Stat(name); err != nil {
		cobra.CheckErr(fmt.Errorf("config file %q not found", name))
	}
	viper.SetConfigFile(name)
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

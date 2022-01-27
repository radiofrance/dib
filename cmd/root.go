package cmd

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	defaultRegistryURL         = "eu.gcr.io/my-test-repository"
	defaultReferentialImage    = "dib-referential"
	defaultLogLevel            = "info"
	defaultBuildPath           = "docker"
	defaultKanikoImage         = "gcr.io/kaniko-project/executor:v1.7.0"
	defaultKubernetesNamespace = "default"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "dib",
	Short: "An Opinionated Docker Image Builder",
	Long: `Docker Image Builder helps building a complex image dependency graph

Run dib --help for mor information`,
}

// Execute runs the root cobra command.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig, initLogLevel)

	desc := `Path to the directory you want to build All Dockerfiles within this directory will be recursively 
found and added to the build graph. You can provide any subdirectory if you want to focus on a reduced set of images, 
as long as it has at least one Dockerfile in it.

It is also required that one of the directories in this path contains a .docker-version file. This directory will 
be considered as the root directory for the hash generation and comparison`

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/.dib.yaml)")
	rootCmd.PersistentFlags().String("build-path", defaultBuildPath, desc)
	rootCmd.PersistentFlags().String("registry-url", defaultRegistryURL, "Docker registry URL where images are stored.")
	rootCmd.PersistentFlags().String("referential-image", defaultReferentialImage, "Name of an image on "+
		"the registry. This image will be used as a reference for checking build completion of previous dib runs. "+
		"Tags will be added to this image but it has no other purpose.")
	rootCmd.PersistentFlags().StringP("log-level", "l", defaultLogLevel, "Log level. Can be any level "+
		"supported by logrus (\"info\", \"debug\", etc...)")

	bindPFlagsSnakeCase(rootCmd.PersistentFlags())
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

type BuildOpts struct {
	BuildPath            string       `mapstructure:"build_path"`
	DisableGenerateGraph bool         `mapstructure:"no_graph"`
	DisableJunitReports  bool         `mapstructure:"no_junit_reports"`
	DisableRunTests      bool         `mapstructure:"no_tests"`
	DryRun               bool         `mapstructure:"dry_run"`
	ForceRebuild         bool         `mapstructure:"force_rebuild"`
	LocalOnly            bool         `mapstructure:"local_only"`
	ReferentialImage     string       `mapstructure:"referential_image"`
	RegistryURL          string       `mapstructure:"registry_url"`
	RetagLatest          bool         `mapstructure:"retag_latest"`
	Backend              string       `mapstructure:"backend"`
	Kaniko               KanikoConfig `mapstructure:"kaniko"`
}

// KanikoConfig holds the configuration for the Kaniko build backend.
type KanikoConfig struct {
	Context struct {
		S3 struct {
			Bucket string `mapstructure:"bucket"`
			Region string `mapstructure:"region"`
		} `mapstructure:"s3"`
	} `mapstructure:"context"`
	Executor struct {
		Docker struct {
			Image string `mapstructure:"image"`
		} `mapstructure:"docker"`
		Kubernetes struct {
			Namespace           string   `mapstructure:"namespace"`
			Image               string   `mapstructure:"image"`
			DockerConfigSecret  string   `mapstructure:"docker_config_secret"`
			ImagePullSecrets    []string `mapstructure:"image_pull_secrets"`
			EnvSecrets          []string `mapstructure:"env_secrets"`
			ContainerOverride   string   `mapstructure:"container_override"`
			PodTemplateOverride string   `mapstructure:"pod_template_override"`
		} `mapstructure:"kubernetes"`
	} `mapstructure:"executor"`
}

func buildOptsFromViper() BuildOpts {
	opts := BuildOpts{}

	// This copies all the viper values into our config struct.
	// The mapping between viper identifiers and struct field names
	// is ensured by `mapstructure` struct tags.
	_ = viper.Unmarshal(&opts)

	return opts
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
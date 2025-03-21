package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/radiofrance/dib/internal/logger"
	"github.com/radiofrance/dib/pkg/dib"
	"github.com/radiofrance/dib/pkg/docker"
	"github.com/radiofrance/dib/pkg/exec"
	"github.com/radiofrance/dib/pkg/goss"
	"github.com/radiofrance/dib/pkg/kaniko"
	"github.com/radiofrance/dib/pkg/preflight"
	"github.com/radiofrance/dib/pkg/ratelimit"
	"github.com/radiofrance/dib/pkg/registry"
	"github.com/radiofrance/dib/pkg/report"
	"github.com/radiofrance/dib/pkg/trivy"
	"github.com/radiofrance/dib/pkg/types"
	"github.com/spf13/cobra"
)

var supportedBackends = []string{
	types.BackendDocker,
	types.BackendKaniko,
}

var supportedTestsRunners = []string{
	types.TestRunnerGoss,
	types.TestRunnerTrivy,
}

var enabledTestsRunner []string

// buildCmd represents the build command.
func BuildCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "build",
		Short: "Run docker images builds",
		Long: `dib build will compute the graph of images, and compare it to the last built state.

For each image, if any file part of its docker context has changed, the image will be rebuilt.
Otherwise, dib will create a new tag based on the previous tag.`,
		RunE: buildAction,
	}
	cmd.Flags().Bool("dry-run", false,
		"Simulate what would happen without actually doing anything dangerous.")
	cmd.Flags().Bool("force-rebuild", false,
		"Forces rebuilding the entire image graph, without regarding if the target version already exists.")
	cmd.Flags().Bool("no-retag", false,
		"Disable re-tagging images after build. "+
			"Note that temporary tags with the \"dev-\" prefix may still be pushed to the registry.")
	cmd.Flags().Bool("no-graph", false,
		"Disable generation of graph during the build process.")
	cmd.Flags().Bool("no-tests", false,
		"Disable execution of tests (unit tests, scans, etc...) after the build.")
	cmd.Flags().StringSlice("include-tests", []string{},
		"List of test runners to exclude during the test phase.")
	cmd.Flags().String("reports-dir", "reports",
		"Path to the directory where the reports are generated.")
	cmd.Flags().Bool("release", false,
		"Enable release mode to tag all images with extra tags found in the `dib.extra-tags` Dockerfile labels.")
	cmd.Flags().Bool("local-only", false,
		"Build docker images locally, do not push on remote registry")
	cmd.Flags().StringP("backend", "b", types.BackendDocker,
		fmt.Sprintf("Build Backend used to run image builds. Supported backends: %v", supportedBackends))
	cmd.Flags().Int("rate-limit", 1,
		"Concurrent number of builds that can run simultaneously")
	cmd.Flags().StringArray("build-arg", []string{},
		"`argument=value` to supply to the builder")

	return cmd
}

func processBuildCommandFlag(cmd *cobra.Command, args []string) (dib.BuildOpts, error) {
	// Bind command flags to viper configuration using snake_case
	bindPFlagsSnakeCase(cmd.Flags())

	opts := dib.BuildOpts{}
	hydrateOptsFromViper(&opts)

	if opts.Backend == types.BuildKitBackend {
		// Ping the buildkit host to ensure its availability.
		// Based on the ping result, we may override the host (e.g., fallback to default buildkit host).
		var err error
		opts.BuildkitHost, err = getBuildkitHost(cmd)
		if err != nil {
			return dib.BuildOpts{}, err
		}

		buildContext := args[0]
		if buildContext == "-" || strings.Contains(buildContext, "://") {
			return dib.BuildOpts{}, fmt.Errorf("unsupported build context: %q", buildContext)
		}
	}

	return opts, nil
}

func buildAction(cmd *cobra.Command, args []string) error {
	opts, err := processBuildCommandFlag(cmd, args)
	if err != nil {
		return fmt.Errorf("error while processing build command flag: %w", err)
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

	if err := doBuild(opts, buildArgs); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}
	logger.Infof("Build process completed")
	return nil
}

func doBuild(opts dib.BuildOpts, buildArgs map[string]string) error {
	if opts.Backend == types.BackendKaniko && opts.LocalOnly {
		logger.Warnf("Using Backend \"kaniko\" with the --local-only flag is partially supported.")
	}

	checkRequirements(opts)

	workingDir, err := getWorkingDir()
	if err != nil {
		logger.Fatalf("failed to get current working directory: %v", err)
	}

	buildPath := path.Join(workingDir, opts.BuildPath)
	logger.Infof("Building images in directory \"%s\"", buildPath)

	logger.Debugf("Generate DAG")
	graph, err := dib.GenerateDAG(buildPath, opts.RegistryURL, opts.HashListFilePath, buildArgs)
	if err != nil {
		return fmt.Errorf("cannot generate DAG: %w", err)
	}
	logger.Debugf("Generate DAG -- Done")

	dibBuilder := dib.Builder{
		Version:     version,
		Graph:       graph,
		TestRunners: getTestRunners(opts, workingDir),
		BuildOpts:   opts,
	}

	gcrRegistry, err := registry.NewRegistry(opts.RegistryURL, opts.DryRun)
	if err != nil {
		return fmt.Errorf("cannot create registry: %w", err)
	}

	if err := dibBuilder.Plan(gcrRegistry); err != nil {
		return fmt.Errorf("cannot plan build: %w", err)
	}

	shell := &exec.ShellExecutor{
		Dir: workingDir,
	}
	dockerBuilderTagger := docker.NewImageBuilderTagger(shell, opts.DryRun)

	var builder types.ImageBuilder
	switch opts.Backend {
	case types.BackendDocker:
		builder = dockerBuilderTagger
	case types.BackendKaniko:
		builder = kaniko.CreateBuilder(opts.Kaniko, shell, workingDir, opts.LocalOnly, opts.DryRun)
	default:
		return fmt.Errorf("invalid backend \"%s\": not supported", opts.Backend)
	}

	res := dibBuilder.RebuildGraph(builder, ratelimit.NewChannelRateLimiter(opts.RateLimit), buildArgs)

	res.Print()
	if err := report.Generate(res, dibBuilder.Graph); err != nil {
		return fmt.Errorf("cannot generate report: %w", err)
	}

	if err := res.CheckError(); err != nil {
		return err
	}

	if opts.NoRetag {
		return nil
	}

	var tagger types.ImageTagger
	if opts.LocalOnly {
		tagger = dockerBuilderTagger
	} else {
		tagger = gcrRegistry
	}

	if err := dib.Retag(graph, tagger, opts.PlaceholderTag, opts.Release); err != nil {
		return fmt.Errorf("cannot retag images: %w", err)
	}

	return nil
}

func checkRequirements(opts dib.BuildOpts) {
	var requiredBinaries []string
	if opts.Backend == types.BackendDocker {
		requiredBinaries = []string{"docker"}
	}

	if !opts.NoTests {
		for _, includedRunner := range opts.IncludeTests {
			if !isTestRunnerEnabled(includedRunner, supportedTestsRunners) {
				logger.Fatalf(
					"invalid test runner provided: %s (available: [%s])",
					includedRunner, strings.Join(supportedTestsRunners, ","))
			}

			enabledTestsRunner = append(enabledTestsRunner, includedRunner)
			if opts.Backend == types.BackendDocker {
				requiredBinaries = append(requiredBinaries, includedRunner)
			}
		}
	}
	preflight.RunPreflightChecks(requiredBinaries)
}

func getTestRunners(opts dib.BuildOpts, workingDir string) []types.TestRunner {
	var testRunners []types.TestRunner
	if !opts.NoTests {
		if isTestRunnerEnabled(types.TestRunnerGoss, enabledTestsRunner) {
			gossRunner, err := goss.CreateTestRunner(opts.Goss, opts.LocalOnly, workingDir)
			if err != nil {
				logger.Fatalf("cannot create goss test runner: %v", err)
			}
			testRunners = append(testRunners, gossRunner)
		}
		if isTestRunnerEnabled(types.TestRunnerTrivy, enabledTestsRunner) {
			trivyRunner, err := trivy.CreateTestRunner(opts.Trivy, opts.LocalOnly, workingDir)
			if err != nil {
				logger.Fatalf("cannot create trivy test runner: %v", err)
			}
			testRunners = append(testRunners, trivyRunner)
		}
	}
	return testRunners
}

func isTestRunnerEnabled(runner string, list []string) bool {
	for _, enabled := range list {
		if runner == enabled {
			return true
		}
	}
	return false
}

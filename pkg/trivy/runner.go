package trivy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/radiofrance/dib/pkg/kubernetes"
	"github.com/radiofrance/dib/pkg/types"
	"github.com/radiofrance/kubecli"
)

var ErrCommandFailed = errors.New("trivy command failed")

// Executor is an interface for executing trivy tests.
type Executor interface {
	Execute(ctx context.Context, output io.Writer, args ...string) error
}

// TestRunner implements types.TestRunner.
type TestRunner struct {
	Executor
	TestRunnerOptions
}

// TestRunnerOptions are the configuration options for TestRunner.
type TestRunnerOptions struct {
	WorkingDirectory string
}

// Config holds the configuration for the Trivy test runner.
type Config struct {
	Executor struct {
		Kubernetes struct {
			Enabled             bool     `mapstructure:"enabled"`
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

// NewTestRunner creates a new instance of TestRunner.
func NewTestRunner(executor Executor, opts TestRunnerOptions) *TestRunner {
	return &TestRunner{executor, opts}
}

// IsConfigured returns true if a goss.yaml file is found at the target context path.
func (b *TestRunner) IsConfigured(_ types.RunTestOptions) bool {
	return true
}

// Name returns the name of the test runner.
func (b *TestRunner) Name() string {
	return types.TestRunnerTrivy
}

// RunTest executes trivy tests on the given image.
func (b *TestRunner) RunTest(ctx context.Context, opts types.RunTestOptions) error {
	var stdout bytes.Buffer

	args := []string{
		"image",
		"--quiet",
		// "--severity CRITICAL", // Filter by vulnerability type
		// "--ignore-unfixed", // ignore vulnerabilities that we can't fix even if we update all packages
		"--format",
		"json",
		opts.ImageReference,
	}

	err := os.MkdirAll(opts.ReportTrivyDir, 0o750)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", opts.ReportTrivyDir, err)
	}

	scanError := b.Execute(ctx, &stdout, args...)

	err = b.exportTrivyReport(opts, stdout.String())
	if err != nil {
		return fmt.Errorf("trivy tests failed, could not export scan report: %w", err)
	}

	if scanError != nil && !errors.Is(scanError, ErrCommandFailed) {
		return fmt.Errorf("unable to run trivy tests: %w", scanError)
	}

	if scanError != nil {
		return fmt.Errorf("trivy tests failed: %w", scanError)
	}

	return nil
}

// exportTrivyReport write stdout of Trivy scan report to json file.
func (b *TestRunner) exportTrivyReport(opts types.RunTestOptions, stdout string) error {
	trivyReportFile := path.Join(
		opts.ReportTrivyDir,
		fmt.Sprintf("%s.json", strings.ReplaceAll(opts.ImageName, "/", "_")),
	)

	err := os.WriteFile(trivyReportFile, []byte(stdout), 0o644)
	if err != nil {
		return fmt.Errorf("could not write trivy report to file %s: %w", trivyReportFile, err)
	}

	return nil
}

func CreateTestRunner(config Config, localOnly bool, workingDir string) (*TestRunner, error) {
	runnerOpts := TestRunnerOptions{
		WorkingDirectory: workingDir,
	}

	if config.Executor.Kubernetes.Enabled && !localOnly {
		executor, err := createTrivyKubernetesExecutor(config)
		if err != nil {
			return nil, err
		}

		return NewTestRunner(executor, runnerOpts), nil
	}

	return NewTestRunner(NewLocalExecutor(), runnerOpts), nil
}

func createTrivyKubernetesExecutor(cfg Config) (*KubernetesExecutor, error) {
	k8sClient, err := kubecli.New("")
	if err != nil {
		return nil, fmt.Errorf("could not get kube client from context: %w", err)
	}

	executor := NewKubernetesExecutor(k8sClient.ClientSet, kubernetes.PodConfig{
		Namespace:     cfg.Executor.Kubernetes.Namespace,
		NameGenerator: kubernetes.UniquePodName("trivy"),
		Labels: map[string]string{
			"app.kubernetes.io/managed-by": "dib",
		},
		Image:             cfg.Executor.Kubernetes.Image,
		ImagePullSecrets:  cfg.Executor.Kubernetes.ImagePullSecrets,
		EnvSecrets:        cfg.Executor.Kubernetes.EnvSecrets,
		PodOverride:       cfg.Executor.Kubernetes.PodTemplateOverride,
		ContainerOverride: cfg.Executor.Kubernetes.ContainerOverride,
	})
	executor.DockerConfigSecret = cfg.Executor.Kubernetes.DockerConfigSecret

	return executor, nil
}

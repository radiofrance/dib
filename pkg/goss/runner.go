package goss

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/radiofrance/dib/pkg/buildkit"
	"github.com/radiofrance/dib/pkg/exec"
	"github.com/radiofrance/dib/pkg/kubernetes"
	"github.com/radiofrance/dib/pkg/types"
	"github.com/radiofrance/kubecli"
)

const gossFilename = "goss.yaml"

var ErrCommandFailed = errors.New("goss command failed")

// Executor is an interface for executing goss tests.
type Executor interface {
	Execute(ctx context.Context, output io.Writer, opts types.RunTestOptions, args ...string) error
}

// TestRunner implements types.TestRunner.
type TestRunner struct {
	Executor
	TestRunnerOptions
}

// TestRunnerOptions are the configuration options for TestRunner.
type TestRunnerOptions struct {
	ReportsDirectory string
	WorkingDirectory string
}

// Config holds the configuration for the Goss test runner.
type Config struct {
	Executor struct {
		Kubernetes struct {
			Enabled           bool     `mapstructure:"enabled"`
			Namespace         string   `mapstructure:"namespace"`
			Image             string   `mapstructure:"image"`
			ImagePullSecrets  []string `mapstructure:"image_pull_secrets"`
			ContainerOverride string   `mapstructure:"container_override"`
			PodOverride       string   `mapstructure:"pod_override"`
		} `mapstructure:"kubernetes"`
	} `mapstructure:"executor"`
}

// NewTestRunner creates a new instance of TestRunner.
func NewTestRunner(executor Executor, opts TestRunnerOptions) *TestRunner {
	return &TestRunner{executor, opts}
}

// Name returns the name of the test runner.
func (b TestRunner) Name() string {
	return types.TestRunnerGoss
}

// IsConfigured returns true if a goss.yaml file is found at the target context path.
func (b TestRunner) IsConfigured(opts types.RunTestOptions) bool {
	if _, err := os.Stat(path.Join(opts.DockerContextPath, gossFilename)); err != nil {
		return false
	}

	return true
}

// RunTest executes goss tests on the given image. goss.yaml file is expected to be present in the given path.
func (b TestRunner) RunTest(opts types.RunTestOptions) error {
	if err := os.MkdirAll(opts.ReportJunitDir, 0o750); err != nil {
		return err
	}

	gossFile := path.Join(opts.DockerContextPath, gossFilename)
	if _, err := os.Stat(gossFile); err != nil {
		return fmt.Errorf("cannot run goss tests: %w", err)
	}

	var stdout bytes.Buffer
	args := []string{"--format", "junit"}

	testError := b.Execute(context.Background(), &stdout, opts, args...)

	if err := b.exportJunitReport(opts, stdout.String()); err != nil {
		return fmt.Errorf("goss tests failed, could not export junit report: %w", err)
	}

	if testError != nil {
		return fmt.Errorf("goss tests failed: %w", testError)
	}

	return nil
}

// exportJunitReport write stdout of goss tests to xml file (junit style).
func (b TestRunner) exportJunitReport(opts types.RunTestOptions, stdout string) error {
	stdout = strings.ReplaceAll(
		stdout,
		"<testcase name=\"",
		fmt.Sprintf(
			"<testcase classname=\"goss-%s\" file=\"%s\" name=\"",
			opts.ImageName,
			strings.ReplaceAll(opts.DockerContextPath, b.WorkingDirectory+"/", ""),
		),
	)

	junitFilename := path.Join(
		opts.ReportJunitDir,
		fmt.Sprintf("junit-%s.xml", strings.ReplaceAll(opts.ImageName, "/", "_")),
	)

	if err := os.WriteFile(junitFilename, []byte(stdout), 0o644); err != nil {
		return fmt.Errorf("could not write junit report to file %s: %w", junitFilename, err)
	}

	return nil
}

// DetectBuildkitContainerdWorker checks if BuildKit is using containerd as its worker.
// This is extracted to a separate function to make it easier to test.
var DetectBuildkitContainerdWorker = func() bool {
	buildctlBinary, err := buildkit.BuildctlBinary()
	if err != nil {
		return false
	}

	buildkitHost, err := buildkit.GetBuildkitHostAdress()
	if err != nil {
		return false
	}

	workerType, err := buildkit.GetBuildkitWorkerType(buildctlBinary, buildkitHost, &exec.ShellExecutor{})
	return err == nil && workerType == buildkit.ContainerdExecutorType
}

func CreateTestRunner(config Config, localOnly bool, workingDir string, backend string) (*TestRunner, error) {
	runnerOpts := TestRunnerOptions{
		WorkingDirectory: workingDir,
	}

	if config.Executor.Kubernetes.Enabled && !localOnly {
		executor, err := createGossKubernetesExecutor(config)
		if err != nil {
			return nil, err
		}
		return NewTestRunner(executor, runnerOpts), nil
	}

	// Choose executor based on backend
	// BackendDocker is deprecated from v0.25.0
	if backend == types.BackendDocker {
		return NewTestRunner(NewDGossExecutor(), runnerOpts), nil
	}

	// Use ContainerdGossExecutor if BuildKit is using containerd as its worker
	if DetectBuildkitContainerdWorker() {
		return NewTestRunner(NewContainerdGossExecutor(), runnerOpts), nil
	}

	return nil, fmt.Errorf("BuildKit is not using containerd as its worker")
}

func createGossKubernetesExecutor(cfg Config) (*KubernetesExecutor, error) {
	k8sClient, err := kubecli.New("")
	if err != nil {
		return nil, fmt.Errorf("could not get kube client from context: %w", err)
	}
	executor := NewKubernetesExecutor(*k8sClient.Config, k8sClient.ClientSet, kubernetes.PodConfig{
		NameGenerator:     kubernetes.UniquePodName("goss"),
		Namespace:         cfg.Executor.Kubernetes.Namespace,
		Image:             cfg.Executor.Kubernetes.Image,
		ImagePullSecrets:  cfg.Executor.Kubernetes.ImagePullSecrets,
		PodOverride:       cfg.Executor.Kubernetes.PodOverride,
		ContainerOverride: cfg.Executor.Kubernetes.ContainerOverride,
	})

	return executor, nil
}

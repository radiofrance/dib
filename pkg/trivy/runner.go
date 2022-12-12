package trivy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/radiofrance/dib/pkg/types"
)

var errGossCommandFailed = errors.New("trivy command failed")

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

// NewTestRunner creates a new instance of TestRunner.
func NewTestRunner(executor Executor, opts TestRunnerOptions) *TestRunner {
	return &TestRunner{executor, opts}
}

// Supports returns true if a goss.yaml file is found at the target context path.
func (b TestRunner) Supports(_ types.RunTestOptions) bool {
	return true
}

// RunTest executes trivy tests on the given image.
func (b TestRunner) RunTest(opts types.RunTestOptions) error {
	args := []string{
		"image",
		"--quiet",
		"--format",
		"json",
		opts.ImageReference,
	}

	err := os.MkdirAll(path.Join(opts.ReportRootDir, "trivy"), 0o755)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path.Join(opts.ReportRootDir, "trivy"), err)
	}
	filePath := path.Join(opts.ReportRootDir, "trivy",
		fmt.Sprintf("%s.json", strings.ReplaceAll(opts.ImageName, "/", "_")))
	fileOutput, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}

	testError := b.Executor.Execute(context.Background(), fileOutput, args...)
	if testError != nil && !errors.Is(testError, errGossCommandFailed) {
		return fmt.Errorf("unable to run trivy tests: %w", testError)
	}

	if testError != nil {
		return fmt.Errorf("trivy tests failed: %w", testError)
	}

	return nil
}

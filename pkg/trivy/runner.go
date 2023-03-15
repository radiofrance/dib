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

	"github.com/radiofrance/dib/pkg/types"
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

// NewTestRunner creates a new instance of TestRunner.
func NewTestRunner(executor Executor, opts TestRunnerOptions) *TestRunner {
	return &TestRunner{executor, opts}
}

// Supports returns true if a goss.yaml file is found at the target context path.
func (b TestRunner) Supports(_ types.RunTestOptions) bool {
	return true
}

// Name returns the name of the test runner.
func (b TestRunner) Name() string {
	return "trivy"
}

// RunTest executes trivy tests on the given image.
func (b TestRunner) RunTest(opts types.RunTestOptions) error {
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

	err := os.MkdirAll(opts.ReportTrivyDir, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", opts.ReportTrivyDir, err)
	}

	scanError := b.Executor.Execute(context.Background(), &stdout, args...)
	if err := b.exportTrivyReport(opts, stdout.String()); err != nil {
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
func (b TestRunner) exportTrivyReport(opts types.RunTestOptions, stdout string) error {
	trivyReportFile := path.Join(
		opts.ReportTrivyDir,
		fmt.Sprintf("%s.json", strings.ReplaceAll(opts.ImageName, "/", "_")),
	)
	if err := os.WriteFile(trivyReportFile, []byte(stdout), 0o644); err != nil { //nolint:gosec
		return fmt.Errorf("could not write trivy report to file %s: %w", trivyReportFile, err)
	}
	return nil
}

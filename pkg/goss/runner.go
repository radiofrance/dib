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

	"github.com/radiofrance/dib/pkg/types"
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

// NewTestRunner creates a new instance of TestRunner.
func NewTestRunner(executor Executor, opts TestRunnerOptions) *TestRunner {
	return &TestRunner{executor, opts}
}

// Name returns the name of the test runner.
func (b TestRunner) Name() string {
	return "goss"
}

// Supports returns true if a goss.yaml file is found at the target context path.
func (b TestRunner) Supports(opts types.RunTestOptions) bool {
	if _, err := os.Stat(path.Join(opts.DockerContextPath, gossFilename)); err != nil {
		return false
	}

	return true
}

// RunTest executes goss tests on the given image. goss.yaml file is expected to be present in the given path.
func (b TestRunner) RunTest(opts types.RunTestOptions) error {
	if err := os.MkdirAll(opts.ReportJunitDir, 0o755); err != nil {
		return err
	}

	gossFile := path.Join(opts.DockerContextPath, gossFilename)
	if _, err := os.Stat(gossFile); err != nil {
		return fmt.Errorf("cannot run goss tests: %w", err)
	}

	var stdout bytes.Buffer
	args := []string{"--format", "junit"}

	testError := b.Executor.Execute(context.Background(), &stdout, opts, args...)

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

	if err := os.WriteFile(junitFilename, []byte(stdout), 0o644); err != nil { //nolint:gosec
		return fmt.Errorf("could not write junit report to file %s: %w", junitFilename, err)
	}

	return nil
}

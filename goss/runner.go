package goss

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/radiofrance/dib/types"
)

const gossFilename = "goss.yaml"

var errGossCommandFailed = errors.New("goss command failed")

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
	JUnitReports     bool
}

// NewTestRunner creates a new instance of TestRunner.
func NewTestRunner(executor Executor, opts TestRunnerOptions) *TestRunner {
	return &TestRunner{executor, opts}
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
	gossFile := path.Join(opts.DockerContextPath, gossFilename)
	if _, err := os.Stat(gossFile); err != nil {
		return fmt.Errorf("cannot run goss tests: %w", err)
	}

	var (
		args   []string
		stdout bytes.Buffer
	)
	if b.JUnitReports {
		args = []string{"--format", "junit"}
	}
	testError := b.Executor.Execute(context.Background(), &stdout, opts, args...)
	if testError != nil && !errors.Is(testError, errGossCommandFailed) {
		return fmt.Errorf("unable to run goss tests: %w", testError)
	}

	if b.JUnitReports {
		if err := b.exportJunitReport(opts, stdout.String(), b); err != nil {
			return fmt.Errorf("goss tests failed, could not export junit report: %w", err)
		}
	}

	if testError != nil {
		return fmt.Errorf("goss tests failed: %w", testError)
	}

	return nil
}

func (b TestRunner) exportJunitReport(opts types.RunTestOptions, stdout string, testRunner TestRunner) error {
	stdout = strings.ReplaceAll(
		stdout,
		"<testcase name=\"",
		fmt.Sprintf(
			"<testcase classname=\"goss-%s\" file=\"%s\" name=\"",
			opts.ImageName,
			strings.ReplaceAll(opts.DockerContextPath, b.WorkingDirectory+"/", ""),
		),
	)

	if err := os.MkdirAll(testRunner.ReportsDirectory, 0o755); err != nil {
		return fmt.Errorf("could not create directory %s: %w", testRunner.ReportsDirectory, err)
	}

	junitFilename := path.Join(testRunner.ReportsDirectory, fmt.Sprintf("junit-%s.xml", opts.ImageName))
	if err := ioutil.WriteFile(junitFilename, []byte(stdout), 0o644); err != nil { // nolint: gosec
		return fmt.Errorf("could not write junit report to file %s: %w", junitFilename, err)
	}
	return nil
}

package dgoss

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/radiofrance/dib/types"

	"github.com/radiofrance/dib/exec"
)

// TestRunner implements types.TestRunner.
type TestRunner struct {
	TestRunnerOptions
}

// TestRunnerOptions are the configuration options for TestRunner.
type TestRunnerOptions struct {
	ReportsDirectory string
	WorkingDirectory string
	JUnitReports     bool
}

// NewTestRunner creates a new instance of TestRunner.
func NewTestRunner(opts TestRunnerOptions) TestRunner {
	return TestRunner{opts}
}

// RunTest executes goss tests on the given image. goss.yaml file is expected to be present in the given path.
func (b TestRunner) RunTest(opts types.RunTestOptions) error {
	shell := &exec.ShellExecutor{
		Dir: opts.DockerContextPath,
		Env: append(os.Environ(), "GOSS_OPTS=--format junit"),
	}

	if _, err := os.Stat(path.Join(opts.DockerContextPath, "goss.yaml")); err == nil {
		var stdout, stderr bytes.Buffer
		testError := shell.ExecuteWithWriters(&stdout, &stderr, "/bin/bash", "-c",
			fmt.Sprintf("dgoss run %s yes", opts.ImageReference))

		if b.JUnitReports {
			if err := b.exportJunitReport(opts, stdout.String(), b); err != nil {
				return fmt.Errorf("dgoss tests failed, could not export junit report: %w", err)
			}
		}

		if testError != nil {
			return fmt.Errorf("dgoss tests failed: %w", testError)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("dgoss tests failed: %w", err)
	}
	return nil
}

func (b TestRunner) exportJunitReport(opts types.RunTestOptions, stdout string, testRunner TestRunner) error {
	stdout = strings.ReplaceAll(stdout, "<testcase name=\"", fmt.Sprintf(
		"<testcase classname=\"goss-%s\" file=\"%s\" name=\"", opts.ImageName,
		strings.ReplaceAll(opts.DockerContextPath, b.WorkingDirectory+"/", "")))

	if err := os.MkdirAll(testRunner.ReportsDirectory, 0o755); err != nil {
		return fmt.Errorf("could not create directory %s: %w", testRunner.ReportsDirectory, err)
	}

	junitFilename := path.Join(testRunner.ReportsDirectory, fmt.Sprintf("junit-%s.xml", opts.ImageName))
	if err := ioutil.WriteFile(junitFilename, []byte(stdout), 0o644); err != nil { // nolint: gosec
		return fmt.Errorf("could not write junit report to file %s: %w", junitFilename, err)
	}
	return nil
}

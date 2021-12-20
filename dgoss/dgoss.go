package dgoss

import (
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/radiofrance/dib/exec"
)

// TestRunner implements dag.TestRunner.
type TestRunner struct{}

// RunTest executes goss tests on the given image. goss.yaml file is expected to be present in the given path.
func (b TestRunner) RunTest(ref, filePath string) error {
	shell := &exec.ShellExecutor{
		Dir: filePath,
	}

	if _, err := os.Stat(path.Join(filePath, "goss.yaml")); err == nil {
		return shell.ExecuteStdout("/bin/bash", "-c", fmt.Sprintf("dgoss run %s yes", ref))
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

package exec

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/radiofrance/dib/internal/logger"
)

// ShellExecutor is an implementation of Executor that uses the standard exec package to run shell commands.
type ShellExecutor struct {
	Dir string
	Env []string
}

// NewShellExecutor initializes a ShellExecutor with the specified working directory and environment variables.
func NewShellExecutor(workingDir string, env []string) *ShellExecutor {
	return &ShellExecutor{
		Dir: workingDir,
		Env: env,
	}
}

// Execute a shell command and return the standard output.
func (e ShellExecutor) Execute(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...) //nolint:noctx
	cmd.Env = e.Env

	var stdout, stderr bytes.Buffer

	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	cmd.Dir = e.Dir

	logger.Debugf("Exec cmd: %s", cmd)

	err := cmd.Run()
	if err != nil {
		return stderr.String(), fmt.Errorf("failed to execute command: %s: %w", cmd, err)
	}

	return stdout.String(), nil
}

// ExecuteWithWriters executes a command and forwards stdout and stderr to an io.Writer.
func (e ShellExecutor) ExecuteWithWriters(stdout, stderr io.Writer, name string, args ...string) error {
	cmd := exec.Command(name, args...) //nolint:noctx
	cmd.Env = e.Env
	cmd.Stderr = stderr
	cmd.Stdout = stdout
	cmd.Dir = e.Dir

	logger.Debugf("Exec cmd: %s", cmd)

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to execute command: %s: %w", cmd, err)
	}

	return nil
}

// ExecuteWithWriter executes a command and forwards both stdout and stderr to a single io.Writer.
func (e ShellExecutor) ExecuteWithWriter(writer io.Writer, name string, args ...string) error {
	return e.ExecuteWithWriters(writer, writer, name, args...)
}

// ExecuteStdout executes a shell command and prints to the standard output.
func (e ShellExecutor) ExecuteStdout(name string, args ...string) error {
	return e.ExecuteWithWriters(os.Stdout, os.Stderr, name, args...)
}

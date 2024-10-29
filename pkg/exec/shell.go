package exec

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/radiofrance/dib/internal/logger"
)

// Executor is an interface for dealing with command execution.
type Executor interface {
	// Execute a command and return the standard output.
	Execute(name string, args ...string) (string, error)
	// ExecuteStdout executes a command and prints the standard output instead of returning it.
	ExecuteStdout(name string, args ...string) error
	// ExecuteWithWriter executes a command and forwards both stdout and stderr to a single io.Writer
	ExecuteWithWriter(writer io.Writer, name string, args ...string) error
	// ExecuteWithWriters executes a command and forwards stdout and stderr to an io.Writer
	ExecuteWithWriters(stdout, stderr io.Writer, name string, args ...string) error
}

// ShellExecutor is an implementation of Executor that uses the standard exec package to run shell commands.
type ShellExecutor struct {
	Dir string
	Env []string
}

// Execute a shell command and return the standard output.
func (e ShellExecutor) Execute(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Env = e.Env

	var stdout, stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	cmd.Dir = e.Dir

	logger.Debugf("Exec cmd: %s", cmd)
	if err := cmd.Run(); err != nil {
		return stderr.String(), fmt.Errorf("failed to execute command: %s: %w", cmd, err)
	}

	return stdout.String(), nil
}

// ExecuteWithWriters executes a command and forwards stdout and stderr to an io.Writer.
func (e ShellExecutor) ExecuteWithWriters(stdout, stderr io.Writer, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Env = e.Env
	cmd.Stderr = stderr
	cmd.Stdout = stdout
	cmd.Dir = e.Dir

	logger.Debugf("Exec cmd: %s", cmd)
	if err := cmd.Run(); err != nil {
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

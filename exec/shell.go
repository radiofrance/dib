package exec

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

// Executor is an interface for dealing with command execution.
type Executor interface {
	// Execute a command and return the standard output.
	Execute(name string, args ...string) (string, error)
	// ExecuteStdout executes a command and prints the standard output instead of returning it.
	ExecuteStdout(name string, args ...string) error
	// ExecuteWithBuffers executes a command and forwards stdout and stderr
	ExecuteWithBuffers(stdout, stderr *bytes.Buffer, name string, args ...string) error
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

	if err := cmd.Run(); err != nil {
		return stderr.String(), fmt.Errorf("failed to execute command `%s`: %w", name, err)
	}

	return stdout.String(), nil
}

func (e ShellExecutor) ExecuteWithBuffers(stdout, stderr *bytes.Buffer, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Env = e.Env

	cmd.Stderr = stderr
	cmd.Stdout = stdout
	cmd.Dir = e.Dir

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute command `%s`: %w", name, err)
	}

	return nil
}

// ExecuteStdout executes a shell command and prints to the standard output.
func (e ShellExecutor) ExecuteStdout(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Env = e.Env

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Dir = e.Dir

	return cmd.Run()
}

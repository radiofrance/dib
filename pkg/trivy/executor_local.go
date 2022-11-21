package trivy

import (
	"context"
	"io"
	"os"
	"strings"

	"github.com/radiofrance/dib/pkg/exec"
)

const defaultShell = "/bin/bash"

// LocalExecutor executes trivy command.
type LocalExecutor struct {
	Shell string
}

// NewLocalExecutor creates a new instance of LocalExecutor.
func NewLocalExecutor() *LocalExecutor {
	shell, exists := os.LookupEnv("SHELL")
	if !exists {
		shell = defaultShell
	}

	return &LocalExecutor{
		Shell: shell,
	}
}

// Execute goss tests on the given image. goss.yaml file is expected to be present in the given path.
func (e LocalExecutor) Execute(_ context.Context, output io.Writer, args ...string) error {
	shell := &exec.ShellExecutor{}

	// We want to discard trivy logs, and only get the HTML output, that's why we set stderr to io.Discard
	return shell.ExecuteWithWriters(output, io.Discard, e.Shell, "-c", strings.Join(args, " "))
}

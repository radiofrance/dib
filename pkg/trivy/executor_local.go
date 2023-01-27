package trivy

import (
	"context"
	"fmt"
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

// Execute trivy scan on the given image.
func (e LocalExecutor) Execute(_ context.Context, output io.Writer, args ...string) error {
	shell := &exec.ShellExecutor{}

	cmd := fmt.Sprintf("trivy %s", strings.Join(args, " "))

	// We want to discard trivy logs, and only get the HTML output, that's why we set stderr to io.Discard
	return shell.ExecuteWithWriters(output, io.Discard, e.Shell, "-c", cmd)
}

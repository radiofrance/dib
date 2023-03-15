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

	return shell.ExecuteWithWriter(output, e.Shell, "-c", cmd)
}

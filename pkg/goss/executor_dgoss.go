package goss

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/radiofrance/dib/pkg/exec"
	"github.com/radiofrance/dib/pkg/types"
)

// DGossExecutor executes goss tests using the dgoss wrapper script.
type DGossExecutor struct {
	Shell string
}

// NewDGossExecutor creates a new instance of DGossExecutor.
func NewDGossExecutor() *DGossExecutor {
	shell, exists := os.LookupEnv("SHELL")
	if !exists {
		shell = defaultShell
	}

	return &DGossExecutor{
		Shell: shell,
	}
}

// Execute goss tests on the given image. goss.yaml file is expected to be present in the given path.
func (e DGossExecutor) Execute(_ context.Context, output io.Writer, opts types.RunTestOptions, args ...string) error {
	shell := &exec.ShellExecutor{
		Dir: opts.DockerContextPath,
		Env: append(os.Environ(), fmt.Sprintf("GOSS_OPTS=%s", strings.Join(args, " "))),
	}

	cmd := fmt.Sprintf("dgoss run --rm --tty --entrypoint='' %s sh", opts.ImageReference)

	return shell.ExecuteWithWriter(output, e.Shell, "-c", cmd)
}

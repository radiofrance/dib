package goss

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/radiofrance/dib/exec"
	"github.com/radiofrance/dib/types"
)

// DGossExecutor executes goss tests using the dgoss wrapper script.
type DGossExecutor struct{}

// Execute goss tests on the given image. goss.yaml file is expected to be present in the given path.
func (e DGossExecutor) Execute(_ context.Context, output io.Writer, opts types.RunTestOptions, args ...string) error {
	shell := &exec.ShellExecutor{
		Dir: opts.DockerContextPath,
		Env: append(os.Environ(), fmt.Sprintf("GOSS_OPTS=%s", strings.Join(args, " "))),
	}

	cmd := fmt.Sprintf("dgoss run %s yes", opts.ImageReference)

	return shell.ExecuteWithWriter(output, "/bin/bash", "-c", cmd)
}

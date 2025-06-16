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

const defaultShell = "/bin/bash"

// ContainerdGossExecutor executes goss tests using containerd via nerdctl.
type ContainerdGossExecutor struct {
	Shell string
}

// NewContainerdGossExecutor creates a new instance of ContainerdGossExecutor.
func NewContainerdGossExecutor() *ContainerdGossExecutor {
	shell, exists := os.LookupEnv("SHELL")
	if !exists {
		shell = defaultShell
	}

	return &ContainerdGossExecutor{
		Shell: shell,
	}
}

// Execute goss tests on the given image using nerdctl. goss.yaml file is expected to be present in the given path.
func (e ContainerdGossExecutor) Execute(
	_ context.Context,
	output io.Writer,
	opts types.RunTestOptions,
	args ...string,
) error {
	shell := &exec.ShellExecutor{
		Dir: opts.DockerContextPath,
		Env: append(os.Environ(), fmt.Sprintf("GOSS_OPTS=%s", strings.Join(args, " "))),
	}

	// Use nerdctl to run the container and execute goss inside it
	cmd := fmt.Sprintf("nerdctl run --rm --tty --entrypoint='' %s sh -c 'goss validate %s'",
		opts.ImageReference,
		strings.Join(args, " "))

	return shell.ExecuteWithWriter(output, e.Shell, "-c", cmd)
}

package executor

import (
	"context"
	"io"

	"k8s.io/apimachinery/pkg/runtime"
)

/*
Package executor defines interfaces for execution functionalities (producer)
that can be utilized by various builders, such as Kubernetes-based or shell-based executors.
*/

// KubernetesExecutor defines an interface for executing Kubernetes-based Buildkit builds.
type KubernetesExecutor interface {
	// ApplyWithWriters executes a Kubernetes Pod and streams its logs to the provided stdout and stderr writers.
	ApplyWithWriters(ctx context.Context, stdout, stderr io.Writer, k8sObject runtime.Object, containerNames string) error
}

// ShellExecutor defines an interface for executing shell commands with various output handling options.
type ShellExecutor interface {
	// Execute a command and return the standard output.
	Execute(name string, args ...string) (string, error)
	// ExecuteStdout executes a command and prints the standard output instead of returning it.
	ExecuteStdout(name string, args ...string) error
	// ExecuteWithWriter executes a command and forwards both stdout and stderr to a single io.Writer
	ExecuteWithWriter(writer io.Writer, name string, args ...string) error
	// ExecuteWithWriters executes a command and forwards stdout and stderr to an io.Writer
	ExecuteWithWriters(stdout, stderr io.Writer, name string, args ...string) error
}

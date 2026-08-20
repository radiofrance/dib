package executor

import (
	"context"
	"io"

	corev1 "k8s.io/api/core/v1"
)

// KubernetesExecutor defines an interface for executing Kubernetes-based builds.
type KubernetesExecutor interface {
	CreateAndWatchPod(ctx context.Context, stdout, stderr io.Writer, pod *corev1.Pod) error
}

// ShellExecutor defines an interface for executing shell commands with various output handling options.
type ShellExecutor interface {
	Execute(name string, args ...string) (string, error)
	ExecuteStdout(name string, args ...string) error
	ExecuteWithWriter(writer io.Writer, name string, args ...string) error
	ExecuteWithWriters(stdout, stderr io.Writer, name string, args ...string) error
}

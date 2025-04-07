package kubernetes

import (
	"fmt"
	"io"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/cmd/exec"
)

// ExecOptions wraps the exec.ExecOptions struct to add helper methods.
type ExecOptions struct {
	exec.ExecOptions
}

// NewExecOptions creates a new instance of ExecOptions with default values.
func NewExecOptions(clientSet kubernetes.Interface, restConfig rest.Config) *ExecOptions {
	restConfig.APIPath = "/api"
	restConfig.GroupVersion = &schema.GroupVersion{Group: "", Version: "v1"}
	restConfig.NegotiatedSerializer = scheme.Codecs.WithoutConversion()

	return &ExecOptions{
		exec.ExecOptions{
			StreamOptions: exec.StreamOptions{
				IOStreams: genericclioptions.IOStreams{
					In:     os.Stdin,
					Out:    os.Stdout,
					ErrOut: os.Stderr,
				},
				Stdin: false,
			},
			FilenameOptions: resource.FilenameOptions{},
			Executor:        &exec.DefaultRemoteExecutor{},
			PodClient:       clientSet.CoreV1(),
			Config:          &restConfig,
		},
	}
}

// WithContainer returns a copy of ExecOptions with pod options set to the given pod.
func (o ExecOptions) WithContainer(pod *corev1.Pod, container string) *ExecOptions {
	o.Pod = pod
	o.Namespace = pod.Namespace
	o.PodName = pod.GetName()
	o.ContainerName = container

	return &o
}

// WithWriters returns a copy of ExecOptions with the given standard output and error output writers.
func (o ExecOptions) WithWriters(out, err io.Writer) *ExecOptions {
	o.Out = out
	o.ErrOut = err

	return &o
}

// Exec executes a command in a running pod.
func Exec(o ExecOptions, cmd []string) error {
	opts := o.ExecOptions
	opts.Command = cmd
	opts.Executor = &exec.DefaultRemoteExecutor{}

	if err := opts.Validate(); err != nil {
		return fmt.Errorf("invalid exec options: %w", err)
	}

	if err := opts.Run(); err != nil {
		return fmt.Errorf("error running command: %w", err)
	}
	return nil
}

// CopyToContainer copies a file to a container in a running pod.
func CopyToContainer(opts ExecOptions, src string, dest string) error {
	file, err := os.Open(src) //nolint:gosec
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	opts.Command = []string{"tee", dest}

	opts.In = file
	opts.Out = io.Discard
	opts.Stdin = true
	opts.Executor = &exec.DefaultRemoteExecutor{}

	if err := opts.Validate(); err != nil {
		return fmt.Errorf("invalid exec options: %w", err)
	}

	if err := opts.Run(); err != nil {
		return fmt.Errorf("error running command: %w", err)
	}
	return nil
}

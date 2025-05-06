package mock

import (
	"context"
	"io"

	"k8s.io/apimachinery/pkg/runtime"
)

type ExecutorCommand struct {
	Command string
	Args    []string
	Output  string
	Error   error
}

type ExecutorResult struct {
	Output string
	Error  error
}

type ShellExecutor struct {
	Executed []ExecutorCommand
	Expected []ExecutorResult
}

func NewShellExecutor(expected []ExecutorResult) *ShellExecutor {
	return &ShellExecutor{
		Executed: []ExecutorCommand{},
		Expected: expected,
	}
}

type KubernetesExecutor struct {
	Applied  runtime.Object
	Expected runtime.Object
}

func NewKubernetesExecutor(expected runtime.Object) *KubernetesExecutor {
	return &KubernetesExecutor{
		Applied:  nil,
		Expected: expected,
	}
}

func (e *ShellExecutor) Execute(name string, args ...string) (string, error) {
	e.Executed = append(e.Executed, ExecutorCommand{
		Command: name,
		Args:    args,
	})

	if len(e.Expected) >= len(e.Executed) {
		currentIndex := len(e.Executed) - 1
		return e.Expected[currentIndex].Output, e.Expected[currentIndex].Error
	}

	return "", nil
}

func (e *ShellExecutor) ExecuteStdout(name string, args ...string) error {
	_, err := e.Execute(name, args...)
	return err
}

func (e *ShellExecutor) ExecuteWithWriters(writer, _ io.Writer, name string, args ...string) error {
	output, err := e.Execute(name, args...)
	_, _ = writer.Write([]byte(output))
	return err
}

func (e *ShellExecutor) ExecuteWithWriter(writer io.Writer, name string, args ...string) error {
	output, err := e.Execute(name, args...)
	_, _ = writer.Write([]byte(output))
	return err
}

//nolint:lll
func (m *KubernetesExecutor) ApplyWithWriters(_ context.Context, _, _ io.Writer, k8sObject runtime.Object, _ string) error {
	m.Applied = k8sObject
	return nil
}

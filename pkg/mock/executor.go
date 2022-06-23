package mock

import "io"

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

type Executor struct {
	Executed []ExecutorCommand
	Expected []ExecutorResult
}

func NewExecutor(expected []ExecutorResult) *Executor {
	return &Executor{
		Executed: []ExecutorCommand{},
		Expected: expected,
	}
}

func (e *Executor) Execute(name string, args ...string) (string, error) {
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

func (e *Executor) ExecuteStdout(name string, args ...string) error {
	_, err := e.Execute(name, args...)
	return err
}

func (e *Executor) ExecuteWithWriters(writer, _ io.Writer, name string, args ...string) error {
	output, err := e.Execute(name, args...)
	_, _ = writer.Write([]byte(output))
	return err
}

func (e *Executor) ExecuteWithWriter(writer io.Writer, name string, args ...string) error {
	output, err := e.Execute(name, args...)
	_, _ = writer.Write([]byte(output))
	return err
}

package dibtest

import (
	"os/exec"

	"github.com/containerd/nerdctl/mod/tigron/test"
	"github.com/containerd/nerdctl/mod/tigron/tig"
)

func newDibCommand(t tig.T, _ test.Config) *dibCommand {
	t.Helper()

	binary, err := exec.LookPath("dib")
	if err != nil {
		t.Fail()
	}

	genericCommand, ok := test.NewGenericCommand().(*test.GenericCommand)
	if !ok {
		t.Fail()
	}

	ret := &dibCommand{
		GenericCommand: *genericCommand,
	}

	ret.WithBinary(binary)

	return ret
}

type dibCommand struct {
	test.GenericCommand
}

func (nc *dibCommand) Run(expect *test.Expected) {
	nc.T().Helper()
	nc.GenericCommand.Run(expect)
}

func (nc *dibCommand) Background() {
	nc.GenericCommand.Background()
}

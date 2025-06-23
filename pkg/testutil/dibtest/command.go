package dibtest

import (
	"os/exec"
	"testing"

	"github.com/containerd/nerdctl/mod/tigron/test"
)

func newDibCommand(_ test.Config, t *testing.T) *dibCommand {
	binary, err := exec.LookPath("dib")
	if err != nil {
		t.Fatalf("unable to find binary 'dib': %v", err)
	}

	ret := &dibCommand{
		GenericCommand: *(test.NewGenericCommand().(*test.GenericCommand)),
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

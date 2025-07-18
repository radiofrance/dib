package dibtest

import (
	"github.com/containerd/nerdctl/mod/tigron/test"
	"github.com/containerd/nerdctl/mod/tigron/tig"
)

func Setup() *test.Case {
	test.Customize(&dibSetup{})

	return &test.Case{
		Env: map[string]string{},
	}
}

type dibSetup struct{}

func (ns *dibSetup) CustomCommand(testCase *test.Case, t tig.T) test.CustomizableCommand {
	return newDibCommand(t, testCase.Config)
}

func (ns *dibSetup) AmbientRequirements(*test.Case, tig.T) {
	// No requirements
}

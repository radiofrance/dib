package dibtest

import (
	"testing"

	"github.com/containerd/nerdctl/mod/tigron/test"
)

func Setup() *test.Case {
	test.Customize(&dibSetup{})
	return &test.Case{
		Env: map[string]string{},
	}
}

type dibSetup struct {
}

func (ns *dibSetup) CustomCommand(testCase *test.Case, t *testing.T) test.CustomizableCommand {
	return newDibCommand(testCase.Config, t)
}

func (ns *dibSetup) AmbientRequirements(testCase *test.Case, t *testing.T) {
	return
}

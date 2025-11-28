//nolint:paralleltest
package cmd_test

import (
	"testing"

	"github.com/containerd/nerdctl/mod/tigron/expect"
	"github.com/containerd/nerdctl/mod/tigron/test"
	"github.com/radiofrance/dib/pkg/testutil/dibtest"
)

func TestIntegVersion(t *testing.T) {
	testCase := dibtest.Setup()
	testCase.Command = test.Command("version")
	testCase.Expected = test.Expects(0, nil, expect.Contains("version"))
	testCase.Run(t)
}

package mock

import "github.com/radiofrance/dib/pkg/types"

type TestRunner struct {
	ReturnedError error
}

func (t *TestRunner) Name() string {
	return "testing"
}

func (t *TestRunner) IsConfigured(_ types.RunTestOptions) bool {
	return true
}

func (t *TestRunner) RunTest(_ types.RunTestOptions) error {
	return t.ReturnedError
}

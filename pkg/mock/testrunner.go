package mock

import "github.com/radiofrance/dib/pkg/types"

type TestRunner struct {
	CallCount     int
	ExpectedError error
	ShouldSupport bool
}

func (t *TestRunner) Name() string {
	return "testing"
}

func (t *TestRunner) Supports(_ types.RunTestOptions) bool {
	return t.ShouldSupport
}

func (t *TestRunner) RunTest(_ types.RunTestOptions) error {
	t.CallCount++
	return t.ExpectedError
}

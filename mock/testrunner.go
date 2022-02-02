package mock

import "github.com/radiofrance/dib/types"

type TestRunner struct {
	CallCount     int
	ExpectedError error
}

func (t *TestRunner) RunTest(_ types.RunTestOptions) error {
	t.CallCount++
	return t.ExpectedError
}

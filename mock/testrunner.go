package mock

import "github.com/radiofrance/dib/types"

type TestRunner struct {
	CallCount int
}

func (t *TestRunner) RunTest(_ types.RunTestOptions) error {
	t.CallCount++
	return nil
}

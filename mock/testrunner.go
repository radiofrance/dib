package mock

type TestRunner struct {
	CallCount int
}

func (t *TestRunner) RunTest(_, _ string) error {
	t.CallCount++
	return nil
}

package mock

type Registry struct {
	RefExistsCallCount int
	RetagCallCount     int
}

func (r *Registry) RefExists(_ string) (bool, error) {
	r.RefExistsCallCount++
	return true, nil
}

func (r *Registry) Retag(_, _ string) error {
	r.RetagCallCount++
	return nil
}

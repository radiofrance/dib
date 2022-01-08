package mock

type Registry struct {
	RefExistsCallCount int
	RetagCallCount     int
}

func (r *Registry) RefExists(_ string) (bool, error) {
	r.RefExistsCallCount++
	return true, nil
}

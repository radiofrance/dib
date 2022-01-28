package mock

type Registry struct {
	RefExistsCallCount int
	ExistingRefs       []string
	Error              error
}

func (r *Registry) RefExists(ref string) (bool, error) {
	r.RefExistsCallCount++

	for _, existingRef := range r.ExistingRefs {
		if ref == existingRef {
			return true, r.Error
		}
	}

	return false, r.Error
}

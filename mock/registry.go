package mock

import "sync"

type Registry struct {
	RefExistsCallCount int
	ExistingRefs       []string
	Error              error
	Lock               sync.Locker
}

func (r *Registry) RefExists(ref string) (bool, error) {
	r.Lock.Lock()
	defer r.Lock.Unlock()

	r.RefExistsCallCount++

	for _, existingRef := range r.ExistingRefs {
		if ref == existingRef {
			return true, r.Error
		}
	}

	return false, r.Error
}

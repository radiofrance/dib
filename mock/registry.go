package mock

import "sync"

type Registry struct {
	RefExistsCallCount int32
	ExistingRefs       []string
	Error              error
	lock               sync.Mutex
}

func (r *Registry) RefExists(ref string) (bool, error) {
	r.lock.Lock()
	r.RefExistsCallCount++
	r.lock.Unlock()

	for _, existingRef := range r.ExistingRefs {
		if ref == existingRef {
			return true, r.Error
		}
	}

	return false, r.Error
}

package mock

import (
	"slices"
	"sync"
)

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

	if slices.Contains(r.ExistingRefs, ref) {
		return true, r.Error
	}

	return false, r.Error
}

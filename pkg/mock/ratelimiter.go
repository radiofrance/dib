package mock

// RateLimiter is an noop implementation of RateLimiter.
type RateLimiter struct{}

func (r RateLimiter) Acquire() {
}

func (r RateLimiter) Release() {
}

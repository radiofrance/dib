package ratelimit

// RateLimiter is an abstraction for rate limiting.
type RateLimiter interface {
	// Acquire waits until rate limit is available for the build
	Acquire()
	// Release tells the build is done
	Release()
}

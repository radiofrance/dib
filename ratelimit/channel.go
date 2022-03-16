package ratelimit

// NewChannelRateLimiter returns an instance of ChannelRateLimiter.
func NewChannelRateLimiter(concurrency int) *ChannelRateLimiter {
	return &ChannelRateLimiter{
		limiter:   make(chan struct{}, concurrency),
		unlimited: concurrency == 0,
	}
}

// ChannelRateLimiter is an implementation of RateLimiter based on a single channel.
type ChannelRateLimiter struct {
	limiter   chan struct{}
	unlimited bool
}

// Acquire holds on the channel until it can send a message.
func (r *ChannelRateLimiter) Acquire() {
	if r.unlimited {
		return
	}
	r.limiter <- struct{}{}
}

// Release receives a message from the channel to unlock the next one.
func (r *ChannelRateLimiter) Release() {
	if r.unlimited {
		return
	}
	<-r.limiter
}

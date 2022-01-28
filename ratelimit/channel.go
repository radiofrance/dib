package ratelimit

// NewChannelRateLimiter returns an instance of ChannelRateLimiter.
func NewChannelRateLimiter(concurrency int) *ChannelRateLimiter {
	return &ChannelRateLimiter{
		limiter: make(chan struct{}, concurrency),
	}
}

// ChannelRateLimiter is an implementation of RateLimiter based on a single channel.
type ChannelRateLimiter struct {
	limiter chan struct{}
}

// Acquire holds on the channel until it can send a message.
func (r *ChannelRateLimiter) Acquire() {
	r.limiter <- struct{}{}
}

// Release receives a message from the channel to unlock the next one.
func (r *ChannelRateLimiter) Release() {
	<-r.limiter
}

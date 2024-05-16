package vpn

import (
	"container/ring"
	"sync"
	"time"
)

// RateLimiter can be used to limit requests within a specific period of time.
// initialized with Init(...) and used with Allow()
type RateLimiter struct {
	buffer    *ring.Ring
	expiresIn time.Duration
	size      int
	mutex     sync.Mutex
	now       func() time.Time
}

// NewRateLimiter initializes the RateLimiter with the current time
// expirationDuration What is the time duration each token expires in
// rateLimit is the amount of requests per expirationDiration, like 1000 requests per Day
func NewRateLimiter(expirationDuration time.Duration, rateLimit int) *RateLimiter {
	r := &RateLimiter{
		now: time.Now,
	}

	r.size = rateLimit
	r.buffer = ring.New(rateLimit)
	r.expiresIn = expirationDuration

	initialValue := r.now()

	for i := 0; i < r.size; i++ {
		r.buffer = r.buffer.Prev()
		r.buffer.Value = initialValue
	}

	return r
}

// Allow returns true if the rate has not yet been exceeded, returns false otherwise.
// Info: goroutine safe
func (r *RateLimiter) Allow() bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	now := r.now()
	nextTokenExpiresAt := r.buffer.Next().Value.(time.Time)
	// is look ahead next token expired
	if now.After(nextTokenExpiresAt) {
		// if expired, we go to the next token
		r.buffer = r.buffer.Next()
		// and inser our new expiration for the action that is going to happen after this function call
		r.buffer.Value = now.Add(r.expiresIn)
		return true
	}
	// the next token has not yet expired, so we cannot do any more requests
	return false
}

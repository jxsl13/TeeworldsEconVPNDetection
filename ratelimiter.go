package main

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
	mutex     *sync.Mutex
}

// NewRateLimiter initializes the RateLimiter with the current time
// expirationDuration What is the time duration each token expires in
// rateLimit is the amount of requests per expirationDiration, like 1000 requests per Day
func NewRateLimiter(expirationDuration time.Duration, rateLimit int) *RateLimiter {
	r := new(RateLimiter)
	r.mutex = new(sync.Mutex)
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.size = rateLimit
	r.buffer = ring.New(r.size)
	r.expiresIn = expirationDuration

	initialValue := time.Now()

	for i := 0; i < r.size; i++ {
		r.buffer = r.buffer.Prev()
		r.buffer.Value = initialValue
	}

	return r
}

// Allow returns true if the rate has not yet been exceeded, returns false otherwise.
// Info: threadsafe
func (r *RateLimiter) Allow() bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	now := time.Now()

	valueOfNextElement := r.buffer.Next().Value.(time.Time)

	timeUntilNextTokenExpires := valueOfNextElement.Sub(now)
	if timeUntilNextTokenExpires <= 0*time.Nanosecond {
		// already expired
		r.buffer = r.buffer.Next()
		// set token to the expiration value
		r.buffer.Value = now.Add(r.expiresIn)
		return true
	}
	// the next token has not yet expired, so we cannot do any more requests
	return false
}

package main

import "time"

// CooldownHandler wraps the retry cooldown
type CooldownHandler struct {
	lastRetry, cooldownSecs int64
}

// ResetCooldown : reset cooldown
func (c *CooldownHandler) ResetCooldown() {
	c.lastRetry = 0
	c.cooldownSecs = 0
}

// IncreaseCooldown : double the waiting time on each execution
func (c *CooldownHandler) IncreaseCooldown() {
	c.lastRetry = time.Now().Unix()
	if c.cooldownSecs == 0 {
		c.cooldownSecs++
	} else {
		c.cooldownSecs *= 2
	}
}

// ExpiresAt : when does out waiting time expire
func (c *CooldownHandler) ExpiresAt() int64 {
	return c.lastRetry + c.cooldownSecs
}

// CanRetry : can we retry to fetch data from the api
func (c *CooldownHandler) CanRetry() bool {
	return time.Now().Unix() >= c.ExpiresAt()
}

// RemainingCooldownSecs : how many seconds to wait until next retry
func (c *CooldownHandler) RemainingCooldownSecs() int64 {
	diff := c.ExpiresAt() - time.Now().Unix()

	if diff <= 0 {
		return 0
	}
	return diff
}

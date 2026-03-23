package channels

import (
	"sync"
	"time"
)

// RateLimiter provides per-user rate limiting.
type RateLimiter struct {
	mu       sync.Mutex
	maxMsgs  int
	windowS  int64
	counters map[string]*rateBucket
}

type rateBucket struct {
	count    int
	windowAt int64
}

// NewRateLimiter creates a limiter allowing maxMsgs per windowSeconds.
func NewRateLimiter(maxMsgs int, windowSeconds int64) *RateLimiter {
	return &RateLimiter{
		maxMsgs:  maxMsgs,
		windowS:  windowSeconds,
		counters: make(map[string]*rateBucket),
	}
}

// Allow returns true if the user hasn't exceeded the rate limit.
func (rl *RateLimiter) Allow(userKey string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now().Unix()
	bucket, ok := rl.counters[userKey]
	if !ok || now-bucket.windowAt >= rl.windowS {
		rl.counters[userKey] = &rateBucket{count: 1, windowAt: now}
		return true
	}

	if bucket.count >= rl.maxMsgs {
		return false
	}
	bucket.count++
	return true
}

// Reset clears the rate limit for a user.
func (rl *RateLimiter) Reset(userKey string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.counters, userKey)
}

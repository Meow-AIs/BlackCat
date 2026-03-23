package remote

import (
	"context"
	"sync"
	"time"
)

// RateLimiter implements a token bucket algorithm for rate limiting remote
// command execution. It is safe for concurrent use.
type RateLimiter struct {
	mu     sync.Mutex
	rate   float64   // tokens added per second
	burst  int       // maximum tokens
	tokens float64   // current available tokens
	last   time.Time // last time tokens were updated
}

// NewRateLimiter creates a RateLimiter that allows rate requests per second
// with a maximum burst size. Panics if rate <= 0 or burst <= 0.
func NewRateLimiter(rate float64, burst int) *RateLimiter {
	if rate <= 0 {
		panic("rate must be positive")
	}
	if burst <= 0 {
		panic("burst must be positive")
	}
	return &RateLimiter{
		rate:   rate,
		burst:  burst,
		tokens: float64(burst),
		last:   time.Now(),
	}
}

// Allow reports whether a single request is allowed right now. It consumes
// one token if available, returning true. Otherwise returns false without
// blocking.
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.refill()

	if rl.tokens >= 1.0 {
		rl.tokens--
		return true
	}
	return false
}

// Wait blocks until a token is available or the context is cancelled.
// Returns nil when a token was consumed, or the context error otherwise.
func (rl *RateLimiter) Wait(ctx context.Context) error {
	for {
		if rl.Allow() {
			return nil
		}

		// Calculate how long until the next token is available.
		rl.mu.Lock()
		waitDur := time.Duration(float64(time.Second) / rl.rate)
		rl.mu.Unlock()

		// Use a shorter poll interval to stay responsive.
		poll := waitDur
		if poll > 5*time.Millisecond {
			poll = 5 * time.Millisecond
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(poll):
			// Try again on next iteration.
		}
	}
}

// refill adds tokens based on elapsed time. Must be called with mu held.
func (rl *RateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(rl.last).Seconds()
	rl.last = now

	rl.tokens += elapsed * rl.rate
	if rl.tokens > float64(rl.burst) {
		rl.tokens = float64(rl.burst)
	}
}

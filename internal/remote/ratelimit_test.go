package remote

import (
	"context"
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(10.0, 5)
	if rl == nil {
		t.Fatal("expected non-nil rate limiter")
	}
}

func TestRateLimiter_AllowWithinBurst(t *testing.T) {
	rl := NewRateLimiter(1.0, 3) // 1 per second, burst of 3

	// Should allow up to burst count immediately
	for i := 0; i < 3; i++ {
		if !rl.Allow() {
			t.Errorf("expected Allow()=true for request %d within burst", i+1)
		}
	}
}

func TestRateLimiter_DenyAfterBurstExhausted(t *testing.T) {
	rl := NewRateLimiter(1.0, 2) // 1 per second, burst of 2

	// Exhaust the burst
	rl.Allow()
	rl.Allow()

	// Next should be denied (no time to refill)
	if rl.Allow() {
		t.Error("expected Allow()=false after burst exhausted")
	}
}

func TestRateLimiter_RefillAfterTime(t *testing.T) {
	rl := NewRateLimiter(100.0, 1) // 100 per second, burst of 1

	// Use the one token
	rl.Allow()

	// Wait for refill (at 100/sec, should refill in ~10ms)
	time.Sleep(20 * time.Millisecond)

	if !rl.Allow() {
		t.Error("expected Allow()=true after refill time")
	}
}

func TestRateLimiter_WaitSuccess(t *testing.T) {
	rl := NewRateLimiter(100.0, 1) // fast rate for testing

	// Use the burst
	rl.Allow()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := rl.Wait(ctx)
	if err != nil {
		t.Errorf("expected Wait to succeed, got error: %v", err)
	}
}

func TestRateLimiter_WaitContextCancelled(t *testing.T) {
	rl := NewRateLimiter(0.1, 1) // Very slow: 1 per 10 seconds

	// Exhaust burst
	rl.Allow()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := rl.Wait(ctx)
	if err == nil {
		t.Error("expected error when context is cancelled")
	}
}

func TestRateLimiter_ZeroBurstPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for zero burst")
		}
	}()
	NewRateLimiter(1.0, 0)
}

func TestRateLimiter_ZeroRatePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for zero rate")
		}
	}()
	NewRateLimiter(0, 5)
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	rl := NewRateLimiter(1000.0, 10)
	done := make(chan bool, 20)

	for i := 0; i < 20; i++ {
		go func() {
			rl.Allow()
			done <- true
		}()
	}

	for i := 0; i < 20; i++ {
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for concurrent access")
		}
	}
}

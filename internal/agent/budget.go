package agent

import "sync/atomic"

// IterationBudget is a thread-safe shared iteration counter for parent and
// child agents. Once the budget reaches zero, further Consume calls return
// false without decrementing.
type IterationBudget struct {
	remaining atomic.Int64
	max       int64
}

// NewBudget creates an IterationBudget with the given maximum. A max <= 0
// creates a budget that is immediately exhausted.
func NewBudget(max int64) *IterationBudget {
	b := &IterationBudget{max: max}
	if max > 0 {
		b.remaining.Store(max)
	}
	return b
}

// Consume atomically decrements the budget by 1 and returns true.
// Returns false without decrementing when the budget is already exhausted.
func (b *IterationBudget) Consume() bool {
	for {
		current := b.remaining.Load()
		if current <= 0 {
			return false
		}
		if b.remaining.CompareAndSwap(current, current-1) {
			return true
		}
		// Another goroutine changed the value; retry.
	}
}

// Remaining returns the number of iterations left.
func (b *IterationBudget) Remaining() int64 {
	v := b.remaining.Load()
	if v < 0 {
		return 0
	}
	return v
}

// Max returns the maximum budget set at construction.
func (b *IterationBudget) Max() int64 {
	return b.max
}

// IsExhausted returns true when no iterations remain.
func (b *IterationBudget) IsExhausted() bool {
	return b.remaining.Load() <= 0
}

// Reset restores the budget to its original maximum value.
func (b *IterationBudget) Reset() {
	if b.max > 0 {
		b.remaining.Store(b.max)
	} else {
		b.remaining.Store(0)
	}
}

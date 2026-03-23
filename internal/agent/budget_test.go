package agent

import (
	"sync"
	"testing"
)

func TestNewBudgetHasCorrectRemaining(t *testing.T) {
	b := NewBudget(10)
	if b.Remaining() != 10 {
		t.Errorf("expected remaining 10, got %d", b.Remaining())
	}
}

func TestNewBudgetMaxIsCorrect(t *testing.T) {
	b := NewBudget(42)
	if b.Max() != 42 {
		t.Errorf("expected max 42, got %d", b.Max())
	}
}

func TestConsumeDecrementsRemaining(t *testing.T) {
	b := NewBudget(5)
	ok := b.Consume()
	if !ok {
		t.Error("expected Consume to return true")
	}
	if b.Remaining() != 4 {
		t.Errorf("expected remaining 4, got %d", b.Remaining())
	}
}

func TestConsumeReturnsFalseWhenExhausted(t *testing.T) {
	b := NewBudget(1)
	if !b.Consume() {
		t.Error("first Consume should return true")
	}
	if b.Consume() {
		t.Error("second Consume should return false when exhausted")
	}
}

func TestMultipleConsumesWork(t *testing.T) {
	b := NewBudget(5)
	for i := 0; i < 5; i++ {
		if !b.Consume() {
			t.Errorf("Consume %d should return true", i+1)
		}
	}
	if b.Remaining() != 0 {
		t.Errorf("expected 0 remaining, got %d", b.Remaining())
	}
	if b.Consume() {
		t.Error("Consume after exhaustion should return false")
	}
}

func TestIsExhaustedWhenRemainingZero(t *testing.T) {
	b := NewBudget(1)
	if b.IsExhausted() {
		t.Error("should not be exhausted initially")
	}
	b.Consume()
	if !b.IsExhausted() {
		t.Error("should be exhausted after consuming all budget")
	}
}

func TestResetRestoresToMax(t *testing.T) {
	b := NewBudget(5)
	b.Consume()
	b.Consume()
	if b.Remaining() != 3 {
		t.Errorf("expected 3 remaining after 2 consumes, got %d", b.Remaining())
	}
	b.Reset()
	if b.Remaining() != 5 {
		t.Errorf("expected 5 remaining after reset, got %d", b.Remaining())
	}
}

func TestBudgetOfZeroConsumeAlwaysFalse(t *testing.T) {
	b := NewBudget(0)
	if b.Remaining() != 0 {
		t.Errorf("expected 0 remaining for zero budget, got %d", b.Remaining())
	}
	if b.Consume() {
		t.Error("Consume on zero budget should return false")
	}
}

func TestNegativeBudgetConsumeReturnsFalse(t *testing.T) {
	b := NewBudget(-5)
	if b.Consume() {
		t.Error("Consume on negative budget should return false")
	}
}

func TestNegativeBudgetIsExhausted(t *testing.T) {
	b := NewBudget(-1)
	if !b.IsExhausted() {
		t.Error("negative budget should be considered exhausted")
	}
}

func TestConcurrentConsume(t *testing.T) {
	const initial = 100
	b := NewBudget(initial)

	var wg sync.WaitGroup
	successCount := int64(0)
	var mu sync.Mutex

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if b.Consume() {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	if successCount != initial {
		t.Errorf("expected exactly %d successful consumes, got %d", initial, successCount)
	}
	if b.Remaining() != 0 {
		t.Errorf("expected 0 remaining after concurrent exhaustion, got %d", b.Remaining())
	}
}

func TestResetAfterConcurrentConsume(t *testing.T) {
	b := NewBudget(10)
	for i := 0; i < 10; i++ {
		b.Consume()
	}
	b.Reset()
	if b.Remaining() != 10 {
		t.Errorf("expected 10 after reset, got %d", b.Remaining())
	}
	if b.IsExhausted() {
		t.Error("should not be exhausted after reset")
	}
}

func TestBudgetRemainingNeverGoesNegative(t *testing.T) {
	b := NewBudget(2)
	b.Consume()
	b.Consume()
	b.Consume() // extra consume — remaining must stay at 0
	if b.Remaining() < 0 {
		t.Errorf("remaining went negative: %d", b.Remaining())
	}
}

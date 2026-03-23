package agent

import (
	"strings"
	"testing"
)

func TestNewNudgeTrackerHasZeroCounters(t *testing.T) {
	nt := NewNudgeTracker()
	if nt == nil {
		t.Fatal("expected non-nil NudgeTracker")
	}
	if nt.totalTurns != 0 {
		t.Errorf("expected totalTurns=0, got %d", nt.totalTurns)
	}
	for _, nudgeType := range []NudgeType{NudgeMemoryWrite, NudgeSkillCreate, NudgeToolDiversity} {
		if nt.counters[nudgeType] != 0 {
			t.Errorf("expected counter for %q to be 0, got %d", nudgeType, nt.counters[nudgeType])
		}
	}
}

func TestNewNudgeTrackerHasDefaultThresholds(t *testing.T) {
	nt := NewNudgeTracker()
	if nt.thresholds[NudgeMemoryWrite] != 5 {
		t.Errorf("expected MemoryWrite threshold=5, got %d", nt.thresholds[NudgeMemoryWrite])
	}
	if nt.thresholds[NudgeSkillCreate] != 10 {
		t.Errorf("expected SkillCreate threshold=10, got %d", nt.thresholds[NudgeSkillCreate])
	}
	if nt.thresholds[NudgeToolDiversity] != 8 {
		t.Errorf("expected ToolDiversity threshold=8, got %d", nt.thresholds[NudgeToolDiversity])
	}
}

func TestRecordActionResetsCounter(t *testing.T) {
	nt := NewNudgeTracker()

	// Advance 3 turns without action to build up counters
	nt.Advance()
	nt.Advance()
	nt.Advance()

	if nt.counters[NudgeMemoryWrite] != 3 {
		t.Errorf("expected counter=3 after 3 advances, got %d", nt.counters[NudgeMemoryWrite])
	}

	// Recording action should reset only that counter
	nt.RecordAction(NudgeMemoryWrite)
	if nt.counters[NudgeMemoryWrite] != 0 {
		t.Errorf("expected counter reset to 0 after RecordAction, got %d", nt.counters[NudgeMemoryWrite])
	}

	// Other counters should remain unchanged
	if nt.counters[NudgeSkillCreate] != 3 {
		t.Errorf("expected SkillCreate counter still 3, got %d", nt.counters[NudgeSkillCreate])
	}
}

func TestAdvanceIncrementsAllCounters(t *testing.T) {
	nt := NewNudgeTracker()
	nt.Advance()

	for _, nudgeType := range []NudgeType{NudgeMemoryWrite, NudgeSkillCreate, NudgeToolDiversity} {
		if nt.counters[nudgeType] != 1 {
			t.Errorf("expected counter for %q=1 after one Advance, got %d", nudgeType, nt.counters[nudgeType])
		}
	}
	if nt.totalTurns != 1 {
		t.Errorf("expected totalTurns=1, got %d", nt.totalTurns)
	}
}

func TestAdvanceIncrementsTotalTurns(t *testing.T) {
	nt := NewNudgeTracker()
	for i := 0; i < 7; i++ {
		nt.Advance()
	}
	if nt.totalTurns != 7 {
		t.Errorf("expected totalTurns=7, got %d", nt.totalTurns)
	}
}

func TestNudgeTriggeredAfterThreshold(t *testing.T) {
	nt := NewNudgeTracker()

	// Advance exactly threshold turns for MemoryWrite (threshold=5)
	var nudges []string
	for i := 0; i < 5; i++ {
		nudges = nt.Advance()
	}

	found := false
	for _, msg := range nudges {
		if strings.Contains(msg, "preferences") || strings.Contains(msg, "remembering") ||
			strings.Contains(msg, "worth remembering") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected MemoryWrite nudge after 5 turns, got nudges: %v", nudges)
	}
}

func TestNudgeNotTriggeredBeforeThreshold(t *testing.T) {
	nt := NewNudgeTracker()

	// Advance 4 turns — one before the MemoryWrite threshold of 5
	var nudges []string
	for i := 0; i < 4; i++ {
		nudges = nt.Advance()
	}

	for _, msg := range nudges {
		if strings.Contains(msg, "preferences") || strings.Contains(msg, "worth remembering") {
			t.Errorf("expected no MemoryWrite nudge before threshold, got: %q", msg)
		}
	}
}

func TestNudgeMessageContentMemoryWrite(t *testing.T) {
	nt := NewNudgeTracker()
	nt.SetThreshold(NudgeMemoryWrite, 2)

	var nudges []string
	for i := 0; i < 2; i++ {
		nudges = nt.Advance()
	}

	expected := "[System: You've had several exchanges. Consider: has the user shared preferences, project context, or coding patterns worth remembering?]"
	found := false
	for _, msg := range nudges {
		if msg == expected {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected exact MemoryWrite nudge message, got nudges: %v", nudges)
	}
}

func TestNudgeMessageContentSkillCreate(t *testing.T) {
	nt := NewNudgeTracker()
	nt.SetThreshold(NudgeSkillCreate, 2)

	var nudges []string
	for i := 0; i < 2; i++ {
		nudges = nt.Advance()
	}

	expected := "[System: The previous task involved many tool calls. Consider saving the approach as a skill for future reuse.]"
	found := false
	for _, msg := range nudges {
		if msg == expected {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected exact SkillCreate nudge message, got nudges: %v", nudges)
	}
}

func TestNudgeMessageContentToolDiversity(t *testing.T) {
	nt := NewNudgeTracker()
	nt.SetThreshold(NudgeToolDiversity, 2)

	var nudges []string
	for i := 0; i < 2; i++ {
		nudges = nt.Advance()
	}

	expected := "[System: Consider if there are other tools available that could help with this task. Check the tool list.]"
	found := false
	for _, msg := range nudges {
		if msg == expected {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected exact ToolDiversity nudge message, got nudges: %v", nudges)
	}
}

func TestCooldownPreventsReNudgingWithinNTurns(t *testing.T) {
	nt := NewNudgeTracker()
	nt.SetThreshold(NudgeMemoryWrite, 2)
	// default cooldown = 5

	// Trigger the nudge at turn 2
	for i := 0; i < 2; i++ {
		nt.Advance()
	}

	// Advance 3 more turns — still within the 5-turn cooldown window
	// The counter for MemoryWrite keeps incrementing past threshold,
	// but the nudge should NOT fire again due to cooldown.
	nudgeCount := 0
	for i := 0; i < 3; i++ {
		msgs := nt.Advance()
		for _, msg := range msgs {
			if strings.Contains(msg, "worth remembering") {
				nudgeCount++
			}
		}
	}

	if nudgeCount > 0 {
		t.Errorf("expected no repeat MemoryWrite nudge within cooldown, got %d repeats", nudgeCount)
	}
}

func TestCooldownAllowsNudgeAfterCooldownExpires(t *testing.T) {
	nt := NewNudgeTracker()
	nt.SetThreshold(NudgeMemoryWrite, 1)
	// cooldown = 5

	// Trigger nudge at turn 1
	nt.Advance()

	// Advance 5 more turns to exhaust the cooldown
	for i := 0; i < 5; i++ {
		nt.Advance()
	}

	// Turn 7: counter is now 6 turns past threshold; cooldown expired at turn 6.
	// The nudge should fire again on or after turn 6.
	found := false
	msgs := nt.Advance()
	for _, msg := range msgs {
		if strings.Contains(msg, "worth remembering") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected MemoryWrite nudge to fire again after cooldown expires")
	}
}

func TestMultipleNudgeTypesCanFireSameTurn(t *testing.T) {
	nt := NewNudgeTracker()
	// Set all thresholds to 2 so they all fire on turn 2
	nt.SetThreshold(NudgeMemoryWrite, 2)
	nt.SetThreshold(NudgeSkillCreate, 2)
	nt.SetThreshold(NudgeToolDiversity, 2)

	var nudges []string
	for i := 0; i < 2; i++ {
		nudges = nt.Advance()
	}

	if len(nudges) < 3 {
		t.Errorf("expected 3 nudges in same turn, got %d: %v", len(nudges), nudges)
	}
}

func TestSetThresholdChangesWhenNudgeFires(t *testing.T) {
	nt := NewNudgeTracker()
	nt.SetThreshold(NudgeMemoryWrite, 3)

	// Should NOT fire at turn 2 (old threshold was 5, new is 3)
	var nudges []string
	for i := 0; i < 2; i++ {
		nudges = nt.Advance()
	}
	for _, msg := range nudges {
		if strings.Contains(msg, "worth remembering") {
			t.Errorf("expected nudge not to fire at turn 2 with threshold=3")
		}
	}

	// Should fire at turn 3
	nudges = nt.Advance()
	found := false
	for _, msg := range nudges {
		if strings.Contains(msg, "worth remembering") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected MemoryWrite nudge to fire at turn 3 with threshold=3")
	}
}

func TestRecordActionForUnknownTypeDoesNotPanic(t *testing.T) {
	nt := NewNudgeTracker()
	// This should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RecordAction panicked for unknown type: %v", r)
		}
	}()
	nt.RecordAction(NudgeType("unknown_type"))
}

func TestAdvanceReturnsEmptySliceBeforeThreshold(t *testing.T) {
	nt := NewNudgeTracker()
	nudges := nt.Advance()
	if len(nudges) != 0 {
		t.Errorf("expected empty nudges on first advance, got: %v", nudges)
	}
}

func TestRecordActionResetsCounterToZeroNotNegative(t *testing.T) {
	nt := NewNudgeTracker()
	nt.RecordAction(NudgeMemoryWrite)
	if nt.counters[NudgeMemoryWrite] != 0 {
		t.Errorf("expected counter=0 after RecordAction on fresh tracker, got %d", nt.counters[NudgeMemoryWrite])
	}
}

func TestNudgeCounterResetsAfterNudgeFires(t *testing.T) {
	nt := NewNudgeTracker()
	nt.SetThreshold(NudgeMemoryWrite, 2)

	// Fire the nudge
	for i := 0; i < 2; i++ {
		nt.Advance()
	}

	// Counter should reset after nudge fires
	if nt.counters[NudgeMemoryWrite] != 0 {
		t.Errorf("expected counter reset to 0 after nudge fires, got %d", nt.counters[NudgeMemoryWrite])
	}
}

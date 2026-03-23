package agent

// NudgeType identifies a category of behavioral nudge.
type NudgeType string

const (
	NudgeMemoryWrite   NudgeType = "memory_write"
	NudgeSkillCreate   NudgeType = "skill_create"
	NudgeToolDiversity NudgeType = "tool_diversity"
)

var nudgeMessages = map[NudgeType]string{
	NudgeMemoryWrite:   "[System: You've had several exchanges. Consider: has the user shared preferences, project context, or coding patterns worth remembering?]",
	NudgeSkillCreate:   "[System: The previous task involved many tool calls. Consider saving the approach as a skill for future reuse.]",
	NudgeToolDiversity: "[System: Consider if there are other tools available that could help with this task. Check the tool list.]",
}

// NudgeTracker tracks turn counts per nudge type and fires nudges at thresholds.
type NudgeTracker struct {
	totalTurns int
	counters   map[NudgeType]int
	thresholds map[NudgeType]int
	lastFired  map[NudgeType]int // turn number when last fired
	cooldown   int               // minimum turns between repeated nudges
}

// NewNudgeTracker creates a NudgeTracker with default thresholds.
func NewNudgeTracker() *NudgeTracker {
	return &NudgeTracker{
		counters: map[NudgeType]int{
			NudgeMemoryWrite:   0,
			NudgeSkillCreate:   0,
			NudgeToolDiversity: 0,
		},
		thresholds: map[NudgeType]int{
			NudgeMemoryWrite:   5,
			NudgeSkillCreate:   10,
			NudgeToolDiversity: 8,
		},
		lastFired: map[NudgeType]int{
			NudgeMemoryWrite:   -100,
			NudgeSkillCreate:   -100,
			NudgeToolDiversity: -100,
		},
		cooldown: 5,
	}
}

// SetThreshold changes the turn threshold for a nudge type.
func (nt *NudgeTracker) SetThreshold(nudgeType NudgeType, threshold int) {
	nt.thresholds[nudgeType] = threshold
}

// RecordAction resets the counter for a nudge type (the user/agent performed the action).
func (nt *NudgeTracker) RecordAction(nudgeType NudgeType) {
	nt.counters[nudgeType] = 0
}

// Advance increments all counters and totalTurns, returning any nudge messages
// that should be injected into the conversation.
func (nt *NudgeTracker) Advance() []string {
	nt.totalTurns++
	for nudgeType := range nt.counters {
		nt.counters[nudgeType]++
	}

	var nudges []string
	for nudgeType, count := range nt.counters {
		threshold := nt.thresholds[nudgeType]
		if threshold <= 0 {
			continue
		}
		if count >= threshold && (nt.totalTurns-nt.lastFired[nudgeType]) > nt.cooldown {
			if msg, ok := nudgeMessages[nudgeType]; ok {
				nudges = append(nudges, msg)
			}
			nt.counters[nudgeType] = 0
			nt.lastFired[nudgeType] = nt.totalTurns
		}
	}
	return nudges
}

package llm

// CacheControl holds the Anthropic-style prompt caching hint.
// Type is always "ephemeral" for the rolling-window strategy.
type CacheControl struct {
	Type string `json:"type"` // "ephemeral"
}

// CacheableMessage extends Message with an optional cache hint.
// When CacheControl is non-nil, Anthropic will create a cache breakpoint
// after this message, reusing the prefix on subsequent requests.
type CacheableMessage struct {
	Message
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}

// ephemeralBreakpoint is the singleton value placed on cached messages.
var ephemeralBreakpoint = &CacheControl{Type: "ephemeral"}

// ApplyCacheBreakpoints applies the "system_and_3" caching strategy:
//   - The system message always receives a breakpoint.
//   - The last 3 non-system messages receive breakpoints (rolling window).
//
// Returns a new slice of CacheableMessage; the input slice is never mutated.
func ApplyCacheBreakpoints(messages []Message) []CacheableMessage {
	result := make([]CacheableMessage, len(messages))

	// First pass: copy messages without any cache markers.
	for i, m := range messages {
		result[i] = CacheableMessage{Message: m}
	}

	// Collect indices of non-system messages to determine the rolling window.
	var nonSystemIdx []int
	for i, m := range messages {
		if m.Role != RoleSystem {
			nonSystemIdx = append(nonSystemIdx, i)
		}
	}

	// Apply breakpoint to the system message (if present).
	for i, m := range messages {
		if m.Role == RoleSystem {
			result[i].CacheControl = ephemeralBreakpoint
		}
	}

	// Apply breakpoints to the last 3 non-system messages.
	const windowSize = 3
	start := len(nonSystemIdx) - windowSize
	if start < 0 {
		start = 0
	}
	for _, idx := range nonSystemIdx[start:] {
		result[idx].CacheControl = ephemeralBreakpoint
	}

	return result
}

// EstimateTokens gives a rough token count for a slice of messages.
// Uses the heuristic chars/4, rounded up (standard GPT/Claude approximation).
func EstimateTokens(messages []Message) int {
	totalChars := 0
	for _, m := range messages {
		totalChars += len(m.Content)
	}
	return (totalChars + 3) / 4
}

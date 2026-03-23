package agent

import (
	"fmt"
	"strings"

	"github.com/meowai/blackcat/internal/llm"
)

// CompressOptions controls compression behavior.
type CompressOptions struct {
	MaxTokens     int     // target token limit
	ProtectFirst  int     // number of turns to protect at start (default 2)
	ProtectLast   int     // number of turns to protect at end (default 4)
	TokensPerChar float64 // chars-to-tokens ratio (default 0.25)
}

// DefaultCompressOptions returns sensible defaults.
func DefaultCompressOptions() CompressOptions {
	return CompressOptions{
		MaxTokens:     8000,
		ProtectFirst:  2,
		ProtectLast:   4,
		TokensPerChar: 0.25,
	}
}

// CompressResult holds the output of compression.
type CompressResult struct {
	Messages     []llm.Message
	Compressed   bool
	OriginalSize int // estimated tokens before
	NewSize      int // estimated tokens after
	Summary      string
}

// CompressMessages compresses middle turns of a conversation.
// It preserves the system prompt, protects first N and last N turns,
// and replaces middle turns with a summary message.
// CRITICAL: Tool call/result pairs are never split.
func CompressMessages(messages []llm.Message, opts CompressOptions) CompressResult {
	if len(messages) == 0 {
		return CompressResult{}
	}

	originalSize := EstimateMessageTokens(messages)

	// No compression needed
	if originalSize <= opts.MaxTokens {
		copied := make([]llm.Message, len(messages))
		copy(copied, messages)
		return CompressResult{
			Messages:     copied,
			Compressed:   false,
			OriginalSize: originalSize,
			NewSize:      originalSize,
		}
	}

	// Single message: return as-is
	if len(messages) == 1 {
		return CompressResult{
			Messages:     []llm.Message{messages[0]},
			Compressed:   false,
			OriginalSize: originalSize,
			NewSize:      originalSize,
		}
	}

	// Separate system message(s) from conversation messages.
	var systemMsgs []llm.Message
	var convMsgs []llm.Message
	for _, m := range messages {
		if m.Role == llm.RoleSystem {
			systemMsgs = append(systemMsgs, m)
		} else {
			convMsgs = append(convMsgs, m)
		}
	}

	// If there is nothing to compress beyond system messages
	if len(convMsgs) == 0 {
		copied := make([]llm.Message, len(messages))
		copy(copied, messages)
		return CompressResult{
			Messages:     copied,
			Compressed:   false,
			OriginalSize: originalSize,
			NewSize:      originalSize,
		}
	}

	// Group conversation messages into "turn groups" that respect tool-pair integrity.
	// A turn group is either:
	//   - A plain user or assistant message (single entry)
	//   - A tool-call group: the assistant message with ToolCalls + all following tool-result messages
	groups := groupByToolPairs(convMsgs)

	protectFirst := opts.ProtectFirst
	protectLast := opts.ProtectLast

	// Clamp so we never exceed available groups
	total := len(groups)
	if protectFirst+protectLast >= total {
		// Not enough middle to compress — return unmodified
		copied := make([]llm.Message, len(messages))
		copy(copied, messages)
		return CompressResult{
			Messages:     copied,
			Compressed:   false,
			OriginalSize: originalSize,
			NewSize:      originalSize,
		}
	}

	firstGroups := groups[:protectFirst]
	middleGroups := groups[protectFirst : total-protectLast]
	lastGroups := groups[total-protectLast:]

	// Build summary from middle turns
	var summaryLines []string
	for _, grp := range middleGroups {
		for _, m := range grp {
			if m.Content != "" {
				preview := m.Content
				if len(preview) > 80 {
					preview = preview[:80] + "..."
				}
				summaryLines = append(summaryLines, fmt.Sprintf("[%s]: %s", string(m.Role), preview))
			}
		}
	}
	summary := strings.Join(summaryLines, "\n")

	summaryMsg := llm.Message{
		Role:    llm.RoleSystem,
		Content: "[CONTEXT COMPRESSED] The following turns were summarized:\n" + summary,
	}

	// Assemble final message list
	var result []llm.Message
	result = append(result, systemMsgs...)
	for _, grp := range firstGroups {
		result = append(result, grp...)
	}
	result = append(result, summaryMsg)
	for _, grp := range lastGroups {
		result = append(result, grp...)
	}

	newSize := EstimateMessageTokens(result)

	return CompressResult{
		Messages:     result,
		Compressed:   true,
		OriginalSize: originalSize,
		NewSize:      newSize,
		Summary:      summary,
	}
}

// groupByToolPairs groups conversation messages into atomic chunks.
// Each chunk is either:
//   - [assistantMsg with ToolCalls, toolResult1, toolResult2, ...] (must stay together)
//   - [singleMsg] for any other role or plain assistant message
func groupByToolPairs(msgs []llm.Message) [][]llm.Message {
	var groups [][]llm.Message
	i := 0
	for i < len(msgs) {
		m := msgs[i]
		if m.Role == llm.RoleAssistant && len(m.ToolCalls) > 0 {
			// Collect the tool-call message plus all immediately following tool-result messages
			group := []llm.Message{m}
			j := i + 1
			for j < len(msgs) && msgs[j].Role == llm.RoleTool {
				group = append(group, msgs[j])
				j++
			}
			groups = append(groups, group)
			i = j
		} else {
			groups = append(groups, []llm.Message{m})
			i++
		}
	}
	return groups
}

// FindToolPairBoundaries identifies safe split points that don't break
// tool_call / tool_result pairs. Returns indices of messages that are
// safe boundaries (not in the middle of a tool pair).
func FindToolPairBoundaries(messages []llm.Message) []int {
	if len(messages) == 0 {
		return nil
	}

	// Mark indices that belong to a tool-call group (unsafe)
	unsafe := make(map[int]bool)
	i := 0
	for i < len(messages) {
		m := messages[i]
		if m.Role == llm.RoleAssistant && len(m.ToolCalls) > 0 {
			unsafe[i] = true
			j := i + 1
			for j < len(messages) && messages[j].Role == llm.RoleTool {
				unsafe[j] = true
				j++
			}
			i = j
		} else {
			i++
		}
	}

	var boundaries []int
	for idx := range messages {
		if !unsafe[idx] {
			boundaries = append(boundaries, idx)
		}
	}
	return boundaries
}

// EstimateMessageTokens returns rough token count for a message slice.
// Uses a simple char-based heuristic: 1 token ≈ 4 characters.
func EstimateMessageTokens(messages []llm.Message) int {
	const charsPerToken = 4
	total := 0
	for _, m := range messages {
		// role overhead (~4 tokens each)
		total += 4
		total += len(m.Content) / charsPerToken
		for _, tc := range m.ToolCalls {
			total += (len(tc.Name) + len(tc.Arguments)) / charsPerToken
		}
		total += len(m.ToolCallID) / charsPerToken
	}
	return total
}

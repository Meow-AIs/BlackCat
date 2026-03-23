package agent

import (
	"strings"
	"testing"

	"github.com/meowai/blackcat/internal/llm"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func msg(role llm.Role, content string) llm.Message {
	return llm.Message{Role: role, Content: content}
}

func toolCallMsg(content string, ids ...string) llm.Message {
	calls := make([]llm.ToolCall, len(ids))
	for i, id := range ids {
		calls[i] = llm.ToolCall{ID: id, Name: "some_tool", Arguments: "{}"}
	}
	return llm.Message{Role: llm.RoleAssistant, Content: content, ToolCalls: calls}
}

func toolResultMsg(content, callID string) llm.Message {
	return llm.Message{Role: llm.RoleTool, Content: content, ToolCallID: callID}
}

// buildConversation constructs a realistic conversation of `turns` user+assistant pairs.
// Each assistant message is a plain text message (no tool calls).
func buildConversation(turns int) []llm.Message {
	msgs := []llm.Message{msg(llm.RoleSystem, "You are a helpful assistant.")}
	for i := 0; i < turns; i++ {
		msgs = append(msgs,
			msg(llm.RoleUser, strings.Repeat("user message content ", 20)),
			msg(llm.RoleAssistant, strings.Repeat("assistant reply content ", 20)),
		)
	}
	return msgs
}

// ---------------------------------------------------------------------------
// DefaultCompressOptions
// ---------------------------------------------------------------------------

func TestDefaultCompressOptions(t *testing.T) {
	opts := DefaultCompressOptions()

	if opts.MaxTokens <= 0 {
		t.Errorf("MaxTokens should be positive, got %d", opts.MaxTokens)
	}
	if opts.ProtectFirst < 0 {
		t.Errorf("ProtectFirst should be >= 0, got %d", opts.ProtectFirst)
	}
	if opts.ProtectLast < 0 {
		t.Errorf("ProtectLast should be >= 0, got %d", opts.ProtectLast)
	}
	if opts.TokensPerChar <= 0 {
		t.Errorf("TokensPerChar should be positive, got %f", opts.TokensPerChar)
	}
	// Spec says defaults are ProtectFirst=2, ProtectLast=4, TokensPerChar=0.25
	if opts.ProtectFirst != 2 {
		t.Errorf("ProtectFirst default = %d, want 2", opts.ProtectFirst)
	}
	if opts.ProtectLast != 4 {
		t.Errorf("ProtectLast default = %d, want 4", opts.ProtectLast)
	}
	if opts.TokensPerChar != 0.25 {
		t.Errorf("TokensPerChar default = %f, want 0.25", opts.TokensPerChar)
	}
}

// ---------------------------------------------------------------------------
// EstimateMessageTokens
// ---------------------------------------------------------------------------

func TestEstimateMessageTokens_Empty(t *testing.T) {
	n := EstimateMessageTokens(nil)
	if n != 0 {
		t.Errorf("empty slice: got %d tokens, want 0", n)
	}
}

func TestEstimateMessageTokens_SingleMessage(t *testing.T) {
	// 100-char content @ 0.25 ratio = ~25 tokens
	m := msg(llm.RoleUser, strings.Repeat("a", 100))
	n := EstimateMessageTokens([]llm.Message{m})
	if n <= 0 {
		t.Errorf("expected positive token count for non-empty message, got %d", n)
	}
}

func TestEstimateMessageTokens_GrowsWithContent(t *testing.T) {
	short := EstimateMessageTokens([]llm.Message{msg(llm.RoleUser, "hi")})
	long := EstimateMessageTokens([]llm.Message{msg(llm.RoleUser, strings.Repeat("x", 1000))})
	if long <= short {
		t.Errorf("longer message should have more tokens: short=%d, long=%d", short, long)
	}
}

// ---------------------------------------------------------------------------
// CompressMessages — short conversation (no compression needed)
// ---------------------------------------------------------------------------

func TestCompressMessages_ShortConversation_NotCompressed(t *testing.T) {
	msgs := buildConversation(2) // system + 4 messages = small
	opts := DefaultCompressOptions()
	opts.MaxTokens = 100_000 // well above any real estimate

	result := CompressMessages(msgs, opts)

	if result.Compressed {
		t.Error("short conversation should not be compressed")
	}
	if len(result.Messages) != len(msgs) {
		t.Errorf("messages altered: got %d, want %d", len(result.Messages), len(msgs))
	}
}

// ---------------------------------------------------------------------------
// CompressMessages — long conversation (compression triggered)
// ---------------------------------------------------------------------------

func TestCompressMessages_LongConversation_IsCompressed(t *testing.T) {
	msgs := buildConversation(20) // 41 messages
	opts := DefaultCompressOptions()
	opts.MaxTokens = 50          // force compression
	opts.ProtectFirst = 2
	opts.ProtectLast = 4

	result := CompressMessages(msgs, opts)

	if !result.Compressed {
		t.Error("long conversation should be compressed")
	}
	if result.OriginalSize <= result.NewSize {
		t.Errorf("OriginalSize %d should exceed NewSize %d", result.OriginalSize, result.NewSize)
	}
}

// ---------------------------------------------------------------------------
// CompressMessages — system prompt always preserved
// ---------------------------------------------------------------------------

func TestCompressMessages_SystemMessageAlwaysPreserved(t *testing.T) {
	msgs := buildConversation(20)
	opts := DefaultCompressOptions()
	opts.MaxTokens = 50

	result := CompressMessages(msgs, opts)

	if len(result.Messages) == 0 {
		t.Fatal("result has no messages")
	}
	first := result.Messages[0]
	if first.Role != llm.RoleSystem {
		t.Errorf("first message role = %q, want system", first.Role)
	}
	if first.Content != msgs[0].Content {
		t.Errorf("system message content changed: got %q, want %q", first.Content, msgs[0].Content)
	}
}

// ---------------------------------------------------------------------------
// CompressMessages — summary message format
// ---------------------------------------------------------------------------

func TestCompressMessages_SummaryMessage_HasCorrectFormat(t *testing.T) {
	msgs := buildConversation(20)
	opts := DefaultCompressOptions()
	opts.MaxTokens = 50
	opts.ProtectFirst = 2
	opts.ProtectLast = 4

	result := CompressMessages(msgs, opts)

	if !result.Compressed {
		t.Skip("not compressed, cannot test summary")
	}

	// Find the summary message (should be a system-role message with [CONTEXT COMPRESSED])
	var found bool
	for _, m := range result.Messages {
		if m.Role == llm.RoleSystem && strings.Contains(m.Content, "[CONTEXT COMPRESSED]") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("no summary message with role=system and [CONTEXT COMPRESSED] prefix found in: %v", result.Messages)
	}
}

func TestCompressMessages_SummaryField_NotEmpty(t *testing.T) {
	msgs := buildConversation(20)
	opts := DefaultCompressOptions()
	opts.MaxTokens = 50

	result := CompressMessages(msgs, opts)

	if result.Compressed && result.Summary == "" {
		t.Error("Summary field should not be empty when compressed")
	}
}

// ---------------------------------------------------------------------------
// CompressMessages — ProtectFirst / ProtectLast exact counts
// ---------------------------------------------------------------------------

func TestCompressMessages_ProtectFirst2_ProtectLast4(t *testing.T) {
	// Build: system + 10 turns = 21 messages
	msgs := buildConversation(10)
	opts := CompressOptions{
		MaxTokens:     50,
		ProtectFirst:  2,
		ProtectLast:   4,
		TokensPerChar: 0.25,
	}

	result := CompressMessages(msgs, opts)

	if !result.Compressed {
		t.Skip("not compressed")
	}

	// system always at index 0
	if result.Messages[0].Role != llm.RoleSystem {
		t.Error("first message must be system")
	}

	// The real conversation messages (non-system) after system prompt.
	// We protect first 2 turns = first 4 non-system messages (user+assistant pairs).
	// We protect last 4 turns = last 8 non-system messages.
	// Locate the summary message — it should appear between the protected groups.
	var summaryIdx int = -1
	for i, m := range result.Messages {
		if m.Role == llm.RoleSystem && strings.Contains(m.Content, "[CONTEXT COMPRESSED]") {
			summaryIdx = i
			break
		}
	}
	if summaryIdx == -1 {
		t.Fatal("no summary message found")
	}

	// Messages before summary (after system) should be the protected first turns
	before := result.Messages[1:summaryIdx]
	after := result.Messages[summaryIdx+1:]

	if len(before) == 0 {
		t.Error("expected some protected-first messages before summary")
	}
	if len(after) == 0 {
		t.Error("expected some protected-last messages after summary")
	}
}

// ---------------------------------------------------------------------------
// CompressMessages — empty input
// ---------------------------------------------------------------------------

func TestCompressMessages_EmptyMessages_ReturnsEmpty(t *testing.T) {
	result := CompressMessages(nil, DefaultCompressOptions())
	if len(result.Messages) != 0 {
		t.Errorf("expected empty messages, got %d", len(result.Messages))
	}
	if result.Compressed {
		t.Error("empty messages should not report compressed")
	}
}

// ---------------------------------------------------------------------------
// CompressMessages — single message
// ---------------------------------------------------------------------------

func TestCompressMessages_SingleMessage_ReturnedAsIs(t *testing.T) {
	single := []llm.Message{msg(llm.RoleSystem, "system")}
	result := CompressMessages(single, DefaultCompressOptions())
	if len(result.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(result.Messages))
	}
	if result.Compressed {
		t.Error("single message should not be compressed")
	}
}

// ---------------------------------------------------------------------------
// CompressMessages — tool pairs never split
// ---------------------------------------------------------------------------

func TestCompressMessages_ToolPairs_NeverSplit(t *testing.T) {
	// Build a conversation where the middle contains a tool call pair.
	// Even under heavy compression the pair must stay together.
	msgs := []llm.Message{
		msg(llm.RoleSystem, "system prompt"),
		// turn 1 (protected first)
		msg(llm.RoleUser, strings.Repeat("u1 ", 30)),
		msg(llm.RoleAssistant, strings.Repeat("a1 ", 30)),
		// turn 2 (protected first)
		msg(llm.RoleUser, strings.Repeat("u2 ", 30)),
		msg(llm.RoleAssistant, strings.Repeat("a2 ", 30)),
		// turn 3 — middle, has tool call that should be kept with its result
		msg(llm.RoleUser, strings.Repeat("u3 ", 30)),
		toolCallMsg("calling tool", "tc-1"),
		toolResultMsg("tool result", "tc-1"),
		// turn 4 — middle
		msg(llm.RoleUser, strings.Repeat("u4 ", 30)),
		msg(llm.RoleAssistant, strings.Repeat("a4 ", 30)),
		// last 4 turns (protected last)
		msg(llm.RoleUser, strings.Repeat("u5 ", 30)),
		msg(llm.RoleAssistant, strings.Repeat("a5 ", 30)),
		msg(llm.RoleUser, strings.Repeat("u6 ", 30)),
		msg(llm.RoleAssistant, strings.Repeat("a6 ", 30)),
		msg(llm.RoleUser, strings.Repeat("u7 ", 30)),
		msg(llm.RoleAssistant, strings.Repeat("a7 ", 30)),
		msg(llm.RoleUser, strings.Repeat("u8 ", 30)),
		msg(llm.RoleAssistant, strings.Repeat("a8 ", 30)),
	}

	opts := CompressOptions{
		MaxTokens:     50,
		ProtectFirst:  2,
		ProtectLast:   4,
		TokensPerChar: 0.25,
	}
	result := CompressMessages(msgs, opts)

	// Verify that tool_call message and its tool_result are either BOTH present
	// or BOTH absent — never one without the other.
	var hasCall, hasResult bool
	for _, m := range result.Messages {
		if m.Role == llm.RoleAssistant && len(m.ToolCalls) > 0 && m.ToolCalls[0].ID == "tc-1" {
			hasCall = true
		}
		if m.Role == llm.RoleTool && m.ToolCallID == "tc-1" {
			hasResult = true
		}
	}
	if hasCall != hasResult {
		t.Errorf("tool pair integrity violated: hasCall=%v hasResult=%v", hasCall, hasResult)
	}
}

// ---------------------------------------------------------------------------
// CompressMessages — multi-tool call (consecutive tool results grouped)
// ---------------------------------------------------------------------------

func TestCompressMessages_MultiToolCall_GroupedCorrectly(t *testing.T) {
	msgs := []llm.Message{
		msg(llm.RoleSystem, "system"),
		msg(llm.RoleUser, "do many things"),
		// assistant calls two tools at once
		toolCallMsg("calling two tools", "tc-a", "tc-b"),
		toolResultMsg("result a", "tc-a"),
		toolResultMsg("result b", "tc-b"),
		// final assistant response
		msg(llm.RoleAssistant, strings.Repeat("done ", 20)),
	}

	opts := CompressOptions{
		MaxTokens:     50,
		ProtectFirst:  0,
		ProtectLast:   1,
		TokensPerChar: 0.25,
	}
	result := CompressMessages(msgs, opts)

	// Count how many of the pair-members made it into the result
	var hasCall, hasResultA, hasResultB bool
	for _, m := range result.Messages {
		if m.Role == llm.RoleAssistant && len(m.ToolCalls) >= 2 {
			hasCall = true
		}
		if m.Role == llm.RoleTool && m.ToolCallID == "tc-a" {
			hasResultA = true
		}
		if m.Role == llm.RoleTool && m.ToolCallID == "tc-b" {
			hasResultB = true
		}
	}

	// All three must be present together or all absent together
	if hasCall != hasResultA || hasCall != hasResultB {
		t.Errorf("multi-tool group split: hasCall=%v hasResultA=%v hasResultB=%v",
			hasCall, hasResultA, hasResultB)
	}
}

// ---------------------------------------------------------------------------
// CompressMessages — OriginalSize and NewSize are reasonable
// ---------------------------------------------------------------------------

func TestCompressMessages_SizeEstimates_AreReasonable(t *testing.T) {
	msgs := buildConversation(20)
	opts := DefaultCompressOptions()
	opts.MaxTokens = 50

	result := CompressMessages(msgs, opts)

	if result.OriginalSize <= 0 {
		t.Errorf("OriginalSize should be positive, got %d", result.OriginalSize)
	}
	if result.Compressed && result.NewSize <= 0 {
		t.Errorf("NewSize should be positive when compressed, got %d", result.NewSize)
	}
}

// ---------------------------------------------------------------------------
// FindToolPairBoundaries
// ---------------------------------------------------------------------------

func TestFindToolPairBoundaries_NoTools(t *testing.T) {
	msgs := []llm.Message{
		msg(llm.RoleSystem, "system"),
		msg(llm.RoleUser, "hello"),
		msg(llm.RoleAssistant, "hi"),
		msg(llm.RoleUser, "how are you"),
		msg(llm.RoleAssistant, "fine"),
	}

	boundaries := FindToolPairBoundaries(msgs)

	// Every index should be a safe boundary (no tool calls)
	if len(boundaries) != len(msgs) {
		t.Errorf("expected %d boundaries (all safe), got %d", len(msgs), len(boundaries))
	}
}

func TestFindToolPairBoundaries_WithToolPair(t *testing.T) {
	msgs := []llm.Message{
		msg(llm.RoleSystem, "system"),           // 0 — safe
		msg(llm.RoleUser, "do something"),        // 1 — safe
		toolCallMsg("calling", "tc-1"),            // 2 — NOT safe (start of pair)
		toolResultMsg("result", "tc-1"),           // 3 — NOT safe (inside pair)
		msg(llm.RoleAssistant, "done"),            // 4 — safe
	}

	boundaries := FindToolPairBoundaries(msgs)

	// Indices 2 and 3 should NOT appear as safe boundaries
	for _, idx := range boundaries {
		if idx == 2 || idx == 3 {
			t.Errorf("index %d should not be a safe boundary (it is part of a tool pair)", idx)
		}
	}

	// Indices 0, 1, 4 should all be safe
	safeSet := make(map[int]bool)
	for _, idx := range boundaries {
		safeSet[idx] = true
	}
	for _, expected := range []int{0, 1, 4} {
		if !safeSet[expected] {
			t.Errorf("index %d should be a safe boundary", expected)
		}
	}
}

func TestFindToolPairBoundaries_MultiToolCall(t *testing.T) {
	msgs := []llm.Message{
		msg(llm.RoleUser, "hello"),           // 0 — safe
		toolCallMsg("tools", "tc-a", "tc-b"), // 1 — NOT safe
		toolResultMsg("a", "tc-a"),           // 2 — NOT safe
		toolResultMsg("b", "tc-b"),           // 3 — NOT safe
		msg(llm.RoleAssistant, "done"),       // 4 — safe
	}

	boundaries := FindToolPairBoundaries(msgs)

	safeSet := make(map[int]bool)
	for _, idx := range boundaries {
		safeSet[idx] = true
	}

	for _, unsafe := range []int{1, 2, 3} {
		if safeSet[unsafe] {
			t.Errorf("index %d should NOT be a safe boundary", unsafe)
		}
	}
	for _, safe := range []int{0, 4} {
		if !safeSet[safe] {
			t.Errorf("index %d should be a safe boundary", safe)
		}
	}
}

func TestFindToolPairBoundaries_Empty(t *testing.T) {
	boundaries := FindToolPairBoundaries(nil)
	if len(boundaries) != 0 {
		t.Errorf("expected 0 boundaries for nil input, got %d", len(boundaries))
	}
}

func TestFindToolPairBoundaries_ConsecutiveToolPairs(t *testing.T) {
	msgs := []llm.Message{
		msg(llm.RoleUser, "start"),           // 0 safe
		toolCallMsg("first", "tc-1"),         // 1 unsafe
		toolResultMsg("r1", "tc-1"),          // 2 unsafe
		toolCallMsg("second", "tc-2"),        // 3 unsafe
		toolResultMsg("r2", "tc-2"),          // 4 unsafe
		msg(llm.RoleAssistant, "done"),       // 5 safe
	}

	boundaries := FindToolPairBoundaries(msgs)

	safeSet := make(map[int]bool)
	for _, idx := range boundaries {
		safeSet[idx] = true
	}

	for _, unsafe := range []int{1, 2, 3, 4} {
		if safeSet[unsafe] {
			t.Errorf("index %d should NOT be a safe boundary", unsafe)
		}
	}
	for _, safe := range []int{0, 5} {
		if !safeSet[safe] {
			t.Errorf("index %d should be a safe boundary", safe)
		}
	}
}

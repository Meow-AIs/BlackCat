package llm

import (
	"testing"
)

// ---------------------------------------------------------------------------
// ApplyCacheBreakpoints tests
// ---------------------------------------------------------------------------

func TestApplyCacheBreakpoints_EmptyInput(t *testing.T) {
	result := ApplyCacheBreakpoints(nil)
	if result == nil || len(result) != 0 {
		t.Errorf("ApplyCacheBreakpoints(nil) = %v, want empty non-nil slice", result)
	}

	result = ApplyCacheBreakpoints([]Message{})
	if len(result) != 0 {
		t.Errorf("ApplyCacheBreakpoints([]) len = %d, want 0", len(result))
	}
}

func TestApplyCacheBreakpoints_SingleSystemMessage(t *testing.T) {
	msgs := []Message{
		{Role: RoleSystem, Content: "You are a helpful assistant."},
	}

	result := ApplyCacheBreakpoints(msgs)

	if len(result) != 1 {
		t.Fatalf("len = %d, want 1", len(result))
	}
	if result[0].CacheControl == nil {
		t.Error("system message should have CacheControl set")
	}
	if result[0].CacheControl.Type != "ephemeral" {
		t.Errorf("CacheControl.Type = %q, want %q", result[0].CacheControl.Type, "ephemeral")
	}
}

func TestApplyCacheBreakpoints_SystemPlusFiveMessages(t *testing.T) {
	msgs := []Message{
		{Role: RoleSystem, Content: "system prompt"},
		{Role: RoleUser, Content: "msg1"},
		{Role: RoleAssistant, Content: "msg2"},
		{Role: RoleUser, Content: "msg3"},
		{Role: RoleAssistant, Content: "msg4"},
		{Role: RoleUser, Content: "msg5"},
	}

	result := ApplyCacheBreakpoints(msgs)

	if len(result) != len(msgs) {
		t.Fatalf("len = %d, want %d", len(result), len(msgs))
	}

	// System message (index 0) must have cache breakpoint.
	if result[0].CacheControl == nil {
		t.Error("system message (index 0) must have CacheControl")
	}

	// Non-system messages: last 3 are indices 3, 4, 5 (msg3, msg4, msg5).
	// First 2 non-system messages (indices 1, 2) must NOT have cache breakpoints.
	if result[1].CacheControl != nil {
		t.Error("index 1 (msg1) should NOT have CacheControl")
	}
	if result[2].CacheControl != nil {
		t.Error("index 2 (msg2) should NOT have CacheControl")
	}

	// Last 3 non-system messages must have cache breakpoints.
	if result[3].CacheControl == nil {
		t.Error("index 3 (msg3) should have CacheControl (last-3 window)")
	}
	if result[4].CacheControl == nil {
		t.Error("index 4 (msg4) should have CacheControl (last-3 window)")
	}
	if result[5].CacheControl == nil {
		t.Error("index 5 (msg5) should have CacheControl (last-3 window)")
	}
}

func TestApplyCacheBreakpoints_ExactlyThreeNonSystemMessages(t *testing.T) {
	msgs := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleUser, Content: "u1"},
		{Role: RoleAssistant, Content: "a1"},
		{Role: RoleUser, Content: "u2"},
	}

	result := ApplyCacheBreakpoints(msgs)

	// All 3 non-system messages should get cache breakpoints.
	for i := 1; i <= 3; i++ {
		if result[i].CacheControl == nil {
			t.Errorf("index %d should have CacheControl (all 3 non-system messages)", i)
		}
	}
}

func TestApplyCacheBreakpoints_FewerThanThreeNonSystem(t *testing.T) {
	msgs := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleUser, Content: "only one"},
	}

	result := ApplyCacheBreakpoints(msgs)

	if len(result) != 2 {
		t.Fatalf("len = %d, want 2", len(result))
	}
	if result[0].CacheControl == nil {
		t.Error("system message should have CacheControl")
	}
	if result[1].CacheControl == nil {
		t.Error("single non-system message should have CacheControl (falls within last-3 window)")
	}
}

func TestApplyCacheBreakpoints_ImmutabilityCheck(t *testing.T) {
	original := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleUser, Content: "u1"},
		{Role: RoleAssistant, Content: "a1"},
	}

	// Make a deep copy of original to compare later.
	origCopy := make([]Message, len(original))
	copy(origCopy, original)

	ApplyCacheBreakpoints(original)

	// Verify original is unchanged.
	for i, m := range original {
		if m.Role != origCopy[i].Role || m.Content != origCopy[i].Content {
			t.Errorf("original[%d] was mutated: got {%v %q}, want {%v %q}",
				i, m.Role, m.Content, origCopy[i].Role, origCopy[i].Content)
		}
	}
}

func TestApplyCacheBreakpoints_ToolMessagesCountedInWindow(t *testing.T) {
	// Tool messages are non-system and should count in the rolling window.
	msgs := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleUser, Content: "u1"},
		{Role: RoleAssistant, Content: "a1"},
		{Role: RoleTool, Content: "tool-result", ToolCallID: "call_1"},
		{Role: RoleAssistant, Content: "a2"},
		{Role: RoleUser, Content: "u2"},
	}
	// Non-system messages: u1(idx1), a1(idx2), tool(idx3), a2(idx4), u2(idx5)
	// Last 3 non-system: idx3, idx4, idx5

	result := ApplyCacheBreakpoints(msgs)

	if len(result) != len(msgs) {
		t.Fatalf("len = %d, want %d", len(result), len(msgs))
	}

	// First 2 non-system (u1, a1) should NOT have cache.
	if result[1].CacheControl != nil {
		t.Error("index 1 (u1) should NOT have CacheControl")
	}
	if result[2].CacheControl != nil {
		t.Error("index 2 (a1) should NOT have CacheControl")
	}

	// Last 3 non-system (tool, a2, u2) should have cache.
	if result[3].CacheControl == nil {
		t.Error("index 3 (tool) should have CacheControl (last-3 window)")
	}
	if result[4].CacheControl == nil {
		t.Error("index 4 (a2) should have CacheControl (last-3 window)")
	}
	if result[5].CacheControl == nil {
		t.Error("index 5 (u2) should have CacheControl (last-3 window)")
	}
}

func TestApplyCacheBreakpoints_OnlyNonSystemMessages_NoSystem(t *testing.T) {
	// No system message at all — only last 3 non-system should get breakpoints.
	msgs := []Message{
		{Role: RoleUser, Content: "u1"},
		{Role: RoleAssistant, Content: "a1"},
		{Role: RoleUser, Content: "u2"},
		{Role: RoleAssistant, Content: "a2"},
		{Role: RoleUser, Content: "u3"},
	}

	result := ApplyCacheBreakpoints(msgs)

	// u1 and a1 (first 2) should NOT have cache.
	if result[0].CacheControl != nil {
		t.Error("index 0 (u1) should NOT have CacheControl")
	}
	if result[1].CacheControl != nil {
		t.Error("index 1 (a1) should NOT have CacheControl")
	}

	// Last 3: u2, a2, u3 should have cache.
	if result[2].CacheControl == nil {
		t.Error("index 2 (u2) should have CacheControl")
	}
	if result[3].CacheControl == nil {
		t.Error("index 3 (a2) should have CacheControl")
	}
	if result[4].CacheControl == nil {
		t.Error("index 4 (u3) should have CacheControl")
	}
}

func TestApplyCacheBreakpoints_CacheControlTypeIsEphemeral(t *testing.T) {
	msgs := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleUser, Content: "hello"},
	}
	result := ApplyCacheBreakpoints(msgs)
	for i, m := range result {
		if m.CacheControl != nil && m.CacheControl.Type != "ephemeral" {
			t.Errorf("index %d: CacheControl.Type = %q, want %q", i, m.CacheControl.Type, "ephemeral")
		}
	}
}

func TestApplyCacheBreakpoints_PreservesMessageContent(t *testing.T) {
	msgs := []Message{
		{Role: RoleSystem, Content: "system text", ToolCallID: ""},
		{Role: RoleUser, Content: "user text"},
		{Role: RoleTool, Content: "tool result", ToolCallID: "call_42"},
	}

	result := ApplyCacheBreakpoints(msgs)

	for i, orig := range msgs {
		got := result[i]
		if got.Role != orig.Role {
			t.Errorf("index %d: Role = %v, want %v", i, got.Role, orig.Role)
		}
		if got.Content != orig.Content {
			t.Errorf("index %d: Content = %q, want %q", i, got.Content, orig.Content)
		}
		if got.ToolCallID != orig.ToolCallID {
			t.Errorf("index %d: ToolCallID = %q, want %q", i, got.ToolCallID, orig.ToolCallID)
		}
	}
}

// ---------------------------------------------------------------------------
// EstimateTokens tests
// ---------------------------------------------------------------------------

func TestEstimateTokens_Empty(t *testing.T) {
	got := EstimateTokens(nil)
	if got != 0 {
		t.Errorf("EstimateTokens(nil) = %d, want 0", got)
	}

	got = EstimateTokens([]Message{})
	if got != 0 {
		t.Errorf("EstimateTokens([]) = %d, want 0", got)
	}
}

func TestEstimateTokens_SingleMessage(t *testing.T) {
	// "hello" is 5 chars → ceil(5/4) = 2 tokens
	msgs := []Message{{Role: RoleUser, Content: "hello"}}
	got := EstimateTokens(msgs)
	// chars/4 rounded up: (5 + 3) / 4 = 2
	if got < 1 {
		t.Errorf("EstimateTokens([hello]) = %d, want >= 1", got)
	}
}

func TestEstimateTokens_MultipleMessages(t *testing.T) {
	msgs := []Message{
		{Role: RoleSystem, Content: "You are helpful."}, // 16 chars
		{Role: RoleUser, Content: "Hello world!"},       // 12 chars
	}
	// total chars = 28, tokens = ceil(28/4) = 7
	got := EstimateTokens(msgs)
	want := (28 + 3) / 4 // = 7
	if got != want {
		t.Errorf("EstimateTokens = %d, want %d", got, want)
	}
}

func TestEstimateTokens_ExactMultipleOfFour(t *testing.T) {
	// "abcd" is 4 chars → exactly 1 token
	msgs := []Message{{Role: RoleUser, Content: "abcd"}}
	got := EstimateTokens(msgs)
	if got != 1 {
		t.Errorf("EstimateTokens([abcd]) = %d, want 1", got)
	}
}

func TestEstimateTokens_LargeMessage(t *testing.T) {
	// 400 chars → 100 tokens exactly
	content := ""
	for i := 0; i < 100; i++ {
		content += "abcd"
	}
	msgs := []Message{{Role: RoleUser, Content: content}}
	got := EstimateTokens(msgs)
	if got != 100 {
		t.Errorf("EstimateTokens(400-char) = %d, want 100", got)
	}
}

func TestEstimateTokens_ReasonableApproximation(t *testing.T) {
	// A realistic prompt; just verify the rough chars/4 contract holds.
	msgs := []Message{
		{Role: RoleSystem, Content: "You are a helpful assistant specialized in Go programming."},
		{Role: RoleUser, Content: "What is the difference between a goroutine and a thread?"},
	}
	totalChars := len(msgs[0].Content) + len(msgs[1].Content)
	got := EstimateTokens(msgs)
	lower := totalChars / 4
	upper := (totalChars + 3) / 4

	if got < lower || got > upper+1 {
		t.Errorf("EstimateTokens = %d, expected roughly %d-%d (chars/4)", got, lower, upper)
	}
}

func TestEstimateTokens_IncludesAllRoles(t *testing.T) {
	// Tokens should include content from ALL message roles.
	msgs := []Message{
		{Role: RoleSystem, Content: "sys"},       // 3 chars
		{Role: RoleUser, Content: "usr"},         // 3 chars
		{Role: RoleAssistant, Content: "ast"},    // 3 chars
		{Role: RoleTool, Content: "tool result"}, // 11 chars
	}
	// total = 20 chars → ceil(20/4) = 5 tokens
	got := EstimateTokens(msgs)
	want := (20 + 3) / 4 // = 5
	if got != want {
		t.Errorf("EstimateTokens(all-roles) = %d, want %d", got, want)
	}
}

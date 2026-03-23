package agent

import (
	"strings"
	"testing"

	"github.com/meowai/blackcat/internal/llm"
	"github.com/meowai/blackcat/internal/tools"
)

func TestNewReasoner(t *testing.T) {
	r := NewReasoner("You are helpful.", 4096)
	if r == nil {
		t.Fatal("NewReasoner returned nil")
	}
	if r.systemPrompt != "You are helpful." {
		t.Errorf("systemPrompt = %q, want %q", r.systemPrompt, "You are helpful.")
	}
	if r.maxTokens != 4096 {
		t.Errorf("maxTokens = %d, want 4096", r.maxTokens)
	}
}

func TestBuildMessages_SystemPromptFirst(t *testing.T) {
	r := NewReasoner("System prompt here.", 4096)

	history := []llm.Message{
		{Role: llm.RoleUser, Content: "Hello"},
		{Role: llm.RoleAssistant, Content: "Hi there"},
	}

	msgs := r.BuildMessages(history, "", nil)

	if len(msgs) < 3 {
		t.Fatalf("got %d messages, want >= 3", len(msgs))
	}
	if msgs[0].Role != llm.RoleSystem {
		t.Errorf("first message role = %q, want %q", msgs[0].Role, llm.RoleSystem)
	}
	if !strings.Contains(msgs[0].Content, "System prompt here.") {
		t.Error("system prompt not in first message")
	}
}

func TestBuildMessages_MemorySnapshotInjected(t *testing.T) {
	r := NewReasoner("Base prompt.", 4096)

	history := []llm.Message{
		{Role: llm.RoleUser, Content: "What did we discuss?"},
	}

	msgs := r.BuildMessages(history, "Previous discussion about Go testing.", nil)

	if !strings.Contains(msgs[0].Content, "Previous discussion about Go testing.") {
		t.Error("memory snapshot not injected into system message")
	}
}

func TestBuildMessages_NoMemorySnapshot(t *testing.T) {
	r := NewReasoner("Base prompt.", 4096)

	history := []llm.Message{
		{Role: llm.RoleUser, Content: "Hello"},
	}

	msgs := r.BuildMessages(history, "", nil)

	// System message should just be the base prompt without "Memory" section
	if strings.Contains(msgs[0].Content, "## Memory") {
		t.Error("empty snapshot should not add Memory section")
	}
}

func TestBuildMessages_ToolDefinitionsIncluded(t *testing.T) {
	r := NewReasoner("Base prompt.", 4096)

	defs := []tools.Definition{
		{Name: "read_file", Description: "Read a file"},
		{Name: "write_file", Description: "Write a file"},
	}

	msgs := r.BuildMessages(nil, "", defs)

	// System message should reference available tools
	if !strings.Contains(msgs[0].Content, "read_file") {
		t.Error("tool definitions not included in system prompt")
	}
}

func TestBuildMessages_DoesNotMutateHistory(t *testing.T) {
	r := NewReasoner("Prompt.", 4096)

	history := []llm.Message{
		{Role: llm.RoleUser, Content: "Hello"},
	}
	origLen := len(history)

	_ = r.BuildMessages(history, "snapshot", nil)

	if len(history) != origLen {
		t.Error("BuildMessages mutated the history slice")
	}
}

func TestInjectContext(t *testing.T) {
	r := NewReasoner("Base.", 4096)

	messages := []llm.Message{
		{Role: llm.RoleSystem, Content: "Base."},
		{Role: llm.RoleUser, Content: "Hello"},
	}

	injected := r.InjectContext(messages, "Extra context here.")

	// Should not mutate original
	if len(messages) != 2 {
		t.Error("InjectContext mutated original messages")
	}

	// First message should have extra context
	if !strings.Contains(injected[0].Content, "Extra context here.") {
		t.Error("context not injected into system message")
	}
}

func TestInjectContext_EmptyContext(t *testing.T) {
	r := NewReasoner("Base.", 4096)

	messages := []llm.Message{
		{Role: llm.RoleSystem, Content: "Base."},
	}

	injected := r.InjectContext(messages, "")

	if injected[0].Content != "Base." {
		t.Error("empty context should not modify system message")
	}
}

func TestExtractToolCalls(t *testing.T) {
	r := NewReasoner("Base.", 4096)

	resp := llm.ChatResponse{
		Content: "Let me read that file.",
		ToolCalls: []llm.ToolCall{
			{ID: "tc1", Name: "read_file", Arguments: `{"path": "main.go"}`},
			{ID: "tc2", Name: "execute", Arguments: `{"command": "ls"}`},
		},
	}

	calls := r.ExtractToolCalls(resp)

	if len(calls) != 2 {
		t.Fatalf("got %d calls, want 2", len(calls))
	}

	if calls[0].Name != "read_file" {
		t.Errorf("call 0 Name = %q, want %q", calls[0].Name, "read_file")
	}
	if calls[0].Args["path"] != "main.go" {
		t.Errorf("call 0 path arg = %v, want %q", calls[0].Args["path"], "main.go")
	}

	if calls[1].Name != "execute" {
		t.Errorf("call 1 Name = %q, want %q", calls[1].Name, "execute")
	}
}

func TestExtractToolCalls_NoToolCalls(t *testing.T) {
	r := NewReasoner("Base.", 4096)

	resp := llm.ChatResponse{Content: "Just text."}
	calls := r.ExtractToolCalls(resp)

	if len(calls) != 0 {
		t.Errorf("got %d calls, want 0", len(calls))
	}
}

func TestExtractToolCalls_InvalidArguments(t *testing.T) {
	r := NewReasoner("Base.", 4096)

	resp := llm.ChatResponse{
		ToolCalls: []llm.ToolCall{
			{ID: "tc1", Name: "read_file", Arguments: "not json"},
		},
	}

	calls := r.ExtractToolCalls(resp)
	if len(calls) != 1 {
		t.Fatalf("got %d calls, want 1", len(calls))
	}
	// Should still return the call with empty args
	if calls[0].Name != "read_file" {
		t.Errorf("Name = %q, want %q", calls[0].Name, "read_file")
	}
	if calls[0].Args == nil {
		t.Error("Args should be empty map, not nil")
	}
}

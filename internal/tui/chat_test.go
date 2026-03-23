package tui

import (
	"strings"
	"testing"
	"time"
)

func TestNewChatModel(t *testing.T) {
	m := NewChatModel(100)
	if m == nil {
		t.Fatal("expected non-nil ChatModel")
	}
	if m.MaxHistory != 100 {
		t.Errorf("expected MaxHistory 100, got %d", m.MaxHistory)
	}
	if m.MessageCount() != 0 {
		t.Errorf("expected 0 messages, got %d", m.MessageCount())
	}
}

func TestChatModelAddMessage(t *testing.T) {
	m := NewChatModel(100)
	m.AddMessage(ChatMessage{
		Role:      "user",
		Content:   "hello",
		Timestamp: time.Now(),
	})
	if m.MessageCount() != 1 {
		t.Errorf("expected 1 message, got %d", m.MessageCount())
	}
}

func TestChatModelMaxHistoryTruncation(t *testing.T) {
	m := NewChatModel(3)
	for i := 0; i < 5; i++ {
		m.AddMessage(ChatMessage{
			Role:    "user",
			Content: "msg",
		})
	}
	if m.MessageCount() != 3 {
		t.Errorf("expected 3 messages after truncation, got %d", m.MessageCount())
	}
}

func TestChatModelClear(t *testing.T) {
	m := NewChatModel(100)
	m.AddMessage(ChatMessage{Role: "user", Content: "hello"})
	m.AddMessage(ChatMessage{Role: "assistant", Content: "hi"})
	m.Clear()
	if m.MessageCount() != 0 {
		t.Errorf("expected 0 messages after clear, got %d", m.MessageCount())
	}
}

func TestChatModelRender(t *testing.T) {
	m := NewChatModel(100)
	m.AddMessage(ChatMessage{
		Role:      "user",
		Content:   "hello",
		Timestamp: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
	})
	m.AddMessage(ChatMessage{
		Role:      "assistant",
		Content:   "hi there",
		Timestamp: time.Date(2025, 1, 1, 12, 0, 1, 0, time.UTC),
	})

	output := m.Render(80, 24)
	if !strings.Contains(output, "hello") {
		t.Error("expected render to contain 'hello'")
	}
	if !strings.Contains(output, "hi there") {
		t.Error("expected render to contain 'hi there'")
	}
}

func TestChatModelRenderToolMessage(t *testing.T) {
	m := NewChatModel(100)
	m.AddMessage(ChatMessage{
		Role:     "tool",
		Content:  "executed successfully",
		ToolName: "bash",
	})

	output := m.Render(80, 24)
	if !strings.Contains(output, "bash") {
		t.Error("expected render to contain tool name 'bash'")
	}
}

func TestChatModelRenderRoles(t *testing.T) {
	m := NewChatModel(100)
	m.AddMessage(ChatMessage{Role: "user", Content: "question"})
	m.AddMessage(ChatMessage{Role: "assistant", Content: "answer"})
	m.AddMessage(ChatMessage{Role: "system", Content: "notice"})

	output := m.Render(80, 24)
	if !strings.Contains(output, "You") {
		t.Error("expected 'You' label for user role")
	}
	if !strings.Contains(output, "BlackCat") {
		t.Error("expected 'BlackCat' label for assistant role")
	}
	if !strings.Contains(output, "System") {
		t.Error("expected 'System' label for system role")
	}
}

func TestChatModelRenderEmpty(t *testing.T) {
	m := NewChatModel(100)
	output := m.Render(80, 24)
	// Should not panic, should return something
	if output == "" {
		t.Error("expected non-empty render for empty chat (welcome message)")
	}
}

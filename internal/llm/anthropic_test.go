package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAnthropicProviderChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Errorf("expected /v1/messages, got %s", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("expected x-api-key 'test-key', got %q", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("expected anthropic-version header")
		}

		var reqBody map[string]any
		json.NewDecoder(r.Body).Decode(&reqBody)
		if reqBody["model"] != "claude-sonnet-4-6" {
			t.Errorf("expected model 'claude-sonnet-4-6', got %v", reqBody["model"])
		}

		resp := `{
			"id": "msg_123",
			"type": "message",
			"role": "assistant",
			"content": [{"type": "text", "text": "Hello from Claude!"}],
			"model": "claude-sonnet-4-6",
			"stop_reason": "end_turn",
			"usage": {"input_tokens": 12, "output_tokens": 8}
		}`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(resp))
	}))
	defer server.Close()

	provider := NewAnthropicProvider("test-key", server.URL)

	resp, err := provider.Chat(context.Background(), ChatRequest{
		Model: "claude-sonnet-4-6",
		Messages: []Message{
			{Role: RoleSystem, Content: "You are BlackCat"},
			{Role: RoleUser, Content: "Hello"},
		},
	})
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if resp.Content != "Hello from Claude!" {
		t.Errorf("expected 'Hello from Claude!', got %q", resp.Content)
	}
	if resp.Usage.TotalTokens != 20 {
		t.Errorf("expected 20 total tokens, got %d", resp.Usage.TotalTokens)
	}
}

func TestAnthropicProviderToolUse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := `{
			"id": "msg_456",
			"type": "message",
			"role": "assistant",
			"content": [
				{"type": "text", "text": "Let me read that file."},
				{"type": "tool_use", "id": "toolu_01", "name": "read_file", "input": {"path": "main.go"}}
			],
			"model": "claude-sonnet-4-6",
			"stop_reason": "tool_use",
			"usage": {"input_tokens": 20, "output_tokens": 15}
		}`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(resp))
	}))
	defer server.Close()

	provider := NewAnthropicProvider("test-key", server.URL)
	resp, err := provider.Chat(context.Background(), ChatRequest{
		Model:    "claude-sonnet-4-6",
		Messages: []Message{{Role: RoleUser, Content: "read main.go"}},
	})
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if resp.Content != "Let me read that file." {
		t.Errorf("expected text content, got %q", resp.Content)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "read_file" {
		t.Errorf("expected tool 'read_file', got %q", resp.ToolCalls[0].Name)
	}
	if resp.FinishReason != "tool_use" {
		t.Errorf("expected stop_reason 'tool_use', got %q", resp.FinishReason)
	}
}

func TestAnthropicProviderName(t *testing.T) {
	p := NewAnthropicProvider("key", "")
	if p.Name() != "anthropic" {
		t.Errorf("expected 'anthropic', got %q", p.Name())
	}
}

func TestAnthropicProviderServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		w.Write([]byte(`{"error": {"message": "rate limited"}}`))
	}))
	defer server.Close()

	provider := NewAnthropicProvider("key", server.URL)
	_, err := provider.Chat(context.Background(), ChatRequest{
		Model:    "claude-sonnet-4-6",
		Messages: []Message{{Role: RoleUser, Content: "hi"}},
	})
	if err == nil {
		t.Error("expected error for 429, got nil")
	}
}

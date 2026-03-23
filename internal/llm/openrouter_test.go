package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewOpenRouterProvider_Defaults(t *testing.T) {
	p := NewOpenRouterProvider("sk-test-key")
	if p.Name() != "openrouter" {
		t.Errorf("expected name 'openrouter', got %s", p.Name())
	}
	if p.baseURL != "https://openrouter.ai/api/v1" {
		t.Errorf("expected default base URL, got %s", p.baseURL)
	}
	if p.apiKey != "sk-test-key" {
		t.Errorf("expected api key 'sk-test-key', got %s", p.apiKey)
	}
}

func TestNewOpenRouterProvider_WithOptions(t *testing.T) {
	p := NewOpenRouterProvider("sk-key",
		WithReferer("https://myapp.com"),
		WithTitle("My App"),
		WithBaseURL("https://custom.endpoint/v1"),
	)
	if p.referer != "https://myapp.com" {
		t.Errorf("expected referer, got %s", p.referer)
	}
	if p.title != "My App" {
		t.Errorf("expected title, got %s", p.title)
	}
	if p.baseURL != "https://custom.endpoint/v1" {
		t.Errorf("expected custom base URL, got %s", p.baseURL)
	}
}

func TestOpenRouterProvider_Chat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}

		// Verify headers
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Error("expected Bearer auth header")
		}
		if r.Header.Get("HTTP-Referer") != "https://example.com" {
			t.Errorf("expected HTTP-Referer header, got %s", r.Header.Get("HTTP-Referer"))
		}
		if r.Header.Get("X-Title") != "TestApp" {
			t.Errorf("expected X-Title header, got %s", r.Header.Get("X-Title"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("expected application/json content type")
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)

		// Verify model is passed through
		if model, ok := reqBody["model"].(string); !ok || model != "anthropic/claude-sonnet-4-6" {
			t.Errorf("expected model 'anthropic/claude-sonnet-4-6', got %v", reqBody["model"])
		}

		resp := openaiResponse{
			ID:    "gen-123",
			Model: "anthropic/claude-sonnet-4-6",
		}
		resp.Choices = []struct {
			Index   int `json:"index"`
			Message struct {
				Role      string           `json:"role"`
				Content   string           `json:"content"`
				ToolCalls []openaiToolCall `json:"tool_calls,omitempty"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}{
			{
				Index: 0,
				Message: struct {
					Role      string           `json:"role"`
					Content   string           `json:"content"`
					ToolCalls []openaiToolCall `json:"tool_calls,omitempty"`
				}{
					Role:    "assistant",
					Content: "I am Claude via OpenRouter.",
				},
				FinishReason: "stop",
			},
		}
		resp.Usage = struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{
			PromptTokens:     15,
			CompletionTokens: 8,
			TotalTokens:      23,
		}

		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewOpenRouterProvider("sk-test",
		WithReferer("https://example.com"),
		WithTitle("TestApp"),
		WithBaseURL(server.URL),
	)

	resp, err := p.Chat(context.Background(), ChatRequest{
		Model: "anthropic/claude-sonnet-4-6",
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "I am Claude via OpenRouter." {
		t.Errorf("unexpected content: %s", resp.Content)
	}
	if resp.Model != "anthropic/claude-sonnet-4-6" {
		t.Errorf("unexpected model: %s", resp.Model)
	}
	if resp.Usage.TotalTokens != 23 {
		t.Errorf("expected 23 total tokens, got %d", resp.Usage.TotalTokens)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("expected finish_reason 'stop', got %s", resp.FinishReason)
	}
}

func TestOpenRouterProvider_ChatWithTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)

		// Verify tools mapped in OpenAI format
		tools, ok := reqBody["tools"].([]any)
		if !ok || len(tools) != 1 {
			t.Fatalf("expected 1 tool, got %v", reqBody["tools"])
		}

		tool := tools[0].(map[string]any)
		if tool["type"] != "function" {
			t.Errorf("expected tool type 'function', got %v", tool["type"])
		}

		resp := map[string]any{
			"id":    "gen-456",
			"model": "openai/gpt-4.1",
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": "",
						"tool_calls": []map[string]any{
							{
								"id":   "call_abc",
								"type": "function",
								"function": map[string]any{
									"name":      "search",
									"arguments": `{"query":"test"}`,
								},
							},
						},
					},
					"finish_reason": "tool_calls",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     20,
				"completion_tokens": 12,
				"total_tokens":      32,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewOpenRouterProvider("sk-test", WithBaseURL(server.URL))
	resp, err := p.Chat(context.Background(), ChatRequest{
		Model: "openai/gpt-4.1",
		Messages: []Message{
			{Role: RoleUser, Content: "Search for test"},
		},
		Tools: []ToolDefinition{
			{Name: "search", Description: "Search the web", Parameters: map[string]any{"type": "object"}},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].ID != "call_abc" {
		t.Errorf("expected tool call ID 'call_abc', got %s", resp.ToolCalls[0].ID)
	}
	if resp.ToolCalls[0].Name != "search" {
		t.Errorf("expected tool name 'search', got %s", resp.ToolCalls[0].Name)
	}
}

func TestOpenRouterProvider_ChatAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": {"message": "rate limit exceeded"}}`))
	}))
	defer server.Close()

	p := NewOpenRouterProvider("sk-test", WithBaseURL(server.URL))
	_, err := p.Chat(context.Background(), ChatRequest{
		Model:    "openai/gpt-4.1",
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})

	if err == nil {
		t.Fatal("expected error for 429 response")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("expected status code in error, got: %s", err.Error())
	}
}

func TestOpenRouterProvider_Stream(t *testing.T) {
	sseLines := []string{
		`data: {"id":"gen-1","choices":[{"delta":{"content":"Hello"},"index":0}]}`,
		`data: {"id":"gen-1","choices":[{"delta":{"content":" world"},"index":0}]}`,
		`data: {"id":"gen-1","choices":[{"delta":{},"index":0,"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":7}}`,
		`data: [DONE]`,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)

		if stream, ok := reqBody["stream"].(bool); !ok || !stream {
			t.Error("expected stream=true")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		for _, line := range sseLines {
			w.Write([]byte(line + "\n\n"))
			flusher.Flush()
		}
	}))
	defer server.Close()

	p := NewOpenRouterProvider("sk-test", WithBaseURL(server.URL))
	ch, err := p.Stream(context.Background(), ChatRequest{
		Model:    "openai/gpt-4.1",
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var collected []StreamChunk
	timeout := time.After(5 * time.Second)
	for {
		select {
		case chunk, ok := <-ch:
			if !ok {
				goto done
			}
			collected = append(collected, chunk)
		case <-timeout:
			t.Fatal("timeout waiting for stream chunks")
		}
	}
done:

	if len(collected) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(collected))
	}
	if collected[0].Content != "Hello" {
		t.Errorf("expected first chunk 'Hello', got %s", collected[0].Content)
	}
	if collected[1].Content != " world" {
		t.Errorf("expected second chunk ' world', got %s", collected[1].Content)
	}

	// Last chunk should be done
	last := collected[len(collected)-1]
	if !last.Done {
		t.Error("expected last chunk to be done")
	}
}

func TestOpenRouterProvider_Models(t *testing.T) {
	p := NewOpenRouterProvider("sk-test")
	models := p.Models()

	if len(models) == 0 {
		t.Fatal("expected hardcoded models")
	}

	// Check some expected models exist
	modelIDs := make(map[string]bool)
	for _, m := range models {
		modelIDs[m.ID] = true
	}

	expectedModels := []string{
		"anthropic/claude-sonnet-4-6",
		"openai/gpt-4.1",
		"deepseek/deepseek-chat",
	}
	for _, id := range expectedModels {
		if !modelIDs[id] {
			t.Errorf("expected model %s not found", id)
		}
	}
}

func TestOpenRouterProvider_NoExtraHeadersWhenEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// When referer/title not set, headers should be absent
		if r.Header.Get("HTTP-Referer") != "" {
			t.Error("expected no HTTP-Referer header when not set")
		}
		if r.Header.Get("X-Title") != "" {
			t.Error("expected no X-Title header when not set")
		}

		resp := map[string]any{
			"id":    "gen-1",
			"model": "test",
			"choices": []map[string]any{
				{"message": map[string]any{"role": "assistant", "content": "ok"}, "finish_reason": "stop"},
			},
			"usage": map[string]any{"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewOpenRouterProvider("sk-test", WithBaseURL(server.URL))
	_, err := p.Chat(context.Background(), ChatRequest{
		Model:    "test",
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

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

func TestNewXAIProvider_Defaults(t *testing.T) {
	p := NewXAIProvider("xai-test-key")
	if p.Name() != "xai" {
		t.Errorf("expected name 'xai', got %s", p.Name())
	}
	if p.baseURL != "https://api.x.ai/v1" {
		t.Errorf("expected default base URL, got %s", p.baseURL)
	}
	if p.apiKey != "xai-test-key" {
		t.Errorf("expected api key 'xai-test-key', got %s", p.apiKey)
	}
}

func TestNewXAIProvider_WithBaseURL(t *testing.T) {
	p := NewXAIProvider("xai-key", WithXAIBaseURL("https://custom.x.ai/v2"))
	if p.baseURL != "https://custom.x.ai/v2" {
		t.Errorf("expected custom base URL, got %s", p.baseURL)
	}
}

func TestXAIProvider_Chat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer xai-test" {
			t.Errorf("expected Bearer auth, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("expected application/json content type")
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)
		if reqBody["model"] != "grok-3" {
			t.Errorf("expected model 'grok-3', got %v", reqBody["model"])
		}

		resp := map[string]any{
			"id":    "chatcmpl-xai-1",
			"model": "grok-3",
			"choices": []map[string]any{
				{
					"index":         0,
					"message":       map[string]any{"role": "assistant", "content": "Hello from Grok!"},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{"prompt_tokens": 8, "completion_tokens": 4, "total_tokens": 12},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewXAIProvider("xai-test", WithXAIBaseURL(server.URL))
	resp, err := p.Chat(context.Background(), ChatRequest{
		Model:    "grok-3",
		Messages: []Message{{Role: RoleUser, Content: "Hello"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "Hello from Grok!" {
		t.Errorf("expected 'Hello from Grok!', got %q", resp.Content)
	}
	if resp.Model != "grok-3" {
		t.Errorf("expected model 'grok-3', got %q", resp.Model)
	}
	if resp.Usage.TotalTokens != 12 {
		t.Errorf("expected 12 total tokens, got %d", resp.Usage.TotalTokens)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("expected finish_reason 'stop', got %q", resp.FinishReason)
	}
}

func TestXAIProvider_ChatWithToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"id":    "chatcmpl-xai-2",
			"model": "grok-4-1-fast-latest",
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": "",
						"tool_calls": []map[string]any{
							{
								"id":   "call_grok_1",
								"type": "function",
								"function": map[string]any{
									"name":      "read_file",
									"arguments": `{"path":"main.go"}`,
								},
							},
						},
					},
					"finish_reason": "tool_calls",
				},
			},
			"usage": map[string]any{"prompt_tokens": 20, "completion_tokens": 10, "total_tokens": 30},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewXAIProvider("xai-test", WithXAIBaseURL(server.URL))
	resp, err := p.Chat(context.Background(), ChatRequest{
		Model:    "grok-4-1-fast-latest",
		Messages: []Message{{Role: RoleUser, Content: "Read main.go"}},
		Tools: []ToolDefinition{
			{Name: "read_file", Description: "Read a file", Parameters: map[string]any{"type": "object"}},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].ID != "call_grok_1" {
		t.Errorf("expected tool call ID 'call_grok_1', got %s", resp.ToolCalls[0].ID)
	}
	if resp.ToolCalls[0].Name != "read_file" {
		t.Errorf("expected tool name 'read_file', got %s", resp.ToolCalls[0].Name)
	}
	if resp.FinishReason != "tool_calls" {
		t.Errorf("expected finish_reason 'tool_calls', got %q", resp.FinishReason)
	}
}

func TestXAIProvider_ChatError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error": {"message": "forbidden"}}`))
	}))
	defer server.Close()

	p := NewXAIProvider("bad-key", WithXAIBaseURL(server.URL))
	_, err := p.Chat(context.Background(), ChatRequest{
		Model:    "grok-3",
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("expected status code in error, got: %s", err.Error())
	}
}

func TestXAIProvider_Stream(t *testing.T) {
	sseLines := []string{
		`data: {"id":"gen-1","choices":[{"delta":{"content":"Hello"},"index":0}]}`,
		`data: {"id":"gen-1","choices":[{"delta":{"content":" Grok"},"index":0}]}`,
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

	p := NewXAIProvider("xai-test", WithXAIBaseURL(server.URL))
	ch, err := p.Stream(context.Background(), ChatRequest{
		Model:    "grok-3",
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
	if collected[1].Content != " Grok" {
		t.Errorf("expected second chunk ' Grok', got %s", collected[1].Content)
	}
	last := collected[len(collected)-1]
	if !last.Done {
		t.Error("expected last chunk to be done")
	}
}

func TestXAIProvider_StreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error": {"message": "service unavailable"}}`))
	}))
	defer server.Close()

	p := NewXAIProvider("xai-test", WithXAIBaseURL(server.URL))
	_, err := p.Stream(context.Background(), ChatRequest{
		Model:    "grok-3",
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err == nil {
		t.Fatal("expected error for 503 response")
	}
}

func TestXAIProvider_Models(t *testing.T) {
	p := NewXAIProvider("xai-test")
	models := p.Models()

	if len(models) == 0 {
		t.Fatal("expected hardcoded models")
	}

	modelIDs := make(map[string]bool)
	for _, m := range models {
		modelIDs[m.ID] = true
	}

	expected := []string{"grok-4-1-fast-latest", "grok-3", "grok-3-mini"}
	for _, id := range expected {
		if !modelIDs[id] {
			t.Errorf("expected model %s not found", id)
		}
	}
}

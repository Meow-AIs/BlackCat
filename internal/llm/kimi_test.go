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

func TestNewKimiProvider_Defaults(t *testing.T) {
	p := NewKimiProvider("kimi-test-key")
	if p.Name() != "kimi" {
		t.Errorf("expected name 'kimi', got %s", p.Name())
	}
	if p.baseURL != "https://api.moonshot.ai/v1" {
		t.Errorf("expected default base URL, got %s", p.baseURL)
	}
	if p.apiKey != "kimi-test-key" {
		t.Errorf("expected api key 'kimi-test-key', got %s", p.apiKey)
	}
}

func TestNewKimiProvider_WithBaseURL(t *testing.T) {
	p := NewKimiProvider("kimi-key", WithKimiBaseURL("https://custom.moonshot.ai/v2"))
	if p.baseURL != "https://custom.moonshot.ai/v2" {
		t.Errorf("expected custom base URL, got %s", p.baseURL)
	}
}

func TestKimiProvider_Chat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer kimi-test" {
			t.Errorf("expected Bearer auth, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("expected application/json content type")
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)
		if reqBody["model"] != "kimi-k2.5" {
			t.Errorf("expected model 'kimi-k2.5', got %v", reqBody["model"])
		}

		resp := map[string]any{
			"id":    "chatcmpl-kimi-1",
			"model": "kimi-k2.5",
			"choices": []map[string]any{
				{
					"index":         0,
					"message":       map[string]any{"role": "assistant", "content": "Hello from Kimi!"},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewKimiProvider("kimi-test", WithKimiBaseURL(server.URL))
	resp, err := p.Chat(context.Background(), ChatRequest{
		Model:    "kimi-k2.5",
		Messages: []Message{{Role: RoleUser, Content: "Hello"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "Hello from Kimi!" {
		t.Errorf("expected 'Hello from Kimi!', got %q", resp.Content)
	}
	if resp.Model != "kimi-k2.5" {
		t.Errorf("expected model 'kimi-k2.5', got %q", resp.Model)
	}
	if resp.Usage.TotalTokens != 15 {
		t.Errorf("expected 15 total tokens, got %d", resp.Usage.TotalTokens)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("expected finish_reason 'stop', got %q", resp.FinishReason)
	}
}

func TestKimiProvider_ChatWithToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"id":    "chatcmpl-kimi-2",
			"model": "kimi-k2.5",
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": "",
						"tool_calls": []map[string]any{
							{
								"id":   "call_kimi_1",
								"type": "function",
								"function": map[string]any{
									"name":      "search",
									"arguments": `{"query":"golang"}`,
								},
							},
						},
					},
					"finish_reason": "tool_calls",
				},
			},
			"usage": map[string]any{"prompt_tokens": 15, "completion_tokens": 10, "total_tokens": 25},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewKimiProvider("kimi-test", WithKimiBaseURL(server.URL))
	resp, err := p.Chat(context.Background(), ChatRequest{
		Model:    "kimi-k2.5",
		Messages: []Message{{Role: RoleUser, Content: "Search golang"}},
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
	if resp.ToolCalls[0].ID != "call_kimi_1" {
		t.Errorf("expected tool call ID 'call_kimi_1', got %s", resp.ToolCalls[0].ID)
	}
	if resp.ToolCalls[0].Name != "search" {
		t.Errorf("expected tool name 'search', got %s", resp.ToolCalls[0].Name)
	}
	if resp.FinishReason != "tool_calls" {
		t.Errorf("expected finish_reason 'tool_calls', got %q", resp.FinishReason)
	}
}

func TestKimiProvider_ChatError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": {"message": "invalid api key"}}`))
	}))
	defer server.Close()

	p := NewKimiProvider("bad-key", WithKimiBaseURL(server.URL))
	_, err := p.Chat(context.Background(), ChatRequest{
		Model:    "kimi-k2.5",
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected status code in error, got: %s", err.Error())
	}
}

func TestKimiProvider_Stream(t *testing.T) {
	sseLines := []string{
		`data: {"id":"gen-1","choices":[{"delta":{"content":"Hello"},"index":0}]}`,
		`data: {"id":"gen-1","choices":[{"delta":{"content":" Kimi"},"index":0}]}`,
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

	p := NewKimiProvider("kimi-test", WithKimiBaseURL(server.URL))
	ch, err := p.Stream(context.Background(), ChatRequest{
		Model:    "kimi-k2.5",
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
	if collected[1].Content != " Kimi" {
		t.Errorf("expected second chunk ' Kimi', got %s", collected[1].Content)
	}
	last := collected[len(collected)-1]
	if !last.Done {
		t.Error("expected last chunk to be done")
	}
}

func TestKimiProvider_StreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": {"message": "server error"}}`))
	}))
	defer server.Close()

	p := NewKimiProvider("kimi-test", WithKimiBaseURL(server.URL))
	_, err := p.Stream(context.Background(), ChatRequest{
		Model:    "kimi-k2.5",
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestKimiProvider_Models(t *testing.T) {
	p := NewKimiProvider("kimi-test")
	models := p.Models()

	if len(models) == 0 {
		t.Fatal("expected hardcoded models")
	}

	modelIDs := make(map[string]bool)
	for _, m := range models {
		modelIDs[m.ID] = true
	}

	expected := []string{"kimi-k2.5", "kimi-k2.5-mini"}
	for _, id := range expected {
		if !modelIDs[id] {
			t.Errorf("expected model %s not found", id)
		}
	}
}

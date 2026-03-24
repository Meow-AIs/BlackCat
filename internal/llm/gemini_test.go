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

func TestGeminiProvider_Defaults(t *testing.T) {
	p := NewGeminiProvider("gemini-test-key")
	if p.Name() != "gemini" {
		t.Errorf("expected name 'gemini', got %s", p.Name())
	}
	if p.baseURL != "https://generativelanguage.googleapis.com/v1beta/openai" {
		t.Errorf("expected default base URL, got %s", p.baseURL)
	}
	if p.apiKey != "gemini-test-key" {
		t.Errorf("expected api key 'gemini-test-key', got %s", p.apiKey)
	}
}

func TestGeminiProvider_BaseURLOption(t *testing.T) {
	p := NewGeminiProvider("key", WithGeminiBaseURL("https://custom.gemini.api/v1"))
	if p.baseURL != "https://custom.gemini.api/v1" {
		t.Errorf("expected custom base URL, got %s", p.baseURL)
	}
}

func TestGeminiProvider_Chat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("expected application/json content type")
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)
		if reqBody["model"] != "gemini-2.5-pro" {
			t.Errorf("expected model 'gemini-2.5-pro', got %v", reqBody["model"])
		}

		resp := map[string]any{
			"id":    "chatcmpl-gemini-1",
			"model": "gemini-2.5-pro",
			"choices": []map[string]any{
				{
					"index":         0,
					"message":       map[string]any{"role": "assistant", "content": "Hello from Gemini!"},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewGeminiProvider("gemini-test", WithGeminiBaseURL(server.URL))
	resp, err := p.Chat(context.Background(), ChatRequest{
		Model:    "gemini-2.5-pro",
		Messages: []Message{{Role: RoleUser, Content: "Hello"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "Hello from Gemini!" {
		t.Errorf("expected 'Hello from Gemini!', got %q", resp.Content)
	}
	if resp.Model != "gemini-2.5-pro" {
		t.Errorf("expected model 'gemini-2.5-pro', got %q", resp.Model)
	}
	if resp.Usage.TotalTokens != 15 {
		t.Errorf("expected 15 total tokens, got %d", resp.Usage.TotalTokens)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("expected finish_reason 'stop', got %q", resp.FinishReason)
	}
}

func TestGeminiProvider_ChatWithToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"id":    "chatcmpl-gemini-2",
			"model": "gemini-2.5-flash",
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": "",
						"tool_calls": []map[string]any{
							{
								"id":   "call_gemini_1",
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

	p := NewGeminiProvider("gemini-test", WithGeminiBaseURL(server.URL))
	resp, err := p.Chat(context.Background(), ChatRequest{
		Model:    "gemini-2.5-flash",
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
	if resp.ToolCalls[0].ID != "call_gemini_1" {
		t.Errorf("expected tool call ID 'call_gemini_1', got %s", resp.ToolCalls[0].ID)
	}
	if resp.ToolCalls[0].Name != "read_file" {
		t.Errorf("expected tool name 'read_file', got %s", resp.ToolCalls[0].Name)
	}
	if resp.FinishReason != "tool_calls" {
		t.Errorf("expected finish_reason 'tool_calls', got %q", resp.FinishReason)
	}
}

func TestGeminiProvider_ChatError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error": {"message": "forbidden"}}`))
	}))
	defer server.Close()

	p := NewGeminiProvider("bad-key", WithGeminiBaseURL(server.URL))
	_, err := p.Chat(context.Background(), ChatRequest{
		Model:    "gemini-2.5-pro",
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("expected status code in error, got: %s", err.Error())
	}
}

func TestGeminiProvider_Stream(t *testing.T) {
	sseLines := []string{
		`data: {"id":"gen-1","choices":[{"delta":{"content":"Hello"},"index":0}]}`,
		`data: {"id":"gen-1","choices":[{"delta":{"content":" Gemini"},"index":0}]}`,
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

	p := NewGeminiProvider("gemini-test", WithGeminiBaseURL(server.URL))
	ch, err := p.Stream(context.Background(), ChatRequest{
		Model:    "gemini-2.5-pro",
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
	if collected[1].Content != " Gemini" {
		t.Errorf("expected second chunk ' Gemini', got %s", collected[1].Content)
	}
	last := collected[len(collected)-1]
	if !last.Done {
		t.Error("expected last chunk to be done")
	}
}

func TestGeminiProvider_StreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error": {"message": "service unavailable"}}`))
	}))
	defer server.Close()

	p := NewGeminiProvider("gemini-test", WithGeminiBaseURL(server.URL))
	_, err := p.Stream(context.Background(), ChatRequest{
		Model:    "gemini-2.5-pro",
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err == nil {
		t.Fatal("expected error for 503 response")
	}
}

func TestGeminiProvider_Models(t *testing.T) {
	p := NewGeminiProvider("gemini-test")
	models := p.Models()

	if len(models) == 0 {
		t.Fatal("expected hardcoded models")
	}

	modelIDs := make(map[string]bool)
	for _, m := range models {
		modelIDs[m.ID] = true
	}

	expected := []string{"gemini-2.5-pro", "gemini-2.5-flash", "gemini-2.0-flash", "gemini-2.0-flash-lite"}
	for _, id := range expected {
		if !modelIDs[id] {
			t.Errorf("expected model %s not found", id)
		}
	}
}

func TestGeminiProvider_AuthHeader(t *testing.T) {
	var capturedAuthHeader string
	var capturedGoogHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuthHeader = r.Header.Get("Authorization")
		capturedGoogHeader = r.Header.Get("x-goog-api-key")

		resp := map[string]any{
			"id":    "chatcmpl-gemini-auth",
			"model": "gemini-2.5-pro",
			"choices": []map[string]any{
				{
					"index":         0,
					"message":       map[string]any{"role": "assistant", "content": "ok"},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewGeminiProvider("my-gemini-api-key", WithGeminiBaseURL(server.URL))
	_, err := p.Chat(context.Background(), ChatRequest{
		Model:    "gemini-2.5-pro",
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Gemini uses x-goog-api-key, NOT Authorization: Bearer
	if capturedGoogHeader != "my-gemini-api-key" {
		t.Errorf("expected x-goog-api-key 'my-gemini-api-key', got %q", capturedGoogHeader)
	}
	if capturedAuthHeader != "" {
		t.Errorf("expected no Authorization header, got %q", capturedAuthHeader)
	}
}

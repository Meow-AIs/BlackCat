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

func TestNewZAIProvider_Defaults(t *testing.T) {
	p := NewZAIProvider("zai-test-key")
	if p.Name() != "zai" {
		t.Errorf("expected name 'zai', got %s", p.Name())
	}
	if p.baseURL != "https://api.z.ai/api/paas/v4" {
		t.Errorf("expected default base URL, got %s", p.baseURL)
	}
	if p.apiKey != "zai-test-key" {
		t.Errorf("expected api key 'zai-test-key', got %s", p.apiKey)
	}
	if p.useCoding {
		t.Error("expected useCoding to be false by default")
	}
}

func TestNewZAIProvider_WithCodingPlan(t *testing.T) {
	p := NewZAIProvider("zai-key", WithZAICodingPlan())
	if !p.useCoding {
		t.Error("expected useCoding to be true")
	}
	if p.codingURL != "https://api.z.ai/api/coding/paas/v4" {
		t.Errorf("expected coding URL, got %s", p.codingURL)
	}
}

func TestNewZAIProvider_WithBaseURL(t *testing.T) {
	p := NewZAIProvider("zai-key", WithZAIBaseURL("https://custom.z.ai/v1"))
	if p.baseURL != "https://custom.z.ai/v1" {
		t.Errorf("expected custom base URL, got %s", p.baseURL)
	}
}

func TestZAIProvider_Chat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer zai-test" {
			t.Errorf("expected Bearer auth, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("expected application/json content type")
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)
		if reqBody["model"] != "glm-5" {
			t.Errorf("expected model 'glm-5', got %v", reqBody["model"])
		}

		resp := map[string]any{
			"id":    "chatcmpl-zai-1",
			"model": "glm-5",
			"choices": []map[string]any{
				{
					"index":         0,
					"message":       map[string]any{"role": "assistant", "content": "Hello from Z.ai!"},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{"prompt_tokens": 12, "completion_tokens": 6, "total_tokens": 18},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewZAIProvider("zai-test", WithZAIBaseURL(server.URL))
	resp, err := p.Chat(context.Background(), ChatRequest{
		Model:    "glm-5",
		Messages: []Message{{Role: RoleUser, Content: "Hello"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "Hello from Z.ai!" {
		t.Errorf("expected 'Hello from Z.ai!', got %q", resp.Content)
	}
	if resp.Model != "glm-5" {
		t.Errorf("expected model 'glm-5', got %q", resp.Model)
	}
	if resp.Usage.TotalTokens != 18 {
		t.Errorf("expected 18 total tokens, got %d", resp.Usage.TotalTokens)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("expected finish_reason 'stop', got %q", resp.FinishReason)
	}
}

func TestZAIProvider_ChatCodingPlan(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		resp := map[string]any{
			"id":    "chatcmpl-coding-1",
			"model": "glm-5",
			"choices": []map[string]any{
				{
					"index":         0,
					"message":       map[string]any{"role": "assistant", "content": "Coding response"},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{"prompt_tokens": 5, "completion_tokens": 3, "total_tokens": 8},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewZAIProvider("zai-test",
		WithZAICodingPlan(),
		WithZAIBaseURL(server.URL+"/standard"),
	)
	// Override coding URL to point at test server
	p.codingURL = server.URL

	resp, err := p.Chat(context.Background(), ChatRequest{
		Model:    "glm-5",
		Messages: []Message{{Role: RoleUser, Content: "Write code"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "Coding response" {
		t.Errorf("expected 'Coding response', got %q", resp.Content)
	}
}

func TestZAIProvider_ChatError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": {"message": "rate limit exceeded"}}`))
	}))
	defer server.Close()

	p := NewZAIProvider("zai-test", WithZAIBaseURL(server.URL))
	_, err := p.Chat(context.Background(), ChatRequest{
		Model:    "glm-5",
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err == nil {
		t.Fatal("expected error for 429 response")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("expected status code in error, got: %s", err.Error())
	}
}

func TestZAIProvider_Stream(t *testing.T) {
	sseLines := []string{
		`data: {"id":"gen-1","choices":[{"delta":{"content":"Hello"},"index":0}]}`,
		`data: {"id":"gen-1","choices":[{"delta":{"content":" Z.ai"},"index":0}]}`,
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

	p := NewZAIProvider("zai-test", WithZAIBaseURL(server.URL))
	ch, err := p.Stream(context.Background(), ChatRequest{
		Model:    "glm-5",
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
	if collected[1].Content != " Z.ai" {
		t.Errorf("expected second chunk ' Z.ai', got %s", collected[1].Content)
	}
	last := collected[len(collected)-1]
	if !last.Done {
		t.Error("expected last chunk to be done")
	}
}

func TestZAIProvider_StreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": {"message": "server error"}}`))
	}))
	defer server.Close()

	p := NewZAIProvider("zai-test", WithZAIBaseURL(server.URL))
	_, err := p.Stream(context.Background(), ChatRequest{
		Model:    "glm-5",
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestZAIProvider_Models(t *testing.T) {
	p := NewZAIProvider("zai-test")
	models := p.Models()

	if len(models) == 0 {
		t.Fatal("expected hardcoded models")
	}

	modelIDs := make(map[string]bool)
	for _, m := range models {
		modelIDs[m.ID] = true
	}

	expected := []string{"glm-5", "glm-5-turbo", "glm-4.7", "glm-4.7-flash", "glm-4.5"}
	for _, id := range expected {
		if !modelIDs[id] {
			t.Errorf("expected model %s not found", id)
		}
	}
}

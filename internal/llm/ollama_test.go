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

func TestNewOllamaProvider_DefaultBaseURL(t *testing.T) {
	p := NewOllamaProvider("")
	if p.baseURL != "http://localhost:11434" {
		t.Errorf("expected default base URL, got %s", p.baseURL)
	}
	if p.Name() != "ollama" {
		t.Errorf("expected name 'ollama', got %s", p.Name())
	}
}

func TestNewOllamaProvider_CustomBaseURL(t *testing.T) {
	p := NewOllamaProvider("http://myhost:9999")
	if p.baseURL != "http://myhost:9999" {
		t.Errorf("expected custom base URL, got %s", p.baseURL)
	}
}

func TestOllamaProvider_Chat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json content type")
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("failed to parse request body: %v", err)
		}

		// Verify stream is false for Chat
		if stream, ok := reqBody["stream"].(bool); !ok || stream {
			t.Error("expected stream=false in request")
		}

		// Verify model is passed through
		if model, ok := reqBody["model"].(string); !ok || model != "llama3" {
			t.Errorf("expected model 'llama3', got %v", reqBody["model"])
		}

		// Verify messages are mapped correctly
		msgs, ok := reqBody["messages"].([]any)
		if !ok || len(msgs) != 2 {
			t.Fatalf("expected 2 messages, got %v", reqBody["messages"])
		}

		resp := map[string]any{
			"model":      "llama3",
			"message":    map[string]any{"role": "assistant", "content": "Hello there!"},
			"done":       true,
			"total_duration":   5000000000,
			"prompt_eval_count": 10,
			"eval_count":        5,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL)
	resp, err := p.Chat(context.Background(), ChatRequest{
		Model: "llama3",
		Messages: []Message{
			{Role: RoleSystem, Content: "You are helpful."},
			{Role: RoleUser, Content: "Hi"},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "Hello there!" {
		t.Errorf("expected 'Hello there!', got %s", resp.Content)
	}
	if resp.Model != "llama3" {
		t.Errorf("expected model 'llama3', got %s", resp.Model)
	}
	if resp.Usage.PromptTokens != 10 {
		t.Errorf("expected 10 prompt tokens, got %d", resp.Usage.PromptTokens)
	}
	if resp.Usage.CompletionTokens != 5 {
		t.Errorf("expected 5 completion tokens, got %d", resp.Usage.CompletionTokens)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("expected finish_reason 'stop', got %s", resp.FinishReason)
	}
}

func TestOllamaProvider_ChatWithTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)

		// Verify tools are present
		tools, ok := reqBody["tools"].([]any)
		if !ok || len(tools) != 1 {
			t.Fatalf("expected 1 tool, got %v", reqBody["tools"])
		}

		resp := map[string]any{
			"model": "llama3",
			"message": map[string]any{
				"role":    "assistant",
				"content": "",
				"tool_calls": []map[string]any{
					{
						"function": map[string]any{
							"name":      "get_weather",
							"arguments": map[string]any{"city": "London"},
						},
					},
				},
			},
			"done":              true,
			"prompt_eval_count": 20,
			"eval_count":        10,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL)
	resp, err := p.Chat(context.Background(), ChatRequest{
		Model: "llama3",
		Messages: []Message{
			{Role: RoleUser, Content: "Weather in London?"},
		},
		Tools: []ToolDefinition{
			{Name: "get_weather", Description: "Get weather", Parameters: map[string]any{"type": "object"}},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "get_weather" {
		t.Errorf("expected tool name 'get_weather', got %s", resp.ToolCalls[0].Name)
	}
}

func TestOllamaProvider_ChatAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "model not found"}`))
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL)
	_, err := p.Chat(context.Background(), ChatRequest{
		Model:    "nonexistent",
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})

	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected status code in error, got: %s", err.Error())
	}
}

func TestOllamaProvider_Stream(t *testing.T) {
	chunks := []map[string]any{
		{"model": "llama3", "message": map[string]any{"role": "assistant", "content": "Hello"}, "done": false},
		{"model": "llama3", "message": map[string]any{"role": "assistant", "content": " world"}, "done": false},
		{"model": "llama3", "message": map[string]any{"role": "assistant", "content": ""}, "done": true, "prompt_eval_count": 8, "eval_count": 4},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)

		// Verify stream is true
		if stream, ok := reqBody["stream"].(bool); !ok || !stream {
			t.Error("expected stream=true in request")
		}

		w.Header().Set("Content-Type", "application/x-ndjson")
		flusher, _ := w.(http.Flusher)
		for _, chunk := range chunks {
			json.NewEncoder(w).Encode(chunk)
			flusher.Flush()
		}
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL)
	ch, err := p.Stream(context.Background(), ChatRequest{
		Model:    "llama3",
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

	if len(collected) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(collected))
	}
	if collected[0].Content != "Hello" {
		t.Errorf("expected first chunk 'Hello', got %s", collected[0].Content)
	}
	if collected[1].Content != " world" {
		t.Errorf("expected second chunk ' world', got %s", collected[1].Content)
	}
	if !collected[2].Done {
		t.Error("expected last chunk to be done")
	}
	if collected[2].Usage == nil || collected[2].Usage.PromptTokens != 8 {
		t.Error("expected usage in final chunk")
	}
}

func TestOllamaProvider_StreamContextCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a slow response - will be cancelled
		time.Sleep(10 * time.Second)
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL)
	ctx, cancel := context.WithCancel(context.Background())

	ch, err := p.Stream(ctx, ChatRequest{
		Model:    "llama3",
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cancel()

	// Channel should close
	timeout := time.After(2 * time.Second)
	select {
	case _, ok := <-ch:
		if ok {
			// Might get one chunk, but channel should eventually close
			select {
			case <-ch:
			case <-timeout:
				t.Fatal("channel did not close after context cancel")
			}
		}
	case <-timeout:
		t.Fatal("channel did not close after context cancel")
	}
}

func TestOllamaProvider_Models(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}

		resp := map[string]any{
			"models": []map[string]any{
				{"name": "llama3:latest", "size": 4000000000},
				{"name": "mistral:7b", "size": 3800000000},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL)
	models := p.Models()

	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0].ID != "llama3:latest" {
		t.Errorf("expected first model 'llama3:latest', got %s", models[0].ID)
	}
	if models[1].ID != "mistral:7b" {
		t.Errorf("expected second model 'mistral:7b', got %s", models[1].ID)
	}
}

func TestOllamaProvider_ModelsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL)
	models := p.Models()

	// Should return empty slice on error, not panic
	if models == nil {
		t.Error("expected non-nil empty slice")
	}
	if len(models) != 0 {
		t.Errorf("expected 0 models on error, got %d", len(models))
	}
}

func TestOllamaProvider_ChatWithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)

		// Verify options are passed through
		options, ok := reqBody["options"].(map[string]any)
		if !ok {
			t.Fatal("expected options in request")
		}
		if temp, ok := options["temperature"].(float64); !ok || temp != 0.7 {
			t.Errorf("expected temperature 0.7, got %v", options["temperature"])
		}

		resp := map[string]any{
			"model":             "llama3",
			"message":           map[string]any{"role": "assistant", "content": "ok"},
			"done":              true,
			"prompt_eval_count": 5,
			"eval_count":        1,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL)
	temp := 0.7
	_, err := p.Chat(context.Background(), ChatRequest{
		Model:       "llama3",
		Messages:    []Message{{Role: RoleUser, Content: "Hi"}},
		Temperature: &temp,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

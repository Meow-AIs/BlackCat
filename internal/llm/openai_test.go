package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// openaiChatResponse is the mock response structure matching OpenAI API.
type openaiChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int `json:"index"`
		Message      struct {
			Role      string `json:"role"`
			Content   string `json:"content"`
			ToolCalls []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls,omitempty"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func newMockOpenAIServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func TestOpenAIProviderChat(t *testing.T) {
	server := newMockOpenAIServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("expected /v1/chat/completions, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Bearer test-key, got %s", r.Header.Get("Authorization"))
		}
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", contentType)
		}

		// Verify request body
		var reqBody map[string]any
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if reqBody["model"] != "gpt-4.1" {
			t.Errorf("expected model 'gpt-4.1', got %v", reqBody["model"])
		}

		resp := openaiChatResponse{
			ID:     "chatcmpl-123",
			Object: "chat.completion",
			Model:  "gpt-4.1",
		}
		resp.Choices = append(resp.Choices, struct {
			Index        int `json:"index"`
			Message      struct {
				Role      string `json:"role"`
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls,omitempty"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}{
			Index: 0,
			Message: struct {
				Role      string `json:"role"`
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls,omitempty"`
			}{
				Role:    "assistant",
				Content: "Hello from BlackCat!",
			},
			FinishReason: "stop",
		})
		resp.Usage.PromptTokens = 10
		resp.Usage.CompletionTokens = 5
		resp.Usage.TotalTokens = 15

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	provider := NewOpenAIProvider("test-key", server.URL+"/v1", "")

	resp, err := provider.Chat(context.Background(), ChatRequest{
		Model: "gpt-4.1",
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
		},
	})
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if resp.Content != "Hello from BlackCat!" {
		t.Errorf("expected 'Hello from BlackCat!', got %q", resp.Content)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("expected finish_reason 'stop', got %q", resp.FinishReason)
	}
	if resp.Usage.TotalTokens != 15 {
		t.Errorf("expected total_tokens 15, got %d", resp.Usage.TotalTokens)
	}
}

func TestOpenAIProviderChatWithToolCalls(t *testing.T) {
	server := newMockOpenAIServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := `{
			"id": "chatcmpl-456",
			"object": "chat.completion",
			"model": "gpt-4.1",
			"choices": [{
				"index": 0,
				"message": {
					"role": "assistant",
					"content": "",
					"tool_calls": [{
						"id": "call_abc",
						"type": "function",
						"function": {
							"name": "read_file",
							"arguments": "{\"path\": \"main.go\"}"
						}
					}]
				},
				"finish_reason": "tool_calls"
			}],
			"usage": {"prompt_tokens": 20, "completion_tokens": 10, "total_tokens": 30}
		}`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(resp))
	})
	defer server.Close()

	provider := NewOpenAIProvider("test-key", server.URL+"/v1", "")

	resp, err := provider.Chat(context.Background(), ChatRequest{
		Model:    "gpt-4.1",
		Messages: []Message{{Role: RoleUser, Content: "read main.go"}},
	})
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "read_file" {
		t.Errorf("expected tool name 'read_file', got %q", resp.ToolCalls[0].Name)
	}
	if resp.ToolCalls[0].ID != "call_abc" {
		t.Errorf("expected tool call ID 'call_abc', got %q", resp.ToolCalls[0].ID)
	}
	if resp.FinishReason != "tool_calls" {
		t.Errorf("expected finish_reason 'tool_calls', got %q", resp.FinishReason)
	}
}

func TestOpenAIProviderChatServerError(t *testing.T) {
	server := newMockOpenAIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": {"message": "server error"}}`))
	})
	defer server.Close()

	provider := NewOpenAIProvider("test-key", server.URL+"/v1", "")

	_, err := provider.Chat(context.Background(), ChatRequest{
		Model:    "gpt-4.1",
		Messages: []Message{{Role: RoleUser, Content: "Hello"}},
	})
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
}

func TestOpenAIProviderName(t *testing.T) {
	provider := NewOpenAIProvider("key", "https://api.openai.com/v1", "")
	if provider.Name() != "openai" {
		t.Errorf("expected name 'openai', got %q", provider.Name())
	}
}

func TestOpenAIProviderCustomName(t *testing.T) {
	provider := NewOpenAIProvider("key", "https://custom.api.com/v1", "my-provider")
	if provider.Name() != "my-provider" {
		t.Errorf("expected name 'my-provider', got %q", provider.Name())
	}
}

func TestOpenAIProviderModels(t *testing.T) {
	provider := NewOpenAIProvider("key", "https://api.openai.com/v1", "")
	models := provider.Models()
	if len(models) == 0 {
		t.Error("expected at least one model")
	}
}

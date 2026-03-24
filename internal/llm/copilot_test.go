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

func TestNewCopilotProvider_Defaults(t *testing.T) {
	p := NewCopilotProvider()
	if p.Name() != "copilot" {
		t.Errorf("expected name 'copilot', got %s", p.Name())
	}
	if p.baseURL != "https://api.githubcopilot.com" {
		t.Errorf("expected default base URL, got %s", p.baseURL)
	}
	if p.oauth == nil {
		t.Error("expected non-nil oauth client")
	}
	if p.httpClient == nil {
		t.Error("expected non-nil http client")
	}
}

func TestNewCopilotProvider_WithBaseURL(t *testing.T) {
	p := NewCopilotProvider(WithCopilotBaseURL("https://custom.copilot.test"))
	if p.baseURL != "https://custom.copilot.test" {
		t.Errorf("expected custom base URL, got %s", p.baseURL)
	}
}

func TestNewCopilotProvider_WithToken(t *testing.T) {
	p := NewCopilotProvider(WithCopilotToken("gho_existing_token"))
	if !p.IsAuthenticated() {
		t.Error("expected authenticated when token provided")
	}
}

func TestCopilotProvider_Chat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-copilot-token" {
			t.Errorf("expected Bearer auth, got %s", auth)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("expected application/json content type")
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)
		if reqBody["model"] != "gpt-4.1" {
			t.Errorf("expected model 'gpt-4.1', got %v", reqBody["model"])
		}

		resp := map[string]any{
			"id":    "chatcmpl-copilot-1",
			"model": "gpt-4.1",
			"choices": []map[string]any{
				{
					"index":         0,
					"message":       map[string]any{"role": "assistant", "content": "Hello from Copilot!"},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewCopilotProvider(
		WithCopilotBaseURL(server.URL),
		WithCopilotToken("test-copilot-token"),
	)

	resp, err := p.Chat(context.Background(), ChatRequest{
		Model:    "gpt-4.1",
		Messages: []Message{{Role: RoleUser, Content: "Hello"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "Hello from Copilot!" {
		t.Errorf("expected 'Hello from Copilot!', got %q", resp.Content)
	}
	if resp.Model != "gpt-4.1" {
		t.Errorf("expected model 'gpt-4.1', got %q", resp.Model)
	}
	if resp.Usage.TotalTokens != 15 {
		t.Errorf("expected 15 total tokens, got %d", resp.Usage.TotalTokens)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("expected finish_reason 'stop', got %q", resp.FinishReason)
	}
}

func TestCopilotProvider_ChatError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": {"message": "unauthorized"}}`))
	}))
	defer server.Close()

	p := NewCopilotProvider(
		WithCopilotBaseURL(server.URL),
		WithCopilotToken("bad-token"),
	)
	_, err := p.Chat(context.Background(), ChatRequest{
		Model:    "gpt-4.1",
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected status code in error, got: %s", err.Error())
	}
}

func TestCopilotProvider_ChatNotAuthenticated(t *testing.T) {
	p := NewCopilotProvider()
	_, err := p.Chat(context.Background(), ChatRequest{
		Model:    "gpt-4.1",
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err == nil {
		t.Fatal("expected error when not authenticated")
	}
	if !strings.Contains(err.Error(), "not authenticated") {
		t.Errorf("expected 'not authenticated' in error, got: %s", err.Error())
	}
}

func TestCopilotProvider_Stream(t *testing.T) {
	sseLines := []string{
		`data: {"id":"gen-1","choices":[{"delta":{"content":"Hello"},"index":0}]}`,
		`data: {"id":"gen-1","choices":[{"delta":{"content":" Copilot"},"index":0}]}`,
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

	p := NewCopilotProvider(
		WithCopilotBaseURL(server.URL),
		WithCopilotToken("test-token"),
	)
	ch, err := p.Stream(context.Background(), ChatRequest{
		Model:    "gpt-4.1",
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
	if collected[1].Content != " Copilot" {
		t.Errorf("expected second chunk ' Copilot', got %s", collected[1].Content)
	}
	last := collected[len(collected)-1]
	if !last.Done {
		t.Error("expected last chunk to be done")
	}
}

func TestCopilotProvider_StreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": {"message": "server error"}}`))
	}))
	defer server.Close()

	p := NewCopilotProvider(
		WithCopilotBaseURL(server.URL),
		WithCopilotToken("test-token"),
	)
	_, err := p.Stream(context.Background(), ChatRequest{
		Model:    "gpt-4.1",
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestCopilotProvider_StreamNotAuthenticated(t *testing.T) {
	p := NewCopilotProvider()
	_, err := p.Stream(context.Background(), ChatRequest{
		Model:    "gpt-4.1",
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err == nil {
		t.Fatal("expected error when not authenticated")
	}
}

func TestCopilotProvider_Models(t *testing.T) {
	p := NewCopilotProvider()
	models := p.Models()

	if len(models) == 0 {
		t.Fatal("expected models")
	}

	modelIDs := make(map[string]bool)
	for _, m := range models {
		modelIDs[m.ID] = true
	}

	expected := []string{"gpt-5.4", "gpt-4.1", "claude-sonnet-4-6", "claude-opus-4-6", "o4-mini"}
	for _, id := range expected {
		if !modelIDs[id] {
			t.Errorf("expected model %s not found", id)
		}
	}
}

func TestCopilotProvider_Login(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DeviceCodeResponse{
			DeviceCode:      "dev-123",
			UserCode:        "ABCD-5678",
			VerificationURI: "https://github.com/login/device",
			ExpiresIn:       900,
			Interval:        5,
		})
	}))
	defer server.Close()

	config := OAuthConfig{
		ClientID:      GitHubCopilotOAuth.ClientID,
		DeviceCodeURL: server.URL,
		TokenURL:      server.URL + "/token",
		Scopes:        GitHubCopilotOAuth.Scopes,
	}
	p := &CopilotProvider{
		oauth:      NewOAuthClient(config),
		baseURL:    "https://api.githubcopilot.com",
		httpClient: &http.Client{},
	}

	resp, err := p.Login(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.UserCode != "ABCD-5678" {
		t.Errorf("expected user code 'ABCD-5678', got %q", resp.UserCode)
	}
}

func TestCopilotProvider_CompleteLogin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(OAuthToken{
			AccessToken: "gho_completed",
			TokenType:   "bearer",
			ExpiresIn:   28800,
		})
	}))
	defer server.Close()

	config := OAuthConfig{
		ClientID:      GitHubCopilotOAuth.ClientID,
		DeviceCodeURL: server.URL + "/device",
		TokenURL:      server.URL,
		Scopes:        GitHubCopilotOAuth.Scopes,
	}
	p := &CopilotProvider{
		oauth:      NewOAuthClient(config),
		baseURL:    "https://api.githubcopilot.com",
		httpClient: &http.Client{},
	}

	err := p.CompleteLogin(context.Background(), "dev-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.IsAuthenticated() {
		t.Error("expected authenticated after complete login")
	}
}

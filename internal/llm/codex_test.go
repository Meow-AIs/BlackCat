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

func TestNewCodexProvider_Defaults(t *testing.T) {
	p := NewCodexProvider()
	if p.Name() != "codex" {
		t.Errorf("expected name 'codex', got %s", p.Name())
	}
	if p.baseURL != "https://api.openai.com/v1" {
		t.Errorf("expected default base URL, got %s", p.baseURL)
	}
	if p.oauth == nil {
		t.Error("expected non-nil oauth client")
	}
	if p.httpClient == nil {
		t.Error("expected non-nil http client")
	}
}

func TestNewCodexProvider_WithBaseURL(t *testing.T) {
	p := NewCodexProvider(WithCodexBaseURL("https://custom.openai.test/v1"))
	if p.baseURL != "https://custom.openai.test/v1" {
		t.Errorf("expected custom base URL, got %s", p.baseURL)
	}
}

func TestNewCodexProvider_WithToken(t *testing.T) {
	p := NewCodexProvider(WithCodexToken("existing-codex-token"))
	if !p.IsAuthenticated() {
		t.Error("expected authenticated when token provided")
	}
}

func TestCodexProvider_Chat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-codex-token" {
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
			"id":    "chatcmpl-codex-1",
			"model": "gpt-4.1",
			"choices": []map[string]any{
				{
					"index":         0,
					"message":       map[string]any{"role": "assistant", "content": "Hello from Codex!"},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{"prompt_tokens": 8, "completion_tokens": 4, "total_tokens": 12},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewCodexProvider(
		WithCodexBaseURL(server.URL),
		WithCodexToken("test-codex-token"),
	)

	resp, err := p.Chat(context.Background(), ChatRequest{
		Model:    "gpt-4.1",
		Messages: []Message{{Role: RoleUser, Content: "Hello"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "Hello from Codex!" {
		t.Errorf("expected 'Hello from Codex!', got %q", resp.Content)
	}
	if resp.Model != "gpt-4.1" {
		t.Errorf("expected model 'gpt-4.1', got %q", resp.Model)
	}
	if resp.Usage.TotalTokens != 12 {
		t.Errorf("expected 12 total tokens, got %d", resp.Usage.TotalTokens)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("expected finish_reason 'stop', got %q", resp.FinishReason)
	}
}

func TestCodexProvider_ChatError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error": {"message": "forbidden"}}`))
	}))
	defer server.Close()

	p := NewCodexProvider(
		WithCodexBaseURL(server.URL),
		WithCodexToken("bad-token"),
	)
	_, err := p.Chat(context.Background(), ChatRequest{
		Model:    "gpt-4.1",
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("expected status code in error, got: %s", err.Error())
	}
}

func TestCodexProvider_ChatNotAuthenticated(t *testing.T) {
	p := NewCodexProvider()
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

func TestCodexProvider_Stream(t *testing.T) {
	sseLines := []string{
		`data: {"id":"gen-1","choices":[{"delta":{"content":"Hello"},"index":0}]}`,
		`data: {"id":"gen-1","choices":[{"delta":{"content":" Codex"},"index":0}]}`,
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

	p := NewCodexProvider(
		WithCodexBaseURL(server.URL),
		WithCodexToken("test-token"),
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
	if collected[1].Content != " Codex" {
		t.Errorf("expected second chunk ' Codex', got %s", collected[1].Content)
	}
	last := collected[len(collected)-1]
	if !last.Done {
		t.Error("expected last chunk to be done")
	}
}

func TestCodexProvider_StreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error": {"message": "service unavailable"}}`))
	}))
	defer server.Close()

	p := NewCodexProvider(
		WithCodexBaseURL(server.URL),
		WithCodexToken("test-token"),
	)
	_, err := p.Stream(context.Background(), ChatRequest{
		Model:    "gpt-4.1",
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err == nil {
		t.Fatal("expected error for 503 response")
	}
}

func TestCodexProvider_StreamNotAuthenticated(t *testing.T) {
	p := NewCodexProvider()
	_, err := p.Stream(context.Background(), ChatRequest{
		Model:    "gpt-4.1",
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err == nil {
		t.Fatal("expected error when not authenticated")
	}
}

func TestCodexProvider_Models(t *testing.T) {
	p := NewCodexProvider()
	models := p.Models()

	if len(models) == 0 {
		t.Fatal("expected models")
	}

	modelIDs := make(map[string]bool)
	for _, m := range models {
		modelIDs[m.ID] = true
	}

	expected := []string{"gpt-5.4", "gpt-4.1", "o4-mini"}
	for _, id := range expected {
		if !modelIDs[id] {
			t.Errorf("expected model %s not found", id)
		}
	}
}

func TestCodexProvider_Login(t *testing.T) {
	// Login now returns a PKCE auth URL via the DeviceCodeResponse wrapper.
	// It no longer calls a device code endpoint.
	p := NewCodexProvider()

	resp, err := p.Login(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.VerificationURI == "" {
		t.Error("expected VerificationURI (auth URL) to be set")
	}
	if !strings.Contains(resp.VerificationURI, "auth.openai.com") {
		t.Errorf("expected OpenAI auth URL, got %q", resp.VerificationURI)
	}
	// PKCE flow has no user code
	if resp.UserCode != "" {
		t.Errorf("expected empty UserCode for PKCE, got %q", resp.UserCode)
	}
}

func TestCodexProvider_CompleteLogin(t *testing.T) {
	// CompleteLogin should return an error directing to use LoginPKCE instead.
	p := NewCodexProvider()

	err := p.CompleteLogin(context.Background(), "dev-codex-123")
	if err == nil {
		t.Fatal("expected error from CompleteLogin (device flow not supported)")
	}
	if !strings.Contains(err.Error(), "LoginPKCE") {
		t.Errorf("error should mention LoginPKCE, got: %v", err)
	}
}

func TestCodexProvider_LoginPKCE(t *testing.T) {
	// Test LoginPKCE with a mock token server
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(OAuthToken{
			AccessToken: "pkce-codex-token",
			TokenType:   "bearer",
			ExpiresIn:   3600,
		})
	}))
	defer tokenServer.Close()

	// We can't easily test the full LoginPKCE flow (it starts a callback server),
	// but we can test that the provider stores tokens correctly via WithCodexToken.
	p := NewCodexProvider(WithCodexToken("manual-pkce-token"))
	if !p.IsAuthenticated() {
		t.Error("expected authenticated after setting token")
	}
	tok, err := p.oauth.GetToken()
	if err != nil {
		t.Fatalf("GetToken error: %v", err)
	}
	if tok.AccessToken != "manual-pkce-token" {
		t.Errorf("AccessToken = %q, want %q", tok.AccessToken, "manual-pkce-token")
	}
}

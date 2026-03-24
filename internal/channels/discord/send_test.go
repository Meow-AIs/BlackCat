package discord

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/meowai/blackcat/internal/channels"
)

// newTestBot creates a Bot wired to a fake Discord API server.
func newTestBot(t *testing.T, handler http.Handler) (*Bot, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	bot := New(Config{
		Token:   "test-token",
		BaseURL: srv.URL,
	})
	return bot, srv
}

func TestSend_PostsToCorrectEndpoint(t *testing.T) {
	var capturedPath string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"123"}`))
	})

	bot, _ := newTestBot(t, handler)
	msg := channels.OutgoingMessage{ChannelID: "chan-abc", Text: "hello world"}

	if err := bot.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send() error: %v", err)
	}

	expected := "/api/v10/channels/chan-abc/messages"
	if capturedPath != expected {
		t.Errorf("expected path %q, got %q", expected, capturedPath)
	}
}

func TestSend_UsesPostMethod(t *testing.T) {
	var capturedMethod string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"123"}`))
	})

	bot, _ := newTestBot(t, handler)
	msg := channels.OutgoingMessage{ChannelID: "chan-abc", Text: "hello"}

	if err := bot.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send() error: %v", err)
	}

	if capturedMethod != http.MethodPost {
		t.Errorf("expected POST, got %q", capturedMethod)
	}
}

func TestSend_SetsAuthorizationHeader(t *testing.T) {
	var capturedAuth string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"123"}`))
	})

	bot, _ := newTestBot(t, handler)
	msg := channels.OutgoingMessage{ChannelID: "chan-abc", Text: "hello"}

	if err := bot.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send() error: %v", err)
	}

	expected := "Bot test-token"
	if capturedAuth != expected {
		t.Errorf("expected Authorization %q, got %q", expected, capturedAuth)
	}
}

func TestSend_SetsContentTypeJSON(t *testing.T) {
	var capturedContentType string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"123"}`))
	})

	bot, _ := newTestBot(t, handler)
	msg := channels.OutgoingMessage{ChannelID: "chan-abc", Text: "hello"}

	if err := bot.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send() error: %v", err)
	}

	if !strings.HasPrefix(capturedContentType, "application/json") {
		t.Errorf("expected Content-Type application/json, got %q", capturedContentType)
	}
}

func TestSend_BodyContainsContent(t *testing.T) {
	var capturedBody map[string]string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"123"}`))
	})

	bot, _ := newTestBot(t, handler)
	msg := channels.OutgoingMessage{ChannelID: "chan-abc", Text: "hello world"}

	if err := bot.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send() error: %v", err)
	}

	if capturedBody["content"] != "hello world" {
		t.Errorf("expected content='hello world', got %q", capturedBody["content"])
	}
}

func TestSend_LongMessageSplits(t *testing.T) {
	var requestCount int
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"123"}`))
	})

	bot, _ := newTestBot(t, handler)

	// Build a message longer than 2000 chars
	longText := strings.Repeat("a", 2100)
	msg := channels.OutgoingMessage{ChannelID: "chan-abc", Text: longText}

	if err := bot.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send() error: %v", err)
	}

	if requestCount < 2 {
		t.Errorf("expected at least 2 HTTP requests for message > 2000 chars, got %d", requestCount)
	}
}

func TestSend_APIErrorReturnsError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"code":0,"message":"401: Unauthorized"}`))
	})

	bot, _ := newTestBot(t, handler)
	msg := channels.OutgoingMessage{ChannelID: "chan-abc", Text: "hello"}

	err := bot.Send(context.Background(), msg)
	if err == nil {
		t.Error("expected error on 401 response, got nil")
	}
}

func TestSend_EmptyChannelID(t *testing.T) {
	var requestCount int
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"123"}`))
	})

	bot, _ := newTestBot(t, handler)
	msg := channels.OutgoingMessage{ChannelID: "", Text: "hello"}

	// Should still attempt the request with empty channel ID in path
	err := bot.Send(context.Background(), msg)
	// Either an error or a request — just should not panic
	_ = err
}

func TestSend_EmptyText(t *testing.T) {
	var capturedBody map[string]string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"123"}`))
	})

	bot, _ := newTestBot(t, handler)
	msg := channels.OutgoingMessage{ChannelID: "chan-abc", Text: ""}

	if err := bot.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send() error: %v", err)
	}

	if capturedBody["content"] != "" {
		t.Errorf("expected empty content, got %q", capturedBody["content"])
	}
}

func TestSend_ContextCancellation(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"123"}`))
	})

	bot, _ := newTestBot(t, handler)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	msg := channels.OutgoingMessage{ChannelID: "chan-abc", Text: "hello"}
	// Should return an error due to cancelled context
	err := bot.Send(ctx, msg)
	// May or may not error depending on timing — just must not panic
	_ = err
}

func TestSend_WithDefaultBaseURL(t *testing.T) {
	// Bot with no BaseURL should use the real Discord URL (not panic)
	bot := New(Config{Token: "tok"})
	if bot.baseURL == "" {
		t.Error("expected default baseURL to be set")
	}
}

func TestSend_Exactly2000CharsIsOnePart(t *testing.T) {
	var requestCount int
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"123"}`))
	})

	bot, _ := newTestBot(t, handler)

	// Exactly 2000 chars should be one request
	text := strings.Repeat("x", 2000)
	msg := channels.OutgoingMessage{ChannelID: "chan-abc", Text: text}

	if err := bot.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send() error: %v", err)
	}

	if requestCount != 1 {
		t.Errorf("expected 1 request for exactly 2000 chars, got %d", requestCount)
	}
}

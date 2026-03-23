package telegram

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/meowai/blackcat/internal/channels"
)

func TestTelegramBotPlatform(t *testing.T) {
	bot := New(Config{Token: "test"})
	if bot.Platform() != channels.PlatformTelegram {
		t.Errorf("expected telegram, got %s", bot.Platform())
	}
}

func TestTelegramBotSend(t *testing.T) {
	var receivedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	bot := New(Config{Token: "test123", BaseURL: server.URL})

	err := bot.Send(context.Background(), channels.OutgoingMessage{
		ChannelID: "12345",
		Text:      "Hello from BlackCat!",
		Format:    channels.FormatMarkdown,
	})
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if receivedBody["chat_id"] != "12345" {
		t.Errorf("expected chat_id '12345', got %v", receivedBody["chat_id"])
	}
	if receivedBody["text"] != "Hello from BlackCat!" {
		t.Errorf("expected message text, got %v", receivedBody["text"])
	}
	if receivedBody["parse_mode"] != "Markdown" {
		t.Errorf("expected Markdown parse_mode, got %v", receivedBody["parse_mode"])
	}
}

func TestTelegramBotSendPlain(t *testing.T) {
	var receivedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	bot := New(Config{Token: "test", BaseURL: server.URL})
	bot.Send(context.Background(), channels.OutgoingMessage{
		ChannelID: "1", Text: "plain text", Format: channels.FormatPlain,
	})

	if _, ok := receivedBody["parse_mode"]; ok {
		t.Error("expected no parse_mode for plain format")
	}
}

func TestTelegramBotGetUpdates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := `{
			"ok": true,
			"result": [{
				"update_id": 100,
				"message": {
					"message_id": 1,
					"from": {"id": 999, "username": "testuser"},
					"chat": {"id": 999},
					"text": "hello",
					"date": 1700000000
				}
			}]
		}`
		w.Write([]byte(resp))
	}))
	defer server.Close()

	bot := New(Config{Token: "test", BaseURL: server.URL})
	updates, err := bot.getUpdates(context.Background(), 0)
	if err != nil {
		t.Fatalf("getUpdates failed: %v", err)
	}
	if len(updates) != 1 {
		t.Fatalf("expected 1 update, got %d", len(updates))
	}
	if updates[0].Message.Text != "hello" {
		t.Errorf("expected 'hello', got %q", updates[0].Message.Text)
	}
	if updates[0].Message.From.Username != "testuser" {
		t.Errorf("expected 'testuser', got %q", updates[0].Message.From.Username)
	}
}

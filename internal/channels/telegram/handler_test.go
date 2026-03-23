package telegram

import (
	"testing"

	"github.com/meowai/blackcat/internal/channels"
)

func TestNewTelegramHandler(t *testing.T) {
	h := NewTelegramHandler("test-token")
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestTelegramHandlerParseUpdate(t *testing.T) {
	h := NewTelegramHandler("test-token")

	data := []byte(`{
		"update_id": 100,
		"message": {
			"message_id": 1,
			"from": {"id": 42, "username": "alice"},
			"chat": {"id": 42},
			"text": "hello blackcat",
			"date": 1700000000
		}
	}`)

	msg, err := h.ParseUpdate(data)
	if err != nil {
		t.Fatalf("ParseUpdate failed: %v", err)
	}
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	if msg.Platform != channels.PlatformTelegram {
		t.Errorf("expected telegram platform, got %s", msg.Platform)
	}
	if msg.Text != "hello blackcat" {
		t.Errorf("expected text 'hello blackcat', got %q", msg.Text)
	}
	if msg.UserID != "42" {
		t.Errorf("expected user ID '42', got %q", msg.UserID)
	}
	if msg.UserName != "alice" {
		t.Errorf("expected username 'alice', got %q", msg.UserName)
	}
	if msg.ChannelID != "42" {
		t.Errorf("expected channel ID '42', got %q", msg.ChannelID)
	}
	if msg.Timestamp != 1700000000 {
		t.Errorf("expected timestamp 1700000000, got %d", msg.Timestamp)
	}
}

func TestTelegramHandlerParseUpdateNoMessage(t *testing.T) {
	h := NewTelegramHandler("test-token")

	data := []byte(`{"update_id": 101}`)
	msg, err := h.ParseUpdate(data)
	if err != nil {
		t.Fatalf("ParseUpdate failed: %v", err)
	}
	if msg != nil {
		t.Error("expected nil message for update without message")
	}
}

func TestTelegramHandlerParseUpdateInvalidJSON(t *testing.T) {
	h := NewTelegramHandler("test-token")

	_, err := h.ParseUpdate([]byte(`{invalid`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestTelegramHandlerFormatMessage(t *testing.T) {
	h := NewTelegramHandler("test-token")

	result := h.FormatMessage(channels.OutgoingMessage{
		ChannelID: "123",
		Text:      "Hello!",
		Format:    channels.FormatMarkdown,
	})

	if result["chat_id"] != "123" {
		t.Errorf("expected chat_id '123', got %v", result["chat_id"])
	}
	if result["text"] != "Hello!" {
		t.Errorf("expected text 'Hello!', got %v", result["text"])
	}
	if result["parse_mode"] != "Markdown" {
		t.Errorf("expected parse_mode 'Markdown', got %v", result["parse_mode"])
	}
}

func TestTelegramHandlerFormatMessagePlain(t *testing.T) {
	h := NewTelegramHandler("test-token")

	result := h.FormatMessage(channels.OutgoingMessage{
		ChannelID: "123",
		Text:      "plain text",
		Format:    channels.FormatPlain,
	})

	if _, ok := result["parse_mode"]; ok {
		t.Error("expected no parse_mode for plain format")
	}
}

func TestTelegramHandlerFormatMessageCode(t *testing.T) {
	h := NewTelegramHandler("test-token")

	result := h.FormatMessage(channels.OutgoingMessage{
		ChannelID: "123",
		Text:      "fmt.Println(\"hi\")",
		Format:    channels.FormatCode,
	})

	if result["parse_mode"] != "Markdown" {
		t.Errorf("expected Markdown for code format, got %v", result["parse_mode"])
	}
	text, ok := result["text"].(string)
	if !ok {
		t.Fatal("expected text to be string")
	}
	if text != "```\nfmt.Println(\"hi\")\n```" {
		t.Errorf("expected code-wrapped text, got %q", text)
	}
}

func TestTelegramHandlerFormatMessageReplyTo(t *testing.T) {
	h := NewTelegramHandler("test-token")

	result := h.FormatMessage(channels.OutgoingMessage{
		ChannelID: "123",
		Text:      "reply",
		ReplyToID: "456",
		Format:    channels.FormatPlain,
	})

	if result["reply_to_message_id"] != "456" {
		t.Errorf("expected reply_to_message_id '456', got %v", result["reply_to_message_id"])
	}
}

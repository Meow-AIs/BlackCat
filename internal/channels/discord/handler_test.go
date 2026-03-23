package discord

import (
	"testing"

	"github.com/meowai/blackcat/internal/channels"
)

func TestNewDiscordHandler(t *testing.T) {
	h := NewDiscordHandler("test-token")
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestDiscordHandlerParseMessage(t *testing.T) {
	h := NewDiscordHandler("test-token")

	data := []byte(`{
		"id": "msg123",
		"channel_id": "chan456",
		"author": {
			"id": "user789",
			"username": "bob"
		},
		"content": "hello discord",
		"timestamp": "2025-01-01T12:00:00.000000+00:00",
		"guild_id": "guild001"
	}`)

	msg, err := h.ParseMessage(data)
	if err != nil {
		t.Fatalf("ParseMessage failed: %v", err)
	}
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	if msg.Platform != channels.PlatformDiscord {
		t.Errorf("expected discord platform, got %s", msg.Platform)
	}
	if msg.Text != "hello discord" {
		t.Errorf("expected 'hello discord', got %q", msg.Text)
	}
	if msg.UserID != "user789" {
		t.Errorf("expected user ID 'user789', got %q", msg.UserID)
	}
	if msg.UserName != "bob" {
		t.Errorf("expected username 'bob', got %q", msg.UserName)
	}
	if msg.ChannelID != "chan456" {
		t.Errorf("expected channel ID 'chan456', got %q", msg.ChannelID)
	}
}

func TestDiscordHandlerParseMessageInvalidJSON(t *testing.T) {
	h := NewDiscordHandler("test-token")
	_, err := h.ParseMessage([]byte(`{bad`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestDiscordHandlerFormatMessage(t *testing.T) {
	h := NewDiscordHandler("test-token")

	result := h.FormatMessage(channels.OutgoingMessage{
		ChannelID: "chan1",
		Text:      "Hello!",
		Format:    channels.FormatMarkdown,
	})

	if result["content"] != "Hello!" {
		t.Errorf("expected content 'Hello!', got %v", result["content"])
	}
}

func TestDiscordHandlerFormatMessageCode(t *testing.T) {
	h := NewDiscordHandler("test-token")

	result := h.FormatMessage(channels.OutgoingMessage{
		Text:   "some code",
		Format: channels.FormatCode,
	})

	text, ok := result["content"].(string)
	if !ok {
		t.Fatal("expected content to be string")
	}
	if text != "```\nsome code\n```" {
		t.Errorf("expected code block, got %q", text)
	}
}

func TestDiscordHandlerSplitLongMessage(t *testing.T) {
	h := NewDiscordHandler("test-token")

	// Short message - no split
	short := "hello"
	parts := h.SplitLongMessage(short, 2000)
	if len(parts) != 1 {
		t.Errorf("expected 1 part for short message, got %d", len(parts))
	}
	if parts[0] != "hello" {
		t.Errorf("expected 'hello', got %q", parts[0])
	}
}

func TestDiscordHandlerSplitLongMessageMultiPart(t *testing.T) {
	h := NewDiscordHandler("test-token")

	// Create a string longer than 10 chars
	long := "abcdefghijklmnopqrstuvwxyz"
	parts := h.SplitLongMessage(long, 10)

	if len(parts) < 2 {
		t.Fatalf("expected at least 2 parts, got %d", len(parts))
	}

	// Verify no part exceeds max
	for i, p := range parts {
		if len(p) > 10 {
			t.Errorf("part %d exceeds max length: %d", i, len(p))
		}
	}

	// Verify full content is preserved
	joined := ""
	for _, p := range parts {
		joined += p
	}
	if joined != long {
		t.Errorf("split content mismatch: %q vs %q", joined, long)
	}
}

func TestDiscordHandlerSplitLongMessageNewlines(t *testing.T) {
	h := NewDiscordHandler("test-token")

	// Should prefer splitting at newlines
	text := "line1\nline2\nline3\nline4"
	parts := h.SplitLongMessage(text, 12)

	if len(parts) < 2 {
		t.Fatalf("expected at least 2 parts, got %d", len(parts))
	}
	for i, p := range parts {
		if len(p) > 12 {
			t.Errorf("part %d exceeds max: %d chars", i, len(p))
		}
	}
}

func TestDiscordHandlerSplitEmpty(t *testing.T) {
	h := NewDiscordHandler("test-token")
	parts := h.SplitLongMessage("", 2000)
	if len(parts) != 1 {
		t.Errorf("expected 1 part for empty message, got %d", len(parts))
	}
}

package slack

import (
	"testing"

	"github.com/meowai/blackcat/internal/channels"
)

func TestNewSlackHandler(t *testing.T) {
	h := NewSlackHandler("xoxb-token", "xapp-token")
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestSlackHandlerParseEvent(t *testing.T) {
	h := NewSlackHandler("xoxb-token", "xapp-token")

	data := []byte(`{
		"event": {
			"type": "message",
			"text": "hello slack",
			"user": "U12345",
			"channel": "C67890",
			"ts": "1700000000.000100"
		}
	}`)

	msg, err := h.ParseEvent(data)
	if err != nil {
		t.Fatalf("ParseEvent failed: %v", err)
	}
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	if msg.Platform != channels.PlatformSlack {
		t.Errorf("expected slack platform, got %s", msg.Platform)
	}
	if msg.Text != "hello slack" {
		t.Errorf("expected 'hello slack', got %q", msg.Text)
	}
	if msg.UserID != "U12345" {
		t.Errorf("expected user ID 'U12345', got %q", msg.UserID)
	}
	if msg.ChannelID != "C67890" {
		t.Errorf("expected channel ID 'C67890', got %q", msg.ChannelID)
	}
}

func TestSlackHandlerParseEventNonMessage(t *testing.T) {
	h := NewSlackHandler("xoxb-token", "xapp-token")

	data := []byte(`{
		"event": {
			"type": "reaction_added",
			"user": "U12345",
			"reaction": "thumbsup"
		}
	}`)

	msg, err := h.ParseEvent(data)
	if err != nil {
		t.Fatalf("ParseEvent failed: %v", err)
	}
	if msg != nil {
		t.Error("expected nil message for non-message event")
	}
}

func TestSlackHandlerParseEventInvalidJSON(t *testing.T) {
	h := NewSlackHandler("xoxb-token", "xapp-token")
	_, err := h.ParseEvent([]byte(`{bad json`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestSlackHandlerParseEventBotMessage(t *testing.T) {
	h := NewSlackHandler("xoxb-token", "xapp-token")

	data := []byte(`{
		"event": {
			"type": "message",
			"subtype": "bot_message",
			"text": "bot says hi",
			"bot_id": "B12345",
			"channel": "C67890",
			"ts": "1700000000.000200"
		}
	}`)

	msg, err := h.ParseEvent(data)
	if err != nil {
		t.Fatalf("ParseEvent failed: %v", err)
	}
	if msg != nil {
		t.Error("expected nil message for bot messages")
	}
}

func TestSlackHandlerFormatBlocks(t *testing.T) {
	h := NewSlackHandler("xoxb-token", "xapp-token")

	blocks := h.FormatBlocks(channels.OutgoingMessage{
		ChannelID: "C123",
		Text:      "Hello from BlackCat!",
		Format:    channels.FormatMarkdown,
	})

	if len(blocks) == 0 {
		t.Fatal("expected at least one block")
	}

	// First block should be a section
	first := blocks[0]
	if first["type"] != "section" {
		t.Errorf("expected section block, got %v", first["type"])
	}
}

func TestSlackHandlerFormatBlocksCode(t *testing.T) {
	h := NewSlackHandler("xoxb-token", "xapp-token")

	blocks := h.FormatBlocks(channels.OutgoingMessage{
		Text:   "fmt.Println()",
		Format: channels.FormatCode,
	})

	if len(blocks) == 0 {
		t.Fatal("expected blocks for code format")
	}
}

func TestSlackHandlerHandleChallengeTrue(t *testing.T) {
	h := NewSlackHandler("xoxb-token", "xapp-token")

	data := []byte(`{
		"type": "url_verification",
		"challenge": "abc123xyz"
	}`)

	challenge, ok := h.HandleChallenge(data)
	if !ok {
		t.Error("expected challenge to be detected")
	}
	if challenge != "abc123xyz" {
		t.Errorf("expected challenge 'abc123xyz', got %q", challenge)
	}
}

func TestSlackHandlerHandleChallengeNotChallenge(t *testing.T) {
	h := NewSlackHandler("xoxb-token", "xapp-token")

	data := []byte(`{
		"type": "event_callback",
		"event": {"type": "message"}
	}`)

	_, ok := h.HandleChallenge(data)
	if ok {
		t.Error("expected no challenge for event_callback")
	}
}

func TestSlackHandlerHandleChallengeInvalidJSON(t *testing.T) {
	h := NewSlackHandler("xoxb-token", "xapp-token")
	_, ok := h.HandleChallenge([]byte(`{bad`))
	if ok {
		t.Error("expected no challenge for invalid JSON")
	}
}

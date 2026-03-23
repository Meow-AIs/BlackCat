package whatsapp

import (
	"testing"

	"github.com/meowai/blackcat/internal/channels"
)

func TestNewWhatsAppHandler(t *testing.T) {
	h := NewWhatsAppHandler()
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestWhatsAppHandlerParseWebhook(t *testing.T) {
	h := NewWhatsAppHandler()

	data := []byte(`{
		"from": "+6281234567890",
		"to": "+6289876543210",
		"body": "hello whatsapp",
		"timestamp": 1700000000,
		"message_id": "msg_001",
		"push_name": "Alice"
	}`)

	msg, err := h.ParseWebhook(data)
	if err != nil {
		t.Fatalf("ParseWebhook failed: %v", err)
	}
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	if msg.Platform != channels.PlatformWhatsApp {
		t.Errorf("expected whatsapp platform, got %s", msg.Platform)
	}
	if msg.Text != "hello whatsapp" {
		t.Errorf("expected 'hello whatsapp', got %q", msg.Text)
	}
	if msg.UserID != "+6281234567890" {
		t.Errorf("expected user ID '+6281234567890', got %q", msg.UserID)
	}
	if msg.UserName != "Alice" {
		t.Errorf("expected username 'Alice', got %q", msg.UserName)
	}
	if msg.ChannelID != "+6281234567890" {
		t.Errorf("expected channel ID to be sender number, got %q", msg.ChannelID)
	}
	if msg.Timestamp != 1700000000 {
		t.Errorf("expected timestamp 1700000000, got %d", msg.Timestamp)
	}
}

func TestWhatsAppHandlerParseWebhookInvalidJSON(t *testing.T) {
	h := NewWhatsAppHandler()
	_, err := h.ParseWebhook([]byte(`{bad`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestWhatsAppHandlerParseWebhookEmptyBody(t *testing.T) {
	h := NewWhatsAppHandler()

	data := []byte(`{
		"from": "+1234",
		"body": "",
		"timestamp": 1700000000
	}`)

	msg, err := h.ParseWebhook(data)
	if err != nil {
		t.Fatalf("ParseWebhook failed: %v", err)
	}
	if msg != nil {
		t.Error("expected nil message for empty body")
	}
}

func TestWhatsAppHandlerFormatMessage(t *testing.T) {
	h := NewWhatsAppHandler()

	result := h.FormatMessage(channels.OutgoingMessage{
		ChannelID: "+6281234567890",
		Text:      "Hello from BlackCat!",
		Format:    channels.FormatPlain,
	})

	if result["to"] != "+6281234567890" {
		t.Errorf("expected to '+6281234567890', got %v", result["to"])
	}
	if result["body"] != "Hello from BlackCat!" {
		t.Errorf("expected body 'Hello from BlackCat!', got %v", result["body"])
	}
}

func TestWhatsAppHandlerFormatMessageCode(t *testing.T) {
	h := NewWhatsAppHandler()

	result := h.FormatMessage(channels.OutgoingMessage{
		ChannelID: "+1234",
		Text:      "some code",
		Format:    channels.FormatCode,
	})

	body, ok := result["body"].(string)
	if !ok {
		t.Fatal("expected body to be string")
	}
	if body != "```\nsome code\n```" {
		t.Errorf("expected code-wrapped body, got %q", body)
	}
}

func TestWhatsAppHandlerFormatMessageMarkdown(t *testing.T) {
	h := NewWhatsAppHandler()

	result := h.FormatMessage(channels.OutgoingMessage{
		ChannelID: "+1234",
		Text:      "*bold* _italic_",
		Format:    channels.FormatMarkdown,
	})

	if result["body"] != "*bold* _italic_" {
		t.Errorf("expected markdown preserved, got %v", result["body"])
	}
}

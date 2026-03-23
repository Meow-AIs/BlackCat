package whatsapp

import (
	"encoding/json"
	"fmt"

	"github.com/meowai/blackcat/internal/channels"
)

// waWebhook represents an incoming WhatsApp message from the bridge.
type waWebhook struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Body      string `json:"body"`
	Timestamp int64  `json:"timestamp"`
	MessageID string `json:"message_id"`
	PushName  string `json:"push_name"`
}

// WhatsAppHandler processes WhatsApp webhook messages and formats responses.
type WhatsAppHandler struct {
	incoming chan channels.IncomingMessage
}

// NewWhatsAppHandler creates a handler for processing WhatsApp messages.
func NewWhatsAppHandler() *WhatsAppHandler {
	return &WhatsAppHandler{
		incoming: make(chan channels.IncomingMessage, 100),
	}
}

// ParseWebhook parses a raw WhatsApp webhook JSON into an IncomingMessage.
// Returns nil for messages with an empty body.
func (h *WhatsAppHandler) ParseWebhook(data []byte) (*channels.IncomingMessage, error) {
	var wh waWebhook
	if err := json.Unmarshal(data, &wh); err != nil {
		return nil, fmt.Errorf("parse whatsapp webhook: %w", err)
	}

	if wh.Body == "" {
		return nil, nil
	}

	msg := channels.IncomingMessage{
		Platform:  channels.PlatformWhatsApp,
		ChannelID: wh.From, // WhatsApp uses sender number as channel
		UserID:    wh.From,
		UserName:  wh.PushName,
		Text:      wh.Body,
		Timestamp: wh.Timestamp,
	}

	return &msg, nil
}

// FormatMessage converts an OutgoingMessage into WhatsApp bridge parameters.
func (h *WhatsAppHandler) FormatMessage(msg channels.OutgoingMessage) map[string]any {
	text := msg.Text
	if msg.Format == channels.FormatCode {
		text = "```\n" + msg.Text + "\n```"
	}

	return map[string]any{
		"to":   msg.ChannelID,
		"body": text,
	}
}

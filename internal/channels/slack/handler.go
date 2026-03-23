package slack

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/meowai/blackcat/internal/channels"
)

// slackEventWrapper wraps Slack event payloads.
type slackEventWrapper struct {
	Type      string     `json:"type"`
	Challenge string     `json:"challenge,omitempty"`
	Event     slackEvent `json:"event"`
}

type slackEvent struct {
	Type    string `json:"type"`
	SubType string `json:"subtype,omitempty"`
	Text    string `json:"text"`
	User    string `json:"user,omitempty"`
	BotID   string `json:"bot_id,omitempty"`
	Channel string `json:"channel"`
	TS      string `json:"ts"`
}

// SlackHandler processes Slack events and formats outgoing messages using Block Kit.
type SlackHandler struct {
	token    string
	appToken string
	incoming chan channels.IncomingMessage
}

// NewSlackHandler creates a handler for processing Slack events.
func NewSlackHandler(token, appToken string) *SlackHandler {
	return &SlackHandler{
		token:    token,
		appToken: appToken,
		incoming: make(chan channels.IncomingMessage, 100),
	}
}

// ParseEvent parses a Slack event callback JSON into an IncomingMessage.
// Returns nil for non-message events and bot messages.
func (h *SlackHandler) ParseEvent(data []byte) (*channels.IncomingMessage, error) {
	var wrapper slackEventWrapper
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("parse slack event: %w", err)
	}

	evt := wrapper.Event

	// Only handle user messages (not bot messages or other event types)
	if evt.Type != "message" {
		return nil, nil
	}
	if evt.SubType == "bot_message" || evt.BotID != "" {
		return nil, nil
	}

	var timestamp int64
	if evt.TS != "" {
		parts := strings.SplitN(evt.TS, ".", 2)
		if ts, err := strconv.ParseInt(parts[0], 10, 64); err == nil {
			timestamp = ts
		}
	}

	msg := channels.IncomingMessage{
		Platform:  channels.PlatformSlack,
		ChannelID: evt.Channel,
		UserID:    evt.User,
		Text:      evt.Text,
		Timestamp: timestamp,
	}

	return &msg, nil
}

// FormatBlocks converts an OutgoingMessage into Slack Block Kit blocks.
func (h *SlackHandler) FormatBlocks(msg channels.OutgoingMessage) []map[string]any {
	text := msg.Text
	textType := "mrkdwn"

	if msg.Format == channels.FormatPlain {
		textType = "plain_text"
	}
	if msg.Format == channels.FormatCode {
		text = "```\n" + msg.Text + "\n```"
	}

	section := map[string]any{
		"type": "section",
		"text": map[string]any{
			"type": textType,
			"text": text,
		},
	}

	return []map[string]any{section}
}

// HandleChallenge checks if the incoming payload is a Slack URL verification challenge.
// Returns the challenge string and true if it is, empty string and false otherwise.
func (h *SlackHandler) HandleChallenge(data []byte) (string, bool) {
	var payload struct {
		Type      string `json:"type"`
		Challenge string `json:"challenge"`
	}

	if err := json.Unmarshal(data, &payload); err != nil {
		return "", false
	}

	if payload.Type == "url_verification" {
		return payload.Challenge, true
	}

	return "", false
}

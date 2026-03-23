package discord

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/meowai/blackcat/internal/channels"
)

// discordMessage represents a Discord MESSAGE_CREATE event payload.
type discordMessage struct {
	ID        string        `json:"id"`
	ChannelID string        `json:"channel_id"`
	GuildID   string        `json:"guild_id"`
	Author    discordAuthor `json:"author"`
	Content   string        `json:"content"`
	Timestamp string        `json:"timestamp"`
}

type discordAuthor struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

// DiscordHandler processes Discord messages and formats outgoing responses.
type DiscordHandler struct {
	token    string
	incoming chan channels.IncomingMessage
}

// NewDiscordHandler creates a handler for processing Discord messages.
func NewDiscordHandler(token string) *DiscordHandler {
	return &DiscordHandler{
		token:    token,
		incoming: make(chan channels.IncomingMessage, 100),
	}
}

// ParseMessage parses a raw Discord message JSON into an IncomingMessage.
func (h *DiscordHandler) ParseMessage(data []byte) (*channels.IncomingMessage, error) {
	var dm discordMessage
	if err := json.Unmarshal(data, &dm); err != nil {
		return nil, fmt.Errorf("parse discord message: %w", err)
	}

	msg := channels.IncomingMessage{
		Platform:  channels.PlatformDiscord,
		ChannelID: dm.ChannelID,
		UserID:    dm.Author.ID,
		UserName:  dm.Author.Username,
		Text:      dm.Content,
	}

	return &msg, nil
}

// FormatMessage converts an OutgoingMessage into Discord API parameters.
func (h *DiscordHandler) FormatMessage(msg channels.OutgoingMessage) map[string]any {
	text := msg.Text
	if msg.Format == channels.FormatCode {
		text = "```\n" + msg.Text + "\n```"
	}

	return map[string]any{
		"content": text,
	}
}

// SplitLongMessage splits text into chunks that fit within Discord's character limit.
// It prefers splitting at newline boundaries when possible.
func (h *DiscordHandler) SplitLongMessage(text string, maxLen int) []string {
	if len(text) <= maxLen {
		return []string{text}
	}

	var parts []string
	remaining := text

	for len(remaining) > 0 {
		if len(remaining) <= maxLen {
			parts = append(parts, remaining)
			break
		}

		// Try to find a newline to split at
		chunk := remaining[:maxLen]
		splitIdx := strings.LastIndex(chunk, "\n")

		if splitIdx > 0 {
			parts = append(parts, remaining[:splitIdx+1])
			remaining = remaining[splitIdx+1:]
		} else {
			// No newline found, hard split at max
			parts = append(parts, chunk)
			remaining = remaining[maxLen:]
		}
	}

	return parts
}

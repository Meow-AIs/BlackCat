package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/meowai/blackcat/internal/channels"
)

// TelegramHandler processes Telegram updates and formats outgoing messages.
type TelegramHandler struct {
	token    string
	incoming chan channels.IncomingMessage
	client   *http.Client
}

// NewTelegramHandler creates a handler for processing Telegram messages.
func NewTelegramHandler(token string) *TelegramHandler {
	return &TelegramHandler{
		token:    token,
		incoming: make(chan channels.IncomingMessage, 100),
		client:   &http.Client{},
	}
}

// ParseUpdate parses a raw Telegram update JSON into an IncomingMessage.
// Returns nil message (no error) if the update has no text message.
func (h *TelegramHandler) ParseUpdate(data []byte) (*channels.IncomingMessage, error) {
	var update tgUpdate
	if err := json.Unmarshal(data, &update); err != nil {
		return nil, fmt.Errorf("parse telegram update: %w", err)
	}

	if update.Message == nil {
		return nil, nil
	}

	msg := channels.IncomingMessage{
		Platform:  channels.PlatformTelegram,
		ChannelID: fmt.Sprintf("%d", update.Message.Chat.ID),
		UserID:    fmt.Sprintf("%d", update.Message.From.ID),
		UserName:  update.Message.From.Username,
		Text:      update.Message.Text,
		Timestamp: int64(update.Message.Date),
	}

	return &msg, nil
}

// FormatMessage converts an OutgoingMessage into Telegram API parameters.
func (h *TelegramHandler) FormatMessage(msg channels.OutgoingMessage) map[string]any {
	text := msg.Text
	if msg.Format == channels.FormatCode {
		text = "```\n" + msg.Text + "\n```"
	}

	params := map[string]any{
		"chat_id": msg.ChannelID,
		"text":    text,
	}

	if msg.Format != channels.FormatPlain {
		params["parse_mode"] = "Markdown"
	}

	if msg.ReplyToID != "" {
		params["reply_to_message_id"] = msg.ReplyToID
	}

	return params
}

// SendMessage sends a text message to a Telegram chat.
func (h *TelegramHandler) SendMessage(ctx context.Context, chatID string, text string, format channels.Format) error {
	params := h.FormatMessage(channels.OutgoingMessage{
		ChannelID: chatID,
		Text:      text,
		Format:    format,
	})

	payload, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", h.token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(payload)))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("send telegram message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}

	return nil
}

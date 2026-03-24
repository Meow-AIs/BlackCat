package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/meowai/blackcat/internal/channels"
)

// Bot implements the channels.Adapter interface for Telegram.
//
// Current implementation: raw HTTP long-polling against the Telegram Bot API.
// This works for low-traffic bots but lacks features like media handling,
// inline keyboards, and webhook support.
//
// For enhanced features (webhooks, media, MarkdownV2), migrate to
// github.com/go-telegram/bot. See docs/native-channels-guide.md.
type Bot struct {
	token        string
	allowedUsers map[int64]bool
	baseURL      string
	incoming     chan channels.IncomingMessage
	cancel       context.CancelFunc
	client       *http.Client
}

// Config for creating a Telegram bot.
type Config struct {
	Token        string
	AllowedUsers []int64
	BaseURL      string // override for testing, default: https://api.telegram.org
}

// New creates a Telegram bot adapter.
//
// The returned Bot uses raw HTTP polling against the Telegram Bot API.
// For production deployments requiring media handling, webhooks, or
// MarkdownV2 formatting, migrate to go-telegram/bot.
// See docs/native-channels-guide.md#telegram-go-telegrambot.
func New(cfg Config) *Bot {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.telegram.org"
	}

	allowed := make(map[int64]bool, len(cfg.AllowedUsers))
	for _, uid := range cfg.AllowedUsers {
		allowed[uid] = true
	}

	return &Bot{
		token:        cfg.Token,
		allowedUsers: allowed,
		baseURL:      baseURL,
		incoming:     make(chan channels.IncomingMessage, 100),
		client:       &http.Client{Timeout: 35 * time.Second},
	}
}

// Platform returns the platform identifier for this adapter.
func (b *Bot) Platform() channels.Platform { return channels.PlatformTelegram }

// Start connects to Telegram and begins long-polling for updates.
//
// Native upgrade: use go-telegram/bot managed polling.
// See docs/native-channels-guide.md#starting-the-long-poll-loop.
func (b *Bot) Start(ctx context.Context) error {
	ctx, b.cancel = context.WithCancel(ctx)
	go b.pollLoop(ctx)
	return nil
}

// Stop gracefully disconnects from Telegram and closes the incoming channel.
func (b *Bot) Stop(_ context.Context) error {
	if b.cancel != nil {
		b.cancel()
	}
	close(b.incoming)
	return nil
}

// Receive returns the channel that emits incoming messages from Telegram.
func (b *Bot) Receive() <-chan channels.IncomingMessage { return b.incoming }

// Send delivers a message to a Telegram chat.
//
// Native upgrade: use go-telegram/bot SendMessage for MarkdownV2, media, buttons.
// See docs/native-channels-guide.md#sending-messages.
func (b *Bot) Send(ctx context.Context, msg channels.OutgoingMessage) error {
	parseMode := "Markdown"
	if msg.Format == channels.FormatPlain {
		parseMode = ""
	}

	params := map[string]any{
		"chat_id": msg.ChannelID,
		"text":    msg.Text,
	}
	if parseMode != "" {
		params["parse_mode"] = parseMode
	}
	if msg.ReplyToID != "" {
		params["reply_to_message_id"] = msg.ReplyToID
	}

	return b.apiCall(ctx, "sendMessage", params)
}

// pollLoop continuously fetches updates from the Telegram Bot API using
// long-polling. Each update is converted to an IncomingMessage and sent
// to the incoming channel.
//
// Native upgrade: this method is replaced by go-telegram/bot's internal polling.
// Register a default handler instead.
func (b *Bot) pollLoop(ctx context.Context) {
	offset := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		updates, err := b.getUpdates(ctx, offset)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		for _, u := range updates {
			if u.UpdateID >= offset {
				offset = u.UpdateID + 1
			}
			if u.Message == nil {
				continue
			}

			// Check user whitelist
			if len(b.allowedUsers) > 0 && !b.allowedUsers[u.Message.From.ID] {
				continue
			}

			b.incoming <- channels.IncomingMessage{
				Platform:  channels.PlatformTelegram,
				ChannelID: fmt.Sprintf("%d", u.Message.Chat.ID),
				UserID:    fmt.Sprintf("%d", u.Message.From.ID),
				UserName:  u.Message.From.Username,
				Text:      u.Message.Text,
				Timestamp: int64(u.Message.Date),
			}
		}
	}
}

// --- Telegram API types (minimal subset) ---

type tgUpdate struct {
	UpdateID int        `json:"update_id"`
	Message  *tgMessage `json:"message,omitempty"`
}

type tgMessage struct {
	MessageID int    `json:"message_id"`
	From      tgUser `json:"from"`
	Chat      tgChat `json:"chat"`
	Text      string `json:"text"`
	Date      int    `json:"date"`
}

type tgUser struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type tgChat struct {
	ID int64 `json:"id"`
}

func (b *Bot) getUpdates(ctx context.Context, offset int) ([]tgUpdate, error) {
	url := fmt.Sprintf("%s/bot%s/getUpdates?offset=%d&timeout=30", b.baseURL, b.token, offset)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	var result struct {
		OK     bool       `json:"ok"`
		Result []tgUpdate `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return result.Result, nil
}

func (b *Bot) apiCall(ctx context.Context, method string, params map[string]any) error {
	data, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshal params: %w", err)
	}
	url := fmt.Sprintf("%s/bot%s/%s", b.baseURL, b.token, method)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(data)))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API error %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/meowai/blackcat/internal/channels"
)

const (
	discordBaseURL  = "https://discord.com"
	discordMaxChars = 2000
)

// Bot implements the channels.Adapter interface for Discord.
//
// Send() uses the Discord REST API v10 to deliver messages.
// Messages longer than 2000 characters are split at newline boundaries.
type Bot struct {
	token           string
	baseURL         string
	httpClient      *http.Client
	allowedGuilds   map[string]bool
	allowedChannels map[string]bool
	incoming        chan channels.IncomingMessage
	cancel          context.CancelFunc
}

// Config for creating a Discord bot.
type Config struct {
	Token           string
	AllowedGuilds   []string
	AllowedChannels []string
	// BaseURL overrides the Discord API base URL. Used for testing with httptest.
	BaseURL string
}

// New creates a Discord bot adapter.
func New(cfg Config) *Bot {
	guilds := make(map[string]bool, len(cfg.AllowedGuilds))
	for _, g := range cfg.AllowedGuilds {
		guilds[g] = true
	}
	chans := make(map[string]bool, len(cfg.AllowedChannels))
	for _, c := range cfg.AllowedChannels {
		chans[c] = true
	}
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = discordBaseURL
	}
	return &Bot{
		token:           cfg.Token,
		baseURL:         baseURL,
		httpClient:      &http.Client{},
		allowedGuilds:   guilds,
		allowedChannels: chans,
		incoming:        make(chan channels.IncomingMessage, 100),
	}
}

// Platform returns the platform identifier for this adapter.
func (b *Bot) Platform() channels.Platform { return channels.PlatformDiscord }

// Start connects to Discord and begins listening for messages.
func (b *Bot) Start(ctx context.Context) error {
	ctx, b.cancel = context.WithCancel(ctx)
	go b.connectGateway(ctx)
	return nil
}

// Stop gracefully disconnects from Discord and closes the incoming channel.
func (b *Bot) Stop(_ context.Context) error {
	if b.cancel != nil {
		b.cancel()
	}
	close(b.incoming)
	return nil
}

// Receive returns the channel that emits incoming messages from Discord.
func (b *Bot) Receive() <-chan channels.IncomingMessage { return b.incoming }

// Send delivers a message to a Discord channel using the REST API v10.
//
// Messages longer than 2000 characters are split into multiple requests,
// preferring to break at newline boundaries where possible.
// Each part is sent as a POST to /api/v10/channels/{channel_id}/messages
// with Authorization: Bot {token} and Content-Type: application/json.
func (b *Bot) Send(ctx context.Context, msg channels.OutgoingMessage) error {
	parts := splitMessage(msg.Text, discordMaxChars)
	for _, part := range parts {
		if err := b.sendPart(ctx, msg.ChannelID, part); err != nil {
			return err
		}
	}
	return nil
}

// sendPart POSTs a single message chunk to the Discord REST API.
func (b *Bot) sendPart(ctx context.Context, channelID, text string) error {
	url := fmt.Sprintf("%s/api/v10/channels/%s/messages", b.baseURL, channelID)

	body, err := json.Marshal(map[string]string{"content": text})
	if err != nil {
		return fmt.Errorf("discord: marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("discord: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bot "+b.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("discord: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord: API error %d for channel %s", resp.StatusCode, channelID)
	}
	return nil
}

// connectGateway is a placeholder for the Discord WebSocket gateway connection.
func (b *Bot) connectGateway(ctx context.Context) {
	<-ctx.Done()
}

// IsAllowed checks if a message from the given guild/channel should be processed.
func (b *Bot) IsAllowed(guildID, channelID string) bool {
	if len(b.allowedGuilds) > 0 && !b.allowedGuilds[guildID] {
		return false
	}
	if len(b.allowedChannels) > 0 && !b.allowedChannels[channelID] {
		return false
	}
	return true
}

// splitMessage breaks text into chunks respecting maxLen, preferring newline boundaries.
func splitMessage(text string, maxLen int) []string {
	if len(text) <= maxLen {
		return []string{text}
	}

	var chunks []string
	for len(text) > 0 {
		if len(text) <= maxLen {
			chunks = append(chunks, text)
			break
		}
		cutAt := maxLen
		if idx := strings.LastIndex(text[:maxLen], "\n"); idx > 0 {
			cutAt = idx + 1
		}
		chunks = append(chunks, text[:cutAt])
		text = text[cutAt:]
	}
	return chunks
}

// truncate shortens a string to maxLen characters, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

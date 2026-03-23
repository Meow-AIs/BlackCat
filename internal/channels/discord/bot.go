package discord

import (
	"context"
	"fmt"

	"github.com/meowai/blackcat/internal/channels"
)

// Bot implements channels.Adapter for Discord.
type Bot struct {
	token           string
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
}

// New creates a Discord bot adapter.
func New(cfg Config) *Bot {
	guilds := make(map[string]bool)
	for _, g := range cfg.AllowedGuilds {
		guilds[g] = true
	}
	chans := make(map[string]bool)
	for _, c := range cfg.AllowedChannels {
		chans[c] = true
	}
	return &Bot{
		token:           cfg.Token,
		allowedGuilds:   guilds,
		allowedChannels: chans,
		incoming:        make(chan channels.IncomingMessage, 100),
	}
}

func (b *Bot) Platform() channels.Platform { return channels.PlatformDiscord }

func (b *Bot) Start(ctx context.Context) error {
	ctx, b.cancel = context.WithCancel(ctx)
	go b.connectGateway(ctx)
	return nil
}

func (b *Bot) Stop(_ context.Context) error {
	if b.cancel != nil {
		b.cancel()
	}
	close(b.incoming)
	return nil
}

func (b *Bot) Receive() <-chan channels.IncomingMessage { return b.incoming }

func (b *Bot) Send(ctx context.Context, msg channels.OutgoingMessage) error {
	// Discord uses embeds for rich formatting
	// Actual implementation would POST to /channels/{id}/messages
	_ = ctx
	_ = fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages", msg.ChannelID)
	return nil
}

func (b *Bot) connectGateway(ctx context.Context) {
	// Actual implementation:
	// 1. GET /gateway/bot → get websocket URL
	// 2. Connect via WSS
	// 3. Send IDENTIFY payload
	// 4. Handle MESSAGE_CREATE events
	// 5. Filter by guild/channel whitelist
	// 6. Convert to IncomingMessage and push to b.incoming
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

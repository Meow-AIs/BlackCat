package discord

import (
	"context"
	"fmt"
	"strings"

	"github.com/meowai/blackcat/internal/channels"
)

// Bot implements the channels.Adapter interface for Discord.
//
// Current implementation: stub that connects to the gateway conceptually but
// does not perform real WebSocket communication. Send is a no-op.
//
// TODO: Replace with github.com/bwmarrin/discordgo for production use.
// See docs/native-channels-guide.md#discord-discordgo for the full
// migration pattern including:
//   - WebSocket gateway connection via discordgo.Session.Open()
//   - MessageCreate event handler for incoming messages
//   - Slash command registration via ApplicationCommandCreate
//   - Embed formatting for rich responses
//   - 2000-character message splitting
//   - Message Content Intent declaration
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
//
// The returned Bot is a stub that satisfies the channels.Adapter interface
// but does not perform real Discord API communication. For production
// deployments, migrate to discordgo.
// See docs/native-channels-guide.md#discord-discordgo.
func New(cfg Config) *Bot {
	guilds := make(map[string]bool, len(cfg.AllowedGuilds))
	for _, g := range cfg.AllowedGuilds {
		guilds[g] = true
	}
	chans := make(map[string]bool, len(cfg.AllowedChannels))
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

// Platform returns the platform identifier for this adapter.
func (b *Bot) Platform() channels.Platform { return channels.PlatformDiscord }

// Start connects to Discord and begins listening for messages.
//
// TODO(native): Replace with discordgo session setup:
//
//	session, _ := discordgo.New("Bot " + b.token)
//	session.Identify.Intents = discordgo.IntentsGuildMessages |
//	    discordgo.IntentsDirectMessages | discordgo.IntentsMessageContent
//	session.AddHandler(b.onMessageCreate)
//	session.Open()
//
// See docs/native-channels-guide.md#connecting-the-gateway.
func (b *Bot) Start(ctx context.Context) error {
	ctx, b.cancel = context.WithCancel(ctx)
	go b.connectGateway(ctx)
	return nil
}

// Stop gracefully disconnects from Discord and closes the incoming channel.
//
// TODO(native): Also call session.Close() to cleanly disconnect the WebSocket.
func (b *Bot) Stop(_ context.Context) error {
	if b.cancel != nil {
		b.cancel()
	}
	close(b.incoming)
	return nil
}

// Receive returns the channel that emits incoming messages from Discord.
func (b *Bot) Receive() <-chan channels.IncomingMessage { return b.incoming }

// Send delivers a message to a Discord channel.
//
// TODO(native): Replace with discordgo's ChannelMessageSendComplex which supports:
//   - MessageReference for reply threading
//   - Embed objects for rich formatting
//   - 2000-character message splitting (see splitMessage helper in guide)
//   - File attachments via MessageSend.Files
//
// See docs/native-channels-guide.md#sending-messages-with-2000-character-splitting.
func (b *Bot) Send(_ context.Context, msg channels.OutgoingMessage) error {
	// Stub: log the intent but do not actually call the Discord API.
	// In production, this would POST to /channels/{id}/messages via discordgo.
	_ = fmt.Sprintf("discord stub: would send to channel %s: %s",
		msg.ChannelID, truncate(msg.Text, 50))
	return nil
}

// connectGateway is a placeholder for the Discord WebSocket gateway connection.
//
// TODO(native): Replace entirely with discordgo's managed connection.
// The native flow is:
//  1. discordgo.New("Bot " + token) creates the session
//  2. session.Open() connects via WSS to the Discord gateway
//  3. session.AddHandler(onMessageCreate) receives MESSAGE_CREATE events
//  4. session.AddHandler(onInteractionCreate) receives slash command interactions
//  5. Filter events by guild/channel whitelist via IsAllowed
//  6. Convert to channels.IncomingMessage and push to b.incoming
//
// See docs/native-channels-guide.md#handling-incoming-messages-1.
func (b *Bot) connectGateway(ctx context.Context) {
	<-ctx.Done()
}

// IsAllowed checks if a message from the given guild/channel should be processed.
// This filter is applied inside the message handler before converting to
// IncomingMessage.
func (b *Bot) IsAllowed(guildID, channelID string) bool {
	if len(b.allowedGuilds) > 0 && !b.allowedGuilds[guildID] {
		return false
	}
	if len(b.allowedChannels) > 0 && !b.allowedChannels[channelID] {
		return false
	}
	return true
}

// splitMessage breaks text into chunks respecting Discord's 2000-char limit,
// preferring to split at newline boundaries.
//
// TODO(native): Use this helper inside Send when migrating to discordgo.
// See docs/native-channels-guide.md#sending-messages-with-2000-character-splitting.
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

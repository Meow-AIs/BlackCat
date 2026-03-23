package slack

import (
	"context"
	"fmt"

	"github.com/meowai/blackcat/internal/channels"
)

// Bot implements channels.Adapter for Slack via Socket Mode.
type Bot struct {
	appToken        string
	botToken        string
	allowedChannels map[string]bool
	incoming        chan channels.IncomingMessage
	cancel          context.CancelFunc
}

// Config for creating a Slack bot.
type Config struct {
	AppToken        string
	BotToken        string
	AllowedChannels []string
}

// New creates a Slack bot adapter.
func New(cfg Config) *Bot {
	chans := make(map[string]bool)
	for _, c := range cfg.AllowedChannels {
		chans[c] = true
	}
	return &Bot{
		appToken:        cfg.AppToken,
		botToken:        cfg.BotToken,
		allowedChannels: chans,
		incoming:        make(chan channels.IncomingMessage, 100),
	}
}

func (b *Bot) Platform() channels.Platform { return channels.PlatformSlack }

func (b *Bot) Start(ctx context.Context) error {
	ctx, b.cancel = context.WithCancel(ctx)
	go b.connectSocketMode(ctx)
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
	// Actual implementation would use Slack Web API chat.postMessage
	// with Block Kit formatting for rich messages
	_ = ctx
	_ = fmt.Sprintf("https://slack.com/api/chat.postMessage")
	return nil
}

func (b *Bot) connectSocketMode(ctx context.Context) {
	// Actual implementation:
	// 1. POST apps.connections.open with app token → get WSS URL
	// 2. Connect via WebSocket
	// 3. Handle event_callback for message events
	// 4. ACK each envelope
	// 5. Filter by allowed channels
	// 6. Support thread replies (thread_ts)
	<-ctx.Done()
}

// IsAllowed checks if a message from the given channel should be processed.
func (b *Bot) IsAllowed(channelID string) bool {
	if len(b.allowedChannels) == 0 {
		return true
	}
	return b.allowedChannels[channelID]
}

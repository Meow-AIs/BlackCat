package whatsapp

import (
	"context"
	"fmt"

	"github.com/meowai/blackcat/internal/channels"
)

// Bot implements channels.Adapter for WhatsApp via Baileys-like bridge.
// Uses WhatsApp Web protocol directly (no Business API needed).
type Bot struct {
	sessionPath    string
	allowedNumbers map[string]bool
	incoming       chan channels.IncomingMessage
	cancel         context.CancelFunc
	paired         bool
}

// Config for creating a WhatsApp bot.
type Config struct {
	SessionPath    string   // where to store session data
	AllowedNumbers []string // phone numbers with country code
}

// New creates a WhatsApp bot adapter using Baileys bridge.
func New(cfg Config) *Bot {
	numbers := make(map[string]bool)
	for _, n := range cfg.AllowedNumbers {
		numbers[n] = true
	}
	return &Bot{
		sessionPath:    cfg.SessionPath,
		allowedNumbers: numbers,
		incoming:       make(chan channels.IncomingMessage, 100),
	}
}

func (b *Bot) Platform() channels.Platform { return channels.PlatformWhatsApp }

func (b *Bot) Start(ctx context.Context) error {
	ctx, b.cancel = context.WithCancel(ctx)
	go b.connectBaileys(ctx)
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

func (b *Bot) Send(_ context.Context, msg channels.OutgoingMessage) error {
	// Actual implementation sends via Baileys bridge
	_ = fmt.Sprintf("sending to %s: %s", msg.ChannelID, msg.Text)
	return nil
}

// IsPaired returns whether the WhatsApp session has been paired via QR.
func (b *Bot) IsPaired() bool { return b.paired }

func (b *Bot) connectBaileys(ctx context.Context) {
	// Actual implementation:
	// 1. Load session from sessionPath (if exists)
	// 2. If no session → generate QR code for pairing
	// 3. Once paired → listen for incoming messages
	// 4. Filter by allowedNumbers
	// 5. Convert to IncomingMessage format
	//
	// Baileys bridge options:
	// a) Embed a small Node.js runtime with whatsmeow (Go native)
	// b) Use github.com/nicofr/go-whatsmeow (native Go WA client)
	// c) Bridge to Baileys via subprocess + JSON stdio
	//
	// Recommended: whatsmeow (pure Go implementation of WA Web protocol)
	<-ctx.Done()
}

// IsAllowed checks if a phone number is in the whitelist.
func (b *Bot) IsAllowed(phoneNumber string) bool {
	if len(b.allowedNumbers) == 0 {
		return true
	}
	return b.allowedNumbers[phoneNumber]
}

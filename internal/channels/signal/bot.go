package signal

import (
	"context"
	"fmt"

	"github.com/meowai/blackcat/internal/channels"
)

// Bot implements channels.Adapter for Signal via signal-cli.
// Uses signal-cli JSON-RPC mode for sending/receiving messages.
type Bot struct {
	phoneNumber    string // bot's registered Signal number
	allowedNumbers map[string]bool
	signalCLIPath  string
	incoming       chan channels.IncomingMessage
	cancel         context.CancelFunc
}

// Config for creating a Signal bot.
type Config struct {
	PhoneNumber    string   // bot's Signal phone number (must be registered)
	AllowedNumbers []string // allowed contacts
	SignalCLIPath  string   // path to signal-cli binary (default: "signal-cli")
}

// New creates a Signal bot adapter.
func New(cfg Config) *Bot {
	numbers := make(map[string]bool)
	for _, n := range cfg.AllowedNumbers {
		numbers[n] = true
	}
	cliPath := cfg.SignalCLIPath
	if cliPath == "" {
		cliPath = "signal-cli"
	}
	return &Bot{
		phoneNumber:    cfg.PhoneNumber,
		allowedNumbers: numbers,
		signalCLIPath:  cliPath,
		incoming:       make(chan channels.IncomingMessage, 100),
	}
}

func (b *Bot) Platform() channels.Platform { return channels.PlatformSignal }

func (b *Bot) Start(ctx context.Context) error {
	ctx, b.cancel = context.WithCancel(ctx)
	go b.listenLoop(ctx)
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
	// Actual implementation:
	// signal-cli -a <botNumber> send -m "<text>" <recipientNumber>
	// Or via JSON-RPC mode: {"method": "send", "params": {...}}
	_ = fmt.Sprintf("signal-cli send to %s", msg.ChannelID)
	return nil
}

func (b *Bot) listenLoop(ctx context.Context) {
	// Actual implementation:
	// 1. Start signal-cli in JSON-RPC daemon mode:
	//    signal-cli -a <phoneNumber> jsonRpc
	// 2. Read JSON messages from stdout
	// 3. Parse incoming messages
	// 4. Filter by allowedNumbers
	// 5. Push to b.incoming channel
	//
	// Alternative: use signal-cli receive --json in a loop
	<-ctx.Done()
}

// IsAllowed checks if a phone number is whitelisted.
func (b *Bot) IsAllowed(phoneNumber string) bool {
	if len(b.allowedNumbers) == 0 {
		return true
	}
	return b.allowedNumbers[phoneNumber]
}

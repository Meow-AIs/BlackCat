package whatsapp

import (
	"context"
	"fmt"

	"github.com/meowai/blackcat/internal/channels"
)

// Bot implements the channels.Adapter interface for WhatsApp.
//
// Current implementation: stub that satisfies the Adapter interface but does
// not perform real WhatsApp communication. The connectBaileys method blocks
// until context cancellation without establishing a connection.
//
// TODO: Replace with go.mau.fi/whatsmeow for production use.
// whatsmeow is a pure Go implementation of the WhatsApp Web multi-device
// protocol -- no Node.js, no Baileys subprocess, no external runtime.
// See docs/native-channels-guide.md#whatsapp-whatsmeow for the full
// migration pattern including:
//   - QR code pairing flow via client.GetQRChannel
//   - Session persistence in SQLite via sqlstore
//   - Event-driven message handling via client.AddEventHandler
//   - Automatic reconnection on disconnect
//   - Handling session expiry (events.LoggedOut)
//   - Advantages over Baileys bridge (no Node.js, lower memory, in-process)
type Bot struct {
	sessionPath    string
	allowedNumbers map[string]bool
	incoming       chan channels.IncomingMessage
	cancel         context.CancelFunc
	paired         bool
}

// Config for creating a WhatsApp bot.
type Config struct {
	// SessionPath is the directory where WhatsApp session data is stored.
	// With whatsmeow, this becomes a SQLite database path.
	// Example: ~/.blackcat/whatsapp-session
	SessionPath string

	// AllowedNumbers is a whitelist of phone numbers in E.164 format
	// (e.g., "+1234567890"). Empty list allows all numbers.
	AllowedNumbers []string
}

// New creates a WhatsApp bot adapter.
//
// The returned Bot is a stub that satisfies the channels.Adapter interface
// but does not perform real WhatsApp communication. For production
// deployments, migrate to whatsmeow (pure Go, no Node.js needed).
// See docs/native-channels-guide.md#whatsapp-whatsmeow.
func New(cfg Config) *Bot {
	numbers := make(map[string]bool, len(cfg.AllowedNumbers))
	for _, n := range cfg.AllowedNumbers {
		numbers[n] = true
	}
	return &Bot{
		sessionPath:    cfg.SessionPath,
		allowedNumbers: numbers,
		incoming:       make(chan channels.IncomingMessage, 100),
	}
}

// Platform returns the platform identifier for this adapter.
func (b *Bot) Platform() channels.Platform { return channels.PlatformWhatsApp }

// Start connects to WhatsApp and begins listening for messages.
//
// TODO(native): Replace with whatsmeow client setup:
//
//	container, _ := sqlstore.New("sqlite3", dbURI, waLog.Noop)
//	deviceStore, _ := container.GetFirstDevice()
//	b.client = whatsmeow.NewClient(deviceStore, waLog.Noop)
//	b.client.AddEventHandler(b.eventHandler)
//	if b.client.Store.ID == nil {
//	    // No session: start QR pairing flow
//	    qrChan, _ := b.client.GetQRChannel(ctx)
//	    b.client.Connect()
//	    // Display QR codes from qrChan
//	} else {
//	    b.client.Connect() // Reconnect with stored session
//	}
//
// See docs/native-channels-guide.md#qr-code-pairing-flow.
func (b *Bot) Start(ctx context.Context) error {
	ctx, b.cancel = context.WithCancel(ctx)
	go b.connectBaileys(ctx)
	return nil
}

// Stop gracefully disconnects from WhatsApp and closes the incoming channel.
//
// TODO(native): Also call b.client.Disconnect() to cleanly close the
// WhatsApp Web connection.
func (b *Bot) Stop(_ context.Context) error {
	if b.cancel != nil {
		b.cancel()
	}
	close(b.incoming)
	return nil
}

// Receive returns the channel that emits incoming messages from WhatsApp.
func (b *Bot) Receive() <-chan channels.IncomingMessage { return b.incoming }

// Send delivers a message to a WhatsApp chat.
//
// TODO(native): Replace with whatsmeow's SendMessage:
//
//	jid, _ := types.ParseJID(msg.ChannelID)
//	b.client.SendMessage(ctx, jid, &waProto.Message{
//	    Conversation: &msg.Text,
//	})
//
// For formatted messages, use ExtendedTextMessage instead of Conversation.
// See docs/native-channels-guide.md#sending-messages-2.
func (b *Bot) Send(_ context.Context, msg channels.OutgoingMessage) error {
	// Stub: log the intent but do not actually send via WhatsApp.
	_ = fmt.Sprintf("whatsapp stub: would send to %s: %s",
		msg.ChannelID, msg.Text)
	return nil
}

// IsPaired returns whether the WhatsApp session has been paired via QR code.
//
// TODO(native): Check b.client.Store.ID != nil to determine pairing status.
func (b *Bot) IsPaired() bool { return b.paired }

// connectBaileys is a placeholder for the WhatsApp connection.
//
// TODO(native): Replace entirely with whatsmeow's event-driven architecture.
// The native flow is:
//  1. Initialize sqlstore for session persistence
//  2. Create whatsmeow.Client with device store
//  3. Register event handler for *events.Message, *events.LoggedOut,
//     *events.Disconnected
//  4. If no stored session, start QR pairing via client.GetQRChannel
//  5. If stored session exists, call client.Connect() to reconnect
//  6. In the event handler, filter by allowedNumbers and convert to
//     channels.IncomingMessage
//
// See docs/native-channels-guide.md#whatsapp-whatsmeow.
func (b *Bot) connectBaileys(ctx context.Context) {
	<-ctx.Done()
}

// IsAllowed checks if a phone number is in the whitelist.
// Phone numbers should be in E.164 format (e.g., "+1234567890").
// An empty whitelist allows all numbers.
func (b *Bot) IsAllowed(phoneNumber string) bool {
	if len(b.allowedNumbers) == 0 {
		return true
	}
	return b.allowedNumbers[phoneNumber]
}

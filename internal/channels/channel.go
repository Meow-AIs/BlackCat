package channels

import "context"

// Platform identifies a messaging platform.
type Platform string

const (
	PlatformTelegram Platform = "telegram"
	PlatformDiscord  Platform = "discord"
	PlatformSlack    Platform = "slack"
	PlatformWhatsApp Platform = "whatsapp"
	PlatformSignal   Platform = "signal"
	PlatformEmail    Platform = "email"
	PlatformCLI      Platform = "cli"
)

// IncomingMessage is a normalized message from any platform.
type IncomingMessage struct {
	Platform  Platform `json:"platform"`
	ChannelID string   `json:"channel_id"`
	UserID    string   `json:"user_id"`
	UserName  string   `json:"user_name"`
	Text      string   `json:"text"`
	ReplyToID string   `json:"reply_to_id,omitempty"`
	Timestamp int64    `json:"timestamp"`
}

// OutgoingMessage is a response to send back to a platform.
type OutgoingMessage struct {
	ChannelID string `json:"channel_id"`
	Text      string `json:"text"`
	ReplyToID string `json:"reply_to_id,omitempty"`
	Format    Format `json:"format"`
}

// Format controls how a message is rendered on the platform.
type Format string

const (
	FormatPlain    Format = "plain"
	FormatMarkdown Format = "markdown"
	FormatCode     Format = "code"
)

// Adapter is the interface that each messaging platform implements.
type Adapter interface {
	// Start connects to the platform and begins listening for messages.
	Start(ctx context.Context) error

	// Stop gracefully disconnects from the platform.
	Stop(ctx context.Context) error

	// Send delivers a message to the platform.
	Send(ctx context.Context, msg OutgoingMessage) error

	// Receive returns a channel that emits incoming messages.
	Receive() <-chan IncomingMessage

	// Platform returns which platform this adapter serves.
	Platform() Platform
}

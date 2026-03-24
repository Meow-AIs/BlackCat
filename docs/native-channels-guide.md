# Native Channel Integration Guide

BlackCat channels currently use lightweight stub or raw-HTTP implementations. This guide explains how to upgrade to native Go libraries for production-grade Telegram, Discord, and WhatsApp support.

## Dependencies to Add

```bash
rtk go get github.com/go-telegram/bot@latest      # Telegram
rtk go get github.com/bwmarrin/discordgo@latest    # Discord
rtk go get go.mau.fi/whatsmeow@latest              # WhatsApp (pure Go, no Node.js)
rtk go get go.mau.fi/whatsmeow/store/sqlstore@latest
```

After adding dependencies, run:

```bash
rtk go mod tidy
```

---

## Telegram (go-telegram/bot)

### Setup

1. Create a bot via [@BotFather](https://t.me/BotFather) on Telegram.
2. Copy the bot token (e.g., `123456:ABCdef...`).
3. Configure:

```bash
blackcat config set telegram_bot_token "123456:ABCdef..."
```

### Implementation Pattern

The `go-telegram/bot` library handles long-polling internally, so the adapter
becomes a thin wrapper that converts between library types and BlackCat's
`channels.IncomingMessage` / `channels.OutgoingMessage`.

#### Creating the Bot Client

```go
package telegram

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/meowai/blackcat/internal/channels"
)

type Bot struct {
	token        string
	allowedUsers map[int64]bool
	incoming     chan channels.IncomingMessage
	client       *bot.Bot
	cancel       context.CancelFunc
}

type Config struct {
	Token        string
	AllowedUsers []int64
}

func New(cfg Config) *Bot {
	allowed := make(map[int64]bool, len(cfg.AllowedUsers))
	for _, uid := range cfg.AllowedUsers {
		allowed[uid] = true
	}
	return &Bot{
		token:        cfg.Token,
		allowedUsers: allowed,
		incoming:     make(chan channels.IncomingMessage, 100),
	}
}
```

#### Starting the Long-Poll Loop

`go-telegram/bot` manages the long-polling goroutine. Register a default handler
and call `bot.Start` (blocking) inside a goroutine.

```go
func (b *Bot) Start(ctx context.Context) error {
	ctx, b.cancel = context.WithCancel(ctx)

	opts := []bot.Option{
		bot.WithDefaultHandler(b.onUpdate),
	}
	client, err := bot.New(b.token, opts...)
	if err != nil {
		return fmt.Errorf("telegram bot init: %w", err)
	}
	b.client = client

	// bot.Start blocks, so run in a goroutine
	go b.client.Start(ctx)
	return nil
}
```

#### Handling Incoming Messages

The default handler receives every update. Filter by allowed users and convert
to `channels.IncomingMessage`.

```go
func (b *Bot) onUpdate(ctx context.Context, tgBot *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	from := update.Message.From
	if from == nil {
		return
	}

	// Enforce user whitelist
	if len(b.allowedUsers) > 0 && !b.allowedUsers[from.ID] {
		return
	}

	b.incoming <- channels.IncomingMessage{
		Platform:  channels.PlatformTelegram,
		ChannelID: strconv.FormatInt(update.Message.Chat.ID, 10),
		UserID:    strconv.FormatInt(from.ID, 10),
		UserName:  from.Username,
		Text:      update.Message.Text,
		Timestamp: int64(update.Message.Date),
	}
}
```

#### Sending Messages

Use `bot.SendMessage` with optional parse mode and reply parameters.

```go
func (b *Bot) Send(ctx context.Context, msg channels.OutgoingMessage) error {
	chatID, err := strconv.ParseInt(msg.ChannelID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat ID %q: %w", msg.ChannelID, err)
	}

	parseMode := models.ParseModeMarkdown
	if msg.Format == channels.FormatPlain {
		parseMode = ""
	}

	params := &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      msg.Text,
		ParseMode: parseMode,
	}
	if msg.ReplyToID != "" {
		replyID, _ := strconv.Atoi(msg.ReplyToID)
		params.ReplyParameters = &models.ReplyParameters{
			MessageID: replyID,
		}
	}

	_, err = b.client.SendMessage(ctx, params)
	return err
}
```

#### Message Formatting

Telegram supports MarkdownV2 and HTML. Key formatting rules:

| BlackCat Format | Telegram Parse Mode | Notes |
|-----------------|-------------------|-------|
| `plain` | (none) | Raw text, no formatting |
| `markdown` | `MarkdownV2` | Escape special chars: `_*[]()~>#+\-=\|{}.!` |
| `code` | `MarkdownV2` | Wrap in triple backticks with language hint |

For code blocks, wrap the output:

```go
func formatCode(text, language string) string {
	return fmt.Sprintf("```%s\n%s\n```", language, text)
}
```

#### Webhook Alternative

For high-traffic bots, webhooks avoid polling overhead. The `go-telegram/bot`
library supports webhooks via `bot.StartWebhook`:

```go
func (b *Bot) StartWebhook(ctx context.Context, publicURL string, port int) error {
	ctx, b.cancel = context.WithCancel(ctx)
	client, err := bot.New(b.token, bot.WithDefaultHandler(b.onUpdate))
	if err != nil {
		return err
	}
	b.client = client

	go b.client.StartWebhook(ctx)

	// Set webhook URL with Telegram
	_, err = b.client.SetWebhook(ctx, &bot.SetWebhookParams{
		URL: publicURL + "/telegram/webhook",
	})
	if err != nil {
		return fmt.Errorf("set webhook: %w", err)
	}

	// Serve webhook HTTP handler
	go http.ListenAndServe(
		fmt.Sprintf(":%d", port),
		b.client.WebhookHandler(),
	)
	return nil
}
```

#### Media Handling

Send images, documents, or voice messages:

```go
// Send a photo from a URL
func (b *Bot) SendPhoto(ctx context.Context, chatID int64, photoURL, caption string) error {
	_, err := b.client.SendPhoto(ctx, &bot.SendPhotoParams{
		ChatID:  chatID,
		Photo:   &models.InputFileString{Data: photoURL},
		Caption: caption,
	})
	return err
}

// Send a document (file upload)
func (b *Bot) SendDocument(ctx context.Context, chatID int64, fileData io.Reader, filename string) error {
	_, err := b.client.SendDocument(ctx, &bot.SendDocumentParams{
		ChatID: chatID,
		Document: &models.InputFileUpload{
			Filename: filename,
			Data:     fileData,
		},
	})
	return err
}
```

---

## Discord (discordgo)

### Setup

1. Create an application at [discord.com/developers](https://discord.com/developers/applications).
2. Add a Bot under the application, copy the token.
3. Enable **Message Content Intent** under Privileged Gateway Intents.
4. Invite the bot with scopes `bot` + `applications.commands` and permissions: Send Messages, Read Message History, Read Messages/View Channels.
5. Configure:

```bash
blackcat config set discord_bot_token "MTIzNDU2..."
```

### Implementation Pattern

`discordgo` manages the WebSocket gateway connection. The adapter registers
event handlers for `MessageCreate` and optionally registers slash commands.

#### Creating the Session

```go
package discord

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/meowai/blackcat/internal/channels"
)

type Bot struct {
	token           string
	allowedGuilds   map[string]bool
	allowedChannels map[string]bool
	incoming        chan channels.IncomingMessage
	session         *discordgo.Session
	cancel          context.CancelFunc
}

type Config struct {
	Token           string
	AllowedGuilds   []string
	AllowedChannels []string
}

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
```

#### Connecting the Gateway

```go
func (b *Bot) Start(ctx context.Context) error {
	ctx, b.cancel = context.WithCancel(ctx)

	session, err := discordgo.New("Bot " + b.token)
	if err != nil {
		return fmt.Errorf("discord session: %w", err)
	}
	b.session = session

	// Declare intents
	session.Identify.Intents = discordgo.IntentsGuildMessages |
		discordgo.IntentsDirectMessages |
		discordgo.IntentsMessageContent

	// Register handlers
	session.AddHandler(b.onMessageCreate)

	if err := session.Open(); err != nil {
		return fmt.Errorf("discord gateway: %w", err)
	}

	// Close session when context is cancelled
	go func() {
		<-ctx.Done()
		session.Close()
	}()

	return nil
}

func (b *Bot) Stop(_ context.Context) error {
	if b.cancel != nil {
		b.cancel()
	}
	if b.session != nil {
		b.session.Close()
	}
	close(b.incoming)
	return nil
}
```

#### Handling Incoming Messages

```go
func (b *Bot) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Enforce guild whitelist
	if len(b.allowedGuilds) > 0 && !b.allowedGuilds[m.GuildID] {
		return
	}
	// Enforce channel whitelist
	if len(b.allowedChannels) > 0 && !b.allowedChannels[m.ChannelID] {
		return
	}

	b.incoming <- channels.IncomingMessage{
		Platform:  channels.PlatformDiscord,
		ChannelID: m.ChannelID,
		UserID:    m.Author.ID,
		UserName:  m.Author.Username,
		Text:      m.Content,
		ReplyToID: m.MessageReference.MessageID, // empty if not a reply
		Timestamp: m.Timestamp.Unix(),
	}
}
```

#### Sending Messages with 2000-Character Splitting

Discord enforces a 2000-character message limit. Long responses must be split.

```go
const discordMaxLen = 2000

func (b *Bot) Send(ctx context.Context, msg channels.OutgoingMessage) error {
	chunks := splitMessage(msg.Text, discordMaxLen)
	for _, chunk := range chunks {
		data := &discordgo.MessageSend{
			Content: chunk,
		}
		if msg.ReplyToID != "" {
			data.Reference = &discordgo.MessageReference{
				MessageID: msg.ReplyToID,
			}
		}
		_, err := b.session.ChannelMessageSendComplex(msg.ChannelID, data)
		if err != nil {
			return fmt.Errorf("discord send: %w", err)
		}
	}
	return nil
}

// splitMessage breaks text into chunks that respect the max length,
// preferring to split at newline boundaries.
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
		// Find last newline within limit
		cutAt := maxLen
		if idx := strings.LastIndex(text[:maxLen], "\n"); idx > 0 {
			cutAt = idx + 1
		}
		chunks = append(chunks, text[:cutAt])
		text = text[cutAt:]
	}
	return chunks
}
```

#### Embed Formatting

For rich responses, use Discord embeds instead of plain text:

```go
func (b *Bot) sendEmbed(channelID, title, description string, color int) error {
	_, err := b.session.ChannelMessageSendEmbed(channelID, &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       color, // e.g., 0x00FF00 for green
	})
	return err
}
```

#### Slash Command Registration

Register application commands on startup for a better user experience:

```go
func (b *Bot) registerSlashCommands() error {
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "ask",
			Description: "Ask BlackCat a question",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "prompt",
					Description: "Your question",
					Required:    true,
				},
			},
		},
		{
			Name:        "status",
			Description: "Show BlackCat system status",
		},
		{
			Name:        "cost",
			Description: "Show session token usage and cost",
		},
	}

	for _, cmd := range commands {
		_, err := b.session.ApplicationCommandCreate(b.session.State.User.ID, "", cmd)
		if err != nil {
			return fmt.Errorf("register command %s: %w", cmd.Name, err)
		}
	}
	return nil
}

func (b *Bot) onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Convert slash command to IncomingMessage with "/" prefix
	// so it flows through the same InputMiddleware as text commands.
	data := i.ApplicationCommandData()
	text := "/" + data.Name
	for _, opt := range data.Options {
		text += " " + fmt.Sprintf("%v", opt.Value)
	}

	b.incoming <- channels.IncomingMessage{
		Platform:  channels.PlatformDiscord,
		ChannelID: i.ChannelID,
		UserID:    i.Member.User.ID,
		UserName:  i.Member.User.Username,
		Text:      text,
		Timestamp: time.Now().Unix(),
	}
}
```

---

## WhatsApp (whatsmeow)

### Setup

1. No external runtime needed. `whatsmeow` is a pure Go implementation of the WhatsApp Web protocol.
2. QR code pairing on first run (displayed in terminal).
3. Session stored in SQLite (managed by whatsmeow's `sqlstore`).

### Implementation Pattern

`whatsmeow` handles the WhatsApp Web multi-device protocol entirely in Go.
The adapter manages QR pairing, session persistence, and message routing.

#### Creating the Client

```go
package whatsapp

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	"github.com/meowai/blackcat/internal/channels"

	_ "github.com/mattn/go-sqlite3" // SQLite driver for session store
)

type Bot struct {
	sessionPath    string
	allowedNumbers map[string]bool
	incoming       chan channels.IncomingMessage
	client         *whatsmeow.Client
	cancel         context.CancelFunc
}

type Config struct {
	SessionPath    string
	AllowedNumbers []string
}

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
```

#### QR Code Pairing Flow

On first run there is no stored session, so whatsmeow generates QR codes that
the user scans with their phone. On subsequent runs the stored session is
reused automatically.

```go
func (b *Bot) Start(ctx context.Context) error {
	ctx, b.cancel = context.WithCancel(ctx)

	// Initialize SQLite session store
	dbURI := fmt.Sprintf("file:%s/whatsmeow.db?_journal_mode=WAL", b.sessionPath)
	container, err := sqlstore.New("sqlite3", dbURI, waLog.Noop)
	if err != nil {
		return fmt.Errorf("whatsmeow store: %w", err)
	}

	// Get or create device store
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		return fmt.Errorf("whatsmeow device: %w", err)
	}

	b.client = whatsmeow.NewClient(deviceStore, waLog.Noop)
	b.client.AddEventHandler(b.eventHandler)

	if b.client.Store.ID == nil {
		// No session: perform QR code pairing
		go b.pairWithQR(ctx)
	} else {
		// Existing session: reconnect
		go b.connect(ctx)
	}

	return nil
}

func (b *Bot) pairWithQR(ctx context.Context) {
	qrChan, _ := b.client.GetQRChannel(ctx)
	err := b.client.Connect()
	if err != nil {
		fmt.Printf("WhatsApp connect error: %v\n", err)
		return
	}

	for evt := range qrChan {
		switch evt.Event {
		case "code":
			// Display QR code in terminal.
			// In production, use a QR rendering library like
			// github.com/mdp/qrterminal/v3:
			//   qrterminal.Generate(evt.Code, qrterminal.L, os.Stdout)
			fmt.Printf("WhatsApp QR code: %s\n", evt.Code)
			fmt.Println("Scan this QR code with WhatsApp > Linked Devices")
		case "success":
			fmt.Println("WhatsApp paired successfully")
		case "timeout":
			fmt.Println("WhatsApp QR timeout, retrying...")
		}
	}
}

func (b *Bot) connect(ctx context.Context) {
	err := b.client.Connect()
	if err != nil {
		fmt.Printf("WhatsApp reconnect error: %v\n", err)
	}
}
```

#### Handling Incoming Messages

```go
func (b *Bot) eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		// Extract sender JID (phone@s.whatsapp.net)
		sender := v.Info.Sender.User // phone number without @suffix
		chatJID := v.Info.Chat.String()

		// Enforce number whitelist
		if len(b.allowedNumbers) > 0 && !b.allowedNumbers["+"+sender] {
			return
		}

		// Extract text content
		text := ""
		if v.Message.GetConversation() != "" {
			text = v.Message.GetConversation()
		} else if v.Message.GetExtendedTextMessage() != nil {
			text = v.Message.GetExtendedTextMessage().GetText()
		}
		if text == "" {
			return // skip non-text messages (images, etc.)
		}

		b.incoming <- channels.IncomingMessage{
			Platform:  channels.PlatformWhatsApp,
			ChannelID: chatJID,
			UserID:    sender,
			UserName:  sender, // WhatsApp doesn't expose display names reliably
			Text:      text,
			Timestamp: v.Info.Timestamp.Unix(),
		}
	}
}
```

#### Sending Messages

```go
func (b *Bot) Send(ctx context.Context, msg channels.OutgoingMessage) error {
	jid, err := types.ParseJID(msg.ChannelID)
	if err != nil {
		return fmt.Errorf("invalid JID %q: %w", msg.ChannelID, err)
	}

	_, err = b.client.SendMessage(ctx, jid, &waProto.Message{
		Conversation: &msg.Text,
	})
	return err
}
```

Note: For `waProto`, import `go.mau.fi/whatsmeow/binary/proto`. The `Conversation`
field is the simplest text message type. For formatted messages, use
`ExtendedTextMessage`.

#### Session Management

whatsmeow persists the session (keys, device identity) in its SQLite store. Key
considerations:

- **Session path**: Must be writable and persisted across container restarts if
  running in Docker. Mount a volume to `~/.blackcat/whatsapp-session/`.
- **Session expiry**: WhatsApp may unlink devices after ~14 days of inactivity.
  Handle `events.LoggedOut` to notify the operator and trigger re-pairing.
- **Reconnection**: Handle `events.Disconnected` with exponential backoff.

```go
case *events.LoggedOut:
	fmt.Println("WhatsApp session expired. Please re-pair via QR code.")
	// Trigger re-pairing or notify admin through another channel

case *events.Disconnected:
	fmt.Println("WhatsApp disconnected, reconnecting...")
	go func() {
		time.Sleep(5 * time.Second)
		b.client.Connect()
	}()
```

#### Advantages Over Baileys Bridge

| Aspect | whatsmeow (Go) | Baileys (Node.js bridge) |
|--------|----------------|--------------------------|
| Runtime dependency | None (pure Go) | Requires Node.js 18+ |
| Binary size | +~2MB to Go binary | +50MB (Node.js runtime) |
| Memory usage | ~30MB | ~80-120MB |
| Process management | In-process goroutine | Child process + JSON stdio |
| Session storage | SQLite (same as BlackCat) | JSON files |
| Crash recovery | Goroutine restart | Process respawn |

---

## Integration with BlackCat

### Implementing the Adapter Interface

Every channel must satisfy `channels.Adapter` from `internal/channels/channel.go`:

```go
type Adapter interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Send(ctx context.Context, msg OutgoingMessage) error
	Receive() <-chan IncomingMessage
	Platform() Platform
}
```

Each native implementation follows the same pattern:

```go
func (b *Bot) Platform() channels.Platform { return channels.PlatformXxx }
func (b *Bot) Receive() <-chan channels.IncomingMessage { return b.incoming }
```

`Start` connects to the platform and spawns a listener goroutine (or registers
callbacks). `Stop` cancels the context and closes the incoming channel. `Send`
translates `OutgoingMessage` into platform-specific API calls.

### Message Flow

```
User sends message on Telegram/Discord/WhatsApp
  |
  v
Platform SDK receives event (callback or poll)
  |
  v
Adapter converts to channels.IncomingMessage
  |
  v
channels.Gateway.routeMessages picks it up
  |
  v
Gateway.handleMessage runs:
  1. Rate limiting check (channels.RateLimiter)
  2. Calls MessageHandler (which runs InputMiddleware)
  3. InputMiddleware checks for "/" prefix:
     - If slash command: execute directly, return result
     - If regular text: forward to Agent Core for LLM processing
  4. Response string returned to Gateway
  |
  v
Gateway calls adapter.Send() with OutgoingMessage
  |
  v
Adapter translates to platform API call (sendMessage, etc.)
```

### Slash Commands via InputMiddleware

Slash commands are handled identically across all channels. The `InputMiddleware`
in `internal/commands/middleware.go` detects the `/` prefix and dispatches to the
appropriate command handler without invoking the LLM.

For Discord slash commands specifically, the `onInteractionCreate` handler
converts the interaction into a text message with a `/` prefix so it flows
through the same middleware:

```go
// In Discord adapter's interaction handler:
text := "/" + data.Name
for _, opt := range data.Options {
	text += " " + fmt.Sprintf("%v", opt.Value)
}
// Push to b.incoming as a normal IncomingMessage
```

This means `/status`, `/cost`, `/memory search X`, etc. all work the same way
on every platform.

### Registering Adapters in the Gateway

In `blackcat serve`, adapters are created from config and registered:

```go
gw := channels.NewGateway(messageHandler)

if cfg.Channels.Telegram.Enabled {
	tg := telegram.New(telegram.Config{
		Token:        cfg.Channels.Telegram.Token,
		AllowedUsers: cfg.Channels.Telegram.AllowedUsers,
	})
	gw.Register(tg)
}

if cfg.Channels.Discord.Enabled {
	dc := discord.New(discord.Config{
		Token:           cfg.Channels.Discord.Token,
		AllowedGuilds:   cfg.Channels.Discord.AllowedGuilds,
		AllowedChannels: cfg.Channels.Discord.AllowedChannels,
	})
	gw.Register(dc)
}

if cfg.Channels.WhatsApp.Enabled {
	wa := whatsapp.New(whatsapp.Config{
		SessionPath:    cfg.Channels.WhatsApp.SessionPath,
		AllowedNumbers: cfg.Channels.WhatsApp.AllowedNumbers,
	})
	gw.Register(wa)
}

gw.Start(ctx)
```

### Error Handling

All adapters should:

1. **Never panic** -- catch all errors and log them.
2. **Reconnect on disconnect** -- use exponential backoff.
3. **Respect context cancellation** -- stop cleanly when `ctx` is cancelled.
4. **Rate-limit API calls** -- respect platform-specific rate limits (Telegram: 30 msg/sec, Discord: 5 msg/sec per channel, WhatsApp: undocumented but ~20 msg/min is safe).

### Testing

Each adapter should have:

- **Unit tests**: Mock the platform SDK, verify IncomingMessage conversion.
- **Integration tests**: Connect to a test bot token, send/receive a message.
- **Stub mode**: The current stubs remain useful for testing the gateway without real platform connections.

Run tests:

```bash
rtk go test ./internal/channels/telegram/... -v
rtk go test ./internal/channels/discord/... -v
rtk go test ./internal/channels/whatsapp/... -v
```

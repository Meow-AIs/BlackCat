package email

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"github.com/meowai/blackcat/internal/channels"
)

// Bot implements channels.Adapter for Email via IMAP + SMTP.
type Bot struct {
	imapHost       string
	imapPort       int
	smtpHost       string
	smtpPort       int
	username       string
	password       string
	fromAddr       string
	allowedSenders map[string]bool
	pollInterval   time.Duration
	incoming       chan channels.IncomingMessage
	cancel         context.CancelFunc
}

// Config for creating an Email bot.
type Config struct {
	IMAPHost       string
	IMAPPort       int
	SMTPHost       string
	SMTPPort       int
	Username       string
	Password       string
	FromAddress    string
	AllowedSenders []string // email addresses allowed to interact
	PollInterval   time.Duration
}

// New creates an Email bot adapter.
func New(cfg Config) *Bot {
	senders := make(map[string]bool)
	for _, s := range cfg.AllowedSenders {
		senders[strings.ToLower(s)] = true
	}
	pollInterval := cfg.PollInterval
	if pollInterval == 0 {
		pollInterval = 30 * time.Second
	}
	return &Bot{
		imapHost:       cfg.IMAPHost,
		imapPort:       cfg.IMAPPort,
		smtpHost:       cfg.SMTPHost,
		smtpPort:       cfg.SMTPPort,
		username:       cfg.Username,
		password:       cfg.Password,
		fromAddr:       cfg.FromAddress,
		allowedSenders: senders,
		pollInterval:   pollInterval,
		incoming:       make(chan channels.IncomingMessage, 100),
	}
}

func (b *Bot) Platform() channels.Platform { return channels.PlatformEmail }

func (b *Bot) Start(ctx context.Context) error {
	ctx, b.cancel = context.WithCancel(ctx)
	go b.pollLoop(ctx)
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
	to := msg.ChannelID // ChannelID = recipient email address
	subject := "BlackCat Response"
	body := msg.Text

	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		b.fromAddr, to, subject, body)

	auth := smtp.PlainAuth("", b.username, b.password, b.smtpHost)
	addr := fmt.Sprintf("%s:%d", b.smtpHost, b.smtpPort)

	return smtp.SendMail(addr, auth, b.fromAddr, []string{to}, []byte(message))
}

func (b *Bot) pollLoop(ctx context.Context) {
	// Actual implementation:
	// 1. Connect to IMAP server via TLS
	// 2. SELECT INBOX
	// 3. SEARCH UNSEEN messages
	// 4. FETCH and parse each message
	// 5. Filter by allowedSenders
	// 6. Mark as SEEN
	// 7. Push to b.incoming as IncomingMessage:
	//    - ChannelID = sender email
	//    - UserID = sender email
	//    - Text = email body (text/plain)
	// 8. Sleep pollInterval, repeat
	//
	// Uses: net/textproto or go-imap library
	ticker := time.NewTicker(b.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.checkInbox(ctx)
		}
	}
}

func (b *Bot) checkInbox(_ context.Context) {
	// Placeholder: actual IMAP fetch implementation
	// Would use crypto/tls + net/textproto for IMAP commands
	// or github.com/emersion/go-imap library
}

// IsAllowed checks if an email sender is whitelisted.
func (b *Bot) IsAllowed(senderEmail string) bool {
	if len(b.allowedSenders) == 0 {
		return true
	}
	return b.allowedSenders[strings.ToLower(senderEmail)]
}

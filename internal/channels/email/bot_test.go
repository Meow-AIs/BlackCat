package email

import (
	"testing"
	"time"

	"github.com/meowai/blackcat/internal/channels"
)

func TestEmailBotPlatform(t *testing.T) {
	bot := New(Config{IMAPHost: "imap.example.com"})
	if bot.Platform() != channels.PlatformEmail {
		t.Errorf("expected email, got %s", bot.Platform())
	}
}

func TestEmailBotIsAllowed(t *testing.T) {
	bot := New(Config{AllowedSenders: []string{"alice@example.com", "Bob@Example.COM"}})
	if !bot.IsAllowed("alice@example.com") {
		t.Error("expected alice allowed")
	}
	if !bot.IsAllowed("bob@example.com") {
		t.Error("expected bob allowed (case insensitive)")
	}
	if bot.IsAllowed("eve@example.com") {
		t.Error("expected eve not allowed")
	}
}

func TestEmailBotNoWhitelist(t *testing.T) {
	bot := New(Config{})
	if !bot.IsAllowed("anyone@anywhere.com") {
		t.Error("expected all allowed with no whitelist")
	}
}

func TestEmailBotDefaultPollInterval(t *testing.T) {
	bot := New(Config{})
	if bot.pollInterval != 30*time.Second {
		t.Errorf("expected 30s default, got %v", bot.pollInterval)
	}
}

func TestEmailBotCustomPollInterval(t *testing.T) {
	bot := New(Config{PollInterval: 5 * time.Minute})
	if bot.pollInterval != 5*time.Minute {
		t.Errorf("expected 5m, got %v", bot.pollInterval)
	}
}

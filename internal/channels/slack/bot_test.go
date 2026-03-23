package slack

import (
	"testing"

	"github.com/meowai/blackcat/internal/channels"
)

func TestSlackBotPlatform(t *testing.T) {
	bot := New(Config{AppToken: "xapp-test", BotToken: "xoxb-test"})
	if bot.Platform() != channels.PlatformSlack {
		t.Errorf("expected slack, got %s", bot.Platform())
	}
}

func TestSlackBotIsAllowed(t *testing.T) {
	bot := New(Config{AllowedChannels: []string{"C01GENERAL"}})
	if !bot.IsAllowed("C01GENERAL") {
		t.Error("expected C01GENERAL allowed")
	}
	if bot.IsAllowed("C02RANDOM") {
		t.Error("expected C02RANDOM not allowed")
	}
}

func TestSlackBotNoWhitelist(t *testing.T) {
	bot := New(Config{})
	if !bot.IsAllowed("anything") {
		t.Error("expected all allowed with no whitelist")
	}
}

package discord

import (
	"testing"

	"github.com/meowai/blackcat/internal/channels"
)

func TestDiscordBotPlatform(t *testing.T) {
	bot := New(Config{Token: "test"})
	if bot.Platform() != channels.PlatformDiscord {
		t.Errorf("expected discord, got %s", bot.Platform())
	}
}

func TestDiscordBotIsAllowed(t *testing.T) {
	bot := New(Config{
		Token:           "test",
		AllowedGuilds:   []string{"guild1"},
		AllowedChannels: []string{"chan1"},
	})

	if !bot.IsAllowed("guild1", "chan1") {
		t.Error("expected guild1/chan1 allowed")
	}
	if bot.IsAllowed("guild2", "chan1") {
		t.Error("expected guild2 not allowed")
	}
	if bot.IsAllowed("guild1", "chan2") {
		t.Error("expected chan2 not allowed")
	}
}

func TestDiscordBotNoWhitelist(t *testing.T) {
	bot := New(Config{Token: "test"})
	if !bot.IsAllowed("any", "any") {
		t.Error("expected all allowed with no whitelist")
	}
}

package whatsapp

import (
	"testing"

	"github.com/meowai/blackcat/internal/channels"
)

func TestWhatsAppBotPlatform(t *testing.T) {
	bot := New(Config{SessionPath: "/tmp/wa"})
	if bot.Platform() != channels.PlatformWhatsApp {
		t.Errorf("expected whatsapp, got %s", bot.Platform())
	}
}

func TestWhatsAppBotIsAllowed(t *testing.T) {
	bot := New(Config{AllowedNumbers: []string{"+6281234567890"}})
	if !bot.IsAllowed("+6281234567890") {
		t.Error("expected allowed number")
	}
	if bot.IsAllowed("+1234567890") {
		t.Error("expected not allowed")
	}
}

func TestWhatsAppBotNoWhitelist(t *testing.T) {
	bot := New(Config{})
	if !bot.IsAllowed("+anything") {
		t.Error("expected all allowed with no whitelist")
	}
}

func TestWhatsAppBotNotPairedInitially(t *testing.T) {
	bot := New(Config{})
	if bot.IsPaired() {
		t.Error("expected not paired initially")
	}
}

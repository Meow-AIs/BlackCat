package signal

import (
	"testing"

	"github.com/meowai/blackcat/internal/channels"
)

func TestSignalBotPlatform(t *testing.T) {
	bot := New(Config{PhoneNumber: "+1234567890"})
	if bot.Platform() != channels.PlatformSignal {
		t.Errorf("expected signal, got %s", bot.Platform())
	}
}

func TestSignalBotIsAllowed(t *testing.T) {
	bot := New(Config{AllowedNumbers: []string{"+1234567890"}})
	if !bot.IsAllowed("+1234567890") {
		t.Error("expected allowed")
	}
	if bot.IsAllowed("+9999999999") {
		t.Error("expected not allowed")
	}
}

func TestSignalBotDefaultCLIPath(t *testing.T) {
	bot := New(Config{PhoneNumber: "+1"})
	if bot.signalCLIPath != "signal-cli" {
		t.Errorf("expected default 'signal-cli', got %q", bot.signalCLIPath)
	}
}

func TestSignalBotCustomCLIPath(t *testing.T) {
	bot := New(Config{PhoneNumber: "+1", SignalCLIPath: "/usr/local/bin/signal-cli"})
	if bot.signalCLIPath != "/usr/local/bin/signal-cli" {
		t.Errorf("expected custom path, got %q", bot.signalCLIPath)
	}
}

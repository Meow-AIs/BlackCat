package channels

import (
	"testing"

	"github.com/meowai/blackcat/internal/commands"
)

func newTestRegistry(t *testing.T) *commands.Registry {
	t.Helper()
	reg := commands.NewRegistry()
	err := reg.Register(commands.CommandDef{
		Name:        "status",
		Description: "Show agent status",
		Category:    "general",
		Handler: func(args []string) commands.CommandResult {
			return commands.CommandResult{Output: "Agent is running"}
		},
	})
	if err != nil {
		t.Fatalf("register status: %v", err)
	}
	err = reg.Register(commands.CommandDef{
		Name:        "memory",
		Description: "Memory operations",
		Category:    "memory",
		Handler: func(args []string) commands.CommandResult {
			return commands.CommandResult{Output: "memory help"}
		},
		SubCommands: map[string]commands.CommandDef{
			"search": {
				Name:        "search",
				Description: "Search memory",
				Handler: func(args []string) commands.CommandResult {
					if len(args) == 0 {
						return commands.CommandResult{Error: "search requires a query"}
					}
					return commands.CommandResult{Output: "results for: " + args[0]}
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("register memory: %v", err)
	}
	return reg
}

func TestCommandRouterInterceptsSlashCommand(t *testing.T) {
	reg := newTestRegistry(t)
	router := NewCommandRouter(reg)

	msg := IncomingMessage{
		Platform:  PlatformTelegram,
		ChannelID: "chat-123",
		UserID:    "user-1",
		UserName:  "alice",
		Text:      "/status",
	}

	out := router.ProcessMessage(msg)
	if out == nil {
		t.Fatal("expected outgoing message for slash command, got nil")
	}
	if out.ChannelID != "chat-123" {
		t.Errorf("expected channel 'chat-123', got %q", out.ChannelID)
	}
	if out.Text != "Agent is running" {
		t.Errorf("expected 'Agent is running', got %q", out.Text)
	}
	if out.Format != FormatPlain {
		t.Errorf("expected plain format, got %q", out.Format)
	}
}

func TestCommandRouterPassesThroughRegularMessage(t *testing.T) {
	reg := newTestRegistry(t)
	router := NewCommandRouter(reg)

	msg := IncomingMessage{
		Platform:  PlatformDiscord,
		ChannelID: "general",
		UserID:    "user-2",
		Text:      "hello blackcat, how are you?",
	}

	out := router.ProcessMessage(msg)
	if out != nil {
		t.Errorf("expected nil for regular message, got %+v", out)
	}
}

func TestCommandRouterEmptyMessage(t *testing.T) {
	reg := newTestRegistry(t)
	router := NewCommandRouter(reg)

	msg := IncomingMessage{
		Platform:  PlatformSlack,
		ChannelID: "random",
		UserID:    "user-3",
		Text:      "",
	}

	out := router.ProcessMessage(msg)
	if out != nil {
		t.Errorf("expected nil for empty message, got %+v", out)
	}
}

func TestCommandRouterWithArgs(t *testing.T) {
	reg := newTestRegistry(t)
	router := NewCommandRouter(reg)

	msg := IncomingMessage{
		Platform:  PlatformWhatsApp,
		ChannelID: "wa-chat",
		UserID:    "user-4",
		Text:      "/memory search auth",
	}

	out := router.ProcessMessage(msg)
	if out == nil {
		t.Fatal("expected outgoing message for command with args, got nil")
	}
	if out.Text != "results for: auth" {
		t.Errorf("expected 'results for: auth', got %q", out.Text)
	}
}

func TestCommandRouterUnknownCommand(t *testing.T) {
	reg := newTestRegistry(t)
	router := NewCommandRouter(reg)

	msg := IncomingMessage{
		Platform:  PlatformTelegram,
		ChannelID: "chat-1",
		UserID:    "user-5",
		Text:      "/nonexistent",
	}

	out := router.ProcessMessage(msg)
	if out == nil {
		t.Fatal("expected outgoing message for unknown command, got nil")
	}
	if out.Text == "" {
		t.Error("expected error text for unknown command")
	}
}

func TestCommandRouterErrorResult(t *testing.T) {
	reg := newTestRegistry(t)
	router := NewCommandRouter(reg)

	msg := IncomingMessage{
		Platform:  PlatformTelegram,
		ChannelID: "chat-1",
		UserID:    "user-6",
		Text:      "/memory search",
	}

	out := router.ProcessMessage(msg)
	if out == nil {
		t.Fatal("expected outgoing message for command error, got nil")
	}
	if out.Text != "search requires a query" {
		t.Errorf("expected error text, got %q", out.Text)
	}
}

func TestCommandRouterSilentCommand(t *testing.T) {
	reg := commands.NewRegistry()
	err := reg.Register(commands.CommandDef{
		Name:        "quiet",
		Description: "Silent command",
		Category:    "general",
		Handler: func(args []string) commands.CommandResult {
			return commands.CommandResult{Silent: true}
		},
	})
	if err != nil {
		t.Fatalf("register quiet: %v", err)
	}

	router := NewCommandRouter(reg)

	msg := IncomingMessage{
		Platform:  PlatformTelegram,
		ChannelID: "chat-1",
		UserID:    "user-7",
		Text:      "/quiet",
	}

	out := router.ProcessMessage(msg)
	if out != nil {
		t.Errorf("expected nil for silent command, got %+v", out)
	}
}

func TestCommandRouterPreservesReplyToID(t *testing.T) {
	reg := newTestRegistry(t)
	router := NewCommandRouter(reg)

	msg := IncomingMessage{
		Platform:  PlatformTelegram,
		ChannelID: "chat-1",
		UserID:    "user-8",
		Text:      "/status",
		ReplyToID: "msg-999",
	}

	out := router.ProcessMessage(msg)
	if out == nil {
		t.Fatal("expected outgoing message, got nil")
	}
	if out.ReplyToID != "msg-999" {
		t.Errorf("expected reply_to 'msg-999', got %q", out.ReplyToID)
	}
}

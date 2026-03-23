package commands

import (
	"strings"
	"testing"
)

func newTestRegistry() *Registry {
	r := NewRegistry()
	_ = r.Register(CommandDef{
		Name:        "help",
		Aliases:     []string{"h"},
		Description: "Show help",
		Usage:       "/help [command]",
		Category:    "general",
		Handler: func(args []string) CommandResult {
			return CommandResult{Output: "help output"}
		},
	})
	_ = r.Register(CommandDef{
		Name:        "memory",
		Description: "Memory commands",
		Usage:       "/memory <subcommand>",
		Category:    "memory",
		Handler: func(args []string) CommandResult {
			return CommandResult{Output: "memory top-level"}
		},
		SubCommands: map[string]CommandDef{
			"search": {
				Name:        "search",
				Description: "Search memory",
				Usage:       "/memory search <query>",
				Category:    "memory",
				Handler: func(args []string) CommandResult {
					return CommandResult{Output: "search: " + strings.Join(args, " ")}
				},
			},
			"stats": {
				Name:        "stats",
				Description: "Memory stats",
				Usage:       "/memory stats",
				Category:    "memory",
				Handler: func(args []string) CommandResult {
					return CommandResult{Output: "stats output"}
				},
			},
		},
	})
	_ = r.Register(CommandDef{
		Name:        "clear",
		Description: "Clear conversation",
		Usage:       "/clear",
		Category:    "general",
		Handler: func(args []string) CommandResult {
			return CommandResult{Output: "cleared"}
		},
	})
	return r
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}
	if len(r.List()) != 0 {
		t.Errorf("new registry should be empty, got %d commands", len(r.List()))
	}
}

func TestRegister(t *testing.T) {
	r := NewRegistry()
	err := r.Register(CommandDef{
		Name:        "test",
		Description: "A test command",
		Category:    "general",
		Handler:     func(args []string) CommandResult { return CommandResult{} },
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if len(r.List()) != 1 {
		t.Errorf("expected 1 command, got %d", len(r.List()))
	}
}

func TestRegisterDuplicate(t *testing.T) {
	r := NewRegistry()
	cmd := CommandDef{
		Name:     "test",
		Category: "general",
		Handler:  func(args []string) CommandResult { return CommandResult{} },
	}
	_ = r.Register(cmd)
	err := r.Register(cmd)
	if err == nil {
		t.Error("expected error when registering duplicate command")
	}
}

func TestRegisterNoHandler(t *testing.T) {
	r := NewRegistry()
	err := r.Register(CommandDef{Name: "bad", Category: "general"})
	if err == nil {
		t.Error("expected error when registering command without handler")
	}
}

func TestGetByName(t *testing.T) {
	r := newTestRegistry()
	cmd, ok := r.Get("help")
	if !ok {
		t.Fatal("Get('help') returned false")
	}
	if cmd.Name != "help" {
		t.Errorf("expected name 'help', got %q", cmd.Name)
	}
}

func TestGetByAlias(t *testing.T) {
	r := newTestRegistry()
	cmd, ok := r.Get("h")
	if !ok {
		t.Fatal("Get('h') returned false")
	}
	if cmd.Name != "help" {
		t.Errorf("expected name 'help', got %q", cmd.Name)
	}
}

func TestGetUnknown(t *testing.T) {
	r := newTestRegistry()
	_, ok := r.Get("nonexistent")
	if ok {
		t.Error("Get should return false for unknown command")
	}
}

func TestIsCommand(t *testing.T) {
	r := newTestRegistry()
	tests := []struct {
		input string
		want  bool
	}{
		{"/help", true},
		{"/unknown", true},
		{"hello", false},
		{"", false},
		{" /help", false},
		{"/", true},
	}
	for _, tt := range tests {
		got := r.IsCommand(tt.input)
		if got != tt.want {
			t.Errorf("IsCommand(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseInput(t *testing.T) {
	r := newTestRegistry()
	tests := []struct {
		input    string
		wantName string
		wantArgs []string
	}{
		{"/help", "help", nil},
		{"/help commands", "help", []string{"commands"}},
		{"/memory search hello world", "memory", []string{"search", "hello", "world"}},
		{"", "", nil},
		{"not a command", "", nil},
	}
	for _, tt := range tests {
		name, args := r.ParseInput(tt.input)
		if name != tt.wantName {
			t.Errorf("ParseInput(%q) name = %q, want %q", tt.input, name, tt.wantName)
		}
		if len(args) != len(tt.wantArgs) {
			t.Errorf("ParseInput(%q) args len = %d, want %d", tt.input, len(args), len(tt.wantArgs))
			continue
		}
		for i, a := range args {
			if a != tt.wantArgs[i] {
				t.Errorf("ParseInput(%q) args[%d] = %q, want %q", tt.input, i, a, tt.wantArgs[i])
			}
		}
	}
}

func TestExecute(t *testing.T) {
	r := newTestRegistry()

	// Execute known command
	result, ok := r.Execute("/help")
	if !ok {
		t.Fatal("Execute('/help') returned false")
	}
	if result.Output != "help output" {
		t.Errorf("expected 'help output', got %q", result.Output)
	}

	// Execute via alias
	result, ok = r.Execute("/h")
	if !ok {
		t.Fatal("Execute('/h') returned false")
	}
	if result.Output != "help output" {
		t.Errorf("expected 'help output', got %q", result.Output)
	}

	// Non-command input
	_, ok = r.Execute("hello")
	if ok {
		t.Error("Execute should return false for non-command input")
	}

	// Unknown command
	result, ok = r.Execute("/nonexistent")
	if !ok {
		t.Fatal("Execute('/nonexistent') should return true with error")
	}
	if result.Error == "" {
		t.Error("Execute('/nonexistent') should have error message")
	}
}

func TestExecuteSubCommand(t *testing.T) {
	r := newTestRegistry()

	result, ok := r.Execute("/memory search cats dogs")
	if !ok {
		t.Fatal("Execute sub-command returned false")
	}
	if result.Output != "search: cats dogs" {
		t.Errorf("expected 'search: cats dogs', got %q", result.Output)
	}

	result, ok = r.Execute("/memory stats")
	if !ok {
		t.Fatal("Execute sub-command returned false")
	}
	if result.Output != "stats output" {
		t.Errorf("expected 'stats output', got %q", result.Output)
	}

	// Unknown sub-command falls through to parent handler
	result, ok = r.Execute("/memory unknown")
	if !ok {
		t.Fatal("Execute unknown sub-command returned false")
	}
	if result.Output != "memory top-level" {
		t.Errorf("expected 'memory top-level', got %q", result.Output)
	}
}

func TestList(t *testing.T) {
	r := newTestRegistry()
	list := r.List()
	if len(list) != 3 {
		t.Errorf("expected 3 commands, got %d", len(list))
	}
}

func TestListByCategory(t *testing.T) {
	r := newTestRegistry()
	general := r.ListByCategory("general")
	if len(general) != 2 {
		t.Errorf("expected 2 general commands, got %d", len(general))
	}
	memory := r.ListByCategory("memory")
	if len(memory) != 1 {
		t.Errorf("expected 1 memory command, got %d", len(memory))
	}
	empty := r.ListByCategory("nonexistent")
	if len(empty) != 0 {
		t.Errorf("expected 0 commands for unknown category, got %d", len(empty))
	}
}

func TestSuggest(t *testing.T) {
	r := newTestRegistry()

	suggestions := r.Suggest("he")
	if len(suggestions) != 1 {
		t.Fatalf("expected 1 suggestion for 'he', got %d", len(suggestions))
	}
	if suggestions[0].Name != "help" {
		t.Errorf("expected suggestion 'help', got %q", suggestions[0].Name)
	}

	suggestions = r.Suggest("mem")
	if len(suggestions) != 1 {
		t.Fatalf("expected 1 suggestion for 'mem', got %d", len(suggestions))
	}

	suggestions = r.Suggest("cl")
	if len(suggestions) != 1 {
		t.Fatalf("expected 1 suggestion for 'cl', got %d", len(suggestions))
	}

	// Empty prefix returns all
	suggestions = r.Suggest("")
	if len(suggestions) != 3 {
		t.Errorf("expected 3 suggestions for empty prefix, got %d", len(suggestions))
	}

	// No match
	suggestions = r.Suggest("zzz")
	if len(suggestions) != 0 {
		t.Errorf("expected 0 suggestions for 'zzz', got %d", len(suggestions))
	}
}

func TestFormatHelp(t *testing.T) {
	r := newTestRegistry()
	help := r.FormatHelp()
	if help == "" {
		t.Error("FormatHelp returned empty string")
	}
	if !strings.Contains(help, "help") {
		t.Error("FormatHelp should contain 'help'")
	}
	if !strings.Contains(help, "memory") {
		t.Error("FormatHelp should contain 'memory'")
	}
}

func TestFormatCommandHelp(t *testing.T) {
	r := newTestRegistry()

	helpText := r.FormatCommandHelp("help")
	if helpText == "" {
		t.Error("FormatCommandHelp('help') returned empty")
	}
	if !strings.Contains(helpText, "help") {
		t.Error("FormatCommandHelp should contain command name")
	}

	// Unknown command
	unknown := r.FormatCommandHelp("nonexistent")
	if !strings.Contains(unknown, "Unknown") && !strings.Contains(unknown, "unknown") {
		t.Error("FormatCommandHelp for unknown should mention 'unknown'")
	}

	// Command with sub-commands
	memHelp := r.FormatCommandHelp("memory")
	if !strings.Contains(memHelp, "search") {
		t.Error("FormatCommandHelp('memory') should list sub-commands")
	}
}

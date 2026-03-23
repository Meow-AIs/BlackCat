package commands

import (
	"strings"
	"testing"
)

func registryWithBuiltins() *Registry {
	r := NewRegistry()
	RegisterBuiltinCommands(r)
	return r
}

func TestRegisterBuiltinCommands(t *testing.T) {
	r := registryWithBuiltins()
	cmds := r.List()
	if len(cmds) == 0 {
		t.Fatal("no builtin commands registered")
	}
	// Should have at least the major ones
	expectedNames := []string{
		"help", "clear", "compact", "cost", "model", "think", "fast",
		"status", "version", "exit",
		"memory", "skills", "config", "domain", "plugin", "hooks",
		"diff", "commit", "undo",
		"doctor", "tokens", "context", "debug",
	}
	for _, name := range expectedNames {
		if _, ok := r.Get(name); !ok {
			t.Errorf("expected builtin command %q to be registered", name)
		}
	}
}

func TestBuiltinCategories(t *testing.T) {
	r := registryWithBuiltins()
	categories := []string{"general", "memory", "skills", "config", "git", "debug"}
	for _, cat := range categories {
		cmds := r.ListByCategory(cat)
		if len(cmds) == 0 {
			t.Errorf("expected at least one command in category %q", cat)
		}
	}
}

func TestHelpCommand(t *testing.T) {
	r := registryWithBuiltins()
	// /help with no args
	result, ok := r.Execute("/help")
	if !ok {
		t.Fatal("Execute('/help') returned false")
	}
	if result.Output == "" {
		t.Error("/help returned empty output")
	}
	if result.Error != "" {
		t.Errorf("/help returned error: %s", result.Error)
	}

	// /help for specific command
	result, ok = r.Execute("/help memory")
	if !ok {
		t.Fatal("Execute('/help memory') returned false")
	}
	if result.Output == "" {
		t.Error("/help memory returned empty output")
	}
}

func TestHelpAlias(t *testing.T) {
	r := registryWithBuiltins()
	result, ok := r.Execute("/h")
	if !ok {
		t.Fatal("Execute('/h') returned false")
	}
	if result.Output == "" {
		t.Error("/h returned empty output")
	}
}

func TestGeneralCommands(t *testing.T) {
	r := registryWithBuiltins()
	cmds := []string{"/clear", "/compact", "/cost", "/model", "/think", "/fast", "/status", "/version", "/exit"}
	for _, c := range cmds {
		result, ok := r.Execute(c)
		if !ok {
			t.Errorf("Execute(%q) returned false", c)
			continue
		}
		if result.Output == "" && result.Error == "" {
			t.Errorf("Execute(%q) returned empty output and no error", c)
		}
	}
}

func TestMemorySubCommands(t *testing.T) {
	r := registryWithBuiltins()
	tests := []struct {
		input    string
		contains string
	}{
		{"/memory search test query", "search"},
		{"/memory stats", "stats"},
		{"/memory forget abc123", "forget"},
		{"/memory list", "memor"},
		{"/memory list episodic", "memor"},
		{"/memory export", "export"},
	}
	for _, tt := range tests {
		result, ok := r.Execute(tt.input)
		if !ok {
			t.Errorf("Execute(%q) returned false", tt.input)
			continue
		}
		if result.Output == "" {
			t.Errorf("Execute(%q) returned empty output", tt.input)
			continue
		}
		lower := strings.ToLower(result.Output)
		if !strings.Contains(lower, tt.contains) {
			t.Errorf("Execute(%q) output %q does not contain %q", tt.input, result.Output, tt.contains)
		}
	}
}

func TestSkillsSubCommands(t *testing.T) {
	r := registryWithBuiltins()
	cmds := []string{
		"/skills search test",
		"/skills install test-skill",
		"/skills uninstall test-skill",
		"/skills list",
		"/skills update",
	}
	for _, c := range cmds {
		result, ok := r.Execute(c)
		if !ok {
			t.Errorf("Execute(%q) returned false", c)
			continue
		}
		if result.Output == "" {
			t.Errorf("Execute(%q) returned empty output", c)
		}
	}
}

func TestConfigSubCommands(t *testing.T) {
	r := registryWithBuiltins()
	cmds := []string{"/config show", "/config set key value", "/config reset"}
	for _, c := range cmds {
		result, ok := r.Execute(c)
		if !ok {
			t.Errorf("Execute(%q) returned false", c)
			continue
		}
		if result.Output == "" {
			t.Errorf("Execute(%q) returned empty output", c)
		}
	}
}

func TestDomainSubCommands(t *testing.T) {
	r := registryWithBuiltins()
	cmds := []string{"/domain", "/domain set devsecops", "/domain detect"}
	for _, c := range cmds {
		result, ok := r.Execute(c)
		if !ok {
			t.Errorf("Execute(%q) returned false", c)
			continue
		}
		if result.Output == "" {
			t.Errorf("Execute(%q) returned empty output", c)
		}
	}
}

func TestPluginSubCommands(t *testing.T) {
	r := registryWithBuiltins()
	cmds := []string{
		"/plugin list",
		"/plugin install test-plugin",
		"/plugin start test-plugin",
		"/plugin stop test-plugin",
	}
	for _, c := range cmds {
		result, ok := r.Execute(c)
		if !ok {
			t.Errorf("Execute(%q) returned false", c)
			continue
		}
		if result.Output == "" {
			t.Errorf("Execute(%q) returned empty output", c)
		}
	}
}

func TestHooksSubCommands(t *testing.T) {
	r := registryWithBuiltins()
	cmds := []string{"/hooks list", "/hooks enable hook-1", "/hooks disable hook-1"}
	for _, c := range cmds {
		result, ok := r.Execute(c)
		if !ok {
			t.Errorf("Execute(%q) returned false", c)
			continue
		}
		if result.Output == "" {
			t.Errorf("Execute(%q) returned empty output", c)
		}
	}
}

func TestGitCommands(t *testing.T) {
	r := registryWithBuiltins()
	cmds := []string{"/diff", "/commit test message", "/undo"}
	for _, c := range cmds {
		result, ok := r.Execute(c)
		if !ok {
			t.Errorf("Execute(%q) returned false", c)
			continue
		}
		if result.Output == "" {
			t.Errorf("Execute(%q) returned empty output", c)
		}
	}
}

func TestDebugCommands(t *testing.T) {
	r := registryWithBuiltins()
	cmds := []string{"/doctor", "/tokens", "/context", "/debug", "/debug on", "/debug off"}
	for _, c := range cmds {
		result, ok := r.Execute(c)
		if !ok {
			t.Errorf("Execute(%q) returned false", c)
			continue
		}
		if result.Output == "" {
			t.Errorf("Execute(%q) returned empty output", c)
		}
	}
}

func TestSuggestionsIncludeBuiltins(t *testing.T) {
	r := registryWithBuiltins()
	suggestions := r.Suggest("he")
	found := false
	for _, s := range suggestions {
		if s.Name == "help" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Suggest('he') should include 'help'")
	}
}

func TestHelpFormattingIncludesAllCategories(t *testing.T) {
	r := registryWithBuiltins()
	help := r.FormatHelp()
	categories := []string{"general", "memory", "skills", "config", "git", "debug"}
	for _, cat := range categories {
		if !strings.Contains(help, cat) {
			t.Errorf("FormatHelp should contain category %q", cat)
		}
	}
}

func TestModelWithArg(t *testing.T) {
	r := registryWithBuiltins()
	result, ok := r.Execute("/model gpt-4")
	if !ok {
		t.Fatal("Execute('/model gpt-4') returned false")
	}
	if !strings.Contains(result.Output, "gpt-4") {
		t.Errorf("expected output to contain 'gpt-4', got %q", result.Output)
	}
}

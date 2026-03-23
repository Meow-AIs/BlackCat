package commands

import (
	"strings"
	"testing"
)

func newTestMiddleware() *InputMiddleware {
	r := NewRegistry()
	RegisterBuiltinCommands(r)
	return NewInputMiddleware(r)
}

func TestProcessSlashCommand(t *testing.T) {
	m := newTestMiddleware()
	result, ok := m.Process("/help")
	if !ok {
		t.Fatal("Process('/help') returned false")
	}
	if result.Output == "" {
		t.Error("Process('/help') returned empty output")
	}
}

func TestProcessRegularInput(t *testing.T) {
	m := newTestMiddleware()
	_, ok := m.Process("hello world")
	if ok {
		t.Error("Process should return false for regular input")
	}
}

func TestProcessEmptyInput(t *testing.T) {
	m := newTestMiddleware()
	_, ok := m.Process("")
	if ok {
		t.Error("Process should return false for empty input")
	}
}

func TestProcessUnknownCommand(t *testing.T) {
	m := newTestMiddleware()
	result, ok := m.Process("/nonexistent")
	if !ok {
		t.Fatal("Process should return true for unknown slash command")
	}
	if result.Error == "" {
		t.Error("Process should return error for unknown command")
	}
}

func TestShouldBypassLLM(t *testing.T) {
	m := newTestMiddleware()
	tests := []struct {
		input string
		want  bool
	}{
		{"/help", true},
		{"/clear", true},
		{"/memory search test", true},
		{"hello", false},
		{"", false},
		{" /help", false},
	}
	for _, tt := range tests {
		got := m.ShouldBypassLLM(tt.input)
		if got != tt.want {
			t.Errorf("ShouldBypassLLM(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestAutocomplete(t *testing.T) {
	m := newTestMiddleware()

	suggestions := m.Autocomplete("/he")
	found := false
	for _, s := range suggestions {
		if s == "/help" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Autocomplete('/he') should include '/help', got %v", suggestions)
	}

	suggestions = m.Autocomplete("/mem")
	found = false
	for _, s := range suggestions {
		if s == "/memory" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Autocomplete('/mem') should include '/memory', got %v", suggestions)
	}

	// Non-slash input returns empty
	suggestions = m.Autocomplete("hello")
	if len(suggestions) != 0 {
		t.Errorf("Autocomplete for non-slash input should be empty, got %v", suggestions)
	}
}

func TestAutocompleteAllCommands(t *testing.T) {
	m := newTestMiddleware()
	suggestions := m.Autocomplete("/")
	if len(suggestions) == 0 {
		t.Error("Autocomplete('/') should return all commands")
	}
	for _, s := range suggestions {
		if !strings.HasPrefix(s, "/") {
			t.Errorf("suggestion %q should start with '/'", s)
		}
	}
}

func TestProcessSubCommand(t *testing.T) {
	m := newTestMiddleware()
	result, ok := m.Process("/memory stats")
	if !ok {
		t.Fatal("Process('/memory stats') returned false")
	}
	if result.Output == "" {
		t.Error("Process('/memory stats') returned empty output")
	}
}

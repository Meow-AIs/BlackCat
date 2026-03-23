package tui

import (
	"strings"
	"testing"
)

func TestNewInputModel(t *testing.T) {
	m := NewInputModel(1000)
	if m == nil {
		t.Fatal("expected non-nil InputModel")
	}
	if m.MaxLength != 1000 {
		t.Errorf("expected MaxLength 1000, got %d", m.MaxLength)
	}
	if m.Value != "" {
		t.Errorf("expected empty value, got %q", m.Value)
	}
}

func TestInputModelInsert(t *testing.T) {
	m := NewInputModel(100)
	m.Insert('h')
	m.Insert('i')
	if m.Value != "hi" {
		t.Errorf("expected 'hi', got %q", m.Value)
	}
}

func TestInputModelInsertMaxLength(t *testing.T) {
	m := NewInputModel(3)
	m.Insert('a')
	m.Insert('b')
	m.Insert('c')
	m.Insert('d') // should be ignored
	if m.Value != "abc" {
		t.Errorf("expected 'abc', got %q", m.Value)
	}
}

func TestInputModelBackspace(t *testing.T) {
	m := NewInputModel(100)
	m.Insert('a')
	m.Insert('b')
	m.Backspace()
	if m.Value != "a" {
		t.Errorf("expected 'a', got %q", m.Value)
	}
}

func TestInputModelBackspaceEmpty(t *testing.T) {
	m := NewInputModel(100)
	m.Backspace() // should not panic
	if m.Value != "" {
		t.Errorf("expected empty, got %q", m.Value)
	}
}

func TestInputModelSubmit(t *testing.T) {
	m := NewInputModel(100)
	m.Insert('h')
	m.Insert('i')
	result := m.Submit()
	if result != "hi" {
		t.Errorf("expected 'hi', got %q", result)
	}
	if m.Value != "" {
		t.Errorf("expected empty after submit, got %q", m.Value)
	}
}

func TestInputModelSubmitAddsToHistory(t *testing.T) {
	m := NewInputModel(100)
	m.Insert('a')
	m.Submit()
	m.Insert('b')
	m.Submit()

	if len(m.History) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(m.History))
	}
	if m.History[0] != "a" {
		t.Errorf("expected first history 'a', got %q", m.History[0])
	}
	if m.History[1] != "b" {
		t.Errorf("expected second history 'b', got %q", m.History[1])
	}
}

func TestInputModelSubmitEmpty(t *testing.T) {
	m := NewInputModel(100)
	result := m.Submit()
	if result != "" {
		t.Errorf("expected empty submit result, got %q", result)
	}
	if len(m.History) != 0 {
		t.Error("expected no history entry for empty submit")
	}
}

func TestInputModelHistoryNavigation(t *testing.T) {
	m := NewInputModel(100)
	m.Insert('a')
	m.Submit()
	m.Insert('b')
	m.Submit()
	m.Insert('c')
	m.Submit()

	// Navigate up through history
	m.HistoryUp()
	if m.Value != "c" {
		t.Errorf("expected 'c', got %q", m.Value)
	}
	m.HistoryUp()
	if m.Value != "b" {
		t.Errorf("expected 'b', got %q", m.Value)
	}
	m.HistoryUp()
	if m.Value != "a" {
		t.Errorf("expected 'a', got %q", m.Value)
	}
	// At top, should stay
	m.HistoryUp()
	if m.Value != "a" {
		t.Errorf("expected 'a' at top, got %q", m.Value)
	}

	// Navigate back down
	m.HistoryDown()
	if m.Value != "b" {
		t.Errorf("expected 'b', got %q", m.Value)
	}
	m.HistoryDown()
	if m.Value != "c" {
		t.Errorf("expected 'c', got %q", m.Value)
	}
	// Past bottom clears
	m.HistoryDown()
	if m.Value != "" {
		t.Errorf("expected empty past history bottom, got %q", m.Value)
	}
}

func TestInputModelHistoryUpEmpty(t *testing.T) {
	m := NewInputModel(100)
	m.HistoryUp() // should not panic
	if m.Value != "" {
		t.Errorf("expected empty, got %q", m.Value)
	}
}

func TestInputModelMultiLine(t *testing.T) {
	m := NewInputModel(100)
	if m.MultiLine {
		t.Error("expected multiline off by default")
	}
	m.SetMultiLine(true)
	if !m.MultiLine {
		t.Error("expected multiline on")
	}
	m.SetMultiLine(false)
	if m.MultiLine {
		t.Error("expected multiline off")
	}
}

func TestInputModelRender(t *testing.T) {
	m := NewInputModel(100)
	m.Insert('h')
	m.Insert('i')
	output := m.Render(80)
	if !strings.Contains(output, "hi") {
		t.Error("expected render to contain input value 'hi'")
	}
}

func TestInputModelRenderEmpty(t *testing.T) {
	m := NewInputModel(100)
	output := m.Render(80)
	// Should show a prompt indicator
	if output == "" {
		t.Error("expected non-empty render for empty input")
	}
}

package tui

import (
	"fmt"
	"strings"
)

// InputModel manages the text input field with history support.
type InputModel struct {
	Value      string
	History    []string
	HistoryIdx int
	MultiLine  bool
	MaxLength  int
}

// NewInputModel creates a new input model with the given max length.
func NewInputModel(maxLength int) *InputModel {
	return &InputModel{
		History:    make([]string, 0),
		HistoryIdx: -1,
		MaxLength:  maxLength,
	}
}

// Insert adds a character at the end of the input value.
func (m *InputModel) Insert(ch rune) {
	if len(m.Value) >= m.MaxLength {
		return
	}
	m.Value = m.Value + string(ch)
}

// Backspace removes the last character from the input value.
func (m *InputModel) Backspace() {
	if len(m.Value) == 0 {
		return
	}
	runes := []rune(m.Value)
	m.Value = string(runes[:len(runes)-1])
}

// Submit returns the current value, clears the input, and adds to history.
func (m *InputModel) Submit() string {
	val := m.Value
	m.Value = ""
	m.HistoryIdx = -1

	if val != "" {
		updated := make([]string, len(m.History), len(m.History)+1)
		copy(updated, m.History)
		m.History = append(updated, val)
	}

	return val
}

// HistoryUp navigates to the previous history entry.
func (m *InputModel) HistoryUp() {
	if len(m.History) == 0 {
		return
	}

	if m.HistoryIdx == -1 {
		// Start from most recent
		m.HistoryIdx = len(m.History) - 1
	} else if m.HistoryIdx > 0 {
		m.HistoryIdx--
	}

	m.Value = m.History[m.HistoryIdx]
}

// HistoryDown navigates to the next history entry, or clears if past the end.
func (m *InputModel) HistoryDown() {
	if len(m.History) == 0 || m.HistoryIdx == -1 {
		return
	}

	if m.HistoryIdx < len(m.History)-1 {
		m.HistoryIdx++
		m.Value = m.History[m.HistoryIdx]
	} else {
		// Past bottom of history, clear
		m.HistoryIdx = -1
		m.Value = ""
	}
}

// SetMultiLine enables or disables multi-line input mode.
func (m *InputModel) SetMultiLine(enabled bool) {
	m.MultiLine = enabled
}

// Render produces a string rendering of the input field.
func (m *InputModel) Render(width int) string {
	prompt := "> "
	if m.MultiLine {
		prompt = ">> "
	}

	display := m.Value
	maxDisplay := width - len(prompt) - 1
	if maxDisplay > 0 && len(display) > maxDisplay {
		display = display[len(display)-maxDisplay:]
	}

	return fmt.Sprintf("%s%s%s", prompt, display, strings.Repeat(" ", 0))
}

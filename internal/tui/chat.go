package tui

import (
	"fmt"
	"strings"
	"time"
)

// ChatMessage represents a single message in the chat view.
type ChatMessage struct {
	Role      string // "user", "assistant", "system", "tool"
	Content   string
	Timestamp time.Time
	ToolName  string // for tool messages
}

// ChatModel manages the chat message history and rendering.
type ChatModel struct {
	Messages   []ChatMessage
	MaxHistory int
}

// NewChatModel creates a new chat model with the given max history size.
func NewChatModel(maxHistory int) *ChatModel {
	return &ChatModel{
		Messages:   make([]ChatMessage, 0),
		MaxHistory: maxHistory,
	}
}

// AddMessage appends a message, truncating old messages if over MaxHistory.
func (m *ChatModel) AddMessage(msg ChatMessage) {
	updated := make([]ChatMessage, len(m.Messages), len(m.Messages)+1)
	copy(updated, m.Messages)
	updated = append(updated, msg)

	if len(updated) > m.MaxHistory {
		trimmed := make([]ChatMessage, m.MaxHistory)
		copy(trimmed, updated[len(updated)-m.MaxHistory:])
		updated = trimmed
	}

	m.Messages = updated
}

// Render produces a string rendering of visible messages within the given dimensions.
func (m *ChatModel) Render(width, height int) string {
	if len(m.Messages) == 0 {
		return centerText("Welcome to BlackCat. Type a message to begin.", width)
	}

	var b strings.Builder
	for i, msg := range m.Messages {
		if i > 0 {
			b.WriteString("\n")
		}
		renderMessage(&b, msg, width)
	}
	return b.String()
}

// Clear removes all messages.
func (m *ChatModel) Clear() {
	m.Messages = make([]ChatMessage, 0)
}

// MessageCount returns the number of stored messages.
func (m *ChatModel) MessageCount() int {
	return len(m.Messages)
}

func renderMessage(b *strings.Builder, msg ChatMessage, width int) {
	label := roleLabel(msg.Role)

	if msg.Role == "tool" && msg.ToolName != "" {
		label = fmt.Sprintf("[%s]", msg.ToolName)
	}

	ts := ""
	if !msg.Timestamp.IsZero() {
		ts = msg.Timestamp.Format("15:04")
	}

	header := label
	if ts != "" {
		header = fmt.Sprintf("%s  %s", label, ts)
	}

	b.WriteString(header)
	b.WriteString("\n")

	lines := wrapText(msg.Content, width-2)
	for _, line := range lines {
		b.WriteString("  ")
		b.WriteString(line)
		b.WriteString("\n")
	}
}

func roleLabel(role string) string {
	switch role {
	case "user":
		return "You"
	case "assistant":
		return "BlackCat"
	case "system":
		return "System"
	case "tool":
		return "Tool"
	default:
		return role
	}
}

func wrapText(text string, maxWidth int) []string {
	if maxWidth <= 0 {
		maxWidth = 1
	}
	if text == "" {
		return []string{""}
	}

	var result []string
	for _, line := range strings.Split(text, "\n") {
		if len(line) <= maxWidth {
			result = append(result, line)
			continue
		}
		for len(line) > maxWidth {
			result = append(result, line[:maxWidth])
			line = line[maxWidth:]
		}
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

func centerText(text string, width int) string {
	if len(text) >= width {
		return text
	}
	pad := (width - len(text)) / 2
	return strings.Repeat(" ", pad) + text
}

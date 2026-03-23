package tui

import "fmt"

// ToolbarModel displays status information at the top or bottom of the TUI.
type ToolbarModel struct {
	Model    string
	Project  string
	MemCount int
	Cost     float64
	State    string // "idle", "thinking", "executing"
}

// NewToolbarModel creates a toolbar with default idle state.
func NewToolbarModel() *ToolbarModel {
	return &ToolbarModel{
		State: "idle",
	}
}

// Update sets all toolbar fields at once (immutable style - returns via receiver mutation
// but replaces all fields atomically).
func (m *ToolbarModel) Update(model, project, state string, memCount int, cost float64) {
	m.Model = model
	m.Project = project
	m.State = state
	m.MemCount = memCount
	m.Cost = cost
}

// Render produces a single-line status bar within the given width.
func (m *ToolbarModel) Render(width int) string {
	stateIndicator := m.State
	switch m.State {
	case "thinking":
		stateIndicator = "thinking..."
	case "executing":
		stateIndicator = "executing..."
	}

	left := fmt.Sprintf(" %s | %s ", m.Model, m.Project)
	right := fmt.Sprintf(" %s | mem:%d | $%.2f ", stateIndicator, m.MemCount, m.Cost)

	totalContent := len(left) + len(right)
	if totalContent >= width {
		// Compact mode: just essential info
		return fmt.Sprintf(" %s | %s ", m.Model, stateIndicator)
	}

	gap := width - totalContent
	padding := ""
	for i := 0; i < gap; i++ {
		padding += " "
	}

	return left + padding + right
}

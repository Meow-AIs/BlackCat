package tui

import (
	"fmt"
	"strings"
)

// PermissionChoice represents the user's decision on a permission prompt.
type PermissionChoice string

const (
	PermAllow       PermissionChoice = "allow"
	PermDeny        PermissionChoice = "deny"
	PermAlwaysAllow PermissionChoice = "always_allow"
)

// choices maps selection index to PermissionChoice.
var choices = []PermissionChoice{PermAllow, PermDeny, PermAlwaysAllow}

// choiceLabels maps selection index to display label.
var choiceLabels = []string{"Allow", "Deny", "Always Allow"}

const maxChoiceIdx = 2

// PermissionPrompt presents a tool execution permission dialog.
type PermissionPrompt struct {
	Command  string
	Risk     string
	Selected int // 0=allow, 1=deny, 2=always_allow
	Resolved bool
	Choice   PermissionChoice
}

// NewPermissionPrompt creates a permission prompt for the given command and risk level.
func NewPermissionPrompt(command, risk string) *PermissionPrompt {
	return &PermissionPrompt{
		Command:  command,
		Risk:     risk,
		Selected: 0,
	}
}

// SelectNext moves the selection to the next option.
func (p *PermissionPrompt) SelectNext() {
	if p.Selected < maxChoiceIdx {
		p.Selected++
	}
}

// SelectPrev moves the selection to the previous option.
func (p *PermissionPrompt) SelectPrev() {
	if p.Selected > 0 {
		p.Selected--
	}
}

// Confirm resolves the prompt with the currently selected option.
func (p *PermissionPrompt) Confirm() PermissionChoice {
	p.Choice = choices[p.Selected]
	p.Resolved = true
	return p.Choice
}

// Render produces a string rendering of the permission prompt.
func (p *PermissionPrompt) Render(width int) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Permission Required [risk: %s]\n", p.Risk))
	b.WriteString(fmt.Sprintf("Command: %s\n\n", p.Command))

	for i, label := range choiceLabels {
		if i == p.Selected {
			b.WriteString(fmt.Sprintf("  > [%s]\n", label))
		} else {
			b.WriteString(fmt.Sprintf("    %s\n", label))
		}
	}

	return b.String()
}

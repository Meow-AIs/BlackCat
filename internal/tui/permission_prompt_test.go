package tui

import (
	"strings"
	"testing"
)

func TestNewPermissionPrompt(t *testing.T) {
	p := NewPermissionPrompt("rm -rf /tmp/test", "high")
	if p == nil {
		t.Fatal("expected non-nil PermissionPrompt")
	}
	if p.Command != "rm -rf /tmp/test" {
		t.Errorf("expected command, got %q", p.Command)
	}
	if p.Risk != "high" {
		t.Errorf("expected risk 'high', got %q", p.Risk)
	}
	if p.Selected != 0 {
		t.Errorf("expected selected 0, got %d", p.Selected)
	}
	if p.Resolved {
		t.Error("expected not resolved initially")
	}
}

func TestPermissionPromptSelectNext(t *testing.T) {
	p := NewPermissionPrompt("cmd", "low")
	if p.Selected != 0 {
		t.Fatal("expected start at 0")
	}
	p.SelectNext()
	if p.Selected != 1 {
		t.Errorf("expected 1, got %d", p.Selected)
	}
	p.SelectNext()
	if p.Selected != 2 {
		t.Errorf("expected 2, got %d", p.Selected)
	}
	// Should wrap or clamp at max
	p.SelectNext()
	if p.Selected != 2 {
		t.Errorf("expected clamped at 2, got %d", p.Selected)
	}
}

func TestPermissionPromptSelectPrev(t *testing.T) {
	p := NewPermissionPrompt("cmd", "low")
	p.Selected = 2
	p.SelectPrev()
	if p.Selected != 1 {
		t.Errorf("expected 1, got %d", p.Selected)
	}
	p.SelectPrev()
	if p.Selected != 0 {
		t.Errorf("expected 0, got %d", p.Selected)
	}
	// Should clamp at 0
	p.SelectPrev()
	if p.Selected != 0 {
		t.Errorf("expected clamped at 0, got %d", p.Selected)
	}
}

func TestPermissionPromptConfirmAllow(t *testing.T) {
	p := NewPermissionPrompt("ls", "low")
	p.Selected = 0
	choice := p.Confirm()
	if choice != PermAllow {
		t.Errorf("expected PermAllow, got %s", choice)
	}
	if !p.Resolved {
		t.Error("expected resolved after confirm")
	}
	if p.Choice != PermAllow {
		t.Errorf("expected Choice PermAllow, got %s", p.Choice)
	}
}

func TestPermissionPromptConfirmDeny(t *testing.T) {
	p := NewPermissionPrompt("rm -rf /", "critical")
	p.Selected = 1
	choice := p.Confirm()
	if choice != PermDeny {
		t.Errorf("expected PermDeny, got %s", choice)
	}
}

func TestPermissionPromptConfirmAlwaysAllow(t *testing.T) {
	p := NewPermissionPrompt("git status", "low")
	p.Selected = 2
	choice := p.Confirm()
	if choice != PermAlwaysAllow {
		t.Errorf("expected PermAlwaysAllow, got %s", choice)
	}
}

func TestPermissionPromptRender(t *testing.T) {
	p := NewPermissionPrompt("docker build .", "medium")
	output := p.Render(80)

	if !strings.Contains(output, "docker build .") {
		t.Error("expected render to contain command")
	}
	if !strings.Contains(output, "medium") {
		t.Error("expected render to contain risk level")
	}
	if !strings.Contains(output, "Allow") {
		t.Error("expected render to contain 'Allow' option")
	}
	if !strings.Contains(output, "Deny") {
		t.Error("expected render to contain 'Deny' option")
	}
}

func TestPermissionPromptRenderHighlightsSelected(t *testing.T) {
	p := NewPermissionPrompt("cmd", "low")
	p.Selected = 1 // Deny
	output := p.Render(80)
	// The selected option should be visually distinct
	if !strings.Contains(output, "Deny") {
		t.Error("expected 'Deny' in render")
	}
}

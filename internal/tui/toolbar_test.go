package tui

import (
	"strings"
	"testing"
)

func TestNewToolbarModel(t *testing.T) {
	m := NewToolbarModel()
	if m == nil {
		t.Fatal("expected non-nil ToolbarModel")
	}
	if m.State != "idle" {
		t.Errorf("expected initial state 'idle', got %q", m.State)
	}
}

func TestToolbarModelUpdate(t *testing.T) {
	m := NewToolbarModel()
	m.Update("gpt-4", "blackcat", "thinking", 42, 0.05)

	if m.Model != "gpt-4" {
		t.Errorf("expected model 'gpt-4', got %q", m.Model)
	}
	if m.Project != "blackcat" {
		t.Errorf("expected project 'blackcat', got %q", m.Project)
	}
	if m.State != "thinking" {
		t.Errorf("expected state 'thinking', got %q", m.State)
	}
	if m.MemCount != 42 {
		t.Errorf("expected memcount 42, got %d", m.MemCount)
	}
	if m.Cost != 0.05 {
		t.Errorf("expected cost 0.05, got %f", m.Cost)
	}
}

func TestToolbarModelRender(t *testing.T) {
	m := NewToolbarModel()
	m.Update("claude-sonnet", "myproject", "idle", 100, 1.23)

	output := m.Render(80)
	if !strings.Contains(output, "claude-sonnet") {
		t.Error("expected render to contain model name")
	}
	if !strings.Contains(output, "myproject") {
		t.Error("expected render to contain project name")
	}
	if !strings.Contains(output, "idle") {
		t.Error("expected render to contain state")
	}
}

func TestToolbarModelRenderThinking(t *testing.T) {
	m := NewToolbarModel()
	m.Update("claude-sonnet", "proj", "thinking", 50, 0.0)

	output := m.Render(80)
	if !strings.Contains(output, "thinking") {
		t.Error("expected render to show thinking state")
	}
}

func TestToolbarModelRenderExecuting(t *testing.T) {
	m := NewToolbarModel()
	m.Update("claude-sonnet", "proj", "executing", 50, 0.0)

	output := m.Render(80)
	if !strings.Contains(output, "executing") {
		t.Error("expected render to show executing state")
	}
}

func TestToolbarModelRenderNarrow(t *testing.T) {
	m := NewToolbarModel()
	m.Update("claude-sonnet-4", "my-long-project", "idle", 999, 12.34)

	// Should not panic with narrow width
	output := m.Render(30)
	if output == "" {
		t.Error("expected non-empty render for narrow width")
	}
}

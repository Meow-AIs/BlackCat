package builtin

import (
	"context"
	"strings"
	"testing"

	"github.com/meowai/blackcat/internal/tools"
)

func TestStatusToolInfo(t *testing.T) {
	tool := NewStatusTool()
	info := tool.Info()

	if info.Name != "system_status" {
		t.Errorf("expected name %q, got %q", "system_status", info.Name)
	}
	if info.Category != "system" {
		t.Errorf("expected category %q, got %q", "system", info.Category)
	}

	hasSection := false
	for _, p := range info.Parameters {
		if p.Name == "section" {
			hasSection = true
			if len(p.Enum) == 0 {
				t.Error("expected section parameter to have enum values")
			}
		}
	}
	if !hasSection {
		t.Error("expected 'section' parameter")
	}
}

func TestStatusToolImplementsInterface(t *testing.T) {
	var _ tools.Tool = NewStatusTool()
}

func TestStatusToolAll(t *testing.T) {
	tool := NewStatusTool()
	result, err := tool.Execute(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	// Default should show all sections
	if !strings.Contains(result.Output, "model") {
		t.Errorf("expected all status to contain model info, got %q", result.Output)
	}
}

func TestStatusToolModel(t *testing.T) {
	tool := NewStatusTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"section": "model",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "model") {
		t.Errorf("expected output to contain model info, got %q", result.Output)
	}
}

func TestStatusToolMemory(t *testing.T) {
	tool := NewStatusTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"section": "memory",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "Entries") {
		t.Errorf("expected output to contain memory entries, got %q", result.Output)
	}
}

func TestStatusToolPlugins(t *testing.T) {
	tool := NewStatusTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"section": "plugins",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "plugin") || !strings.Contains(result.Output, "Plugin") {
		if !strings.Contains(strings.ToLower(result.Output), "plugin") {
			t.Errorf("expected output to mention plugins, got %q", result.Output)
		}
	}
}

func TestStatusToolSession(t *testing.T) {
	tool := NewStatusTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"section": "session",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "Session") {
		t.Errorf("expected output to contain session info, got %q", result.Output)
	}
}

func TestStatusToolHealth(t *testing.T) {
	tool := NewStatusTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"section": "health",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "operational") {
		t.Errorf("expected output to contain health status, got %q", result.Output)
	}
}

func TestStatusToolInvalidSection(t *testing.T) {
	tool := NewStatusTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"section": "nonexistent",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1 for invalid section, got %d", result.ExitCode)
	}
}

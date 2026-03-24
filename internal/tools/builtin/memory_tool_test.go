package builtin

import (
	"context"
	"strings"
	"testing"

	"github.com/meowai/blackcat/internal/tools"
)

func TestMemoryToolInfo(t *testing.T) {
	tool := NewMemoryTool()
	info := tool.Info()

	if info.Name != "manage_memory" {
		t.Errorf("expected name %q, got %q", "manage_memory", info.Name)
	}
	if info.Category != "system" {
		t.Errorf("expected category %q, got %q", "system", info.Category)
	}

	hasAction := false
	for _, p := range info.Parameters {
		if p.Name == "action" && p.Required {
			hasAction = true
			if len(p.Enum) == 0 {
				t.Error("expected action parameter to have enum values")
			}
		}
	}
	if !hasAction {
		t.Error("expected required 'action' parameter")
	}
}

func TestMemoryToolImplementsInterface(t *testing.T) {
	var _ tools.Tool = NewMemoryTool()
}

func TestMemoryToolMissingAction(t *testing.T) {
	tool := NewMemoryTool()
	_, err := tool.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Error("expected error for missing action")
	}
}

func TestMemoryToolInvalidAction(t *testing.T) {
	tool := NewMemoryTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "invalid",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", result.ExitCode)
	}
}

func TestMemoryToolSearch(t *testing.T) {
	tool := NewMemoryTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "search",
		"query":  "security scan",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "security scan") {
		t.Errorf("expected output to contain query, got %q", result.Output)
	}
}

func TestMemoryToolStats(t *testing.T) {
	tool := NewMemoryTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "stats",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "episodic") {
		t.Errorf("expected output to mention memory tiers, got %q", result.Output)
	}
}

func TestMemoryToolForget(t *testing.T) {
	tool := NewMemoryTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "forget",
		"id":     "mem-abc123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "mem-abc123") {
		t.Errorf("expected output to contain memory ID, got %q", result.Output)
	}
}

func TestMemoryToolList(t *testing.T) {
	tool := NewMemoryTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "list",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "Recent") {
		t.Errorf("expected output to show recent memories, got %q", result.Output)
	}
}

func TestMemoryToolListWithTier(t *testing.T) {
	tool := NewMemoryTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "list",
		"tier":   "semantic",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "semantic") {
		t.Errorf("expected output to mention tier filter, got %q", result.Output)
	}
}

func TestMemoryToolExport(t *testing.T) {
	tool := NewMemoryTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "export",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "export") || !strings.Contains(result.Output, "Export") {
		// Accept either case
		if !strings.Contains(strings.ToLower(result.Output), "export") {
			t.Errorf("expected output to mention export, got %q", result.Output)
		}
	}
}

package builtin

import (
	"context"
	"strings"
	"testing"

	"github.com/meowai/blackcat/internal/tools"
)

func TestModelToolInfo(t *testing.T) {
	tool := NewModelTool()
	info := tool.Info()

	if info.Name != "change_model" {
		t.Errorf("expected name %q, got %q", "change_model", info.Name)
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

func TestModelToolImplementsInterface(t *testing.T) {
	var _ tools.Tool = NewModelTool()
}

func TestModelToolMissingAction(t *testing.T) {
	tool := NewModelTool()
	_, err := tool.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Error("expected error for missing action")
	}
}

func TestModelToolInvalidAction(t *testing.T) {
	tool := NewModelTool()
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

func TestModelToolSet(t *testing.T) {
	tool := NewModelTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "set",
		"model":  "anthropic/claude-sonnet-4-6",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "claude-sonnet-4-6") {
		t.Errorf("expected output to contain model name, got %q", result.Output)
	}
	if !strings.Contains(result.Output, "anthropic") || !strings.Contains(result.Output, "Anthropic") {
		if !strings.Contains(strings.ToLower(result.Output), "anthropic") {
			t.Errorf("expected output to contain provider, got %q", result.Output)
		}
	}
}

func TestModelToolSetOllama(t *testing.T) {
	tool := NewModelTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "set",
		"model":  "ollama/qwen2.5:32b",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "qwen2.5:32b") {
		t.Errorf("expected output to contain model name, got %q", result.Output)
	}
}

func TestModelToolSetMissingModel(t *testing.T) {
	tool := NewModelTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "set",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1 for missing model, got %d", result.ExitCode)
	}
}

func TestModelToolList(t *testing.T) {
	tool := NewModelTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "list",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "Anthropic") {
		t.Errorf("expected output to list Anthropic models, got %q", result.Output)
	}
}

func TestModelToolInfoAction(t *testing.T) {
	tool := NewModelTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "info",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "Current model") {
		t.Errorf("expected output to show current model, got %q", result.Output)
	}
}

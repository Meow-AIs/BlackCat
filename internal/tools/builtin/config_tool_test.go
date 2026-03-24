package builtin

import (
	"context"
	"strings"
	"testing"

	"github.com/meowai/blackcat/internal/tools"
)

func TestConfigToolInfo(t *testing.T) {
	tool := NewConfigTool()
	info := tool.Info()

	if info.Name != "manage_config" {
		t.Errorf("expected name %q, got %q", "manage_config", info.Name)
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

func TestConfigToolImplementsInterface(t *testing.T) {
	var _ tools.Tool = NewConfigTool()
}

func TestConfigToolMissingAction(t *testing.T) {
	tool := NewConfigTool()
	_, err := tool.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Error("expected error for missing action")
	}
}

func TestConfigToolInvalidAction(t *testing.T) {
	tool := NewConfigTool()
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

func TestConfigToolShow(t *testing.T) {
	tool := NewConfigTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "show",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "model") {
		t.Errorf("expected output to contain config info, got %q", result.Output)
	}
}

func TestConfigToolSetModel(t *testing.T) {
	tool := NewConfigTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "set",
		"key":    "model",
		"value":  "claude-sonnet-4-6",
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
}

func TestConfigToolGet(t *testing.T) {
	tool := NewConfigTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "get",
		"key":    "model",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "model") {
		t.Errorf("expected output to contain key name, got %q", result.Output)
	}
}

func TestConfigToolReset(t *testing.T) {
	tool := NewConfigTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "reset",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "reset") {
		t.Errorf("expected output to confirm reset, got %q", result.Output)
	}
}

func TestConfigToolSetMissingKey(t *testing.T) {
	tool := NewConfigTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "set",
		"value":  "something",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1 for missing key, got %d", result.ExitCode)
	}
}

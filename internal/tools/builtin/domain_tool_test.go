package builtin

import (
	"context"
	"strings"
	"testing"

	"github.com/meowai/blackcat/internal/tools"
)

func TestDomainToolInfo(t *testing.T) {
	tool := NewDomainTool()
	info := tool.Info()

	if info.Name != "manage_domain" {
		t.Errorf("expected name %q, got %q", "manage_domain", info.Name)
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

func TestDomainToolImplementsInterface(t *testing.T) {
	var _ tools.Tool = NewDomainTool()
}

func TestDomainToolMissingAction(t *testing.T) {
	tool := NewDomainTool()
	_, err := tool.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Error("expected error for missing action")
	}
}

func TestDomainToolInvalidAction(t *testing.T) {
	tool := NewDomainTool()
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

func TestDomainToolDetect(t *testing.T) {
	tool := NewDomainTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "detect",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "Detected domain") {
		t.Errorf("expected output to show detected domain, got %q", result.Output)
	}
}

func TestDomainToolSetDevsecops(t *testing.T) {
	tool := NewDomainTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "set",
		"domain": "devsecops",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "DevSecOps") {
		t.Errorf("expected output to confirm domain switch, got %q", result.Output)
	}
}

func TestDomainToolSetArchitect(t *testing.T) {
	tool := NewDomainTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "set",
		"domain": "architect",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "Solution Architect") {
		t.Errorf("expected output to confirm domain switch, got %q", result.Output)
	}
}

func TestDomainToolSetInvalidDomain(t *testing.T) {
	tool := NewDomainTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "set",
		"domain": "unknown",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1 for unknown domain, got %d", result.ExitCode)
	}
}

func TestDomainToolInfo_Action(t *testing.T) {
	tool := NewDomainTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "info",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "Current domain") {
		t.Errorf("expected output to show current domain, got %q", result.Output)
	}
}

func TestDomainToolList(t *testing.T) {
	tool := NewDomainTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "list",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "general") {
		t.Errorf("expected output to list general domain, got %q", result.Output)
	}
	if !strings.Contains(result.Output, "devsecops") {
		t.Errorf("expected output to list devsecops domain, got %q", result.Output)
	}
	if !strings.Contains(result.Output, "architect") {
		t.Errorf("expected output to list architect domain, got %q", result.Output)
	}
}

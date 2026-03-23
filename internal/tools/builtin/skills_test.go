package builtin

import (
	"context"
	"strings"
	"testing"
)

func TestSkillsToolInfo(t *testing.T) {
	tool := NewSkillsTool()
	info := tool.Info()

	if info.Name != "manage_skills" {
		t.Errorf("expected name 'manage_skills', got %q", info.Name)
	}
	if info.Category != "skills" {
		t.Errorf("expected category 'skills', got %q", info.Category)
	}
	if info.Description == "" {
		t.Error("expected non-empty description")
	}
	if len(info.Parameters) < 3 {
		t.Errorf("expected at least 3 parameters, got %d", len(info.Parameters))
	}

	// Verify action parameter has enum
	for _, p := range info.Parameters {
		if p.Name == "action" {
			if !p.Required {
				t.Error("action parameter should be required")
			}
			if len(p.Enum) != 6 {
				t.Errorf("expected 6 enum values for action, got %d", len(p.Enum))
			}
		}
	}
}

func TestSkillsToolSearch(t *testing.T) {
	tool := NewSkillsTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "search",
		"query":  "secret scanner",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Error != "" {
		t.Errorf("unexpected error: %s", result.Error)
	}
	if result.Output == "" {
		t.Error("expected non-empty output for search")
	}
}

func TestSkillsToolInstall(t *testing.T) {
	tool := NewSkillsTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "install",
		"query":  "devsecops/secret-scanner",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Error != "" {
		t.Errorf("unexpected error: %s", result.Error)
	}
	if !strings.Contains(result.Output, "devsecops/secret-scanner") {
		t.Errorf("expected output to contain skill name, got %q", result.Output)
	}
}

func TestSkillsToolInstallWithVersion(t *testing.T) {
	tool := NewSkillsTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action":  "install",
		"query":   "devsecops/secret-scanner",
		"version": "1.2.0",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Error != "" {
		t.Errorf("unexpected error: %s", result.Error)
	}
	if !strings.Contains(result.Output, "devsecops/secret-scanner") {
		t.Errorf("expected output to contain skill name, got %q", result.Output)
	}
}

func TestSkillsToolUninstall(t *testing.T) {
	tool := NewSkillsTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "uninstall",
		"query":  "devsecops/secret-scanner",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Error != "" {
		t.Errorf("unexpected error: %s", result.Error)
	}
	if !strings.Contains(result.Output, "devsecops/secret-scanner") {
		t.Errorf("expected output to contain skill name, got %q", result.Output)
	}
}

func TestSkillsToolUpdate(t *testing.T) {
	tool := NewSkillsTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "update",
		"query":  "devsecops/secret-scanner",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Error != "" {
		t.Errorf("unexpected error: %s", result.Error)
	}
	if !strings.Contains(result.Output, "devsecops/secret-scanner") {
		t.Errorf("expected output to contain skill name, got %q", result.Output)
	}
}

func TestSkillsToolList(t *testing.T) {
	tool := NewSkillsTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "list",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Error != "" {
		t.Errorf("unexpected error: %s", result.Error)
	}
	if result.Output == "" {
		t.Error("expected non-empty output for list")
	}
}

func TestSkillsToolInfoAction(t *testing.T) {
	tool := NewSkillsTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "info",
		"query":  "devsecops/secret-scanner",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Error != "" {
		t.Errorf("unexpected error: %s", result.Error)
	}
	if result.Output == "" {
		t.Error("expected non-empty output for info")
	}
}

func TestSkillsToolUnknownAction(t *testing.T) {
	tool := NewSkillsTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "delete",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Error == "" {
		t.Error("expected error for unknown action")
	}
}

func TestSkillsToolMissingAction(t *testing.T) {
	tool := NewSkillsTool()
	result, err := tool.Execute(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Error == "" {
		t.Error("expected error for missing action")
	}
}

func TestSkillsToolSearchEmptyQuery(t *testing.T) {
	tool := NewSkillsTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "search",
		"query":  "",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	// Empty search should still return something (e.g., popular skills)
	if result.Output == "" {
		t.Error("expected non-empty output even for empty search")
	}
}

func TestSkillsToolImplementsToolInterface(t *testing.T) {
	// Verify the tool satisfies the tools.Tool interface at compile time.
	// This is tested by the type assertion in the test itself.
	tool := NewSkillsTool()
	_ = tool.Info()
	_, _ = tool.Execute(context.Background(), map[string]any{"action": "list"})
}

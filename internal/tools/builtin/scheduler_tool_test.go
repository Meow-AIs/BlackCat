package builtin

import (
	"context"
	"strings"
	"testing"

	"github.com/meowai/blackcat/internal/tools"
)

func TestSchedulerToolInfo(t *testing.T) {
	tool := NewSchedulerTool()
	info := tool.Info()

	if info.Name != "manage_scheduler" {
		t.Errorf("expected name %q, got %q", "manage_scheduler", info.Name)
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

func TestSchedulerToolImplementsInterface(t *testing.T) {
	var _ tools.Tool = NewSchedulerTool()
}

func TestSchedulerToolMissingAction(t *testing.T) {
	tool := NewSchedulerTool()
	_, err := tool.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Error("expected error for missing action")
	}
}

func TestSchedulerToolInvalidAction(t *testing.T) {
	tool := NewSchedulerTool()
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

func TestSchedulerToolAdd(t *testing.T) {
	tool := NewSchedulerTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "add",
		"name":   "daily-scan",
		"cron":   "0 9 * * *",
		"task":   "Run security scan",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "daily-scan") {
		t.Errorf("expected output to contain schedule name, got %q", result.Output)
	}
	if !strings.Contains(result.Output, "0 9 * * *") {
		t.Errorf("expected output to contain cron expression, got %q", result.Output)
	}
}

func TestSchedulerToolList(t *testing.T) {
	tool := NewSchedulerTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "list",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "schedule") {
		t.Errorf("expected output to mention schedules, got %q", result.Output)
	}
}

func TestSchedulerToolRemove(t *testing.T) {
	tool := NewSchedulerTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "remove",
		"name":   "daily-scan",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "Removed") {
		t.Errorf("expected output to confirm removal, got %q", result.Output)
	}
}

func TestSchedulerToolHistory(t *testing.T) {
	tool := NewSchedulerTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "history",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "Recent runs") {
		t.Errorf("expected output to show run history, got %q", result.Output)
	}
}

func TestSchedulerToolPause(t *testing.T) {
	tool := NewSchedulerTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "pause",
		"name":   "daily-scan",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "paused") {
		t.Errorf("expected output to confirm pause, got %q", result.Output)
	}
}

func TestSchedulerToolResume(t *testing.T) {
	tool := NewSchedulerTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "resume",
		"name":   "daily-scan",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "resumed") {
		t.Errorf("expected output to confirm resume, got %q", result.Output)
	}
}

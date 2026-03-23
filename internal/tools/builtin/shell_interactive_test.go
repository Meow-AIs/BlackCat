package builtin

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestInteractiveShellToolInfo(t *testing.T) {
	sm := NewSessionManager(5)
	tool := NewInteractiveShellTool(sm)

	info := tool.Info()
	if info.Name != "interactive_shell" {
		t.Errorf("expected name 'interactive_shell', got %q", info.Name)
	}
	if info.Category != "shell" {
		t.Errorf("expected category 'shell', got %q", info.Category)
	}
	if len(info.Parameters) == 0 {
		t.Error("expected parameters")
	}
}

func TestInteractiveShellToolStartAction(t *testing.T) {
	sm := NewSessionManager(5)
	defer sm.Cleanup()
	tool := NewInteractiveShellTool(sm)

	result, err := tool.Execute(context.Background(), map[string]any{
		"action":     "start",
		"session_id": "t1",
		"command":    "echo hello",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "t1") {
		t.Errorf("expected output containing session ID, got %q", result.Output)
	}
}

func TestInteractiveShellToolSendAction(t *testing.T) {
	sm := NewSessionManager(5)
	defer sm.Cleanup()
	tool := NewInteractiveShellTool(sm)

	// Start a cat session
	_, err := tool.Execute(context.Background(), map[string]any{
		"action":     "start",
		"session_id": "send1",
		"command":    "cat",
	})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Send input
	result, err := tool.Execute(context.Background(), map[string]any{
		"action":     "send",
		"session_id": "send1",
		"input":      "test input",
	})
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d; error: %s", result.ExitCode, result.Error)
	}

	// Kill
	tool.Execute(context.Background(), map[string]any{
		"action":     "kill",
		"session_id": "send1",
	})
}

func TestInteractiveShellToolReadAction(t *testing.T) {
	sm := NewSessionManager(5)
	defer sm.Cleanup()
	tool := NewInteractiveShellTool(sm)

	// Start echo which produces output and exits
	tool.Execute(context.Background(), map[string]any{
		"action":     "start",
		"session_id": "read1",
		"command":    "echo test_output",
	})

	time.Sleep(500 * time.Millisecond)

	result, err := tool.Execute(context.Background(), map[string]any{
		"action":     "read",
		"session_id": "read1",
	})
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if !strings.Contains(result.Output, "test_output") {
		t.Errorf("expected output containing 'test_output', got %q", result.Output)
	}
}

func TestInteractiveShellToolListAction(t *testing.T) {
	sm := NewSessionManager(5)
	defer sm.Cleanup()
	tool := NewInteractiveShellTool(sm)

	tool.Execute(context.Background(), map[string]any{
		"action":     "start",
		"session_id": "list1",
		"command":    "echo a",
	})

	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "list",
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if !strings.Contains(result.Output, "list1") {
		t.Errorf("expected output containing 'list1', got %q", result.Output)
	}
}

func TestInteractiveShellToolKillAction(t *testing.T) {
	sm := NewSessionManager(5)
	defer sm.Cleanup()
	tool := NewInteractiveShellTool(sm)

	tool.Execute(context.Background(), map[string]any{
		"action":     "start",
		"session_id": "kill1",
		"command":    "cat",
	})

	result, err := tool.Execute(context.Background(), map[string]any{
		"action":     "kill",
		"session_id": "kill1",
	})
	if err != nil {
		t.Fatalf("Kill failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d; error: %s", result.ExitCode, result.Error)
	}
}

func TestInteractiveShellToolMissingAction(t *testing.T) {
	sm := NewSessionManager(5)
	tool := NewInteractiveShellTool(sm)

	_, err := tool.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Error("expected error for missing action")
	}
}

func TestInteractiveShellToolInvalidAction(t *testing.T) {
	sm := NewSessionManager(5)
	tool := NewInteractiveShellTool(sm)

	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "invalid",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Error == "" {
		t.Error("expected error for invalid action")
	}
}

func TestInteractiveShellToolStartMissingCommand(t *testing.T) {
	sm := NewSessionManager(5)
	tool := NewInteractiveShellTool(sm)

	result, err := tool.Execute(context.Background(), map[string]any{
		"action":     "start",
		"session_id": "x",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Error == "" {
		t.Error("expected error for missing command")
	}
}

func TestInteractiveShellToolSendMissingSessionID(t *testing.T) {
	sm := NewSessionManager(5)
	tool := NewInteractiveShellTool(sm)

	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "send",
		"input":  "hello",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Error == "" {
		t.Error("expected error for missing session_id")
	}
}

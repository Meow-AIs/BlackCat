package builtin

import (
	"context"
	"runtime"
	"strings"
	"testing"

	"github.com/meowai/blackcat/internal/security"
)

func TestShellToolInfo(t *testing.T) {
	tool := NewShellTool(nil)
	if tool.Info().Name != "execute" {
		t.Errorf("expected name 'execute', got %q", tool.Info().Name)
	}
	if tool.Info().Category != "shell" {
		t.Errorf("expected category 'shell', got %q", tool.Info().Category)
	}
}

func TestShellToolExecute(t *testing.T) {
	tool := NewShellTool(nil)

	cmd := "echo blackcat"
	if runtime.GOOS == "windows" {
		cmd = "cmd /c echo blackcat"
	}

	result, err := tool.Execute(context.Background(), map[string]any{"command": cmd})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(result.Output, "blackcat") {
		t.Errorf("expected output containing 'blackcat', got %q", result.Output)
	}
}

func TestShellToolDeniedCommand(t *testing.T) {
	checker := security.NewPermissionChecker()
	tool := NewShellTool(checker)

	result, err := tool.Execute(context.Background(), map[string]any{"command": "rm -rf /"})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.ExitCode != -1 {
		t.Errorf("expected exit code -1 for denied command, got %d", result.ExitCode)
	}
	if result.Error == "" {
		t.Error("expected error message for denied command")
	}
}

func TestShellToolMissingArg(t *testing.T) {
	tool := NewShellTool(nil)
	_, err := tool.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Error("expected error for missing command arg")
	}
}

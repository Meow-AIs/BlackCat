package tui

import (
	"strings"
	"testing"
	"time"
)

func TestNewApp(t *testing.T) {
	app := NewApp(80, 24)
	if app == nil {
		t.Fatal("expected non-nil App")
	}
	if app.Width != 80 {
		t.Errorf("expected width 80, got %d", app.Width)
	}
	if app.Height != 24 {
		t.Errorf("expected height 24, got %d", app.Height)
	}
	if app.State != AppChat {
		t.Errorf("expected initial state AppChat, got %s", app.State)
	}
	if app.Chat == nil {
		t.Error("expected non-nil Chat")
	}
	if app.Input == nil {
		t.Error("expected non-nil Input")
	}
	if app.Toolbar == nil {
		t.Error("expected non-nil Toolbar")
	}
	if app.Permission != nil {
		t.Error("expected nil Permission prompt initially")
	}
	if app.DiffView != nil {
		t.Error("expected nil DiffView initially")
	}
}

func TestAppRenderChatState(t *testing.T) {
	app := NewApp(80, 24)
	app.Chat.AddMessage(ChatMessage{
		Role:    "user",
		Content: "hello",
	})
	output := app.Render()
	if !strings.Contains(output, "hello") {
		t.Error("expected render to contain chat message")
	}
}

func TestAppRenderPermissionState(t *testing.T) {
	app := NewApp(80, 24)
	app.Permission = NewPermissionPrompt("rm -rf /tmp", "high")
	app.State = AppPermission

	output := app.Render()
	if !strings.Contains(output, "rm -rf /tmp") {
		t.Error("expected render to contain permission prompt command")
	}
}

func TestAppRenderDiffState(t *testing.T) {
	app := NewApp(80, 24)
	app.DiffView = []DiffLine{
		{Type: DiffAdded, Content: "new code", LineNum: 1},
	}
	app.State = AppDiff

	output := app.Render()
	if !strings.Contains(output, "new code") {
		t.Error("expected render to contain diff content")
	}
}

func TestAppResize(t *testing.T) {
	app := NewApp(80, 24)
	app.Resize(120, 40)
	if app.Width != 120 {
		t.Errorf("expected width 120, got %d", app.Width)
	}
	if app.Height != 40 {
		t.Errorf("expected height 40, got %d", app.Height)
	}
}

func TestAppStateTransitions(t *testing.T) {
	app := NewApp(80, 24)

	// Chat -> Permission
	app.Permission = NewPermissionPrompt("cmd", "low")
	app.State = AppPermission
	if app.State != AppPermission {
		t.Error("expected AppPermission state")
	}

	// Permission -> Chat (after resolving)
	app.Permission.Confirm()
	app.Permission = nil
	app.State = AppChat
	if app.State != AppChat {
		t.Error("expected AppChat state")
	}

	// Chat -> Diff
	app.DiffView = []DiffLine{{Type: DiffAdded, Content: "x"}}
	app.State = AppDiff
	if app.State != AppDiff {
		t.Error("expected AppDiff state")
	}
}

func TestAppRenderIncludesToolbar(t *testing.T) {
	app := NewApp(80, 24)
	app.Toolbar.Update("claude-sonnet", "proj", "idle", 10, 0.5)

	output := app.Render()
	if !strings.Contains(output, "claude-sonnet") {
		t.Error("expected render to include toolbar model name")
	}
}

func TestAppRenderIncludesInput(t *testing.T) {
	app := NewApp(80, 24)
	app.Input.Insert('t')
	app.Input.Insert('e')
	app.Input.Insert('s')
	app.Input.Insert('t')

	output := app.Render()
	if !strings.Contains(output, "test") {
		t.Error("expected render to include input value")
	}
}

func TestAppRenderWithMessages(t *testing.T) {
	app := NewApp(80, 24)
	app.Chat.AddMessage(ChatMessage{
		Role:      "user",
		Content:   "What is Go?",
		Timestamp: time.Now(),
	})
	app.Chat.AddMessage(ChatMessage{
		Role:      "assistant",
		Content:   "Go is a programming language.",
		Timestamp: time.Now(),
	})

	output := app.Render()
	if !strings.Contains(output, "What is Go?") {
		t.Error("expected user message in render")
	}
	if !strings.Contains(output, "Go is a programming language.") {
		t.Error("expected assistant message in render")
	}
}

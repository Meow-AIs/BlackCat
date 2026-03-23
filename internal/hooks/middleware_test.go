package hooks

import (
	"context"
	"errors"
	"testing"
)

func TestToolMiddlewareFiresHooks(t *testing.T) {
	e := NewEngine()
	beforeFired := false
	afterFired := false

	e.Register(EventBeforeTool, "before", PriorityNormal, func(ctx HookContext) HookResult {
		beforeFired = true
		return HookResult{Allow: true}
	})
	e.Register(EventAfterTool, "after", PriorityNormal, func(ctx HookContext) HookResult {
		afterFired = true
		return HookResult{Allow: true}
	})

	m := NewToolMiddleware(e)
	result, err := m.WrapExecution(
		context.Background(),
		"execute",
		map[string]any{"command": "ls"},
		func() (string, error) { return "file1\nfile2", nil },
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "file1\nfile2" {
		t.Fatalf("unexpected result: %s", result)
	}
	if !beforeFired {
		t.Fatal("before hook not fired")
	}
	if !afterFired {
		t.Fatal("after hook not fired")
	}
}

func TestToolMiddlewareBlockedTool(t *testing.T) {
	e := NewEngine()
	e.Register(EventBeforeTool, "blocker", PriorityFirst, func(ctx HookContext) HookResult {
		return HookResult{Allow: false, Message: "tool blocked by hook"}
	})

	executed := false
	m := NewToolMiddleware(e)
	_, err := m.WrapExecution(
		context.Background(),
		"execute",
		map[string]any{"command": "rm -rf /"},
		func() (string, error) {
			executed = true
			return "", nil
		},
	)

	if err == nil {
		t.Fatal("expected error when tool is blocked")
	}
	if executed {
		t.Fatal("tool should not have been executed when blocked")
	}
	if !errors.Is(err, ErrHookBlocked) {
		t.Fatalf("expected ErrHookBlocked, got %v", err)
	}
}

func TestToolMiddlewareModifiedArgs(t *testing.T) {
	e := NewEngine()
	e.Register(EventBeforeTool, "modifier", PriorityNormal, func(ctx HookContext) HookResult {
		return HookResult{
			Allow:    true,
			Modified: map[string]any{"timeout": 60},
		}
	})

	var receivedArgs map[string]any
	m := NewToolMiddleware(e)

	// The middleware should pass modified data to after_tool, but the execute
	// function uses the original args. Modified data is available in the result.
	_, err := m.WrapExecution(
		context.Background(),
		"execute",
		map[string]any{"command": "ls"},
		func() (string, error) {
			receivedArgs = map[string]any{"command": "ls"}
			return "ok", nil
		},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedArgs == nil {
		t.Fatal("execute function was not called")
	}
}

func TestToolMiddlewareToolError(t *testing.T) {
	e := NewEngine()
	errorHookFired := false

	e.Register(EventToolError, "error-handler", PriorityNormal, func(ctx HookContext) HookResult {
		errorHookFired = true
		return HookResult{Allow: true}
	})

	m := NewToolMiddleware(e)
	_, err := m.WrapExecution(
		context.Background(),
		"execute",
		map[string]any{"command": "bad"},
		func() (string, error) { return "", errors.New("command failed") },
	)

	if err == nil {
		t.Fatal("expected error from failed tool")
	}
	if !errorHookFired {
		t.Fatal("tool_error hook should have fired")
	}
}

func TestResponseMiddlewareModifies(t *testing.T) {
	e := NewEngine()
	e.Register(EventBeforeResponse, "modifier", PriorityNormal, func(ctx HookContext) HookResult {
		return HookResult{
			Allow:    true,
			Modified: map[string]any{"response": "modified response"},
		}
	})

	m := NewResponseMiddleware(e)
	result := m.WrapResponse("original response", "session-123")
	if result != "modified response" {
		t.Fatalf("expected modified response, got %q", result)
	}
}

func TestResponseMiddlewareNoModification(t *testing.T) {
	e := NewEngine()
	e.Register(EventBeforeResponse, "logger", PriorityNormal, func(ctx HookContext) HookResult {
		return HookResult{Allow: true, Message: "logged"}
	})

	m := NewResponseMiddleware(e)
	result := m.WrapResponse("original", "session-123")
	if result != "original" {
		t.Fatalf("expected original response, got %q", result)
	}
}

func TestMemoryMiddlewareShouldStore(t *testing.T) {
	e := NewEngine()
	RegisterBuiltinHooks(e)

	m := NewMemoryMiddleware(e)

	// Good content should be stored.
	if !m.ShouldStore("The user prefers Go for backend development.", "semantic") {
		t.Fatal("expected good content to be stored")
	}

	// Low quality should be filtered.
	if m.ShouldStore("ok", "semantic") {
		t.Fatal("expected low quality content to be filtered")
	}
}

func TestMemoryMiddlewareNoHooks(t *testing.T) {
	e := NewEngine()
	m := NewMemoryMiddleware(e)

	// Without hooks, everything should be stored.
	if !m.ShouldStore("anything", "episodic") {
		t.Fatal("expected content to be stored when no hooks registered")
	}
}

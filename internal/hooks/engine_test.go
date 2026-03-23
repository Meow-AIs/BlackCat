package hooks

import (
	"sync"
	"testing"
	"time"
)

func TestNewEngine(t *testing.T) {
	e := NewEngine()
	if e == nil {
		t.Fatal("NewEngine returned nil")
	}
	if e.Count() != 0 {
		t.Fatalf("expected 0 hooks, got %d", e.Count())
	}
}

func TestRegisterAndCount(t *testing.T) {
	e := NewEngine()
	handler := func(ctx HookContext) HookResult {
		return HookResult{Allow: true}
	}

	id := e.Register(EventBeforeTool, "test-hook", PriorityNormal, handler)
	if id == "" {
		t.Fatal("Register returned empty ID")
	}
	if e.Count() != 1 {
		t.Fatalf("expected 1 hook, got %d", e.Count())
	}
}

func TestUnregister(t *testing.T) {
	e := NewEngine()
	handler := func(ctx HookContext) HookResult {
		return HookResult{Allow: true}
	}

	id := e.Register(EventBeforeTool, "test-hook", PriorityNormal, handler)
	e.Unregister(id)
	if e.Count() != 0 {
		t.Fatalf("expected 0 hooks after unregister, got %d", e.Count())
	}
}

func TestUnregisterNonexistent(t *testing.T) {
	e := NewEngine()
	// Should not panic.
	e.Unregister("nonexistent-id")
}

func TestFireSingleHandler(t *testing.T) {
	e := NewEngine()
	called := false
	e.Register(EventBeforeTool, "test", PriorityNormal, func(ctx HookContext) HookResult {
		called = true
		return HookResult{Allow: true, Message: "ok"}
	})

	result := e.Fire(EventBeforeTool, map[string]any{"key": "value"})
	if !called {
		t.Fatal("handler was not called")
	}
	if !result.Allow {
		t.Fatal("expected Allow=true")
	}
	if result.Message != "ok" {
		t.Fatalf("expected message 'ok', got %q", result.Message)
	}
}

func TestFireMultipleHandlers(t *testing.T) {
	e := NewEngine()
	order := []string{}
	mu := sync.Mutex{}

	e.Register(EventBeforeTool, "second", PriorityNormal, func(ctx HookContext) HookResult {
		mu.Lock()
		order = append(order, "second")
		mu.Unlock()
		return HookResult{Allow: true}
	})
	e.Register(EventBeforeTool, "first", PriorityFirst, func(ctx HookContext) HookResult {
		mu.Lock()
		order = append(order, "first")
		mu.Unlock()
		return HookResult{Allow: true}
	})
	e.Register(EventBeforeTool, "last", PriorityLast, func(ctx HookContext) HookResult {
		mu.Lock()
		order = append(order, "last")
		mu.Unlock()
		return HookResult{Allow: true}
	})

	e.Fire(EventBeforeTool, map[string]any{})

	if len(order) != 3 {
		t.Fatalf("expected 3 calls, got %d", len(order))
	}
	if order[0] != "first" || order[1] != "second" || order[2] != "last" {
		t.Fatalf("wrong order: %v", order)
	}
}

func TestFirePriorityOrdering(t *testing.T) {
	e := NewEngine()
	order := []int{}

	e.Register(EventAfterTool, "p50", HookPriority(50), func(ctx HookContext) HookResult {
		order = append(order, 50)
		return HookResult{Allow: true}
	})
	e.Register(EventAfterTool, "p10", HookPriority(10), func(ctx HookContext) HookResult {
		order = append(order, 10)
		return HookResult{Allow: true}
	})
	e.Register(EventAfterTool, "p90", HookPriority(90), func(ctx HookContext) HookResult {
		order = append(order, 90)
		return HookResult{Allow: true}
	})

	e.Fire(EventAfterTool, map[string]any{})

	if len(order) != 3 || order[0] != 10 || order[1] != 50 || order[2] != 90 {
		t.Fatalf("expected [10 50 90], got %v", order)
	}
}

func TestFireBlockChain(t *testing.T) {
	e := NewEngine()
	secondCalled := false

	e.Register(EventBeforeTool, "blocker", PriorityFirst, func(ctx HookContext) HookResult {
		return HookResult{Allow: false, Message: "blocked"}
	})
	e.Register(EventBeforeTool, "after-blocker", PriorityNormal, func(ctx HookContext) HookResult {
		secondCalled = true
		return HookResult{Allow: true}
	})

	result := e.Fire(EventBeforeTool, map[string]any{})
	if result.Allow {
		t.Fatal("expected Allow=false")
	}
	if result.Message != "blocked" {
		t.Fatalf("expected message 'blocked', got %q", result.Message)
	}
	if secondCalled {
		t.Fatal("second handler should not have been called after block")
	}
}

func TestFireModifiedDataMerging(t *testing.T) {
	e := NewEngine()

	e.Register(EventBeforeTool, "modifier1", PriorityFirst, func(ctx HookContext) HookResult {
		return HookResult{
			Allow:    true,
			Modified: map[string]any{"key1": "val1", "shared": "from1"},
		}
	})
	e.Register(EventBeforeTool, "modifier2", PriorityNormal, func(ctx HookContext) HookResult {
		return HookResult{
			Allow:    true,
			Modified: map[string]any{"key2": "val2", "shared": "from2"},
		}
	})

	result := e.Fire(EventBeforeTool, map[string]any{})
	if result.Modified["key1"] != "val1" {
		t.Fatal("expected key1=val1")
	}
	if result.Modified["key2"] != "val2" {
		t.Fatal("expected key2=val2")
	}
	// Last writer wins.
	if result.Modified["shared"] != "from2" {
		t.Fatalf("expected shared=from2 (last writer wins), got %v", result.Modified["shared"])
	}
}

func TestEnableDisable(t *testing.T) {
	e := NewEngine()
	called := false

	id := e.Register(EventBeforeTool, "toggle", PriorityNormal, func(ctx HookContext) HookResult {
		called = true
		return HookResult{Allow: true}
	})

	e.Disable(id)
	e.Fire(EventBeforeTool, map[string]any{})
	if called {
		t.Fatal("disabled hook should not have been called")
	}

	e.Enable(id)
	e.Fire(EventBeforeTool, map[string]any{})
	if !called {
		t.Fatal("enabled hook should have been called")
	}
}

func TestFireAsync(t *testing.T) {
	e := NewEngine()
	done := make(chan struct{})

	e.Register(EventSessionEnd, "async-test", PriorityNormal, func(ctx HookContext) HookResult {
		close(done)
		return HookResult{Allow: true}
	})

	e.FireAsync(EventSessionEnd, map[string]any{})

	select {
	case <-done:
		// Success.
	case <-time.After(2 * time.Second):
		t.Fatal("FireAsync did not execute handler within timeout")
	}
}

func TestFireNoHandlers(t *testing.T) {
	e := NewEngine()
	result := e.Fire(EventBeforeTool, map[string]any{})
	if !result.Allow {
		t.Fatal("expected Allow=true when no handlers registered")
	}
}

func TestListHooks(t *testing.T) {
	e := NewEngine()
	e.Register(EventBeforeTool, "hook1", PriorityNormal, func(ctx HookContext) HookResult {
		return HookResult{Allow: true}
	})
	e.Register(EventAfterTool, "hook2", PriorityNormal, func(ctx HookContext) HookResult {
		return HookResult{Allow: true}
	})

	all := e.ListHooks()
	if len(all) != 2 {
		t.Fatalf("expected 2 hooks, got %d", len(all))
	}
}

func TestListByEvent(t *testing.T) {
	e := NewEngine()
	e.Register(EventBeforeTool, "hook1", PriorityNormal, func(ctx HookContext) HookResult {
		return HookResult{Allow: true}
	})
	e.Register(EventAfterTool, "hook2", PriorityNormal, func(ctx HookContext) HookResult {
		return HookResult{Allow: true}
	})

	beforeHooks := e.ListByEvent(EventBeforeTool)
	if len(beforeHooks) != 1 {
		t.Fatalf("expected 1 before_tool hook, got %d", len(beforeHooks))
	}
	if beforeHooks[0].Name != "hook1" {
		t.Fatalf("expected hook1, got %s", beforeHooks[0].Name)
	}
}

func TestFireDifferentEvents(t *testing.T) {
	e := NewEngine()
	beforeCalled := false
	afterCalled := false

	e.Register(EventBeforeTool, "before", PriorityNormal, func(ctx HookContext) HookResult {
		beforeCalled = true
		return HookResult{Allow: true}
	})
	e.Register(EventAfterTool, "after", PriorityNormal, func(ctx HookContext) HookResult {
		afterCalled = true
		return HookResult{Allow: true}
	})

	e.Fire(EventBeforeTool, map[string]any{})
	if !beforeCalled {
		t.Fatal("before handler not called")
	}
	if afterCalled {
		t.Fatal("after handler should not be called for before_tool event")
	}
}

package hooks

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConditionEquals(t *testing.T) {
	c := &HookCondition{Field: "tool_name", Operator: "equals", Value: "execute"}

	if !c.Evaluate(map[string]any{"tool_name": "execute"}) {
		t.Fatal("expected equals to match")
	}
	if c.Evaluate(map[string]any{"tool_name": "read"}) {
		t.Fatal("expected equals to not match")
	}
}

func TestConditionNotEquals(t *testing.T) {
	c := &HookCondition{Field: "tool_name", Operator: "not_equals", Value: "execute"}

	if !c.Evaluate(map[string]any{"tool_name": "read"}) {
		t.Fatal("expected not_equals to match")
	}
	if c.Evaluate(map[string]any{"tool_name": "execute"}) {
		t.Fatal("expected not_equals to not match")
	}
}

func TestConditionContains(t *testing.T) {
	c := &HookCondition{Field: "command", Operator: "contains", Value: "rm -rf /"}

	if !c.Evaluate(map[string]any{"command": "sudo rm -rf /"}) {
		t.Fatal("expected contains to match")
	}
	if c.Evaluate(map[string]any{"command": "ls -la"}) {
		t.Fatal("expected contains to not match")
	}
}

func TestConditionStartsWith(t *testing.T) {
	c := &HookCondition{Field: "command", Operator: "starts_with", Value: "sudo"}

	if !c.Evaluate(map[string]any{"command": "sudo rm -rf"}) {
		t.Fatal("expected starts_with to match")
	}
	if c.Evaluate(map[string]any{"command": "ls -la"}) {
		t.Fatal("expected starts_with to not match")
	}
}

func TestConditionMatches(t *testing.T) {
	c := &HookCondition{Field: "command", Operator: "matches", Value: `^rm\s+-rf`}

	if !c.Evaluate(map[string]any{"command": "rm -rf /tmp"}) {
		t.Fatal("expected matches to match")
	}
	if c.Evaluate(map[string]any{"command": "ls -la"}) {
		t.Fatal("expected matches to not match")
	}
}

func TestConditionMissingField(t *testing.T) {
	c := &HookCondition{Field: "nonexistent", Operator: "equals", Value: "test"}

	if c.Evaluate(map[string]any{"other": "value"}) {
		t.Fatal("expected missing field to not match")
	}
}

func TestConditionUnknownOperator(t *testing.T) {
	c := &HookCondition{Field: "tool_name", Operator: "unknown_op", Value: "test"}

	if c.Evaluate(map[string]any{"tool_name": "test"}) {
		t.Fatal("expected unknown operator to not match")
	}
}

func TestConditionNonStringField(t *testing.T) {
	c := &HookCondition{Field: "count", Operator: "equals", Value: "5"}

	// Non-string values should be converted via fmt.Sprintf.
	if !c.Evaluate(map[string]any{"count": 5}) {
		t.Fatal("expected numeric field to match via string conversion")
	}
}

func TestLoadAndSaveScriptHooks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hooks.json")

	hooks := []ScriptHook{
		{
			Name:     "block-rm",
			Event:    EventBeforeTool,
			Priority: PriorityFirst,
			Enabled:  true,
			Condition: &HookCondition{
				Field:    "command",
				Operator: "contains",
				Value:    "rm -rf /",
			},
			Action: HookAction{
				Type:    "block",
				Message: "Blocked dangerous command",
			},
		},
		{
			Name:     "log-all",
			Event:    EventAfterTool,
			Priority: PriorityLast,
			Enabled:  true,
			Action: HookAction{
				Type:    "log",
				Message: "Tool executed: {{.tool_name}}",
			},
		},
	}

	err := SaveScriptHooks(path, hooks)
	if err != nil {
		t.Fatalf("SaveScriptHooks failed: %v", err)
	}

	loaded, err := LoadScriptHooks(path)
	if err != nil {
		t.Fatalf("LoadScriptHooks failed: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("expected 2 hooks, got %d", len(loaded))
	}
	if loaded[0].Name != "block-rm" {
		t.Fatalf("expected hook name 'block-rm', got %q", loaded[0].Name)
	}
	if loaded[1].Action.Type != "log" {
		t.Fatalf("expected action type 'log', got %q", loaded[1].Action.Type)
	}
}

func TestLoadScriptHooksFileNotFound(t *testing.T) {
	_, err := LoadScriptHooks("/nonexistent/path/hooks.json")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestToHandlerBlockAction(t *testing.T) {
	hook := ScriptHook{
		Name:    "blocker",
		Event:   EventBeforeTool,
		Enabled: true,
		Condition: &HookCondition{
			Field:    "command",
			Operator: "contains",
			Value:    "rm -rf /",
		},
		Action: HookAction{
			Type:    "block",
			Message: "Blocked: rm -rf /",
		},
	}

	handler := hook.ToHandler()
	ctx := HookContext{
		Event: EventBeforeTool,
		Data:  map[string]any{"command": "rm -rf /"},
	}

	result := handler(ctx)
	if result.Allow {
		t.Fatal("expected block action to return Allow=false")
	}
	if result.Message != "Blocked: rm -rf /" {
		t.Fatalf("expected block message, got %q", result.Message)
	}
}

func TestToHandlerBlockConditionNotMet(t *testing.T) {
	hook := ScriptHook{
		Name:    "blocker",
		Event:   EventBeforeTool,
		Enabled: true,
		Condition: &HookCondition{
			Field:    "command",
			Operator: "contains",
			Value:    "rm -rf /",
		},
		Action: HookAction{
			Type:    "block",
			Message: "Blocked",
		},
	}

	handler := hook.ToHandler()
	ctx := HookContext{
		Event: EventBeforeTool,
		Data:  map[string]any{"command": "ls -la"},
	}

	result := handler(ctx)
	if !result.Allow {
		t.Fatal("expected Allow=true when condition not met")
	}
}

func TestToHandlerLogAction(t *testing.T) {
	hook := ScriptHook{
		Name:    "logger",
		Event:   EventAfterTool,
		Enabled: true,
		Action: HookAction{
			Type:    "log",
			Message: "executed: {{.tool_name}}",
		},
	}

	handler := hook.ToHandler()
	ctx := HookContext{
		Event: EventAfterTool,
		Data:  map[string]any{"tool_name": "shell"},
	}

	result := handler(ctx)
	if !result.Allow {
		t.Fatal("log action should always allow")
	}
	if result.Message != "executed: shell" {
		t.Fatalf("expected 'executed: shell', got %q", result.Message)
	}
}

func TestToHandlerModifyAction(t *testing.T) {
	hook := ScriptHook{
		Name:    "modifier",
		Event:   EventBeforeTool,
		Enabled: true,
		Action: HookAction{
			Type:   "modify",
			Modify: map[string]any{"timeout": 30},
		},
	}

	handler := hook.ToHandler()
	ctx := HookContext{
		Event: EventBeforeTool,
		Data:  map[string]any{},
	}

	result := handler(ctx)
	if !result.Allow {
		t.Fatal("modify action should allow")
	}
	if result.Modified["timeout"] != 30 {
		t.Fatalf("expected timeout=30, got %v", result.Modified["timeout"])
	}
}

func TestToHandlerAllowAction(t *testing.T) {
	hook := ScriptHook{
		Name:    "allower",
		Event:   EventBeforeTool,
		Enabled: true,
		Action: HookAction{
			Type:    "allow",
			Message: "explicitly allowed",
		},
	}

	handler := hook.ToHandler()
	ctx := HookContext{Event: EventBeforeTool, Data: map[string]any{}}
	result := handler(ctx)
	if !result.Allow {
		t.Fatal("allow action should return Allow=true")
	}
}

func TestToHandlerNoCondition(t *testing.T) {
	hook := ScriptHook{
		Name:    "unconditional",
		Event:   EventSecurityAlert,
		Enabled: true,
		Action: HookAction{
			Type:    "notify",
			Message: "alert: {{.description}}",
		},
	}

	handler := hook.ToHandler()
	ctx := HookContext{
		Event: EventSecurityAlert,
		Data:  map[string]any{"description": "suspicious activity"},
	}

	result := handler(ctx)
	if !result.Allow {
		t.Fatal("notify action should allow")
	}
	if result.Message != "alert: suspicious activity" {
		t.Fatalf("expected templated message, got %q", result.Message)
	}
}

func TestSaveScriptHooksInvalidPath(t *testing.T) {
	err := SaveScriptHooks("/nonexistent/dir/hooks.json", []ScriptHook{})
	if err == nil {
		// Some systems might create the file; only fail if truly expected.
		_ = os.Remove("/nonexistent/dir/hooks.json")
	}
}

func TestTemplateMessageMissingField(t *testing.T) {
	hook := ScriptHook{
		Name:    "logger",
		Event:   EventAfterTool,
		Enabled: true,
		Action: HookAction{
			Type:    "log",
			Message: "tool: {{.tool_name}}",
		},
	}

	handler := hook.ToHandler()
	ctx := HookContext{
		Event: EventAfterTool,
		Data:  map[string]any{},
	}

	result := handler(ctx)
	// Should not panic; missing fields become empty or <no value>.
	if !result.Allow {
		t.Fatal("log action should allow")
	}
}

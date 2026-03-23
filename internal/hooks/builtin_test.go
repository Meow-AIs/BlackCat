package hooks

import (
	"strings"
	"testing"
)

func TestRegisterBuiltinHooks(t *testing.T) {
	e := NewEngine()
	RegisterBuiltinHooks(e)

	if e.Count() < 6 {
		t.Fatalf("expected at least 6 builtin hooks, got %d", e.Count())
	}
}

func TestSafetyGuardBlocksDangerousCommands(t *testing.T) {
	e := NewEngine()
	RegisterBuiltinHooks(e)

	dangerous := []string{
		"rm -rf /",
		"rm -rf ~",
		"mkfs /dev/sda",
		"dd if=/dev/zero of=/dev/sda",
	}

	for _, cmd := range dangerous {
		result := e.Fire(EventBeforeTool, map[string]any{
			"tool_name": "execute",
			"command":   cmd,
		})
		if result.Allow {
			t.Fatalf("safety guard should block: %s", cmd)
		}
	}
}

func TestSafetyGuardAllowsSafeCommands(t *testing.T) {
	e := NewEngine()
	RegisterBuiltinHooks(e)

	safe := []string{
		"ls -la",
		"cat /etc/hosts",
		"go test ./...",
		"rm -rf /tmp/test",
	}

	for _, cmd := range safe {
		result := e.Fire(EventBeforeTool, map[string]any{
			"tool_name": "execute",
			"command":   cmd,
		})
		if !result.Allow {
			t.Fatalf("safety guard should allow: %s", cmd)
		}
	}
}

func TestOutputSanitizerRedactsSecrets(t *testing.T) {
	e := NewEngine()
	RegisterBuiltinHooks(e)

	tests := []struct {
		name   string
		output string
		want   string // substring that should appear
		deny   string // substring that should NOT appear
	}{
		{
			name:   "api key",
			output: "API_KEY=sk-abc123secret456",
			deny:   "sk-abc123secret456",
			want:   "[REDACTED]",
		},
		{
			name:   "bearer token",
			output: "Authorization: Bearer eyJhbGciOiJIUz.long.token",
			deny:   "eyJhbGciOiJIUz",
			want:   "[REDACTED]",
		},
		{
			name:   "password in env",
			output: "PASSWORD=mysecretpassword123",
			deny:   "mysecretpassword123",
			want:   "[REDACTED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.Fire(EventAfterTool, map[string]any{
				"output": tt.output,
			})
			if !result.Allow {
				t.Fatal("sanitizer should allow, not block")
			}
			if result.Modified == nil {
				t.Fatal("expected modified output")
			}
			out, ok := result.Modified["output"].(string)
			if !ok {
				t.Fatal("modified output should be string")
			}
			if strings.Contains(out, tt.deny) {
				t.Fatalf("output still contains secret: %s", out)
			}
			if !strings.Contains(out, tt.want) {
				t.Fatalf("output missing redaction marker: %s", out)
			}
		})
	}
}

func TestOutputSanitizerCleanOutput(t *testing.T) {
	e := NewEngine()
	RegisterBuiltinHooks(e)

	result := e.Fire(EventAfterTool, map[string]any{
		"output": "Hello, world!",
	})
	// Clean output should not be modified.
	if result.Modified != nil {
		out, ok := result.Modified["output"].(string)
		if ok && out != "Hello, world!" {
			t.Fatalf("clean output should not be modified, got %q", out)
		}
	}
}

func TestCostWarning(t *testing.T) {
	e := NewEngine()
	RegisterBuiltinHooks(e)

	// High cost should produce a message.
	result := e.Fire(EventAfterResponse, map[string]any{
		"cost": 5.0,
	})
	if result.Message == "" {
		t.Fatal("expected cost warning message for high cost")
	}

	// Low cost should not produce a warning.
	result = e.Fire(EventAfterResponse, map[string]any{
		"cost": 0.01,
	})
	if result.Message != "" {
		t.Fatalf("expected no warning for low cost, got %q", result.Message)
	}
}

func TestMemoryQualityFilter(t *testing.T) {
	e := NewEngine()
	RegisterBuiltinHooks(e)

	lowQuality := []string{
		"ok",
		"thanks",
		"yes",
		"short",
	}

	for _, content := range lowQuality {
		result := e.Fire(EventMemoryStore, map[string]any{
			"content": content,
		})
		if result.Allow {
			t.Fatalf("memory quality should block low-quality entry: %q", content)
		}
	}

	// Good quality entry should pass.
	result := e.Fire(EventMemoryStore, map[string]any{
		"content": "The user prefers TypeScript for frontend development and Go for backend services.",
	})
	if !result.Allow {
		t.Fatal("memory quality should allow good content")
	}
}

func TestPermissionAudit(t *testing.T) {
	e := NewEngine()
	RegisterBuiltinHooks(e)

	result := e.Fire(EventPermissionAsk, map[string]any{
		"tool":   "execute",
		"action": "run shell command",
	})
	if !result.Allow {
		t.Fatal("permission audit should not block, only log")
	}
	if result.Message == "" {
		t.Fatal("permission audit should produce a log message")
	}
}

func TestSessionLogger(t *testing.T) {
	e := NewEngine()
	RegisterBuiltinHooks(e)

	result := e.Fire(EventSessionEnd, map[string]any{
		"tool_count": 5,
		"duration":   "2m30s",
		"errors":     0,
	})
	if !result.Allow {
		t.Fatal("session logger should not block")
	}
	if result.Message == "" {
		t.Fatal("session logger should produce a summary message")
	}
}

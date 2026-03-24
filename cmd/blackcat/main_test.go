package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

// captureOutput captures stdout during function execution.
func captureOutput(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestRunNoArgs(t *testing.T) {
	out := captureOutput(func() {
		code := run([]string{"blackcat"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})
	if !strings.Contains(out, "BlackCat") {
		t.Error("expected interactive banner")
	}
}

func TestRunVersion(t *testing.T) {
	out := captureOutput(func() {
		code := run([]string{"blackcat", "version"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})
	if !strings.Contains(out, "BlackCat v") {
		t.Errorf("expected version output, got %q", out)
	}
	if !strings.Contains(out, "commit:") {
		t.Error("expected commit info in version output")
	}
}

func TestRunVersionFlags(t *testing.T) {
	for _, flag := range []string{"--version", "-v"} {
		t.Run(flag, func(t *testing.T) {
			out := captureOutput(func() {
				run([]string{"blackcat", flag})
			})
			if !strings.Contains(out, "BlackCat v") {
				t.Errorf("expected version output for %s", flag)
			}
		})
	}
}

func TestRunHelp(t *testing.T) {
	for _, flag := range []string{"help", "--help", "-h"} {
		t.Run(flag, func(t *testing.T) {
			out := captureOutput(func() {
				code := run([]string{"blackcat", flag})
				if code != 0 {
					t.Errorf("expected exit code 0, got %d", code)
				}
			})
			if !strings.Contains(out, "Usage:") {
				t.Error("expected usage info in help")
			}
			if !strings.Contains(out, "Commands:") {
				t.Error("expected commands list in help")
			}
		})
	}
}

func TestRunOneShot(t *testing.T) {
	out := captureOutput(func() {
		code := run([]string{"blackcat", "do", "something"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})
	if !strings.Contains(out, "Processing: do something") {
		t.Errorf("expected one-shot processing, got %q", out)
	}
}

func TestRunServe(t *testing.T) {
	out := captureOutput(func() {
		code := run([]string{"blackcat", "serve"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})
	if !strings.Contains(out, "Starting BlackCat gateway") {
		t.Error("expected serve output")
	}
}

func TestRunDoctor(t *testing.T) {
	out := captureOutput(func() {
		run([]string{"blackcat", "doctor"})
	})
	if !strings.Contains(out, "System Health Check") {
		t.Error("expected doctor output")
	}
	if !strings.Contains(out, "Go runtime:") {
		t.Error("expected Go runtime check")
	}
	if !strings.Contains(out, "Platform:") {
		t.Error("expected platform info")
	}
}

func TestRunMemoryStats(t *testing.T) {
	out := captureOutput(func() {
		code := run([]string{"blackcat", "memory", "stats"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})
	if !strings.Contains(out, "Memory Statistics") {
		t.Error("expected memory stats output")
	}
}

func TestRunMemorySearch(t *testing.T) {
	out := captureOutput(func() {
		code := run([]string{"blackcat", "memory", "search", "test query"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})
	if !strings.Contains(out, "Searching memory for: test query") {
		t.Error("expected memory search output")
	}
}

func TestRunMemoryList(t *testing.T) {
	out := captureOutput(func() {
		code := run([]string{"blackcat", "memory", "list"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})
	if !strings.Contains(out, "Memory entries") {
		t.Error("expected memory list output")
	}
}

func TestRunMemoryNoSubcommand(t *testing.T) {
	out := captureOutput(func() {
		code := run([]string{"blackcat", "memory"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
	})
	if !strings.Contains(out, "Usage:") {
		t.Error("expected usage hint")
	}
}

func TestRunScheduleList(t *testing.T) {
	out := captureOutput(func() {
		code := run([]string{"blackcat", "schedule", "list"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})
	if !strings.Contains(out, "Schedules:") {
		t.Error("expected schedule list output")
	}
}

func TestRunScheduleAdd(t *testing.T) {
	out := captureOutput(func() {
		code := run([]string{"blackcat", "schedule", "add", "*/5 * * * *", "check status"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})
	if !strings.Contains(out, "Added schedule") {
		t.Error("expected schedule add confirmation")
	}
}

func TestRunScheduleRemove(t *testing.T) {
	out := captureOutput(func() {
		code := run([]string{"blackcat", "schedule", "remove", "abc123"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})
	if !strings.Contains(out, "Removed schedule") {
		t.Error("expected schedule remove confirmation")
	}
}

func TestRunSkillsList(t *testing.T) {
	out := captureOutput(func() {
		code := run([]string{"blackcat", "skills", "list"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})
	if !strings.Contains(out, "Skills:") {
		t.Error("expected skills list output")
	}
}

func TestRunSkillsShow(t *testing.T) {
	out := captureOutput(func() {
		code := run([]string{"blackcat", "skills", "show", "web-search"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})
	if !strings.Contains(out, "Skill: web-search") {
		t.Error("expected skill detail output")
	}
}

func TestRunMCPList(t *testing.T) {
	out := captureOutput(func() {
		code := run([]string{"blackcat", "mcp", "list"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})
	if !strings.Contains(out, "MCP servers:") {
		t.Error("expected MCP list output")
	}
}

func TestRunMCPAdd(t *testing.T) {
	out := captureOutput(func() {
		code := run([]string{"blackcat", "mcp", "add", "myserver", "npx", "my-mcp-server"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})
	if !strings.Contains(out, "Added MCP server: myserver") {
		t.Error("expected MCP add confirmation")
	}
}

func TestRunMCPRemove(t *testing.T) {
	out := captureOutput(func() {
		code := run([]string{"blackcat", "mcp", "remove", "myserver"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})
	if !strings.Contains(out, "Removed MCP server") {
		t.Error("expected MCP remove confirmation")
	}
}

func TestRunConfigNoSubcommand(t *testing.T) {
	out := captureOutput(func() {
		code := run([]string{"blackcat", "config"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
	})
	if !strings.Contains(out, "Usage:") {
		t.Error("expected usage hint")
	}
}

func TestRunConfigSet(t *testing.T) {
	out := captureOutput(func() {
		code := run([]string{"blackcat", "config", "set", "provider=anthropic"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})
	if !strings.Contains(out, "Set provider = anthropic") {
		t.Error("expected config set confirmation")
	}
}

func TestRunInit(t *testing.T) {
	// Use a temp dir to avoid modifying real home
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	out := captureOutput(func() {
		code := run([]string{"blackcat", "init"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})
	if !strings.Contains(out, "Initialized BlackCat") {
		t.Errorf("expected init output, got %q", out)
	}

	// Run again - should say already exists
	out = captureOutput(func() {
		code := run([]string{"blackcat", "init"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})
	if !strings.Contains(out, "already exists") {
		t.Error("expected 'already exists' on second init")
	}
}

func TestDefaultConfigYAML(t *testing.T) {
	cfg := defaultConfigYAML()
	if !strings.Contains(cfg, "provider:") {
		t.Error("expected provider key in default config")
	}
	if !strings.Contains(cfg, "memory:") {
		t.Error("expected memory key in default config")
	}
	if !strings.Contains(cfg, "security:") {
		t.Error("expected security key in default config")
	}
}

func TestSubcommandErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"memory search no query", []string{"blackcat", "memory", "search"}},
		{"schedule add missing args", []string{"blackcat", "schedule", "add"}},
		{"schedule remove no id", []string{"blackcat", "schedule", "remove"}},
		{"skills show no name", []string{"blackcat", "skills", "show"}},
		{"mcp add missing args", []string{"blackcat", "mcp", "add"}},
		{"mcp remove no name", []string{"blackcat", "mcp", "remove"}},
		{"config set no value", []string{"blackcat", "config", "set"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = captureOutput(func() {
				code := run(tt.args)
				if code != 1 {
					t.Errorf("expected exit code 1 for %s, got %d", tt.name, code)
				}
			})
		})
	}
}

func TestVersionVariables(t *testing.T) {
	// Verify the package-level vars exist and have defaults
	if version == "" {
		t.Error("version should not be empty")
	}
	if commit == "" {
		t.Error("commit should not be empty")
	}
	_ = fmt.Sprintf("v%s (%s)", version, commit)
}

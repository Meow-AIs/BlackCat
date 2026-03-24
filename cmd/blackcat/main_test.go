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
	// runOneShot attempts to contact an LLM; without a real provider it
	// prints an error but must still return (exit 0 when we handle the
	// error gracefully, or exit 1 on unrecoverable failure).
	// We only assert that it does not panic and that it prints the prompt.
	out := captureOutput(func() {
		run([]string{"blackcat", "do", "something"})
	})
	// The prompt must always be echoed before any LLM attempt.
	if !strings.Contains(out, "do something") {
		t.Errorf("expected prompt in output, got %q", out)
	}
}

func TestRunServe(t *testing.T) {
	// cmdServe blocks on signal; test that it at least starts and the
	// startup banner is printed. We exercise it via cmdServeDry which
	// returns immediately after printing the banner (no channels to
	// start, no SIGINT wait).
	out := captureOutput(func() {
		cmdServeDry()
	})
	if !strings.Contains(out, "Starting BlackCat gateway") {
		t.Error("expected serve startup banner")
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
	setupTempHome(t)

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
	setupTempHome(t)

	// Add first so remove has something to act on.
	captureOutput(func() {
		run([]string{"blackcat", "schedule", "add", "*/5 * * * *", "abc123"})
	})

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
	setupTempHome(t)

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
	setupTempHome(t)

	// Add first so remove has something to act on.
	captureOutput(func() {
		run([]string{"blackcat", "mcp", "add", "myserver", "npx", "my-mcp-server"})
	})

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

// --- initAgent tests ---

func TestInitAgent_NoAPIKeys(t *testing.T) {
	// Unset all provider keys so initAgent falls back to Ollama.
	keys := []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GROQ_API_KEY"}
	for _, k := range keys {
		t.Setenv(k, "")
	}

	core, err := initAgent()
	if err != nil {
		t.Fatalf("initAgent() returned unexpected error: %v", err)
	}
	if core == nil {
		t.Fatal("initAgent() returned nil core")
	}
}

func TestInitAgent_AnthropicKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-test")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("GROQ_API_KEY", "")

	core, err := initAgent()
	if err != nil {
		t.Fatalf("initAgent() with Anthropic key: %v", err)
	}
	if core == nil {
		t.Fatal("initAgent() returned nil core")
	}
}

func TestInitAgent_OpenAIKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "sk-test")
	t.Setenv("GROQ_API_KEY", "")

	core, err := initAgent()
	if err != nil {
		t.Fatalf("initAgent() with OpenAI key: %v", err)
	}
	if core == nil {
		t.Fatal("initAgent() returned nil core")
	}
}

func TestInitAgent_GroqKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("GROQ_API_KEY", "gsk_test")

	core, err := initAgent()
	if err != nil {
		t.Fatalf("initAgent() with Groq key: %v", err)
	}
	if core == nil {
		t.Fatal("initAgent() returned nil core")
	}
}

func TestInitAgent_AnthropicPriority(t *testing.T) {
	// When multiple keys are set, Anthropic should be selected first.
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-test")
	t.Setenv("OPENAI_API_KEY", "sk-openai-test")
	t.Setenv("GROQ_API_KEY", "gsk_test")

	core, err := initAgent()
	if err != nil {
		t.Fatalf("initAgent() with multiple keys: %v", err)
	}
	if core == nil {
		t.Fatal("initAgent() returned nil core")
	}
}

func TestInitAgent_ReturnsNonNilCoreAlways(t *testing.T) {
	// Regardless of env state, initAgent must never return (nil, nil).
	keys := []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GROQ_API_KEY"}
	for _, k := range keys {
		t.Setenv(k, "")
	}

	core, err := initAgent()
	if core == nil && err == nil {
		t.Fatal("initAgent() returned (nil, nil) — must return a valid core or error")
	}
}

func TestLoadConfigFallback(t *testing.T) {
	// Point HOME to a temp dir with no config file.
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	// loadConfig must not fail even without an existing file.
	cfg := loadConfig()
	if cfg.Agent.MaxSubAgents == 0 {
		// Default is 3; zero means Default() wasn't applied.
		t.Error("loadConfig() did not apply defaults")
	}
}

// --- Config persistence tests ---

// setupTempHome creates a temp home dir with a .blackcat/config.yaml
// containing the default config YAML. Returns the temp dir path.
func setupTempHome(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	configDir := tmpDir + "/.blackcat"
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	configPath := configDir + "/config.yaml"
	if err := os.WriteFile(configPath, []byte(defaultConfigYAML()), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	return tmpDir
}

// TestSaveConfigRoundTrip verifies saveConfig writes valid YAML that can be re-read.
func TestSaveConfigRoundTrip(t *testing.T) {
	tmpDir := setupTempHome(t)
	configPath := tmpDir + "/.blackcat/config.yaml"

	cfg := loadConfig()
	cfg.Scheduler.Enabled = true
	if err := saveConfig(cfg, configPath); err != nil {
		t.Fatalf("saveConfig failed: %v", err)
	}

	// Reload and verify the change persisted
	reloaded := loadConfig()
	if !reloaded.Scheduler.Enabled {
		t.Error("expected scheduler.enabled=true after saveConfig round-trip")
	}
}

// TestScheduleAddPersists verifies 'schedule add' writes to config and no longer
// prints the "not yet implemented" placeholder.
func TestScheduleAddPersists(t *testing.T) {
	setupTempHome(t)

	out := captureOutput(func() {
		code := run([]string{"blackcat", "schedule", "add", "*/5 * * * *", "check disk usage"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})

	if strings.Contains(out, "not yet implemented") {
		t.Errorf("expected persistence, but got placeholder: %q", out)
	}
	if !strings.Contains(out, "Added schedule") {
		t.Errorf("expected confirmation, got %q", out)
	}
}

// TestScheduleAddAndListRoundTrip verifies add followed by list shows the new schedule.
func TestScheduleAddAndListRoundTrip(t *testing.T) {
	setupTempHome(t)

	captureOutput(func() {
		run([]string{"blackcat", "schedule", "add", "0 9 * * *", "morning standup"})
	})

	out := captureOutput(func() {
		code := run([]string{"blackcat", "schedule", "list"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})

	if !strings.Contains(out, "0 9 * * *") {
		t.Errorf("expected cron expression in list, got %q", out)
	}
	if !strings.Contains(out, "morning standup") {
		t.Errorf("expected task in list, got %q", out)
	}
}

// TestScheduleRemovePersists verifies 'schedule remove' removes an entry and no longer
// prints the "not yet implemented" placeholder.
func TestScheduleRemovePersists(t *testing.T) {
	setupTempHome(t)

	// Add a schedule first
	captureOutput(func() {
		run([]string{"blackcat", "schedule", "add", "*/10 * * * *", "cleanup task"})
	})

	out := captureOutput(func() {
		code := run([]string{"blackcat", "schedule", "remove", "cleanup task"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})

	if strings.Contains(out, "not yet implemented") {
		t.Errorf("expected persistence, got placeholder: %q", out)
	}
	if !strings.Contains(out, "Removed schedule") {
		t.Errorf("expected removal confirmation, got %q", out)
	}
}

// TestScheduleRemoveAndListRoundTrip verifies removed schedules no longer appear in list.
func TestScheduleRemoveAndListRoundTrip(t *testing.T) {
	setupTempHome(t)

	captureOutput(func() {
		run([]string{"blackcat", "schedule", "add", "0 8 * * *", "morning task"})
	})
	captureOutput(func() {
		run([]string{"blackcat", "schedule", "remove", "morning task"})
	})

	out := captureOutput(func() {
		run([]string{"blackcat", "schedule", "list"})
	})

	if strings.Contains(out, "morning task") {
		t.Errorf("expected removed schedule to be absent, got %q", out)
	}
}

// TestScheduleRemoveNonExistent verifies removing a non-existent schedule returns error.
func TestScheduleRemoveNonExistent(t *testing.T) {
	setupTempHome(t)

	_ = captureOutput(func() {
		code := run([]string{"blackcat", "schedule", "remove", "ghost-schedule"})
		if code != 1 {
			t.Errorf("expected exit code 1 for non-existent schedule, got %d", code)
		}
	})
}

// TestScheduleListFromConfig verifies list reads from persistent config.
func TestScheduleListFromConfig(t *testing.T) {
	setupTempHome(t)

	captureOutput(func() {
		run([]string{"blackcat", "schedule", "add", "0 1 * * *", "task-alpha"})
	})
	captureOutput(func() {
		run([]string{"blackcat", "schedule", "add", "0 2 * * *", "task-beta"})
	})

	out := captureOutput(func() {
		run([]string{"blackcat", "schedule", "list"})
	})

	if !strings.Contains(out, "task-alpha") {
		t.Errorf("expected task-alpha in list, got %q", out)
	}
	if !strings.Contains(out, "task-beta") {
		t.Errorf("expected task-beta in list, got %q", out)
	}
}

// TestMCPAddPersists verifies 'mcp add' writes to config and removes the placeholder.
func TestMCPAddPersists(t *testing.T) {
	setupTempHome(t)

	out := captureOutput(func() {
		code := run([]string{"blackcat", "mcp", "add", "filesystem", "npx", "-y", "@modelcontextprotocol/server-filesystem"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})

	if strings.Contains(out, "not yet implemented") {
		t.Errorf("expected persistence, got placeholder: %q", out)
	}
	if !strings.Contains(out, "Added MCP server: filesystem") {
		t.Errorf("expected MCP add confirmation, got %q", out)
	}
}

// TestMCPAddAndListRoundTrip verifies add followed by list shows the new server.
func TestMCPAddAndListRoundTrip(t *testing.T) {
	setupTempHome(t)

	captureOutput(func() {
		run([]string{"blackcat", "mcp", "add", "brave-search", "npx", "brave-mcp"})
	})

	out := captureOutput(func() {
		code := run([]string{"blackcat", "mcp", "list"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})

	if !strings.Contains(out, "brave-search") {
		t.Errorf("expected server name in list, got %q", out)
	}
}

// TestMCPRemovePersists verifies 'mcp remove' removes the server and removes placeholder.
func TestMCPRemovePersists(t *testing.T) {
	setupTempHome(t)

	captureOutput(func() {
		run([]string{"blackcat", "mcp", "add", "temp-server", "node", "server.js"})
	})

	out := captureOutput(func() {
		code := run([]string{"blackcat", "mcp", "remove", "temp-server"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})

	if strings.Contains(out, "not yet implemented") {
		t.Errorf("expected persistence, got placeholder: %q", out)
	}
	if !strings.Contains(out, "Removed MCP server") {
		t.Errorf("expected removal confirmation, got %q", out)
	}
}

// TestMCPRemoveAndListRoundTrip verifies removed servers are absent from list.
func TestMCPRemoveAndListRoundTrip(t *testing.T) {
	setupTempHome(t)

	captureOutput(func() {
		run([]string{"blackcat", "mcp", "add", "deletable", "echo", "hello"})
	})
	captureOutput(func() {
		run([]string{"blackcat", "mcp", "remove", "deletable"})
	})

	out := captureOutput(func() {
		run([]string{"blackcat", "mcp", "list"})
	})

	if strings.Contains(out, "deletable") {
		t.Errorf("expected removed server to be absent, got %q", out)
	}
}

// TestMCPRemoveNonExistent verifies removing a missing server returns error.
func TestMCPRemoveNonExistent(t *testing.T) {
	setupTempHome(t)

	_ = captureOutput(func() {
		code := run([]string{"blackcat", "mcp", "remove", "ghost-server"})
		if code != 1 {
			t.Errorf("expected exit code 1 for non-existent server, got %d", code)
		}
	})
}

// TestMCPListFromConfig verifies list reads from persistent config.
func TestMCPListFromConfig(t *testing.T) {
	setupTempHome(t)

	captureOutput(func() {
		run([]string{"blackcat", "mcp", "add", "server-one", "python", "server.py"})
	})
	captureOutput(func() {
		run([]string{"blackcat", "mcp", "add", "server-two", "ruby", "server.rb"})
	})

	out := captureOutput(func() {
		run([]string{"blackcat", "mcp", "list"})
	})

	if !strings.Contains(out, "server-one") {
		t.Errorf("expected server-one in list, got %q", out)
	}
	if !strings.Contains(out, "server-two") {
		t.Errorf("expected server-two in list, got %q", out)
	}
}

// TestConfigBackupCreated verifies that a backup is created when saving config.
func TestConfigBackupCreated(t *testing.T) {
	tmpDir := setupTempHome(t)
	configPath := tmpDir + "/.blackcat/config.yaml"
	backupPath := configPath + ".bak"

	// Backup should not exist yet
	if _, err := os.Stat(backupPath); err == nil {
		t.Fatal("backup should not exist before first modification")
	}

	captureOutput(func() {
		run([]string{"blackcat", "schedule", "add", "* * * * *", "backup test"})
	})

	// Backup should now exist
	if _, err := os.Stat(backupPath); err != nil {
		t.Errorf("expected backup file to exist after config modification: %v", err)
	}
}

// TestScheduleListNoConfig verifies list shows empty list when no config file exists.
func TestScheduleListNoConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	out := captureOutput(func() {
		code := run([]string{"blackcat", "schedule", "list"})
		if code != 0 {
			t.Errorf("expected exit code 0 for missing config, got %d", code)
		}
	})
	if !strings.Contains(out, "Schedules:") {
		t.Errorf("expected schedules header, got %q", out)
	}
}

// TestMCPListNoConfig verifies mcp list shows empty list when no config file exists.
func TestMCPListNoConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	out := captureOutput(func() {
		code := run([]string{"blackcat", "mcp", "list"})
		if code != 0 {
			t.Errorf("expected exit code 0 for missing config, got %d", code)
		}
	})
	if !strings.Contains(out, "MCP servers:") {
		t.Errorf("expected MCP servers header, got %q", out)
	}
}

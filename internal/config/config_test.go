package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()

	// Model
	if cfg.Model != "anthropic/claude-sonnet-4-6" {
		t.Errorf("expected default model 'anthropic/claude-sonnet-4-6', got %q", cfg.Model)
	}

	// Memory defaults
	if !cfg.Memory.Enabled {
		t.Error("expected memory to be enabled by default")
	}
	if cfg.Memory.MaxVectors != 10000 {
		t.Errorf("expected max_vectors 10000, got %d", cfg.Memory.MaxVectors)
	}
	if cfg.Memory.Embedding != "local" {
		t.Errorf("expected embedding 'local', got %q", cfg.Memory.Embedding)
	}
	if cfg.Memory.RetentionDays != 30 {
		t.Errorf("expected retention_days 30, got %d", cfg.Memory.RetentionDays)
	}

	// Agent defaults
	if cfg.Agent.MaxSubAgents != 3 {
		t.Errorf("expected max_sub_agents 3, got %d", cfg.Agent.MaxSubAgents)
	}
	if cfg.Agent.SubAgentModel != "anthropic/claude-haiku-4-5" {
		t.Errorf("expected sub_agent_model 'anthropic/claude-haiku-4-5', got %q", cfg.Agent.SubAgentModel)
	}
	if cfg.Agent.SubAgentTimeout != "300s" {
		t.Errorf("expected sub_agent_timeout '300s', got %q", cfg.Agent.SubAgentTimeout)
	}

	// Scheduler defaults
	if cfg.Scheduler.Enabled {
		t.Error("expected scheduler disabled by default")
	}

	// Ollama defaults
	if cfg.Providers.Ollama.BaseURL != "http://localhost:11434" {
		t.Errorf("expected ollama base_url 'http://localhost:11434', got %q", cfg.Providers.Ollama.BaseURL)
	}
}

func TestLoadFromYAML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	yaml := `
model: openai/gpt-4.1
memory:
  max_vectors: 5000
  embedding: openai
  retention_days: 60
agent:
  max_sub_agents: 5
channels:
  telegram:
    enabled: true
    token: "test-token-123"
    allowed_users: [111, 222]
    mode: private
scheduler:
  enabled: true
  schedules:
    - name: daily-report
      cron: "0 9 * * *"
      task: "generate report"
      channel: telegram
providers:
  anthropic:
    api_key: "sk-ant-test"
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadFromFile(cfgPath)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if cfg.Model != "openai/gpt-4.1" {
		t.Errorf("expected model 'openai/gpt-4.1', got %q", cfg.Model)
	}
	if cfg.Memory.MaxVectors != 5000 {
		t.Errorf("expected max_vectors 5000, got %d", cfg.Memory.MaxVectors)
	}
	if cfg.Memory.Embedding != "openai" {
		t.Errorf("expected embedding 'openai', got %q", cfg.Memory.Embedding)
	}
	if cfg.Memory.RetentionDays != 60 {
		t.Errorf("expected retention_days 60, got %d", cfg.Memory.RetentionDays)
	}
	if cfg.Agent.MaxSubAgents != 5 {
		t.Errorf("expected max_sub_agents 5, got %d", cfg.Agent.MaxSubAgents)
	}
	if !cfg.Channels.Telegram.Enabled {
		t.Error("expected telegram enabled")
	}
	if cfg.Channels.Telegram.Token != "test-token-123" {
		t.Errorf("expected telegram token 'test-token-123', got %q", cfg.Channels.Telegram.Token)
	}
	if len(cfg.Channels.Telegram.AllowedUsers) != 2 {
		t.Errorf("expected 2 allowed users, got %d", len(cfg.Channels.Telegram.AllowedUsers))
	}
	if !cfg.Scheduler.Enabled {
		t.Error("expected scheduler enabled")
	}
	if len(cfg.Scheduler.Schedules) != 1 {
		t.Fatalf("expected 1 schedule, got %d", len(cfg.Scheduler.Schedules))
	}
	if cfg.Scheduler.Schedules[0].Name != "daily-report" {
		t.Errorf("expected schedule name 'daily-report', got %q", cfg.Scheduler.Schedules[0].Name)
	}
	if cfg.Providers.Anthropic.APIKey != "sk-ant-test" {
		t.Errorf("expected anthropic api_key 'sk-ant-test', got %q", cfg.Providers.Anthropic.APIKey)
	}
}

func TestLoadFromYAMLMergesWithDefaults(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	// Partial config — only override model, rest should be defaults
	yaml := `model: ollama/llama3.3`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadFromFile(cfgPath)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Overridden
	if cfg.Model != "ollama/llama3.3" {
		t.Errorf("expected model 'ollama/llama3.3', got %q", cfg.Model)
	}
	// Defaults preserved
	if !cfg.Memory.Enabled {
		t.Error("expected memory enabled (default)")
	}
	if cfg.Memory.MaxVectors != 10000 {
		t.Errorf("expected max_vectors 10000 (default), got %d", cfg.Memory.MaxVectors)
	}
	if cfg.Agent.MaxSubAgents != 3 {
		t.Errorf("expected max_sub_agents 3 (default), got %d", cfg.Agent.MaxSubAgents)
	}
}

func TestLoadFromYAMLFileNotFound(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestLoadFromYAMLInvalidSyntax(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	invalid := `[[[not valid yaml`
	if err := os.WriteFile(cfgPath, []byte(invalid), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	_, err := LoadFromFile(cfgPath)
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

func TestEnvVarExpansion(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	t.Setenv("BLACKCAT_TEST_KEY", "expanded-value")

	yaml := `
providers:
  anthropic:
    api_key: "$BLACKCAT_TEST_KEY"
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadFromFile(cfgPath)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if cfg.Providers.Anthropic.APIKey != "expanded-value" {
		t.Errorf("expected env expanded 'expanded-value', got %q", cfg.Providers.Anthropic.APIKey)
	}
}

func TestMergeProjectConfig(t *testing.T) {
	global := Default()
	global.Model = "anthropic/claude-sonnet-4-6"

	dir := t.TempDir()
	projectPath := filepath.Join(dir, ".blackcat.yaml")

	yaml := `
model: ollama/deepseek-r1
agent:
  max_sub_agents: 1
`
	if err := os.WriteFile(projectPath, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	merged, err := MergeProjectConfig(global, projectPath)
	if err != nil {
		t.Fatalf("MergeProjectConfig failed: %v", err)
	}

	// Project overrides
	if merged.Model != "ollama/deepseek-r1" {
		t.Errorf("expected model 'ollama/deepseek-r1', got %q", merged.Model)
	}
	if merged.Agent.MaxSubAgents != 1 {
		t.Errorf("expected max_sub_agents 1, got %d", merged.Agent.MaxSubAgents)
	}

	// Global preserved
	if !merged.Memory.Enabled {
		t.Error("expected memory enabled from global")
	}
}

func TestMergeProjectConfigFileNotFoundReturnsGlobal(t *testing.T) {
	global := Default()
	merged, err := MergeProjectConfig(global, "/nonexistent/.blackcat.yaml")
	if err != nil {
		t.Fatalf("expected no error for missing project config, got: %v", err)
	}
	if merged.Model != global.Model {
		t.Error("expected global config returned when project config missing")
	}
}

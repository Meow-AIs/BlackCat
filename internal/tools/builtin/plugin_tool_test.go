package builtin

import (
	"context"
	"testing"

	"github.com/meowai/blackcat/internal/plugin"
	"github.com/meowai/blackcat/internal/tools"
)

func TestPluginToolInfo(t *testing.T) {
	dir := t.TempDir()
	mgr := plugin.NewManager(dir)
	tool := NewPluginTool(mgr)

	info := tool.Info()
	if info.Name != "manage_plugins" {
		t.Errorf("expected name manage_plugins, got %s", info.Name)
	}
	if info.Category != "plugin" {
		t.Errorf("expected category plugin, got %s", info.Category)
	}

	// Check that action parameter exists
	found := false
	for _, p := range info.Parameters {
		if p.Name == "action" {
			found = true
			if len(p.Enum) == 0 {
				t.Error("expected action parameter to have enum values")
			}
		}
	}
	if !found {
		t.Error("expected action parameter in tool definition")
	}
}

func TestPluginToolImplementsInterface(t *testing.T) {
	dir := t.TempDir()
	mgr := plugin.NewManager(dir)
	tool := NewPluginTool(mgr)

	var _ tools.Tool = tool
}

func TestPluginToolList(t *testing.T) {
	dir := t.TempDir()
	mgr := plugin.NewManager(dir)

	manifest := plugin.PluginManifest{
		Name:         "test/sample",
		Version:      "1.0.0",
		Type:         plugin.PluginProvider,
		Description:  "A test plugin",
		Author:       "test",
		License:      "MIT",
		Command:      "echo",
		Protocol:     "jsonrpc",
		Capabilities: []string{"chat"},
	}
	mgr.Install(manifest)

	tool := NewPluginTool(mgr)
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "list",
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if result.Output == "" {
		t.Error("expected non-empty output for list")
	}
}

func TestPluginToolInstall(t *testing.T) {
	dir := t.TempDir()
	mgr := plugin.NewManager(dir)
	tool := NewPluginTool(mgr)

	result, err := tool.Execute(context.Background(), map[string]any{
		"action":  "install",
		"name":    "test/new-plugin",
		"version": "1.0.0",
		"type":    "provider",
		"command": "test-binary",
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d; error: %s", result.ExitCode, result.Error)
	}

	// Verify it's actually installed
	if _, ok := mgr.Get("test/new-plugin"); !ok {
		t.Error("expected plugin to be installed")
	}
}

func TestPluginToolUninstall(t *testing.T) {
	dir := t.TempDir()
	mgr := plugin.NewManager(dir)

	manifest := plugin.PluginManifest{
		Name:         "test/removeme",
		Version:      "1.0.0",
		Type:         plugin.PluginProvider,
		Command:      "echo",
		Protocol:     "jsonrpc",
		Capabilities: []string{"chat"},
	}
	mgr.Install(manifest)

	tool := NewPluginTool(mgr)
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "uninstall",
		"name":   "test/removeme",
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d; error: %s", result.ExitCode, result.Error)
	}

	if _, ok := mgr.Get("test/removeme"); ok {
		t.Error("expected plugin to be removed")
	}
}

func TestPluginToolInfo_Action(t *testing.T) {
	dir := t.TempDir()
	mgr := plugin.NewManager(dir)

	manifest := plugin.PluginManifest{
		Name:         "test/info-target",
		Version:      "2.0.0",
		Type:         plugin.PluginChannel,
		Description:  "Info test",
		Author:       "tester",
		Command:      "echo",
		Protocol:     "jsonrpc",
		Capabilities: []string{"send"},
	}
	mgr.Install(manifest)

	tool := NewPluginTool(mgr)
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "info",
		"name":   "test/info-target",
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if result.Output == "" {
		t.Error("expected non-empty info output")
	}
}

func TestPluginToolMissingAction(t *testing.T) {
	dir := t.TempDir()
	mgr := plugin.NewManager(dir)
	tool := NewPluginTool(mgr)

	result, err := tool.Execute(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code for missing action")
	}
}

func TestPluginToolUnknownAction(t *testing.T) {
	dir := t.TempDir()
	mgr := plugin.NewManager(dir)
	tool := NewPluginTool(mgr)

	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "unknown_action",
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code for unknown action")
	}
}

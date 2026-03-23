package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func testManifest(name string) PluginManifest {
	return PluginManifest{
		Name:         name,
		Version:      "1.0.0",
		Type:         PluginProvider,
		Description:  "Test plugin",
		Author:       "test",
		License:      "MIT",
		Command:      "echo",
		Protocol:     "jsonrpc",
		Capabilities: []string{"chat"},
		Config: map[string]PluginConfigField{
			"api_key": {
				Type:        "secret",
				Description: "API key",
				Required:    true,
			},
			"model": {
				Type:        "string",
				Description: "Default model",
				Default:     "gpt-4",
				Required:    false,
			},
		},
	}
}

func setupManager(t *testing.T) *Manager {
	t.Helper()
	dir := t.TempDir()
	return NewManager(dir)
}

func TestNewManager(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)
	if m == nil {
		t.Fatal("expected non-nil manager")
	}
	if m.pluginDir != dir {
		t.Errorf("expected pluginDir %s, got %s", dir, m.pluginDir)
	}
	if m.nextPort != 50051 {
		t.Errorf("expected nextPort 50051, got %d", m.nextPort)
	}
}

func TestInstall(t *testing.T) {
	m := setupManager(t)
	manifest := testManifest("test/myplugin")

	err := m.Install(manifest)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	inst, ok := m.Get("test/myplugin")
	if !ok {
		t.Fatal("expected plugin to exist after install")
	}
	if inst.State != StateInstalled {
		t.Errorf("expected state %s, got %s", StateInstalled, inst.State)
	}
	if inst.Manifest.Name != "test/myplugin" {
		t.Errorf("expected name test/myplugin, got %s", inst.Manifest.Name)
	}
}

func TestInstallDuplicateRejected(t *testing.T) {
	m := setupManager(t)
	manifest := testManifest("test/dup")

	if err := m.Install(manifest); err != nil {
		t.Fatalf("first install failed: %v", err)
	}
	if err := m.Install(manifest); err == nil {
		t.Fatal("expected error on duplicate install")
	}
}

func TestInstallValidation(t *testing.T) {
	m := setupManager(t)

	tests := []struct {
		name     string
		manifest PluginManifest
	}{
		{"empty name", PluginManifest{Version: "1.0.0", Type: PluginProvider, Command: "x", Protocol: "jsonrpc"}},
		{"empty version", PluginManifest{Name: "a/b", Type: PluginProvider, Command: "x", Protocol: "jsonrpc"}},
		{"empty command", PluginManifest{Name: "a/b", Version: "1.0.0", Type: PluginProvider, Protocol: "jsonrpc"}},
		{"empty protocol", PluginManifest{Name: "a/b", Version: "1.0.0", Type: PluginProvider, Command: "x"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := m.Install(tt.manifest); err == nil {
				t.Error("expected validation error")
			}
		})
	}
}

func TestUninstall(t *testing.T) {
	m := setupManager(t)
	manifest := testManifest("test/removeme")

	if err := m.Install(manifest); err != nil {
		t.Fatalf("install failed: %v", err)
	}

	if err := m.Uninstall("test/removeme"); err != nil {
		t.Fatalf("uninstall failed: %v", err)
	}

	if _, ok := m.Get("test/removeme"); ok {
		t.Error("expected plugin to be removed after uninstall")
	}
}

func TestUninstallNotFound(t *testing.T) {
	m := setupManager(t)
	if err := m.Uninstall("nonexistent"); err == nil {
		t.Error("expected error uninstalling nonexistent plugin")
	}
}

func TestStartStop(t *testing.T) {
	m := setupManager(t)
	manifest := testManifest("test/lifecycle")
	manifest.Command = "sleep"
	manifest.Args = []string{"60"}

	if err := m.Install(manifest); err != nil {
		t.Fatalf("install failed: %v", err)
	}

	if err := m.Start("test/lifecycle"); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	if !m.IsRunning("test/lifecycle") {
		t.Error("expected plugin to be running after start")
	}

	inst, _ := m.Get("test/lifecycle")
	if inst.State != StateActive {
		t.Errorf("expected state %s, got %s", StateActive, inst.State)
	}
	if inst.Port == 0 {
		t.Error("expected non-zero port after start")
	}

	if err := m.Stop("test/lifecycle"); err != nil {
		t.Fatalf("stop failed: %v", err)
	}

	if m.IsRunning("test/lifecycle") {
		t.Error("expected plugin to not be running after stop")
	}

	inst, _ = m.Get("test/lifecycle")
	if inst.State != StateStopped {
		t.Errorf("expected state %s, got %s", StateStopped, inst.State)
	}
}

func TestStartNotFound(t *testing.T) {
	m := setupManager(t)
	if err := m.Start("nonexistent"); err == nil {
		t.Error("expected error starting nonexistent plugin")
	}
}

func TestRestart(t *testing.T) {
	m := setupManager(t)
	manifest := testManifest("test/restart")
	manifest.Command = "sleep"
	manifest.Args = []string{"60"}

	if err := m.Install(manifest); err != nil {
		t.Fatalf("install failed: %v", err)
	}
	if err := m.Start("test/restart"); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	if err := m.Restart("test/restart"); err != nil {
		t.Fatalf("restart failed: %v", err)
	}

	if !m.IsRunning("test/restart") {
		t.Error("expected plugin to be running after restart")
	}

	// Cleanup
	_ = m.Stop("test/restart")
}

func TestList(t *testing.T) {
	m := setupManager(t)
	m.Install(testManifest("test/a"))

	bManifest := testManifest("test/b")
	bManifest.Type = PluginChannel
	m.Install(bManifest)

	all := m.List()
	if len(all) != 2 {
		t.Errorf("expected 2 plugins, got %d", len(all))
	}
}

func TestListByType(t *testing.T) {
	m := setupManager(t)
	m.Install(testManifest("test/prov1"))

	chanManifest := testManifest("test/chan1")
	chanManifest.Type = PluginChannel
	m.Install(chanManifest)

	providers := m.ListByType(PluginProvider)
	if len(providers) != 1 {
		t.Errorf("expected 1 provider, got %d", len(providers))
	}
	if providers[0].Manifest.Name != "test/prov1" {
		t.Errorf("expected test/prov1, got %s", providers[0].Manifest.Name)
	}

	channels := m.ListByType(PluginChannel)
	if len(channels) != 1 {
		t.Errorf("expected 1 channel, got %d", len(channels))
	}
}

func TestConfigure(t *testing.T) {
	m := setupManager(t)
	m.Install(testManifest("test/configurable"))

	cfg := map[string]string{"api_key": "sk-123", "model": "gpt-3.5"}
	if err := m.Configure("test/configurable", cfg); err != nil {
		t.Fatalf("configure failed: %v", err)
	}

	inst, _ := m.Get("test/configurable")
	if inst.Config["api_key"] != "sk-123" {
		t.Errorf("expected api_key sk-123, got %s", inst.Config["api_key"])
	}
}

func TestConfigureNotFound(t *testing.T) {
	m := setupManager(t)
	if err := m.Configure("nonexistent", map[string]string{}); err == nil {
		t.Error("expected error configuring nonexistent plugin")
	}
}

func TestValidateConfig(t *testing.T) {
	m := setupManager(t)
	manifest := testManifest("test/validate")

	// Missing required field
	err := m.ValidateConfig(manifest, map[string]string{"model": "gpt-4"})
	if err == nil {
		t.Error("expected validation error for missing required field")
	}

	// All required fields present
	err = m.ValidateConfig(manifest, map[string]string{"api_key": "sk-123"})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestStatePersistence(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)
	m.Install(testManifest("test/persist"))

	cfg := map[string]string{"api_key": "sk-abc"}
	m.Configure("test/persist", cfg)

	if err := m.SaveState(); err != nil {
		t.Fatalf("save state failed: %v", err)
	}

	// Verify file exists
	stateFile := filepath.Join(dir, "plugins.json")
	if _, err := os.Stat(stateFile); err != nil {
		t.Fatalf("state file not found: %v", err)
	}

	// Load into new manager
	m2 := NewManager(dir)
	if err := m2.LoadState(); err != nil {
		t.Fatalf("load state failed: %v", err)
	}

	inst, ok := m2.Get("test/persist")
	if !ok {
		t.Fatal("expected plugin to exist after load")
	}
	if inst.Manifest.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", inst.Manifest.Version)
	}
	if inst.Config["api_key"] != "sk-abc" {
		t.Errorf("expected api_key sk-abc, got %s", inst.Config["api_key"])
	}
}

func TestStopAll(t *testing.T) {
	m := setupManager(t)

	a := testManifest("test/a")
	a.Command = "sleep"
	a.Args = []string{"60"}
	m.Install(a)

	b := testManifest("test/b")
	b.Command = "sleep"
	b.Args = []string{"60"}
	b.Type = PluginChannel
	m.Install(b)

	m.Start("test/a")
	m.Start("test/b")

	if err := m.StopAll(); err != nil {
		t.Fatalf("stop all failed: %v", err)
	}

	if m.IsRunning("test/a") || m.IsRunning("test/b") {
		t.Error("expected all plugins to be stopped")
	}
}

func TestAllocatePort(t *testing.T) {
	m := setupManager(t)

	p1 := m.allocatePort()
	p2 := m.allocatePort()

	if p1 == p2 {
		t.Error("expected unique ports")
	}
	if p1 != 50051 {
		t.Errorf("expected first port 50051, got %d", p1)
	}
	if p2 != 50052 {
		t.Errorf("expected second port 50052, got %d", p2)
	}
}

func TestIsRunningNotFound(t *testing.T) {
	m := setupManager(t)
	if m.IsRunning("nonexistent") {
		t.Error("expected false for nonexistent plugin")
	}
}

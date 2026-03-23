package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// PluginInstance represents a single installed plugin and its runtime state.
type PluginInstance struct {
	Manifest  PluginManifest    `json:"manifest"`
	State     PluginState       `json:"state"`
	Process   *os.Process       `json:"-"` // nil when stopped
	Port      int               `json:"port"`
	StartedAt time.Time         `json:"started_at,omitempty"`
	Error     string            `json:"error,omitempty"`
	Config    map[string]string `json:"config,omitempty"`
}

// Manager manages plugin lifecycle: install, start, stop, configure, persist.
type Manager struct {
	plugins   map[string]*PluginInstance
	pluginDir string
	mu        sync.RWMutex
	nextPort  int
}

// NewManager creates a Manager that stores plugins under pluginDir.
func NewManager(pluginDir string) *Manager {
	return &Manager{
		plugins:   make(map[string]*PluginInstance),
		pluginDir: pluginDir,
		nextPort:  50051,
	}
}

// Install validates the manifest and registers a new plugin in installed state.
func (m *Manager) Install(manifest PluginManifest) error {
	if err := validateManifest(manifest); err != nil {
		return fmt.Errorf("invalid manifest: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.plugins[manifest.Name]; exists {
		return fmt.Errorf("plugin %q is already installed", manifest.Name)
	}

	m.plugins[manifest.Name] = &PluginInstance{
		Manifest: manifest,
		State:    StateInstalled,
		Config:   make(map[string]string),
	}
	return nil
}

// Uninstall stops (if running) and removes the plugin.
func (m *Manager) Uninstall(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	inst, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %q not found", name)
	}

	if inst.State == StateActive && inst.Process != nil {
		_ = inst.Process.Kill()
	}

	delete(m.plugins, name)
	return nil
}

// Start launches the plugin process and sets state to active.
func (m *Manager) Start(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	inst, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %q not found", name)
	}

	if inst.State == StateActive && inst.Process != nil {
		return fmt.Errorf("plugin %q is already running", name)
	}

	port := m.allocatePortLocked()

	cmd := exec.Command(inst.Manifest.Command, inst.Manifest.Args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PLUGIN_PORT=%d", port))

	if err := cmd.Start(); err != nil {
		inst.State = StateError
		inst.Error = err.Error()
		return fmt.Errorf("failed to start plugin %q: %w", name, err)
	}

	inst.Process = cmd.Process
	inst.Port = port
	inst.State = StateActive
	inst.StartedAt = time.Now()
	inst.Error = ""

	return nil
}

// Stop gracefully stops the plugin process.
func (m *Manager) Stop(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	inst, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %q not found", name)
	}

	if inst.Process != nil {
		_ = inst.Process.Kill()
		_, _ = inst.Process.Wait()
		inst.Process = nil
	}

	inst.State = StateStopped
	inst.Error = ""
	return nil
}

// Restart stops and then starts the plugin.
func (m *Manager) Restart(name string) error {
	if err := m.Stop(name); err != nil {
		return err
	}
	return m.Start(name)
}

// Get returns a copy of the plugin instance. Returns false if not found.
func (m *Manager) Get(name string) (*PluginInstance, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	inst, exists := m.plugins[name]
	if !exists {
		return nil, false
	}

	copied := *inst
	return &copied, true
}

// List returns copies of all plugin instances.
func (m *Manager) List() []PluginInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]PluginInstance, 0, len(m.plugins))
	for _, inst := range m.plugins {
		result = append(result, *inst)
	}
	return result
}

// ListByType returns copies of plugins matching the given type.
func (m *Manager) ListByType(pluginType PluginType) []PluginInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []PluginInstance
	for _, inst := range m.plugins {
		if inst.Manifest.Type == pluginType {
			result = append(result, *inst)
		}
	}
	return result
}

// IsRunning returns true if the plugin exists and is in active state.
func (m *Manager) IsRunning(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	inst, exists := m.plugins[name]
	if !exists {
		return false
	}
	return inst.State == StateActive && inst.Process != nil
}

// Configure sets user-provided config values on the plugin.
func (m *Manager) Configure(name string, config map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	inst, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %q not found", name)
	}

	// Create new config map (immutable pattern)
	newConfig := make(map[string]string, len(config))
	for k, v := range config {
		newConfig[k] = v
	}
	inst.Config = newConfig
	return nil
}

// ValidateConfig checks that all required config fields are present.
func (m *Manager) ValidateConfig(manifest PluginManifest, config map[string]string) error {
	for fieldName, field := range manifest.Config {
		if field.Required {
			if _, ok := config[fieldName]; !ok {
				return fmt.Errorf("required config field %q is missing", fieldName)
			}
		}
	}
	return nil
}

// persistedState is the JSON-serializable form of all plugins.
type persistedState struct {
	Plugins map[string]*PluginInstance `json:"plugins"`
}

// SaveState persists plugin state to plugins.json.
func (m *Manager) SaveState() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state := persistedState{Plugins: m.plugins}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	path := filepath.Join(m.pluginDir, "plugins.json")
	if err := os.MkdirAll(m.pluginDir, 0o755); err != nil {
		return fmt.Errorf("create plugin dir: %w", err)
	}

	return os.WriteFile(path, data, 0o644)
}

// LoadState restores plugin state from plugins.json.
func (m *Manager) LoadState() error {
	path := filepath.Join(m.pluginDir, "plugins.json")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read state: %w", err)
	}

	var state persistedState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("unmarshal state: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if state.Plugins != nil {
		m.plugins = state.Plugins
	}

	// Loaded plugins are not running — clear process references and reset active to stopped
	for _, inst := range m.plugins {
		inst.Process = nil
		if inst.State == StateActive {
			inst.State = StateStopped
		}
		if inst.Config == nil {
			inst.Config = make(map[string]string)
		}
	}

	return nil
}

// StopAll gracefully stops all running plugins. Returns the first error encountered.
func (m *Manager) StopAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var firstErr error
	for name, inst := range m.plugins {
		if inst.State == StateActive && inst.Process != nil {
			if err := inst.Process.Kill(); err != nil && firstErr == nil {
				firstErr = fmt.Errorf("stop %s: %w", name, err)
			}
			_, _ = inst.Process.Wait()
			inst.Process = nil
		}
		if inst.State == StateActive {
			inst.State = StateStopped
		}
	}
	return firstErr
}

// HealthCheck verifies the plugin is responsive (placeholder for pipe-based ping).
func (m *Manager) HealthCheck(name string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	inst, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %q not found", name)
	}
	if inst.State != StateActive {
		return fmt.Errorf("plugin %q is not running", name)
	}
	return nil
}

// allocatePort returns the next available port (thread-safe via caller lock).
func (m *Manager) allocatePort() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.allocatePortLocked()
}

// allocatePortLocked returns the next port. Caller must hold m.mu.
func (m *Manager) allocatePortLocked() int {
	port := m.nextPort
	m.nextPort++
	return port
}

// validateManifest checks required fields on a PluginManifest.
func validateManifest(manifest PluginManifest) error {
	var errs []error
	if manifest.Name == "" {
		errs = append(errs, errors.New("name is required"))
	}
	if manifest.Version == "" {
		errs = append(errs, errors.New("version is required"))
	}
	if manifest.Command == "" {
		errs = append(errs, errors.New("command is required"))
	}
	if manifest.Protocol == "" {
		errs = append(errs, errors.New("protocol is required"))
	}
	return errors.Join(errs...)
}

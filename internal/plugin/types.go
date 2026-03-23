package plugin

// PluginType identifies the kind of extension a plugin provides.
type PluginType string

const (
	PluginProvider PluginType = "provider" // LLM provider
	PluginChannel  PluginType = "channel"  // messaging channel adapter
	PluginDomain   PluginType = "domain"   // domain specialization
	PluginScanner  PluginType = "scanner"  // security scanner
	PluginHook     PluginType = "hook"     // event hook
)

// PluginState represents the current lifecycle state of a plugin.
type PluginState string

const (
	StateInstalled PluginState = "installed"
	StateActive    PluginState = "active"
	StateStopped   PluginState = "stopped"
	StateError     PluginState = "error"
)

// PluginManifest describes a plugin's metadata and how to launch it.
type PluginManifest struct {
	Name         string                       `json:"name"`                   // unique id: "author/name"
	Version      string                       `json:"version"`                // semver
	Type         PluginType                   `json:"type"`
	Description  string                       `json:"description"`
	Author       string                       `json:"author"`
	License      string                       `json:"license"`
	Command      string                       `json:"command"`                // binary to execute
	Args         []string                     `json:"args,omitempty"`
	Protocol     string                       `json:"protocol"`               // "grpc" or "jsonrpc"
	Port         int                          `json:"port,omitempty"`         // 0 = auto-assign
	Capabilities []string                     `json:"capabilities"`           // what the plugin provides
	Config       map[string]PluginConfigField `json:"config,omitempty"`       // user-configurable fields
	MinVersion   string                       `json:"min_version,omitempty"` // min BlackCat version
}

// PluginConfigField describes a single user-configurable setting for a plugin.
type PluginConfigField struct {
	Type        string `json:"type"`        // "string", "int", "bool", "secret"
	Description string `json:"description"`
	Default     string `json:"default,omitempty"`
	Required    bool   `json:"required"`
}

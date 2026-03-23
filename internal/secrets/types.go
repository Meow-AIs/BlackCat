// Package secrets provides secure secret management for BlackCat.
//
// Architecture:
//   OS Keychain (preferred) > Encrypted File (fallback) > Environment Variables (last resort)
//
// Secrets are never stored in plaintext config files, never passed to LLM context,
// and automatically redacted from all tool outputs, logs, and memory entries.
package secrets

import (
	"encoding/json"
	"time"
)

// SecretType categorizes secrets for access control and UI grouping.
type SecretType string

const (
	TypeAPIKey     SecretType = "api_key"
	TypeSSHKey     SecretType = "ssh_key"
	TypeKubeConfig SecretType = "kube_config"
	TypeDBCred     SecretType = "db_credential"
	TypeVPN        SecretType = "vpn_credential"
	TypeCloudCred  SecretType = "cloud_credential"
	TypeGitToken   SecretType = "git_token"
	TypeCustom     SecretType = "custom"
)

// Scope determines whether a secret is available globally or per-project.
type Scope string

const (
	ScopeGlobal  Scope = "global"
	ScopeProject Scope = "project"
)

// SecretMetadata holds all non-sensitive information about a stored secret.
// The actual secret value is never stored in this struct after initial creation.
type SecretMetadata struct {
	// Name is the unique identifier within its scope (e.g. "openai_api_key").
	Name string `json:"name"`

	// Type categorizes the secret for access control.
	Type SecretType `json:"type"`

	// Scope determines global vs project-level availability.
	Scope Scope `json:"scope"`

	// ProjectPath is set when Scope == ScopeProject. Absolute path to the project root.
	ProjectPath string `json:"project_path,omitempty"`

	// Description is a human-readable explanation (never contains the secret value).
	Description string `json:"description,omitempty"`

	// EnvVar is the environment variable name to inject when executing tools.
	// e.g. "OPENAI_API_KEY". Empty means the secret is not auto-injected.
	EnvVar string `json:"env_var,omitempty"`

	// Tags enable grouping and filtering (e.g. ["llm", "anthropic"]).
	Tags []string `json:"tags,omitempty"`

	// CreatedAt is when the secret was first stored.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the secret value was last changed.
	UpdatedAt time.Time `json:"updated_at"`

	// ExpiresAt is the optional expiry time. Zero value means no expiry.
	ExpiresAt time.Time `json:"expires_at,omitempty"`

	// RotationDays is the recommended rotation interval. 0 means no rotation policy.
	RotationDays int `json:"rotation_days,omitempty"`

	// AllowedTools lists tool names that may access this secret.
	// Empty means all tools can access it (subject to scope).
	AllowedTools []string `json:"allowed_tools,omitempty"`

	// AllowedAgents lists sub-agent IDs that may access this secret.
	// Empty means only the primary agent (not sub-agents) can access it.
	AllowedAgents []string `json:"allowed_agents,omitempty"`

	// ImportedFrom tracks the origin for imported secrets (e.g. "aws_credentials", "dotenv").
	ImportedFrom string `json:"imported_from,omitempty"`

	// Fingerprint is a truncated SHA-256 of the secret value, used for change detection
	// without storing the value in metadata. Format: first 8 hex chars.
	Fingerprint string `json:"fingerprint,omitempty"`
}

// SecretRef is a lightweight reference to a secret, used in access control lists
// and injection mappings. It never contains the secret value.
type SecretRef struct {
	Name  string `json:"name"`
	Scope Scope  `json:"scope"`
}

// AuditEntry records a single access to a secret. The secret value is never logged.
type AuditEntry struct {
	Timestamp time.Time `json:"timestamp"`
	SecretRef SecretRef `json:"secret_ref"`
	Action    string    `json:"action"` // "read", "write", "delete", "rotate", "inject"
	Actor     string    `json:"actor"`  // "agent", "sub-agent:<id>", "tool:<name>", "user", "scheduler"
	Reason    string    `json:"reason,omitempty"`
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`
}

// ExpiryStatus summarizes a secret's rotation health.
type ExpiryStatus struct {
	SecretRef  SecretRef `json:"secret_ref"`
	ExpiresAt  time.Time `json:"expires_at,omitempty"`
	DaysLeft   int       `json:"days_left"`
	IsExpired  bool      `json:"is_expired"`
	NeedsRotation bool  `json:"needs_rotation"`
}

// MarshalJSON prevents accidental serialization of SecretMetadata with any
// attached context that might contain values.
func (m SecretMetadata) MarshalJSON() ([]byte, error) {
	type Alias SecretMetadata
	return json.Marshal(Alias(m))
}

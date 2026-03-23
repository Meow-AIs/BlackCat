package remote

import (
	"errors"
	"fmt"
	"sync"
)

// ConnectionType represents the type of remote connection.
type ConnectionType string

const (
	ConnSSH     ConnectionType = "ssh"
	ConnKubectl ConnectionType = "kubectl"
)

// ConnectionProfile holds configuration for a single remote connection.
type ConnectionProfile struct {
	Name        string         `json:"name" yaml:"name"`
	Type        ConnectionType `json:"type" yaml:"type"`
	Host        string         `json:"host,omitempty" yaml:"host,omitempty"`
	Port        int            `json:"port,omitempty" yaml:"port,omitempty"`
	User        string         `json:"user,omitempty" yaml:"user,omitempty"`
	KeyPath     string         `json:"key_path,omitempty" yaml:"key_path,omitempty"`
	ProxyJump   string         `json:"proxy_jump,omitempty" yaml:"proxy_jump,omitempty"`
	KubeContext string         `json:"kube_context,omitempty" yaml:"kube_context,omitempty"`
	KubeNS      string         `json:"kube_namespace,omitempty" yaml:"kube_namespace,omitempty"`
	Environment string         `json:"environment" yaml:"environment"`
	Permissions RemotePerms    `json:"permissions" yaml:"permissions"`
}

// RemotePerms defines what operations are allowed on a remote connection.
type RemotePerms struct {
	AllowExec       bool     `json:"allow_exec" yaml:"allow_exec"`
	AllowTransfer   bool     `json:"allow_transfer" yaml:"allow_transfer"`
	AllowedCmds     []string `json:"allowed_cmds,omitempty" yaml:"allowed_cmds,omitempty"`
	DeniedCmds      []string `json:"denied_cmds,omitempty" yaml:"denied_cmds,omitempty"`
	MaxConcurrent   int      `json:"max_concurrent" yaml:"max_concurrent"`
	RequireApproval bool     `json:"require_approval" yaml:"require_approval"`
}

// ProfileStore defines operations for managing connection profiles.
type ProfileStore interface {
	Get(name string) (ConnectionProfile, error)
	List() []ConnectionProfile
	Save(profile ConnectionProfile) error
	Delete(name string) error
}

// MemoryProfileStore is a thread-safe in-memory implementation of ProfileStore.
type MemoryProfileStore struct {
	mu       sync.RWMutex
	profiles map[string]ConnectionProfile
}

// NewMemoryProfileStore creates a new empty in-memory profile store.
func NewMemoryProfileStore() *MemoryProfileStore {
	return &MemoryProfileStore{
		profiles: make(map[string]ConnectionProfile),
	}
}

// Get retrieves a profile by name. Returns a copy to preserve immutability.
func (s *MemoryProfileStore) Get(name string) (ConnectionProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.profiles[name]
	if !ok {
		return ConnectionProfile{}, fmt.Errorf("profile %q not found", name)
	}
	return copyProfile(p), nil
}

// List returns all profiles as copies to preserve immutability.
func (s *MemoryProfileStore) List() []ConnectionProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]ConnectionProfile, 0, len(s.profiles))
	for _, p := range s.profiles {
		result = append(result, copyProfile(p))
	}
	return result
}

// Save stores a profile, overwriting any existing profile with the same name.
func (s *MemoryProfileStore) Save(profile ConnectionProfile) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.profiles[profile.Name] = copyProfile(profile)
	return nil
}

// Delete removes a profile by name.
func (s *MemoryProfileStore) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.profiles[name]; !ok {
		return fmt.Errorf("profile %q not found", name)
	}
	delete(s.profiles, name)
	return nil
}

// copyProfile returns a deep copy of a ConnectionProfile.
func copyProfile(p ConnectionProfile) ConnectionProfile {
	cp := p
	if p.Permissions.AllowedCmds != nil {
		cp.Permissions.AllowedCmds = make([]string, len(p.Permissions.AllowedCmds))
		copy(cp.Permissions.AllowedCmds, p.Permissions.AllowedCmds)
	}
	if p.Permissions.DeniedCmds != nil {
		cp.Permissions.DeniedCmds = make([]string, len(p.Permissions.DeniedCmds))
		copy(cp.Permissions.DeniedCmds, p.Permissions.DeniedCmds)
	}
	return cp
}

var validEnvironments = map[string]bool{
	"dev":     true,
	"staging": true,
	"prod":    true,
}

// ValidateProfile checks that a ConnectionProfile has all required fields.
func ValidateProfile(p ConnectionProfile) error {
	if p.Name == "" {
		return errors.New("profile name is required")
	}
	if p.Type == "" {
		return errors.New("connection type is required")
	}
	if p.Type != ConnSSH && p.Type != ConnKubectl {
		return fmt.Errorf("invalid connection type: %q (must be %q or %q)", p.Type, ConnSSH, ConnKubectl)
	}
	if p.Environment == "" {
		return errors.New("environment is required")
	}
	if !validEnvironments[p.Environment] {
		return fmt.Errorf("invalid environment: %q (must be dev, staging, or prod)", p.Environment)
	}
	if p.Type == ConnSSH && p.Host == "" {
		return errors.New("host is required for SSH connections")
	}
	if p.Type == ConnKubectl && p.KubeContext == "" {
		return errors.New("kube_context is required for kubectl connections")
	}
	return nil
}

// DefaultPermsForEnv returns sensible default permissions for a given environment.
// Unknown environments default to the most restrictive (prod-like) settings.
func DefaultPermsForEnv(env string) RemotePerms {
	switch env {
	case "dev":
		return RemotePerms{
			AllowExec:       true,
			AllowTransfer:   true,
			RequireApproval: false,
			MaxConcurrent:   5,
		}
	case "staging":
		return RemotePerms{
			AllowExec:       true,
			AllowTransfer:   true,
			RequireApproval: true,
			MaxConcurrent:   5,
		}
	default:
		// prod and unknown environments get the most restrictive defaults
		return RemotePerms{
			AllowExec:       true,
			AllowTransfer:   false,
			RequireApproval: true,
			MaxConcurrent:   5,
			DeniedCmds:      []string{"rm -rf", "DROP", "TRUNCATE", "shutdown"},
		}
	}
}

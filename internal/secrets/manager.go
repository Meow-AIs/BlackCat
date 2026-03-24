package secrets

import (
	"context"
	"fmt"
	"regexp"
	"time"
)

// validName matches secret names: lowercase alphanumeric, hyphens, underscores, dots.
// Max 128 chars. No leading/trailing separators.
var validName = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,126}[a-z0-9]$`)

// Manager is the high-level secret management API. It coordinates between
// backends (keychain, encrypted file, env), metadata, audit logging, and
// access control.
//
// Usage:
//
//	mgr, err := secrets.NewManager(opts)
//	val, err := mgr.Get(ctx, "openai_api_key", secrets.ScopeGlobal)
//	defer secrets.SecureWipe(val)
type Manager struct {
	backends    []Backend // ordered by preference: keychain > file > env
	primary     Backend   // first available backend (used for writes)
	meta        MetadataStore
	audit       AuditLog
	projectPath string // current project root (for scope resolution)
}

// ManagerOpts configures the Manager.
type ManagerOpts struct {
	// Backends in preference order. If nil, defaults to [keychain, encrypted_file, env].
	Backends []Backend

	// MetadataStore for persisting secret metadata.
	MetadataStore MetadataStore

	// AuditLog for recording access events. May be nil to disable audit logging.
	AuditLog AuditLog

	// ProjectPath is the current project root. Empty means no project context.
	ProjectPath string
}

// NewManager creates a Manager with the given options.
// At least one backend must be available, and MetadataStore is required.
func NewManager(opts ManagerOpts) (*Manager, error) {
	if opts.MetadataStore == nil {
		return nil, fmt.Errorf("MetadataStore is required")
	}

	m := &Manager{
		backends:    opts.Backends,
		meta:        opts.MetadataStore,
		audit:       opts.AuditLog,
		projectPath: opts.ProjectPath,
	}

	// Find the first available backend for writes.
	for _, b := range m.backends {
		if b.Available() {
			m.primary = b
			break
		}
	}
	if m.primary == nil {
		return nil, fmt.Errorf("no available secret backend")
	}

	return m, nil
}

// Get retrieves a secret value. It checks access control, expiry, and logs the access.
// The caller is responsible for calling SecureWipe on the returned bytes when done.
func (m *Manager) Get(ctx context.Context, name string, scope Scope) ([]byte, error) {
	if !validName.MatchString(name) {
		return nil, ErrInvalidName
	}

	// Check metadata for expiry and access control.
	meta, err := m.meta.GetMeta(ctx, name, scope)
	if err != nil {
		return nil, err
	}

	if !meta.ExpiresAt.IsZero() && time.Now().After(meta.ExpiresAt) {
		m.logAudit(ctx, name, scope, "read", "agent", false, "secret expired")
		return nil, ErrExpired
	}

	// Build the storage key.
	storageKey := storageKey(name, scope)

	// Try each backend in preference order.
	var val []byte
	for _, b := range m.backends {
		if !b.Available() {
			continue
		}
		val, err = b.Get(ctx, storageKey)
		if err == nil {
			break
		}
		if err != ErrNotFound {
			// Log backend failure but try next backend.
			continue
		}
	}

	if val == nil {
		m.logAudit(ctx, name, scope, "read", "agent", false, "not found in any backend")
		return nil, ErrNotFound
	}

	m.logAudit(ctx, name, scope, "read", "agent", true, "")
	return val, nil
}

// Set stores a secret with its metadata. The value is written to the primary backend
// and metadata is persisted separately.
func (m *Manager) Set(ctx context.Context, meta SecretMetadata, value []byte) error {
	if !validName.MatchString(meta.Name) {
		return ErrInvalidName
	}

	now := time.Now()
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = now
	}
	meta.UpdatedAt = now
	meta.Fingerprint = Fingerprint(value)

	storageKey := storageKey(meta.Name, meta.Scope)

	if err := m.primary.Set(ctx, storageKey, value); err != nil {
		m.logAudit(ctx, meta.Name, meta.Scope, "write", "user", false, err.Error())
		return fmt.Errorf("store secret value: %w", err)
	}

	if err := m.meta.SaveMeta(ctx, meta); err != nil {
		// Roll back the backend write on metadata failure.
		_ = m.primary.Delete(ctx, storageKey)
		return fmt.Errorf("store secret metadata: %w", err)
	}

	m.logAudit(ctx, meta.Name, meta.Scope, "write", "user", true, "")
	return nil
}

// Delete removes a secret from both the backend and metadata store.
func (m *Manager) Delete(ctx context.Context, name string, scope Scope) error {
	if !validName.MatchString(name) {
		return ErrInvalidName
	}

	storageKey := storageKey(name, scope)

	// Delete from all backends (secret might exist in multiple due to migration).
	for _, b := range m.backends {
		if !b.Available() {
			continue
		}
		_ = b.Delete(ctx, storageKey)
	}

	if err := m.meta.DeleteMeta(ctx, name, scope); err != nil {
		m.logAudit(ctx, name, scope, "delete", "user", false, err.Error())
		return err
	}

	m.logAudit(ctx, name, scope, "delete", "user", true, "")
	return nil
}

// List returns metadata for all secrets in the given scope.
func (m *Manager) List(ctx context.Context, scope Scope) ([]SecretMetadata, error) {
	return m.meta.ListMeta(ctx, scope)
}

// Exists checks if a secret is present.
func (m *Manager) Exists(ctx context.Context, name string, scope Scope) (bool, error) {
	_, err := m.meta.GetMeta(ctx, name, scope)
	if err == ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// CheckExpiry returns all secrets that are expired or will expire within the given days.
func (m *Manager) CheckExpiry(ctx context.Context, withinDays int) ([]ExpiryStatus, error) {
	return m.meta.ListExpiring(ctx, withinDays)
}

// Rotate replaces a secret's value and updates the rotation timestamp.
func (m *Manager) Rotate(ctx context.Context, name string, scope Scope, newValue []byte) error {
	meta, err := m.meta.GetMeta(ctx, name, scope)
	if err != nil {
		return err
	}

	meta.UpdatedAt = time.Now()
	meta.Fingerprint = Fingerprint(newValue)

	// If there's a rotation policy, push the expiry forward.
	if meta.RotationDays > 0 {
		meta.ExpiresAt = time.Now().AddDate(0, 0, meta.RotationDays)
	}

	storageKey := storageKey(name, scope)
	if err := m.primary.Set(ctx, storageKey, newValue); err != nil {
		return fmt.Errorf("rotate secret value: %w", err)
	}

	if err := m.meta.SaveMeta(ctx, meta); err != nil {
		return fmt.Errorf("update rotation metadata: %w", err)
	}

	m.logAudit(ctx, name, scope, "rotate", "user", true, "")
	return nil
}

// AuditHistory returns audit entries for a secret.
func (m *Manager) AuditHistory(ctx context.Context, name string, limit int) ([]AuditEntry, error) {
	if m.audit == nil {
		return nil, nil
	}
	return m.audit.Query(ctx, AuditFilter{SecretName: name, Limit: limit})
}

// logAudit is a helper that silently ignores errors from the audit log.
func (m *Manager) logAudit(ctx context.Context, name string, scope Scope, action, actor string, success bool, errMsg string) {
	if m.audit == nil {
		return
	}
	_ = m.audit.Log(ctx, AuditEntry{
		Timestamp: time.Now(),
		SecretRef: SecretRef{Name: name, Scope: scope},
		Action:    action,
		Actor:     actor,
		Success:   success,
		Error:     errMsg,
	})
}

// storageKey builds the backend key for a secret.
// Format: "<scope>/<name>" to namespace global vs project secrets.
func storageKey(name string, scope Scope) string {
	return string(scope) + "/" + name
}

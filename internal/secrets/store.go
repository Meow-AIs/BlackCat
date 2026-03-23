package secrets

import (
	"context"
	"errors"
)

// Sentinel errors for the secrets package.
var (
	ErrNotFound       = errors.New("secret not found")
	ErrAlreadyExists  = errors.New("secret already exists")
	ErrAccessDenied   = errors.New("access denied to secret")
	ErrExpired        = errors.New("secret has expired")
	ErrBackendFailure = errors.New("secret backend failure")
	ErrLocked         = errors.New("secret store is locked")
	ErrInvalidName    = errors.New("invalid secret name")
)

// Store is the primary interface for reading and writing secrets.
// Implementations must never log or return the secret value in error messages.
type Store interface {
	// Get retrieves a secret value. Returns ErrNotFound if absent, ErrExpired if past expiry.
	Get(ctx context.Context, name string, scope Scope) ([]byte, error)

	// Set stores or updates a secret value and its metadata.
	Set(ctx context.Context, meta SecretMetadata, value []byte) error

	// Delete removes a secret. Returns ErrNotFound if absent.
	Delete(ctx context.Context, name string, scope Scope) error

	// List returns metadata for all secrets matching the given scope.
	// Never returns secret values.
	List(ctx context.Context, scope Scope) ([]SecretMetadata, error)

	// Exists checks whether a secret is present without retrieving its value.
	Exists(ctx context.Context, name string, scope Scope) (bool, error)
}

// Backend is the underlying storage mechanism. Store dispatches to the appropriate
// Backend based on availability (keychain > encrypted file > env).
type Backend interface {
	// Name returns the backend identifier (e.g. "keychain", "encrypted_file", "env").
	Name() string

	// Available reports whether this backend is usable on the current platform.
	Available() bool

	// Get retrieves the raw secret bytes.
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores the raw secret bytes.
	Set(ctx context.Context, key string, value []byte) error

	// Delete removes a secret.
	Delete(ctx context.Context, key string) error

	// List returns all keys managed by this backend.
	List(ctx context.Context) ([]string, error)
}

// MetadataStore persists SecretMetadata separately from secret values.
// This is always backed by SQLite (the same memory.db or a dedicated secrets.db).
type MetadataStore interface {
	// SaveMeta persists metadata for a secret.
	SaveMeta(ctx context.Context, meta SecretMetadata) error

	// GetMeta retrieves metadata. Returns ErrNotFound if absent.
	GetMeta(ctx context.Context, name string, scope Scope) (SecretMetadata, error)

	// DeleteMeta removes metadata.
	DeleteMeta(ctx context.Context, name string, scope Scope) error

	// ListMeta returns all metadata matching the scope filter.
	ListMeta(ctx context.Context, scope Scope) ([]SecretMetadata, error)

	// ListExpiring returns secrets that expire within the given number of days.
	ListExpiring(ctx context.Context, withinDays int) ([]ExpiryStatus, error)
}

// AuditLog records secret access events. Stored in SQLite.
type AuditLog interface {
	// Log records an audit entry.
	Log(ctx context.Context, entry AuditEntry) error

	// Query returns audit entries matching the filter.
	Query(ctx context.Context, filter AuditFilter) ([]AuditEntry, error)
}

// AuditFilter controls which audit entries are returned.
type AuditFilter struct {
	SecretName string     // filter by secret name (empty = all)
	Action     string     // filter by action (empty = all)
	Actor      string     // filter by actor (empty = all)
	Since      int64      // unix timestamp, 0 = no lower bound
	Limit      int        // max entries to return, 0 = default (100)
}

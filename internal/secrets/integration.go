package secrets

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// DefaultGlobalDir returns the BlackCat home directory: ~/.blackcat
func DefaultGlobalDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".blackcat")
}

// DefaultSecretsFilePath returns the default encrypted secrets file path.
func DefaultSecretsFilePath() string {
	return filepath.Join(DefaultGlobalDir(), "secrets.enc")
}

// SetupOpts configures the full secrets subsystem.
type SetupOpts struct {
	// DB is the shared SQLite database (same as memory.db).
	DB *sql.DB

	// Passphrase for the encrypted file backend. Required when keychain is unavailable.
	// Typically prompted from the user or read from BLACKCAT_MASTER_PASSWORD env var.
	Passphrase []byte

	// ProjectPath is the current project root (may be empty).
	ProjectPath string

	// SecretsFilePath overrides the default encrypted file location.
	SecretsFilePath string
}

// SetupResult contains all the components created by Setup.
type SetupResult struct {
	Manager       *Manager
	SecureManager *SecureManager
	Injector      *Injector
	Sanitizer     *Sanitizer
	Importer      *Importer
}

// Setup initializes the complete secrets subsystem. This is the main entry point
// called during BlackCat startup.
//
// Backend selection order:
//  1. OS Keychain (macOS Keychain, Windows Credential Manager, Linux libsecret)
//  2. Encrypted file (~/.blackcat/secrets.enc) with Argon2id + XChaCha20-Poly1305
//  3. Environment variables (read-only, BLACKCAT_SECRET_* prefix)
func Setup(ctx context.Context, opts SetupOpts) (*SetupResult, error) {
	if opts.DB == nil {
		return nil, fmt.Errorf("database is required for secrets metadata")
	}

	// Initialize metadata store and audit log.
	metaStore, err := NewSQLiteMetadataStore(opts.DB)
	if err != nil {
		return nil, fmt.Errorf("init metadata store: %w", err)
	}
	auditLog := NewSQLiteAuditLog(opts.DB)

	// Build backend chain.
	backends := buildBackendChain(opts)

	// Create manager.
	mgr, err := NewManager(ManagerOpts{
		Backends:      backends,
		MetadataStore: metaStore,
		AuditLog:      auditLog,
		ProjectPath:   opts.ProjectPath,
	})
	if err != nil {
		return nil, fmt.Errorf("init secret manager: %w", err)
	}

	// Create secure manager with default access policy.
	policy := NewDefaultAccessPolicy()
	secureMgr := NewSecureManager(mgr, policy)

	// Create injector and sanitizer.
	injector := NewInjector(secureMgr)
	sanitizer := NewSanitizer()

	// Pre-register existing secrets in the sanitizer for output redaction.
	if err := preloadSanitizer(ctx, mgr, sanitizer); err != nil {
		// Non-fatal: sanitizer works but existing secrets won't be redacted until accessed.
		_ = err
	}

	importer := NewImporter(mgr)

	return &SetupResult{
		Manager:       mgr,
		SecureManager: secureMgr,
		Injector:      injector,
		Sanitizer:     sanitizer,
		Importer:      importer,
	}, nil
}

// buildBackendChain creates backends in preference order.
func buildBackendChain(opts SetupOpts) []Backend {
	var backends []Backend

	// 1. OS Keychain
	kc := NewKeychainBackend()
	if kc.Available() {
		backends = append(backends, kc)
	}

	// 2. Encrypted file (only if passphrase is available)
	if len(opts.Passphrase) > 0 {
		filePath := opts.SecretsFilePath
		if filePath == "" {
			filePath = DefaultSecretsFilePath()
		}

		// Make a copy of passphrase since NewEncryptedFileBackend wipes it.
		passphraseCopy := make([]byte, len(opts.Passphrase))
		copy(passphraseCopy, opts.Passphrase)

		efb, err := NewEncryptedFileBackend(filePath, passphraseCopy)
		if err == nil {
			backends = append(backends, efb)
		}
	}

	// 3. Environment variables (always available, read-only)
	backends = append(backends, NewEnvBackend())

	return backends
}

// preloadSanitizer loads all known secret values into the sanitizer.
func preloadSanitizer(ctx context.Context, mgr *Manager, sanitizer *Sanitizer) error {
	for _, scope := range []Scope{ScopeGlobal, ScopeProject} {
		metas, err := mgr.List(ctx, scope)
		if err != nil {
			continue
		}
		for _, meta := range metas {
			val, err := mgr.Get(ctx, meta.Name, meta.Scope)
			if err != nil {
				continue
			}
			sanitizer.Register(string(val), meta.Name)
			SecureWipe(val)
		}
	}
	return nil
}

// MasterPassphraseFromEnv reads the master passphrase from the
// BLACKCAT_MASTER_PASSWORD environment variable. Returns nil if not set.
// This is intended for headless/CI environments where interactive prompt
// is not available.
func MasterPassphraseFromEnv() []byte {
	val := os.Getenv("BLACKCAT_MASTER_PASSWORD")
	if val == "" {
		return nil
	}
	return []byte(val)
}

// PlatformKeychainInfo returns a human-readable description of the OS keychain
// that would be used on this platform.
func PlatformKeychainInfo() string {
	switch runtime.GOOS {
	case "darwin":
		return "macOS Keychain Services"
	case "windows":
		return "Windows Credential Manager"
	case "linux":
		return "Linux Secret Service (libsecret/GNOME Keyring or KWallet)"
	default:
		return "No OS keychain available; using encrypted file backend"
	}
}

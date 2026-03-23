//go:build cgo

package secrets

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// DefaultGlobalDir / DefaultSecretsFilePath
// ---------------------------------------------------------------------------

func TestDefaultGlobalDir_ContainsBlackcat(t *testing.T) {
	dir := DefaultGlobalDir()
	if dir == "" {
		t.Error("DefaultGlobalDir should return a non-empty path")
	}
	if !strings.Contains(dir, ".blackcat") {
		t.Errorf("DefaultGlobalDir should contain '.blackcat', got %q", dir)
	}
}

func TestDefaultSecretsFilePath_EndsWithSecretsEnc(t *testing.T) {
	path := DefaultSecretsFilePath()
	if !strings.HasSuffix(path, "secrets.enc") {
		t.Errorf("DefaultSecretsFilePath should end with secrets.enc, got %q", path)
	}
	// Must be inside the .blackcat dir.
	dir := filepath.Dir(path)
	if !strings.HasSuffix(dir, ".blackcat") {
		t.Errorf("secrets file should be inside .blackcat dir, got %q", dir)
	}
}

// ---------------------------------------------------------------------------
// MasterPassphraseFromEnv
// ---------------------------------------------------------------------------

func TestMasterPassphraseFromEnv_Set(t *testing.T) {
	t.Setenv("BLACKCAT_MASTER_PASSWORD", "my-master-pass")

	pass := MasterPassphraseFromEnv()
	if string(pass) != "my-master-pass" {
		t.Errorf("MasterPassphraseFromEnv: want %q, got %q", "my-master-pass", pass)
	}
}

func TestMasterPassphraseFromEnv_NotSet(t *testing.T) {
	// Unset the variable.
	t.Setenv("BLACKCAT_MASTER_PASSWORD", "")

	pass := MasterPassphraseFromEnv()
	if pass != nil {
		t.Errorf("MasterPassphraseFromEnv: want nil when unset, got %q", pass)
	}
}

// ---------------------------------------------------------------------------
// PlatformKeychainInfo
// ---------------------------------------------------------------------------

func TestPlatformKeychainInfo_NonEmpty(t *testing.T) {
	info := PlatformKeychainInfo()
	if info == "" {
		t.Error("PlatformKeychainInfo should return a non-empty string")
	}
}

// ---------------------------------------------------------------------------
// Setup — full integration using temp DB and encrypted file backend
// ---------------------------------------------------------------------------

func TestSetup_WithEncryptedFile(t *testing.T) {
	db := openTestDB(t)
	dir := t.TempDir()
	secretsFile := filepath.Join(dir, "test.enc")

	opts := SetupOpts{
		DB:              db,
		Passphrase:      []byte("setup-test-passphrase"),
		SecretsFilePath: secretsFile,
		ProjectPath:     "/test/project",
	}

	result, err := Setup(context.Background(), opts)
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if result.Manager == nil {
		t.Error("Setup should return a non-nil Manager")
	}
	if result.SecureManager == nil {
		t.Error("Setup should return a non-nil SecureManager")
	}
	if result.Injector == nil {
		t.Error("Setup should return a non-nil Injector")
	}
	if result.Sanitizer == nil {
		t.Error("Setup should return a non-nil Sanitizer")
	}
	if result.Importer == nil {
		t.Error("Setup should return a non-nil Importer")
	}
}

func TestSetup_NilDB_ReturnsError(t *testing.T) {
	_, err := Setup(context.Background(), SetupOpts{
		DB:         nil,
		Passphrase: []byte("pass"),
	})
	if err == nil {
		t.Error("Setup with nil DB should return an error")
	}
}

func TestSetup_NoPassphrase_UsesEnvBackendOnly(t *testing.T) {
	db := openTestDB(t)

	// No passphrase means the encrypted file backend is skipped.
	opts := SetupOpts{
		DB: db,
		// Passphrase intentionally nil.
	}

	result, err := Setup(context.Background(), opts)
	if err != nil {
		t.Fatalf("Setup without passphrase: %v", err)
	}
	// Should still succeed — env backend is always available.
	if result.Manager == nil {
		t.Error("Setup should succeed with env backend when no passphrase is given")
	}
}

// ---------------------------------------------------------------------------
// MasterPassphraseFromEnv — integration path
// ---------------------------------------------------------------------------

func TestSetup_PassphraseFromEnv(t *testing.T) {
	db := openTestDB(t)
	dir := t.TempDir()

	t.Setenv("BLACKCAT_MASTER_PASSWORD", "env-passphrase")

	pass := MasterPassphraseFromEnv()
	if pass == nil {
		t.Skip("BLACKCAT_MASTER_PASSWORD not readable — skipping")
	}

	opts := SetupOpts{
		DB:              db,
		Passphrase:      pass,
		SecretsFilePath: filepath.Join(dir, "env.enc"),
	}

	result, err := Setup(context.Background(), opts)
	if err != nil {
		t.Fatalf("Setup with env passphrase: %v", err)
	}
	if result.Manager == nil {
		t.Error("Manager should not be nil")
	}

	// Use os.Unsetenv to clean up (t.Setenv already handles it on cleanup,
	// but BLACKCAT_MASTER_PASSWORD read by other goroutines mid-test would
	// see our test value — acceptable since tests are isolated).
	_ = os.Unsetenv("BLACKCAT_MASTER_PASSWORD")
}

package secrets

import (
	"context"
	"fmt"
	"strings"

	"github.com/zalando/go-keyring"
)

// keychainServicePrefix is prepended to all keychain entries to namespace BlackCat secrets.
const keychainServicePrefix = "blackcat"

// KeychainBackend stores secrets in the OS keychain:
//   - macOS: Keychain Services (via Security.framework)
//   - Windows: Windows Credential Manager
//   - Linux: libsecret (GNOME Keyring) or KWallet
//
// This is the preferred backend because the OS manages encryption, access control,
// and secure memory. Falls back to encrypted file on headless systems where
// no keyring daemon is available.
type KeychainBackend struct{}

// NewKeychainBackend creates a keychain backend.
func NewKeychainBackend() *KeychainBackend {
	return &KeychainBackend{}
}

func (k *KeychainBackend) Name() string {
	return "keychain"
}

// Available checks if the OS keychain is accessible by performing a probe write/delete.
func (k *KeychainBackend) Available() bool {
	testKey := keychainServicePrefix + "/probe"
	err := keyring.Set(testKey, "blackcat-probe", "probe")
	if err != nil {
		return false
	}
	_ = keyring.Delete(testKey, "blackcat-probe")
	return true
}

func (k *KeychainBackend) Get(_ context.Context, key string) ([]byte, error) {
	service := keychainService(key)
	val, err := keyring.Get(service, "blackcat")
	if err != nil {
		if err == keyring.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("keychain get %q: %w", key, err)
	}
	return []byte(val), nil
}

func (k *KeychainBackend) Set(_ context.Context, key string, value []byte) error {
	service := keychainService(key)
	if err := keyring.Set(service, "blackcat", string(value)); err != nil {
		return fmt.Errorf("keychain set %q: %w", key, err)
	}
	return nil
}

func (k *KeychainBackend) Delete(_ context.Context, key string) error {
	service := keychainService(key)
	if err := keyring.Delete(service, "blackcat"); err != nil {
		if err == keyring.ErrNotFound {
			return ErrNotFound
		}
		return fmt.Errorf("keychain delete %q: %w", key, err)
	}
	return nil
}

// List is not natively supported by most OS keychains with go-keyring.
// We rely on the MetadataStore (SQLite) for listing and only use the keychain
// for value retrieval. This method returns nil.
func (k *KeychainBackend) List(_ context.Context) ([]string, error) {
	// OS keychains do not expose enumeration via go-keyring.
	// The MetadataStore is the source of truth for secret names.
	return nil, nil
}

// keychainService builds a namespaced service name for the keychain entry.
// Format: "blackcat/<scope>/<name>" — keeps entries organized and avoids collisions.
func keychainService(key string) string {
	if strings.HasPrefix(key, keychainServicePrefix+"/") {
		return key
	}
	return keychainServicePrefix + "/" + key
}

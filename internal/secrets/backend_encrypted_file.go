package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// EncryptedFileBackend stores secrets in an encrypted JSON file on disk.
// Used as a fallback when the OS keychain is unavailable (headless servers, containers).
//
// File format: Argon2id salt (16) || XChaCha20-Poly1305 nonce (24) || encrypted JSON blob
// The JSON blob is a map[string]string where keys are secret names and values are
// base64-encoded secret bytes.
//
// The file is stored at ~/.blackcat/secrets.enc by default.
type EncryptedFileBackend struct {
	mu        sync.RWMutex
	filePath  string
	masterKey []byte // derived from passphrase, held in memory while store is unlocked
	salt      []byte // Argon2id salt used to derive masterKey; preserved for re-encryption
	cache     map[string][]byte
	dirty     bool
}

// NewEncryptedFileBackend creates an encrypted file backend.
// The passphrase is used to derive the encryption key via Argon2id.
// If the file does not exist, it will be created on the first Set call.
func NewEncryptedFileBackend(filePath string, passphrase []byte) (*EncryptedFileBackend, error) {
	b := &EncryptedFileBackend{
		filePath: filePath,
		cache:    make(map[string][]byte),
	}

	// If the file exists, load and decrypt it.
	data, err := os.ReadFile(filePath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read secrets file: %w", err)
	}

	if len(data) > 0 {
		// File exists: read the embedded salt (first saltLen bytes) and re-derive the key.
		if len(data) < saltLen {
			return nil, fmt.Errorf("decrypt secrets file: file too short")
		}
		embeddedSalt := data[:saltLen]

		key, _, err := DeriveKey(passphrase, embeddedSalt)
		if err != nil {
			return nil, fmt.Errorf("derive master key: %w", err)
		}

		// Decrypt the payload that follows the salt: nonce(24) || ciphertext+tag
		plaintext, err := Decrypt(key, data[saltLen:])
		if err != nil {
			return nil, fmt.Errorf("decrypt secrets file (wrong passphrase?): %w", err)
		}
		if err := json.Unmarshal(plaintext, &b.cache); err != nil {
			SecureWipe(plaintext)
			return nil, fmt.Errorf("parse decrypted secrets: %w", err)
		}
		SecureWipe(plaintext)

		// Preserve salt and key for subsequent flushes.
		b.salt = make([]byte, saltLen)
		copy(b.salt, embeddedSalt)
		b.masterKey = key
	} else {
		// New file: generate a fresh salt and derive the master key.
		key, usedSalt, err := DeriveKey(passphrase, nil)
		if err != nil {
			return nil, fmt.Errorf("derive master key: %w", err)
		}
		b.masterKey = key
		b.salt = usedSalt
	}

	// Wipe the passphrase copy (caller should also wipe theirs).
	SecureWipe(passphrase)

	return b, nil
}

func (b *EncryptedFileBackend) Name() string {
	return "encrypted_file"
}

func (b *EncryptedFileBackend) Available() bool {
	dir := filepath.Dir(b.filePath)
	info, err := os.Stat(dir)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func (b *EncryptedFileBackend) Get(_ context.Context, key string) ([]byte, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	val, ok := b.cache[key]
	if !ok {
		return nil, ErrNotFound
	}

	// Return a copy to prevent mutation of cached value.
	cp := make([]byte, len(val))
	copy(cp, val)
	return cp, nil
}

func (b *EncryptedFileBackend) Set(_ context.Context, key string, value []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Store a copy to prevent external mutation.
	cp := make([]byte, len(value))
	copy(cp, value)
	b.cache[key] = cp
	b.dirty = true

	return b.flush()
}

func (b *EncryptedFileBackend) Delete(_ context.Context, key string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.cache[key]; !ok {
		return ErrNotFound
	}

	delete(b.cache, key)
	b.dirty = true

	return b.flush()
}

func (b *EncryptedFileBackend) List(_ context.Context) ([]string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	keys := make([]string, 0, len(b.cache))
	for k := range b.cache {
		keys = append(keys, k)
	}
	return keys, nil
}

// flush encrypts the cache and writes it to disk.
// Must be called with b.mu held.
// File format: salt (saltLen) || nonce (nonceLen) || ciphertext+tag
func (b *EncryptedFileBackend) flush() error {
	if b.masterKey == nil {
		return ErrLocked
	}

	plaintext, err := json.Marshal(b.cache)
	if err != nil {
		return fmt.Errorf("marshal secrets: %w", err)
	}

	// Encrypt with the pre-derived masterKey (no additional key derivation).
	encryptedPayload, err := Encrypt(b.masterKey, plaintext)
	if err != nil {
		SecureWipe(plaintext)
		return fmt.Errorf("encrypt secrets: %w", err)
	}
	SecureWipe(plaintext)

	// Prepend the salt so re-opening can re-derive the same key.
	encrypted := make([]byte, 0, saltLen+len(encryptedPayload))
	encrypted = append(encrypted, b.salt...)
	encrypted = append(encrypted, encryptedPayload...)

	// Ensure directory exists with restrictive permissions.
	dir := filepath.Dir(b.filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create secrets directory: %w", err)
	}

	// Write atomically: write to temp file, then rename.
	tmpPath := b.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, encrypted, 0600); err != nil {
		return fmt.Errorf("write secrets file: %w", err)
	}
	if err := os.Rename(tmpPath, b.filePath); err != nil {
		return fmt.Errorf("rename secrets file: %w", err)
	}

	b.dirty = false
	return nil
}

// Lock wipes the master key, salt, and cache from memory.
// After calling Lock, the backend cannot perform any operations until
// re-initialized with a passphrase.
func (b *EncryptedFileBackend) Lock() {
	b.mu.Lock()
	defer b.mu.Unlock()

	SecureWipe(b.masterKey)
	b.masterKey = nil

	SecureWipe(b.salt)
	b.salt = nil

	for k, v := range b.cache {
		SecureWipe(v)
		delete(b.cache, k)
	}
}

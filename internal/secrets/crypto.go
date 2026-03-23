package secrets

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

// Crypto parameters. Argon2id is used for key derivation, XChaCha20-Poly1305 for encryption.
// These are chosen for:
//   - Argon2id: winner of the Password Hashing Competition, resistant to GPU/ASIC attacks
//   - XChaCha20-Poly1305: 192-bit nonce (safe for random nonces), no AES-NI dependency,
//     constant-time, and available in Go's extended crypto library
const (
	// Argon2id parameters — OWASP recommended minimum for interactive use.
	argon2Time    = 3           // iterations
	argon2Memory  = 64 * 1024  // 64 MB
	argon2Threads = 4
	argon2KeyLen  = 32         // 256-bit key for XChaCha20-Poly1305

	// Salt length for Argon2id.
	saltLen = 16

	// XChaCha20-Poly1305 nonce length (24 bytes).
	nonceLen = 24
)

var (
	ErrDecryptionFailed = errors.New("decryption failed: wrong password or corrupted data")
	ErrInvalidCiphertext = errors.New("ciphertext too short")
)

// DeriveKey produces a 256-bit encryption key from a passphrase and salt using Argon2id.
// The salt must be exactly saltLen bytes. If nil, a random salt is generated and returned.
func DeriveKey(passphrase []byte, salt []byte) (key []byte, usedSalt []byte, err error) {
	if salt == nil {
		salt = make([]byte, saltLen)
		if _, err := io.ReadFull(rand.Reader, salt); err != nil {
			return nil, nil, fmt.Errorf("generate salt: %w", err)
		}
	}
	if len(salt) != saltLen {
		return nil, nil, fmt.Errorf("salt must be %d bytes, got %d", saltLen, len(salt))
	}

	key = argon2.IDKey(passphrase, salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)
	return key, salt, nil
}

// Encrypt encrypts plaintext using XChaCha20-Poly1305 with the given key.
// Returns: salt (16) || nonce (24) || ciphertext+tag
// The key should be derived via DeriveKey. If deriving from a passphrase, pass the
// passphrase to EncryptWithPassphrase instead.
func Encrypt(key []byte, plaintext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	nonce := make([]byte, nonceLen)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := aead.Seal(nil, nonce, plaintext, nil)

	// nonce || ciphertext (nonce is prepended; salt is handled by caller)
	result := make([]byte, 0, nonceLen+len(ciphertext))
	result = append(result, nonce...)
	result = append(result, ciphertext...)
	return result, nil
}

// Decrypt decrypts data produced by Encrypt using the given key.
// Input format: nonce (24) || ciphertext+tag
func Decrypt(key []byte, data []byte) ([]byte, error) {
	if len(data) < nonceLen {
		return nil, ErrInvalidCiphertext
	}

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	nonce := data[:nonceLen]
	ciphertext := data[nonceLen:]

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// EncryptWithPassphrase derives a key from passphrase and encrypts.
// Returns: salt (16) || nonce (24) || ciphertext+tag
func EncryptWithPassphrase(passphrase []byte, plaintext []byte) ([]byte, error) {
	key, salt, err := DeriveKey(passphrase, nil)
	if err != nil {
		return nil, err
	}

	encrypted, err := Encrypt(key, plaintext)
	if err != nil {
		return nil, err
	}

	// Prepend salt
	result := make([]byte, 0, saltLen+len(encrypted))
	result = append(result, salt...)
	result = append(result, encrypted...)
	return result, nil
}

// DecryptWithPassphrase extracts the salt, derives the key, and decrypts.
// Input format: salt (16) || nonce (24) || ciphertext+tag
func DecryptWithPassphrase(passphrase []byte, data []byte) ([]byte, error) {
	if len(data) < saltLen+nonceLen {
		return nil, ErrInvalidCiphertext
	}

	salt := data[:saltLen]
	encrypted := data[saltLen:]

	key, _, err := DeriveKey(passphrase, salt)
	if err != nil {
		return nil, err
	}

	return Decrypt(key, encrypted)
}

// Fingerprint returns the first 8 hex characters of SHA-256(value).
// Used for change detection without storing the actual value.
func Fingerprint(value []byte) string {
	h := sha256.Sum256(value)
	return hex.EncodeToString(h[:4])
}

// SecureWipe overwrites a byte slice with zeros. Call this after you are done
// with sensitive data (keys, plaintext secrets) to reduce the window where
// secrets exist in memory.
//
// Note: Go's GC may have already copied the data elsewhere. This is a
// best-effort mitigation, not a guarantee. For true memory protection,
// use mlock via syscall on Linux/macOS.
func SecureWipe(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

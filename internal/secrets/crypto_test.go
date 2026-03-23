package secrets

import (
	"bytes"
	"testing"
)

// ---------------------------------------------------------------------------
// DeriveKey
// ---------------------------------------------------------------------------

func TestDeriveKey_GeneratesSalt(t *testing.T) {
	key, salt, err := DeriveKey([]byte("passphrase"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(key) != argon2KeyLen {
		t.Errorf("key length: want %d, got %d", argon2KeyLen, len(key))
	}
	if len(salt) != saltLen {
		t.Errorf("salt length: want %d, got %d", saltLen, len(salt))
	}
}

func TestDeriveKey_Deterministic(t *testing.T) {
	salt := bytes.Repeat([]byte{0xAB}, saltLen)
	k1, _, err1 := DeriveKey([]byte("passphrase"), salt)
	k2, _, err2 := DeriveKey([]byte("passphrase"), append([]byte(nil), salt...))
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected errors: %v / %v", err1, err2)
	}
	if !bytes.Equal(k1, k2) {
		t.Error("same passphrase+salt must produce same key")
	}
}

func TestDeriveKey_DifferentPassphrases(t *testing.T) {
	salt := bytes.Repeat([]byte{0x01}, saltLen)
	k1, _, _ := DeriveKey([]byte("passA"), salt)
	k2, _, _ := DeriveKey([]byte("passB"), salt)
	if bytes.Equal(k1, k2) {
		t.Error("different passphrases must produce different keys")
	}
}

func TestDeriveKey_WrongSaltLength(t *testing.T) {
	tests := []struct {
		name string
		salt []byte
	}{
		{"short salt", []byte("tooshort")},
		{"long salt", bytes.Repeat([]byte{0x01}, saltLen+1)},
		{"empty salt", []byte{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := DeriveKey([]byte("passphrase"), tt.salt)
			if err == nil {
				t.Error("expected error for invalid salt length")
			}
		})
	}
}

func TestDeriveKey_EmptyPassphrase(t *testing.T) {
	// Empty passphrase is valid (unusual but not forbidden by the function).
	key, salt, err := DeriveKey([]byte{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(key) != argon2KeyLen {
		t.Errorf("key length: want %d, got %d", argon2KeyLen, len(key))
	}
	if len(salt) != saltLen {
		t.Errorf("salt length: want %d, got %d", saltLen, len(salt))
	}
}

// ---------------------------------------------------------------------------
// Encrypt / Decrypt round-trip
// ---------------------------------------------------------------------------

func validKey(t *testing.T) []byte {
	t.Helper()
	key, _, err := DeriveKey([]byte("test-passphrase"), bytes.Repeat([]byte{0xCC}, saltLen))
	if err != nil {
		t.Fatalf("derive key: %v", err)
	}
	return key
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	tests := []struct {
		name      string
		plaintext []byte
	}{
		{"simple text", []byte("hello, world")},
		{"empty plaintext", []byte{}},
		{"binary data", []byte{0x00, 0x01, 0xFE, 0xFF}},
		{"unicode", []byte("こんにちは 🔑")},
		{"large data", bytes.Repeat([]byte("A"), 1024*64)},
	}

	key := validKey(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ciphertext, err := Encrypt(key, tt.plaintext)
			if err != nil {
				t.Fatalf("Encrypt: %v", err)
			}
			got, err := Decrypt(key, ciphertext)
			if err != nil {
				t.Fatalf("Decrypt: %v", err)
			}
			if !bytes.Equal(got, tt.plaintext) {
				t.Errorf("round-trip mismatch: want %q, got %q", tt.plaintext, got)
			}
		})
	}
}

func TestEncryptDecrypt_DifferentCiphertexts(t *testing.T) {
	// Two encryptions of the same plaintext must differ (random nonce).
	key := validKey(t)
	c1, _ := Encrypt(key, []byte("secret"))
	c2, _ := Encrypt(key, []byte("secret"))
	if bytes.Equal(c1, c2) {
		t.Error("identical ciphertexts: nonce is not random")
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	key := validKey(t)
	ciphertext, _ := Encrypt(key, []byte("secret data"))

	wrongKey, _, _ := DeriveKey([]byte("wrong-passphrase"), bytes.Repeat([]byte{0xCC}, saltLen))
	_, err := Decrypt(wrongKey, ciphertext)
	if err != ErrDecryptionFailed {
		t.Errorf("expected ErrDecryptionFailed, got %v", err)
	}
}

func TestDecrypt_TamperedCiphertext(t *testing.T) {
	key := validKey(t)
	ciphertext, _ := Encrypt(key, []byte("secret data"))

	// Flip a bit in the ciphertext body (after the nonce).
	tampered := make([]byte, len(ciphertext))
	copy(tampered, ciphertext)
	tampered[nonceLen+1] ^= 0xFF

	_, err := Decrypt(key, tampered)
	if err != ErrDecryptionFailed {
		t.Errorf("expected ErrDecryptionFailed for tampered data, got %v", err)
	}
}

func TestDecrypt_TruncatedInput(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"nil input", nil},
		{"empty input", []byte{}},
		{"too short (nonce only)", bytes.Repeat([]byte{0x01}, nonceLen-1)},
	}
	key := validKey(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decrypt(key, tt.data)
			if err != ErrInvalidCiphertext {
				t.Errorf("expected ErrInvalidCiphertext, got %v", err)
			}
		})
	}
}

func TestDecrypt_InvalidKeyLength(t *testing.T) {
	_, err := Decrypt([]byte("short"), []byte("nonce+ciphertext"))
	if err == nil {
		t.Error("expected error for invalid key length")
	}
}

// ---------------------------------------------------------------------------
// EncryptWithPassphrase / DecryptWithPassphrase
// ---------------------------------------------------------------------------

func TestEncryptDecryptWithPassphrase_RoundTrip(t *testing.T) {
	tests := []struct {
		name       string
		passphrase []byte
		plaintext  []byte
	}{
		{"normal", []byte("my-passphrase"), []byte("super secret value")},
		{"empty plaintext", []byte("pass"), []byte{}},
		{"unicode passphrase", []byte("pässwörñ"), []byte("value")},
		{"binary content", []byte("pass"), []byte{0x00, 0xFF, 0xAB, 0xCD}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ciphertext, err := EncryptWithPassphrase(tt.passphrase, tt.plaintext)
			if err != nil {
				t.Fatalf("EncryptWithPassphrase: %v", err)
			}
			got, err := DecryptWithPassphrase(tt.passphrase, ciphertext)
			if err != nil {
				t.Fatalf("DecryptWithPassphrase: %v", err)
			}
			if !bytes.Equal(got, tt.plaintext) {
				t.Errorf("round-trip mismatch: want %q, got %q", tt.plaintext, got)
			}
		})
	}
}

func TestDecryptWithPassphrase_WrongPassphrase(t *testing.T) {
	ciphertext, err := EncryptWithPassphrase([]byte("correct"), []byte("secret"))
	if err != nil {
		t.Fatalf("EncryptWithPassphrase: %v", err)
	}
	_, err = DecryptWithPassphrase([]byte("wrong"), ciphertext)
	if err != ErrDecryptionFailed {
		t.Errorf("expected ErrDecryptionFailed, got %v", err)
	}
}

func TestDecryptWithPassphrase_TruncatedInput(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"nil", nil},
		{"empty", []byte{}},
		{"only salt", bytes.Repeat([]byte{0x01}, saltLen)},
		{"salt + partial nonce", bytes.Repeat([]byte{0x01}, saltLen+nonceLen-1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecryptWithPassphrase([]byte("pass"), tt.data)
			if err != ErrInvalidCiphertext {
				t.Errorf("expected ErrInvalidCiphertext, got %v", err)
			}
		})
	}
}

func TestEncryptWithPassphrase_SaltIsEmbedded(t *testing.T) {
	// Ciphertext must be at least saltLen + nonceLen + AEAD tag bytes.
	ct, err := EncryptWithPassphrase([]byte("pass"), []byte("v"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	minLen := saltLen + nonceLen
	if len(ct) <= minLen {
		t.Errorf("ciphertext too short: got %d, want > %d", len(ct), minLen)
	}
}

// ---------------------------------------------------------------------------
// Fingerprint
// ---------------------------------------------------------------------------

func TestFingerprint(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{"simple value", []byte("my-api-key-12345")},
		{"empty value", []byte{}},
		{"binary", []byte{0x00, 0xFF}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp := Fingerprint(tt.input)
			if len(fp) != 8 {
				t.Errorf("fingerprint length: want 8, got %d", len(fp))
			}
			// Must be valid hex.
			for _, c := range fp {
				if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
					t.Errorf("fingerprint contains non-hex char: %c", c)
				}
			}
		})
	}
}

func TestFingerprint_Deterministic(t *testing.T) {
	v := []byte("consistent-secret")
	if Fingerprint(v) != Fingerprint(v) {
		t.Error("fingerprint must be deterministic")
	}
}

func TestFingerprint_DifferentInputs(t *testing.T) {
	if Fingerprint([]byte("abc")) == Fingerprint([]byte("xyz")) {
		t.Error("different values should (almost certainly) have different fingerprints")
	}
}

// ---------------------------------------------------------------------------
// SecureWipe
// ---------------------------------------------------------------------------

func TestSecureWipe_ZeroesBytes(t *testing.T) {
	data := []byte("sensitive-secret-value")
	SecureWipe(data)
	for i, b := range data {
		if b != 0 {
			t.Errorf("byte at index %d is not zero after wipe: %d", i, b)
		}
	}
}

func TestSecureWipe_EmptySlice(t *testing.T) {
	// Should not panic.
	SecureWipe([]byte{})
}

func TestSecureWipe_NilSlice(t *testing.T) {
	// Should not panic.
	SecureWipe(nil)
}

func TestSecureWipe_PreservesLength(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	SecureWipe(data)
	if len(data) != 5 {
		t.Errorf("length changed after wipe: want 5, got %d", len(data))
	}
}

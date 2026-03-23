package secrets

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// newTestEncryptedBackend creates a fresh EncryptedFileBackend in a temp dir.
// The returned filePath is the secrets file path.
func newTestEncryptedBackend(t *testing.T, passphrase string) (*EncryptedFileBackend, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "secrets.enc")
	b, err := NewEncryptedFileBackend(path, []byte(passphrase))
	if err != nil {
		t.Fatalf("NewEncryptedFileBackend: %v", err)
	}
	return b, path
}

// ---------------------------------------------------------------------------
// Basic Set / Get round-trip
// ---------------------------------------------------------------------------

func TestEncryptedFileBackend_SetGetRoundTrip(t *testing.T) {
	b, _ := newTestEncryptedBackend(t, "test-passphrase")
	ctx := context.Background()

	key := "global/openai_api_key"
	value := []byte("sk-1234567890abcdef")

	if err := b.Set(ctx, key, value); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := b.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if !bytes.Equal(got, value) {
		t.Errorf("Get returned %q, want %q", got, value)
	}
}

func TestEncryptedFileBackend_GetReturnsIsolatedCopy(t *testing.T) {
	b, _ := newTestEncryptedBackend(t, "pass")
	ctx := context.Background()

	original := []byte("my-secret-value")
	if err := b.Set(ctx, "key", original); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, _ := b.Get(ctx, "key")
	// Mutate the returned copy — cached value must not change.
	got[0] = 'X'

	got2, _ := b.Get(ctx, "key")
	if got2[0] == 'X' {
		t.Error("Get should return an independent copy; mutation of returned value affected cache")
	}
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestEncryptedFileBackend_Delete(t *testing.T) {
	b, _ := newTestEncryptedBackend(t, "pass")
	ctx := context.Background()

	if err := b.Set(ctx, "mykey", []byte("value")); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := b.Delete(ctx, "mykey"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := b.Get(ctx, "mykey")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestEncryptedFileBackend_DeleteNotFound(t *testing.T) {
	b, _ := newTestEncryptedBackend(t, "pass")
	ctx := context.Background()

	err := b.Delete(ctx, "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound for missing key, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestEncryptedFileBackend_List(t *testing.T) {
	b, _ := newTestEncryptedBackend(t, "pass")
	ctx := context.Background()

	keys := []string{"global/key_a", "global/key_b", "project/key_c"}
	for _, k := range keys {
		if err := b.Set(ctx, k, []byte("v")); err != nil {
			t.Fatalf("Set %s: %v", k, err)
		}
	}

	list, err := b.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != len(keys) {
		t.Errorf("List length: want %d, got %d", len(keys), len(list))
	}

	listed := map[string]bool{}
	for _, k := range list {
		listed[k] = true
	}
	for _, want := range keys {
		if !listed[want] {
			t.Errorf("key %q not in List output", want)
		}
	}
}

func TestEncryptedFileBackend_ListEmpty(t *testing.T) {
	b, _ := newTestEncryptedBackend(t, "pass")
	ctx := context.Background()

	list, err := b.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d entries", len(list))
	}
}

// ---------------------------------------------------------------------------
// Lock — wipes memory and blocks Get
// ---------------------------------------------------------------------------

func TestEncryptedFileBackend_LockWipesCache(t *testing.T) {
	b, _ := newTestEncryptedBackend(t, "pass")
	ctx := context.Background()

	if err := b.Set(ctx, "key", []byte("secret")); err != nil {
		t.Fatalf("Set: %v", err)
	}

	b.Lock()

	// After lock, Get must fail (cache is empty, no master key for decryption).
	_, err := b.Get(ctx, "key")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("after Lock, Get should return ErrNotFound, got %v", err)
	}
}

func TestEncryptedFileBackend_LockClearsAllKeys(t *testing.T) {
	b, _ := newTestEncryptedBackend(t, "pass")
	ctx := context.Background()

	for _, k := range []string{"a", "b", "c"} {
		if err := b.Set(ctx, k, []byte("v")); err != nil {
			t.Fatalf("Set: %v", err)
		}
	}

	b.Lock()

	list, err := b.List(ctx)
	if err != nil {
		t.Fatalf("List after Lock: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list after Lock, got %d entries", len(list))
	}
}

// ---------------------------------------------------------------------------
// Wrong passphrase on load
// ---------------------------------------------------------------------------

func TestEncryptedFileBackend_WrongPassphraseOnLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secrets.enc")

	// Create with correct passphrase and write data.
	b, err := NewEncryptedFileBackend(path, []byte("correct-passphrase"))
	if err != nil {
		t.Fatalf("NewEncryptedFileBackend (create): %v", err)
	}
	if err := b.Set(context.Background(), "key", []byte("value")); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Try to re-open with wrong passphrase.
	_, err = NewEncryptedFileBackend(path, []byte("wrong-passphrase"))
	if err == nil {
		t.Fatal("expected error when opening with wrong passphrase, got nil")
	}
}

// ---------------------------------------------------------------------------
// File persistence: close and re-open
// ---------------------------------------------------------------------------

func TestEncryptedFileBackend_Persistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secrets.enc")
	passphrase := "reopen-passphrase"

	// First session: create and write.
	b1, err := NewEncryptedFileBackend(path, []byte(passphrase))
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	ctx := context.Background()
	if err := b1.Set(ctx, "global/my_key", []byte("persisted-value")); err != nil {
		t.Fatalf("Set: %v", err)
	}
	// b1 goes out of scope here; file is already flushed.

	// Second session: re-open with same passphrase.
	b2, err := NewEncryptedFileBackend(path, []byte(passphrase))
	if err != nil {
		t.Fatalf("second open: %v", err)
	}

	got, err := b2.Get(ctx, "global/my_key")
	if err != nil {
		t.Fatalf("Get after reopen: %v", err)
	}
	if !bytes.Equal(got, []byte("persisted-value")) {
		t.Errorf("persisted value mismatch: want %q, got %q", "persisted-value", got)
	}
}

func TestEncryptedFileBackend_PersistenceDeletedKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secrets.enc")
	passphrase := "del-test"

	b1, err := NewEncryptedFileBackend(path, []byte(passphrase))
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	ctx := context.Background()

	if err := b1.Set(ctx, "key1", []byte("v1")); err != nil {
		t.Fatalf("Set key1: %v", err)
	}
	if err := b1.Set(ctx, "key2", []byte("v2")); err != nil {
		t.Fatalf("Set key2: %v", err)
	}
	if err := b1.Delete(ctx, "key1"); err != nil {
		t.Fatalf("Delete key1: %v", err)
	}

	b2, err := NewEncryptedFileBackend(path, []byte(passphrase))
	if err != nil {
		t.Fatalf("second open: %v", err)
	}

	_, err = b2.Get(ctx, "key1")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("deleted key1 should not be found after reopen, got %v", err)
	}

	val, err := b2.Get(ctx, "key2")
	if err != nil {
		t.Fatalf("key2 should still exist: %v", err)
	}
	if !bytes.Equal(val, []byte("v2")) {
		t.Errorf("key2 value mismatch: want v2, got %q", val)
	}
}

// ---------------------------------------------------------------------------
// Available
// ---------------------------------------------------------------------------

func TestEncryptedFileBackend_Available(t *testing.T) {
	b, _ := newTestEncryptedBackend(t, "pass")
	if !b.Available() {
		t.Error("Available should return true when dir exists")
	}
}

// ---------------------------------------------------------------------------
// Name
// ---------------------------------------------------------------------------

func TestEncryptedFileBackend_Name(t *testing.T) {
	b, _ := newTestEncryptedBackend(t, "pass")
	if b.Name() != "encrypted_file" {
		t.Errorf("Name: want encrypted_file, got %q", b.Name())
	}
}

// ---------------------------------------------------------------------------
// Concurrent access safety
// ---------------------------------------------------------------------------

func TestEncryptedFileBackend_ConcurrentAccess(t *testing.T) {
	b, _ := newTestEncryptedBackend(t, "concurrent-pass")
	ctx := context.Background()

	const goroutines = 10
	const keysPerGoroutine = 5

	var wg sync.WaitGroup
	errs := make(chan error, goroutines*keysPerGoroutine*3)

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for k := 0; k < keysPerGoroutine; k++ {
				key := "concurrent-key"
				val := []byte("value")

				if err := b.Set(ctx, key, val); err != nil {
					errs <- err
					return
				}
				if _, err := b.Get(ctx, key); err != nil && !errors.Is(err, ErrNotFound) {
					errs <- err
					return
				}
				if _, err := b.List(ctx); err != nil {
					errs <- err
					return
				}
			}
		}(g)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent access error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Empty key name edge case
// ---------------------------------------------------------------------------

func TestEncryptedFileBackend_GetMissingKey(t *testing.T) {
	b, _ := newTestEncryptedBackend(t, "pass")
	ctx := context.Background()

	_, err := b.Get(ctx, "does-not-exist")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Overwrite existing key
// ---------------------------------------------------------------------------

func TestEncryptedFileBackend_OverwriteKey(t *testing.T) {
	b, _ := newTestEncryptedBackend(t, "pass")
	ctx := context.Background()

	if err := b.Set(ctx, "key", []byte("original")); err != nil {
		t.Fatalf("first Set: %v", err)
	}
	if err := b.Set(ctx, "key", []byte("updated")); err != nil {
		t.Fatalf("second Set: %v", err)
	}

	got, err := b.Get(ctx, "key")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !bytes.Equal(got, []byte("updated")) {
		t.Errorf("want updated, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// File-too-short triggers error on open
// ---------------------------------------------------------------------------

func TestEncryptedFileBackend_FileTooShort(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "short.enc")

	// Write a file shorter than saltLen bytes.
	if err := os.WriteFile(path, []byte{0x01, 0x02}, 0600); err != nil {
		t.Fatalf("write short file: %v", err)
	}

	_, err := NewEncryptedFileBackend(path, []byte("pass"))
	if err == nil {
		t.Error("expected error for file that is too short, got nil")
	}
}

// ---------------------------------------------------------------------------
// Flush error path: Set after Lock
// ---------------------------------------------------------------------------

func TestEncryptedFileBackend_SetAfterLock_ReturnsError(t *testing.T) {
	b, _ := newTestEncryptedBackend(t, "pass")
	ctx := context.Background()

	b.Lock()

	// After Lock, masterKey is nil; flush should return ErrLocked.
	err := b.Set(ctx, "key", []byte("value"))
	if err == nil {
		t.Error("expected error when Set is called after Lock")
	}
}

// ---------------------------------------------------------------------------
// Unicode / special characters in values
// ---------------------------------------------------------------------------

func TestEncryptedFileBackend_UnicodeValue(t *testing.T) {
	b, _ := newTestEncryptedBackend(t, "pass")
	ctx := context.Background()

	value := []byte("こんにちは 🔑 café résumé")
	if err := b.Set(ctx, "unicode_key", value); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := b.Get(ctx, "unicode_key")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !bytes.Equal(got, value) {
		t.Errorf("unicode round-trip failed: want %q, got %q", value, got)
	}
}

func TestEncryptedFileBackend_BinaryValue(t *testing.T) {
	b, _ := newTestEncryptedBackend(t, "pass")
	ctx := context.Background()

	// Binary value including null bytes and high bytes.
	value := []byte{0x00, 0x01, 0xFE, 0xFF, 0x00}
	if err := b.Set(ctx, "binary_key", value); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := b.Get(ctx, "binary_key")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !bytes.Equal(got, value) {
		t.Errorf("binary round-trip failed: want %v, got %v", value, got)
	}
}

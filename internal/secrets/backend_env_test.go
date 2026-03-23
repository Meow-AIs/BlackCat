package secrets

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Name / Available
// ---------------------------------------------------------------------------

func TestEnvBackend_Name(t *testing.T) {
	b := NewEnvBackend()
	if b.Name() != "env" {
		t.Errorf("Name: want env, got %q", b.Name())
	}
}

func TestEnvBackend_Available(t *testing.T) {
	b := NewEnvBackend()
	if !b.Available() {
		t.Error("EnvBackend.Available should always return true")
	}
}

// ---------------------------------------------------------------------------
// Get: env var present → returns value
// ---------------------------------------------------------------------------

func TestEnvBackend_Get_Present(t *testing.T) {
	b := NewEnvBackend()
	ctx := context.Background()

	// The key "global/my_key" maps to BLACKCAT_SECRET_GLOBAL/MY_KEY — but the
	// envVarName function uppercases and replaces '-' with '_', not '/'.
	// Let's verify exact mapping: envVarName("openai_api_key") → BLACKCAT_SECRET_OPENAI_API_KEY
	// Secret name is passed to Get as a raw storage key (scope/name).
	// envVarName normalises: upper case + replace '-' with '_'.
	// So "global/openai_api_key" → BLACKCAT_SECRET_GLOBAL/OPENAI_API_KEY — '/' stays.
	// Test a key without scope prefix to match exact env var name.
	t.Setenv("BLACKCAT_SECRET_MY_TEST_KEY", "my-secret-value")

	// The raw key passed to Get is the storage key, envVarName uppercases it.
	// "my_test_key" → BLACKCAT_SECRET_MY_TEST_KEY
	val, err := b.Get(ctx, "my_test_key")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(val) != "my-secret-value" {
		t.Errorf("Get: want %q, got %q", "my-secret-value", val)
	}
}

func TestEnvBackend_Get_WithDash(t *testing.T) {
	b := NewEnvBackend()
	ctx := context.Background()

	// Dashes in key names should be converted to underscores.
	t.Setenv("BLACKCAT_SECRET_SOME_KEY", "dash-value")

	val, err := b.Get(ctx, "some-key")
	if err != nil {
		t.Fatalf("Get with dash: %v", err)
	}
	if string(val) != "dash-value" {
		t.Errorf("Get with dash: want %q, got %q", "dash-value", val)
	}
}

// ---------------------------------------------------------------------------
// Get: env var missing → ErrNotFound
// ---------------------------------------------------------------------------

func TestEnvBackend_Get_Missing(t *testing.T) {
	b := NewEnvBackend()
	ctx := context.Background()

	_, err := b.Get(ctx, "definitely_not_set_xyz_123")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Set is a no-op (returns nil, does not set env var)
// ---------------------------------------------------------------------------

func TestEnvBackend_Set_IsNoop(t *testing.T) {
	b := NewEnvBackend()
	ctx := context.Background()

	err := b.Set(ctx, "some_key", []byte("value"))
	if err != nil {
		t.Errorf("Set should return nil (no-op), got %v", err)
	}

	// The env var should NOT have been set.
	_, getErr := b.Get(ctx, "some_key")
	if !errors.Is(getErr, ErrNotFound) {
		t.Error("Set should not persist the value to env; Get should still return ErrNotFound")
	}
}

// ---------------------------------------------------------------------------
// Delete is a no-op (returns nil, does not clear env var)
// ---------------------------------------------------------------------------

func TestEnvBackend_Delete_IsNoop(t *testing.T) {
	b := NewEnvBackend()
	ctx := context.Background()

	t.Setenv("BLACKCAT_SECRET_NOOP_DEL", "keep-me")

	err := b.Delete(ctx, "noop_del")
	if err != nil {
		t.Errorf("Delete should return nil (no-op), got %v", err)
	}

	// Env var should still be readable.
	val, getErr := b.Get(ctx, "noop_del")
	if getErr != nil {
		t.Fatalf("Get after no-op Delete: %v", getErr)
	}
	if string(val) != "keep-me" {
		t.Errorf("value should persist after no-op Delete, got %q", val)
	}
}

// ---------------------------------------------------------------------------
// List returns all BLACKCAT_SECRET_* keys
// ---------------------------------------------------------------------------

func TestEnvBackend_List_ReturnsMatchingKeys(t *testing.T) {
	b := NewEnvBackend()
	ctx := context.Background()

	t.Setenv("BLACKCAT_SECRET_LIST_A", "val_a")
	t.Setenv("BLACKCAT_SECRET_LIST_B", "val_b")
	t.Setenv("BLACKCAT_SECRET_LIST_C", "val_c")
	// Non-prefixed var — must NOT appear.
	t.Setenv("SOME_OTHER_VAR", "ignored")

	list, err := b.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	found := map[string]bool{}
	for _, k := range list {
		found[k] = true
	}

	// Keys are returned as lowercase without the prefix.
	for _, want := range []string{"list_a", "list_b", "list_c"} {
		if !found[want] {
			t.Errorf("List missing key %q", want)
		}
	}
	if found["some_other_var"] {
		t.Error("SOME_OTHER_VAR should not appear in List")
	}
}

func TestEnvBackend_List_NoMatchingKeys(t *testing.T) {
	// Unset any BLACKCAT_SECRET_* vars that may exist in the test environment.
	// We can't unset all of them portably, but we can verify the prefix logic.
	b := NewEnvBackend()
	ctx := context.Background()

	list, err := b.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	// All returned keys must have come from BLACKCAT_SECRET_* prefix.
	for _, k := range list {
		if strings.HasPrefix(k, envPrefix) {
			t.Errorf("List returned key with prefix still attached: %q", k)
		}
	}
}

func TestEnvBackend_List_KeysAreLowercase(t *testing.T) {
	b := NewEnvBackend()
	ctx := context.Background()

	t.Setenv("BLACKCAT_SECRET_UPPERCASE_KEY", "v")

	list, err := b.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	for _, k := range list {
		if k != strings.ToLower(k) {
			t.Errorf("List key %q is not lowercase", k)
		}
	}
}

// ---------------------------------------------------------------------------
// envVarName helper
// ---------------------------------------------------------------------------

func TestEnvVarName(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{"openai_api_key", "BLACKCAT_SECRET_OPENAI_API_KEY"},
		{"some-key", "BLACKCAT_SECRET_SOME_KEY"},
		{"mixed_CASE", "BLACKCAT_SECRET_MIXED_CASE"},
		{"global/my_secret", "BLACKCAT_SECRET_GLOBAL/MY_SECRET"},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := envVarName(tt.key)
			if got != tt.want {
				t.Errorf("envVarName(%q): want %q, got %q", tt.key, tt.want, got)
			}
		})
	}
}

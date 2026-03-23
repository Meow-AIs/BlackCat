package secrets

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// In-memory mock Backend (distinct name to avoid clash with importer_test.go)
// ---------------------------------------------------------------------------

type mgrTestBackend struct {
	mu        sync.RWMutex
	data      map[string][]byte
	available bool
}

func newMgrBackend() *mgrTestBackend {
	return &mgrTestBackend{data: make(map[string][]byte), available: true}
}

func (m *mgrTestBackend) Name() string { return "mgr-mem" }
func (m *mgrTestBackend) Available() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.available
}

func (m *mgrTestBackend) Get(_ context.Context, key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.data[key]
	if !ok {
		return nil, ErrNotFound
	}
	cp := make([]byte, len(v))
	copy(cp, v)
	return cp, nil
}

func (m *mgrTestBackend) Set(_ context.Context, key string, value []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]byte, len(value))
	copy(cp, value)
	m.data[key] = cp
	return nil
}

func (m *mgrTestBackend) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[key]; !ok {
		return ErrNotFound
	}
	delete(m.data, key)
	return nil
}

func (m *mgrTestBackend) List(_ context.Context) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	return keys, nil
}

// ---------------------------------------------------------------------------
// In-memory mock MetadataStore
// ---------------------------------------------------------------------------

type mgrMetaStore struct {
	mu   sync.RWMutex
	data map[string]SecretMetadata // key: "<scope>/<name>"
}

func newMgrMetaStore() *mgrMetaStore {
	return &mgrMetaStore{data: make(map[string]SecretMetadata)}
}

func mgrMetaKey(name string, scope Scope) string {
	return string(scope) + "/" + name
}

func (s *mgrMetaStore) SaveMeta(_ context.Context, meta SecretMetadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[mgrMetaKey(meta.Name, meta.Scope)] = meta
	return nil
}

func (s *mgrMetaStore) GetMeta(_ context.Context, name string, scope Scope) (SecretMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.data[mgrMetaKey(name, scope)]
	if !ok {
		return SecretMetadata{}, ErrNotFound
	}
	return m, nil
}

func (s *mgrMetaStore) DeleteMeta(_ context.Context, name string, scope Scope) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := mgrMetaKey(name, scope)
	if _, ok := s.data[k]; !ok {
		return ErrNotFound
	}
	delete(s.data, k)
	return nil
}

func (s *mgrMetaStore) ListMeta(_ context.Context, scope Scope) ([]SecretMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []SecretMetadata
	prefix := string(scope) + "/"
	for k, v := range s.data {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			result = append(result, v)
		}
	}
	return result, nil
}

func (s *mgrMetaStore) ListExpiring(_ context.Context, withinDays int) ([]ExpiryStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := time.Now()
	deadline := now.AddDate(0, 0, withinDays)
	var result []ExpiryStatus
	for _, m := range s.data {
		if m.ExpiresAt.IsZero() {
			continue
		}
		if m.ExpiresAt.Before(deadline) || m.ExpiresAt.Equal(deadline) {
			daysLeft := int(time.Until(m.ExpiresAt).Hours() / 24)
			result = append(result, ExpiryStatus{
				SecretRef: SecretRef{Name: m.Name, Scope: m.Scope},
				ExpiresAt: m.ExpiresAt,
				DaysLeft:  daysLeft,
				IsExpired: m.ExpiresAt.Before(now),
			})
		}
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// In-memory mock AuditLog
// ---------------------------------------------------------------------------

type mgrAuditLog struct {
	mu      sync.Mutex
	entries []AuditEntry
}

func (a *mgrAuditLog) Log(_ context.Context, entry AuditEntry) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.entries = append(a.entries, entry)
	return nil
}

func (a *mgrAuditLog) Query(_ context.Context, f AuditFilter) ([]AuditEntry, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	var result []AuditEntry
	for _, e := range a.entries {
		if f.SecretName != "" && e.SecretRef.Name != f.SecretName {
			continue
		}
		result = append(result, e)
	}
	limit := f.Limit
	if limit == 0 {
		limit = 100
	}
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newMgrTestManager(t *testing.T) (*Manager, *mgrTestBackend, *mgrMetaStore, *mgrAuditLog) {
	t.Helper()
	be := newMgrBackend()
	ms := newMgrMetaStore()
	al := &mgrAuditLog{}
	mgr, err := NewManager(ManagerOpts{
		Backends:      []Backend{be},
		MetadataStore: ms,
		AuditLog:      al,
	})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	return mgr, be, ms, al
}

func mgrSetSecret(t *testing.T, mgr *Manager, name string, scope Scope, value []byte) {
	t.Helper()
	meta := SecretMetadata{
		Name:  name,
		Scope: scope,
		Type:  TypeAPIKey,
	}
	if err := mgr.Set(context.Background(), meta, value); err != nil {
		t.Fatalf("Set(%q): %v", name, err)
	}
}

// ---------------------------------------------------------------------------
// NewManager
// ---------------------------------------------------------------------------

func TestNewManager_RequiresMetadataStore(t *testing.T) {
	be := newMgrBackend()
	_, err := NewManager(ManagerOpts{
		Backends:      []Backend{be},
		MetadataStore: nil,
	})
	if err == nil {
		t.Error("expected error when MetadataStore is nil")
	}
}

func TestNewManager_RequiresAvailableBackend(t *testing.T) {
	be := newMgrBackend()
	be.available = false
	_, err := NewManager(ManagerOpts{
		Backends:      []Backend{be},
		MetadataStore: newMgrMetaStore(),
	})
	if err == nil {
		t.Error("expected error when no backend is available")
	}
}

func TestNewManager_PicksFirstAvailableBackend(t *testing.T) {
	unavailable := newMgrBackend()
	unavailable.available = false
	available := newMgrBackend()

	mgr, err := NewManager(ManagerOpts{
		Backends:      []Backend{unavailable, available},
		MetadataStore: newMgrMetaStore(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mgr.primary != available {
		t.Error("manager should pick the first available backend")
	}
}

// ---------------------------------------------------------------------------
// Manager.Set
// ---------------------------------------------------------------------------

func TestManager_Set_InvalidName(t *testing.T) {
	mgr, _, _, _ := newMgrTestManager(t)
	tests := []struct {
		name  string
		label string
	}{
		{"", "empty name"},
		{"A", "single char uppercase"},
		{"UPPER_CASE", "all uppercase"},
		{"-leading", "leading hyphen"},
		{"trailing-", "trailing hyphen"},
		{"has space", "space in name"},
		{"x", "single lowercase char — too short for regex"},
	}
	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			meta := SecretMetadata{Name: tt.name, Scope: ScopeGlobal}
			err := mgr.Set(context.Background(), meta, []byte("value"))
			if !errors.Is(err, ErrInvalidName) {
				t.Errorf("expected ErrInvalidName, got %v", err)
			}
		})
	}
}

func TestManager_Set_ValidName(t *testing.T) {
	mgr, _, _, _ := newMgrTestManager(t)
	tests := []struct{ name string }{
		{"ab"},
		{"openai-api-key"},
		{"my.secret.value"},
		{"secret-123"},
		{"a1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := SecretMetadata{Name: tt.name, Scope: ScopeGlobal}
			err := mgr.Set(context.Background(), meta, []byte("value"))
			if err != nil {
				t.Errorf("unexpected error for name %q: %v", tt.name, err)
			}
		})
	}
}

func TestManager_Set_SetsFingerprint(t *testing.T) {
	mgr, _, ms, _ := newMgrTestManager(t)
	value := []byte("my-secret-value")
	meta := SecretMetadata{Name: "my-secret", Scope: ScopeGlobal}
	if err := mgr.Set(context.Background(), meta, value); err != nil {
		t.Fatalf("Set: %v", err)
	}
	saved, err := ms.GetMeta(context.Background(), "my-secret", ScopeGlobal)
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	want := Fingerprint(value)
	if saved.Fingerprint != want {
		t.Errorf("fingerprint: want %q, got %q", want, saved.Fingerprint)
	}
}

func TestManager_Set_SetsTimestamps(t *testing.T) {
	mgr, _, ms, _ := newMgrTestManager(t)
	before := time.Now()
	meta := SecretMetadata{Name: "ts-secret", Scope: ScopeGlobal}
	if err := mgr.Set(context.Background(), meta, []byte("v")); err != nil {
		t.Fatalf("Set: %v", err)
	}
	after := time.Now()

	saved, _ := ms.GetMeta(context.Background(), "ts-secret", ScopeGlobal)
	if saved.CreatedAt.Before(before) || saved.CreatedAt.After(after) {
		t.Errorf("CreatedAt out of range: %v", saved.CreatedAt)
	}
	if saved.UpdatedAt.Before(before) || saved.UpdatedAt.After(after) {
		t.Errorf("UpdatedAt out of range: %v", saved.UpdatedAt)
	}
}

// ---------------------------------------------------------------------------
// Manager.Get
// ---------------------------------------------------------------------------

func TestManager_Get_ReturnsValue(t *testing.T) {
	mgr, _, _, _ := newMgrTestManager(t)
	want := []byte("my-secret-value")
	mgrSetSecret(t, mgr, "my-secret", ScopeGlobal, want)

	got, err := mgr.Get(context.Background(), "my-secret", ScopeGlobal)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("Get: want %q, got %q", want, got)
	}
}

func TestManager_Get_NotFound_NoMetadata(t *testing.T) {
	mgr, _, _, _ := newMgrTestManager(t)
	_, err := mgr.Get(context.Background(), "no-meta", ScopeGlobal)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestManager_Get_NotFound_MetadataExistsNoValue(t *testing.T) {
	mgr, _, ms, _ := newMgrTestManager(t)
	// Save metadata without a corresponding backend value.
	_ = ms.SaveMeta(context.Background(), SecretMetadata{
		Name:  "meta-only",
		Scope: ScopeGlobal,
	})
	_, err := mgr.Get(context.Background(), "meta-only", ScopeGlobal)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound when value missing from backend, got %v", err)
	}
}

func TestManager_Get_InvalidName(t *testing.T) {
	mgr, _, _, _ := newMgrTestManager(t)
	_, err := mgr.Get(context.Background(), "INVALID NAME", ScopeGlobal)
	if !errors.Is(err, ErrInvalidName) {
		t.Errorf("expected ErrInvalidName, got %v", err)
	}
}

func TestManager_Get_ExpiredSecret(t *testing.T) {
	mgr, _, ms, _ := newMgrTestManager(t)
	_ = ms.SaveMeta(context.Background(), SecretMetadata{
		Name:      "expired-secret",
		Scope:     ScopeGlobal,
		ExpiresAt: time.Now().Add(-time.Hour),
	})
	_, err := mgr.Get(context.Background(), "expired-secret", ScopeGlobal)
	if !errors.Is(err, ErrExpired) {
		t.Errorf("expected ErrExpired, got %v", err)
	}
}

func TestManager_Get_NotYetExpiredSecret(t *testing.T) {
	mgr, _, _, _ := newMgrTestManager(t)
	meta := SecretMetadata{
		Name:      "not-yet-expired",
		Scope:     ScopeGlobal,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := mgr.Set(context.Background(), meta, []byte("valid-value")); err != nil {
		t.Fatalf("Set: %v", err)
	}
	_, err := mgr.Get(context.Background(), "not-yet-expired", ScopeGlobal)
	if err != nil {
		t.Errorf("expected success for not-yet-expired secret, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Manager.Delete
// ---------------------------------------------------------------------------

func TestManager_Delete_RemovesSecret(t *testing.T) {
	mgr, _, _, _ := newMgrTestManager(t)
	mgrSetSecret(t, mgr, "to-delete", ScopeGlobal, []byte("value"))

	if err := mgr.Delete(context.Background(), "to-delete", ScopeGlobal); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	exists, _ := mgr.Exists(context.Background(), "to-delete", ScopeGlobal)
	if exists {
		t.Error("secret should not exist after deletion")
	}
}

func TestManager_Delete_InvalidName(t *testing.T) {
	mgr, _, _, _ := newMgrTestManager(t)
	err := mgr.Delete(context.Background(), "INVALID", ScopeGlobal)
	if !errors.Is(err, ErrInvalidName) {
		t.Errorf("expected ErrInvalidName, got %v", err)
	}
}

func TestManager_Delete_NotFound(t *testing.T) {
	mgr, _, _, _ := newMgrTestManager(t)
	err := mgr.Delete(context.Background(), "nonexistent", ScopeGlobal)
	if err == nil {
		t.Error("expected error when deleting nonexistent secret")
	}
}

// ---------------------------------------------------------------------------
// Manager.List
// ---------------------------------------------------------------------------

func TestManager_List_ReturnsSecretsInScope(t *testing.T) {
	mgr, _, _, _ := newMgrTestManager(t)
	mgrSetSecret(t, mgr, "global-secret", ScopeGlobal, []byte("v1"))
	mgrSetSecret(t, mgr, "project-secret", ScopeProject, []byte("v2"))

	globals, err := mgr.List(context.Background(), ScopeGlobal)
	if err != nil {
		t.Fatalf("List globals: %v", err)
	}
	if len(globals) != 1 || globals[0].Name != "global-secret" {
		t.Errorf("expected 1 global secret, got %v", globals)
	}

	projects, err := mgr.List(context.Background(), ScopeProject)
	if err != nil {
		t.Fatalf("List projects: %v", err)
	}
	if len(projects) != 1 || projects[0].Name != "project-secret" {
		t.Errorf("expected 1 project secret, got %v", projects)
	}
}

func TestManager_List_EmptyScope(t *testing.T) {
	mgr, _, _, _ := newMgrTestManager(t)
	secrets, err := mgr.List(context.Background(), ScopeGlobal)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(secrets) != 0 {
		t.Errorf("expected empty list, got %v", secrets)
	}
}

// ---------------------------------------------------------------------------
// Manager.Exists
// ---------------------------------------------------------------------------

func TestManager_Exists_PresentAndAbsent(t *testing.T) {
	mgr, _, _, _ := newMgrTestManager(t)
	mgrSetSecret(t, mgr, "present-secret", ScopeGlobal, []byte("v"))

	exists, err := mgr.Exists(context.Background(), "present-secret", ScopeGlobal)
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if !exists {
		t.Error("expected secret to exist")
	}

	absent, err := mgr.Exists(context.Background(), "absent-secret", ScopeGlobal)
	if err != nil {
		t.Fatalf("Exists for absent: %v", err)
	}
	if absent {
		t.Error("expected secret to not exist")
	}
}

// ---------------------------------------------------------------------------
// Manager.CheckExpiry
// ---------------------------------------------------------------------------

func TestManager_CheckExpiry_WithinWindow(t *testing.T) {
	mgr, _, ms, _ := newMgrTestManager(t)
	now := time.Now()

	_ = ms.SaveMeta(context.Background(), SecretMetadata{
		Name:      "soon-expiring",
		Scope:     ScopeGlobal,
		ExpiresAt: now.AddDate(0, 0, 2),
	})
	_ = ms.SaveMeta(context.Background(), SecretMetadata{
		Name:      "later-expiring",
		Scope:     ScopeGlobal,
		ExpiresAt: now.AddDate(0, 0, 10),
	})
	_ = ms.SaveMeta(context.Background(), SecretMetadata{
		Name:      "already-expired",
		Scope:     ScopeGlobal,
		ExpiresAt: now.Add(-time.Hour),
	})

	statuses, err := mgr.CheckExpiry(context.Background(), 5)
	if err != nil {
		t.Fatalf("CheckExpiry: %v", err)
	}

	names := make([]string, 0, len(statuses))
	for _, s := range statuses {
		names = append(names, s.SecretRef.Name)
	}
	if !mgrContainsStr(names, "soon-expiring") {
		t.Error("expected soon-expiring in results")
	}
	if !mgrContainsStr(names, "already-expired") {
		t.Error("expected already-expired in results")
	}
	if mgrContainsStr(names, "later-expiring") {
		t.Error("later-expiring should not be in 5-day window")
	}
}

func mgrContainsStr(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Manager.Rotate
// ---------------------------------------------------------------------------

func TestManager_Rotate_UpdatesValue(t *testing.T) {
	mgr, _, _, _ := newMgrTestManager(t)
	mgrSetSecret(t, mgr, "rotating-secret", ScopeGlobal, []byte("old-value"))

	if err := mgr.Rotate(context.Background(), "rotating-secret", ScopeGlobal, []byte("new-value")); err != nil {
		t.Fatalf("Rotate: %v", err)
	}

	got, err := mgr.Get(context.Background(), "rotating-secret", ScopeGlobal)
	if err != nil {
		t.Fatalf("Get after rotate: %v", err)
	}
	if string(got) != "new-value" {
		t.Errorf("expected new-value, got %q", got)
	}
}

func TestManager_Rotate_UpdatesFingerprint(t *testing.T) {
	mgr, _, ms, _ := newMgrTestManager(t)
	mgrSetSecret(t, mgr, "fp-secret", ScopeGlobal, []byte("old-value"))

	newValue := []byte("brand-new-value")
	if err := mgr.Rotate(context.Background(), "fp-secret", ScopeGlobal, newValue); err != nil {
		t.Fatalf("Rotate: %v", err)
	}

	saved, _ := ms.GetMeta(context.Background(), "fp-secret", ScopeGlobal)
	if saved.Fingerprint != Fingerprint(newValue) {
		t.Error("fingerprint not updated after rotation")
	}
}

func TestManager_Rotate_ExtendsExpiryWithRotationPolicy(t *testing.T) {
	mgr, _, ms, _ := newMgrTestManager(t)
	meta := SecretMetadata{
		Name:         "rotation-policy",
		Scope:        ScopeGlobal,
		RotationDays: 30,
		ExpiresAt:    time.Now().Add(-24 * time.Hour),
	}
	if err := mgr.Set(context.Background(), meta, []byte("old")); err != nil {
		t.Fatalf("Set: %v", err)
	}

	before := time.Now()
	if err := mgr.Rotate(context.Background(), "rotation-policy", ScopeGlobal, []byte("new")); err != nil {
		t.Fatalf("Rotate: %v", err)
	}

	saved, _ := ms.GetMeta(context.Background(), "rotation-policy", ScopeGlobal)
	expected := before.AddDate(0, 0, 30)
	if saved.ExpiresAt.Before(expected.Add(-2*time.Second)) || saved.ExpiresAt.After(expected.Add(2*time.Second)) {
		t.Errorf("ExpiresAt not extended: got %v, expected ~%v", saved.ExpiresAt, expected)
	}
}

func TestManager_Rotate_NotFound(t *testing.T) {
	mgr, _, _, _ := newMgrTestManager(t)
	err := mgr.Rotate(context.Background(), "no-such-secret", ScopeGlobal, []byte("v"))
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Manager.AuditHistory
// ---------------------------------------------------------------------------

func TestManager_AuditHistory_RecordsAccess(t *testing.T) {
	mgr, _, _, _ := newMgrTestManager(t)
	mgrSetSecret(t, mgr, "audited-secret", ScopeGlobal, []byte("value"))
	_, _ = mgr.Get(context.Background(), "audited-secret", ScopeGlobal)

	entries, err := mgr.AuditHistory(context.Background(), "audited-secret", 100)
	if err != nil {
		t.Fatalf("AuditHistory: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected audit entries, got none")
	}
}

func TestManager_AuditHistory_NilAuditLog(t *testing.T) {
	be := newMgrBackend()
	mgr, err := NewManager(ManagerOpts{
		Backends:      []Backend{be},
		MetadataStore: newMgrMetaStore(),
		AuditLog:      nil,
	})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	entries, err := mgr.AuditHistory(context.Background(), "any", 10)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if entries != nil {
		t.Errorf("expected nil, got %v", entries)
	}
}

// ---------------------------------------------------------------------------
// Multi-backend fallback
// ---------------------------------------------------------------------------

func TestManager_Get_FallsBackToSecondBackend(t *testing.T) {
	primary := newMgrBackend()
	secondary := newMgrBackend()
	ms := newMgrMetaStore()

	// Secret only exists in the secondary backend.
	storKey := storageKey("fallback-secret", ScopeGlobal)
	_ = secondary.Set(context.Background(), storKey, []byte("from-secondary"))
	_ = ms.SaveMeta(context.Background(), SecretMetadata{
		Name:  "fallback-secret",
		Scope: ScopeGlobal,
	})

	mgr, err := NewManager(ManagerOpts{
		Backends:      []Backend{primary, secondary},
		MetadataStore: ms,
	})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	got, err := mgr.Get(context.Background(), "fallback-secret", ScopeGlobal)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != "from-secondary" {
		t.Errorf("expected from-secondary, got %q", got)
	}
}

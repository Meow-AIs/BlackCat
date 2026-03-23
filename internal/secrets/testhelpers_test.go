package secrets

// testhelpers_test.go provides shared in-memory test doubles used by
// access_control_test.go and other test files that need lightweight mocks
// without SQLite or OS dependencies.

import (
	"context"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// memMetaStore — in-memory MetadataStore
// ---------------------------------------------------------------------------

type memMetaStore struct {
	mu   sync.RWMutex
	data map[string]SecretMetadata
}

func newMemMetaStore() *memMetaStore {
	return &memMetaStore{data: make(map[string]SecretMetadata)}
}

func memMetaKey(name string, scope Scope) string {
	return string(scope) + "/" + name
}

func (s *memMetaStore) SaveMeta(_ context.Context, meta SecretMetadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[memMetaKey(meta.Name, meta.Scope)] = meta
	return nil
}

func (s *memMetaStore) GetMeta(_ context.Context, name string, scope Scope) (SecretMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.data[memMetaKey(name, scope)]
	if !ok {
		return SecretMetadata{}, ErrNotFound
	}
	return m, nil
}

func (s *memMetaStore) DeleteMeta(_ context.Context, name string, scope Scope) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := memMetaKey(name, scope)
	if _, ok := s.data[k]; !ok {
		return ErrNotFound
	}
	delete(s.data, k)
	return nil
}

func (s *memMetaStore) ListMeta(_ context.Context, scope Scope) ([]SecretMetadata, error) {
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

func (s *memMetaStore) ListExpiring(_ context.Context, withinDays int) ([]ExpiryStatus, error) {
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
// memAuditLog — in-memory AuditLog
// ---------------------------------------------------------------------------

type memAuditLog struct {
	mu      sync.Mutex
	entries []AuditEntry
}

func (a *memAuditLog) Log(_ context.Context, entry AuditEntry) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.entries = append(a.entries, entry)
	return nil
}

func (a *memAuditLog) Query(_ context.Context, f AuditFilter) ([]AuditEntry, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	var result []AuditEntry
	for _, e := range a.entries {
		if f.SecretName != "" && e.SecretRef.Name != f.SecretName {
			continue
		}
		if f.Action != "" && e.Action != f.Action {
			continue
		}
		if f.Actor != "" && e.Actor != f.Actor {
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
// setSecret — convenience helper for tests that use memMetaStore + memBackend
// ---------------------------------------------------------------------------

// setSecret stores a secret using Manager.Set with minimal required metadata.
// Produces a valid name (lowercase alpha+digits only) matching the validName regex.
func setSecret(t *testing.T, mgr *Manager, name string, scope Scope, value []byte) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	meta := SecretMetadata{
		Name:      name,
		Scope:     scope,
		Type:      TypeAPIKey,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := mgr.Set(context.Background(), meta, value); err != nil {
		t.Fatalf("setSecret(%q): %v", name, err)
	}
}

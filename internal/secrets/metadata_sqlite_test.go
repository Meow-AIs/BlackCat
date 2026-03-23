//go:build cgo

package secrets

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// openTestDB opens an in-memory SQLite database for tests.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// newTestStore creates a SQLiteMetadataStore backed by an in-memory DB.
func newTestStore(t *testing.T) *SQLiteMetadataStore {
	t.Helper()
	db := openTestDB(t)
	store, err := NewSQLiteMetadataStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteMetadataStore: %v", err)
	}
	return store
}

// sampleMeta returns a fully-populated SecretMetadata for use in tests.
func sampleMeta(name string, scope Scope) SecretMetadata {
	now := time.Now().UTC().Truncate(time.Second)
	return SecretMetadata{
		Name:          name,
		Scope:         scope,
		Type:          TypeAPIKey,
		ProjectPath:   "/home/user/project",
		Description:   "test secret",
		EnvVar:        "TEST_API_KEY",
		Tags:          []string{"llm", "openai"},
		CreatedAt:     now,
		UpdatedAt:     now,
		RotationDays:  30,
		AllowedTools:  []string{"shell", "http"},
		AllowedAgents: []string{"primary"},
		ImportedFrom:  "dotenv",
		Fingerprint:   "abcd1234",
	}
}

// ---------------------------------------------------------------------------
// SaveMeta + GetMeta round-trip
// ---------------------------------------------------------------------------

func TestSQLiteMetadataStore_SaveGetRoundTrip(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	meta := sampleMeta("openai_api_key", ScopeGlobal)
	if err := store.SaveMeta(ctx, meta); err != nil {
		t.Fatalf("SaveMeta: %v", err)
	}

	got, err := store.GetMeta(ctx, meta.Name, meta.Scope)
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}

	if got.Name != meta.Name {
		t.Errorf("Name: want %q, got %q", meta.Name, got.Name)
	}
	if got.Scope != meta.Scope {
		t.Errorf("Scope: want %q, got %q", meta.Scope, got.Scope)
	}
	if got.Type != meta.Type {
		t.Errorf("Type: want %q, got %q", meta.Type, got.Type)
	}
	if got.Description != meta.Description {
		t.Errorf("Description: want %q, got %q", meta.Description, got.Description)
	}
	if got.EnvVar != meta.EnvVar {
		t.Errorf("EnvVar: want %q, got %q", meta.EnvVar, got.EnvVar)
	}
	if got.RotationDays != meta.RotationDays {
		t.Errorf("RotationDays: want %d, got %d", meta.RotationDays, got.RotationDays)
	}
	if got.Fingerprint != meta.Fingerprint {
		t.Errorf("Fingerprint: want %q, got %q", meta.Fingerprint, got.Fingerprint)
	}
	if got.ImportedFrom != meta.ImportedFrom {
		t.Errorf("ImportedFrom: want %q, got %q", meta.ImportedFrom, got.ImportedFrom)
	}

	// Timestamps are stored at RFC3339 second precision.
	if !got.CreatedAt.Equal(meta.CreatedAt) {
		t.Errorf("CreatedAt: want %v, got %v", meta.CreatedAt, got.CreatedAt)
	}
	if !got.UpdatedAt.Equal(meta.UpdatedAt) {
		t.Errorf("UpdatedAt: want %v, got %v", meta.UpdatedAt, got.UpdatedAt)
	}

	// Tags and JSON arrays round-trip.
	if len(got.Tags) != len(meta.Tags) {
		t.Errorf("Tags length: want %d, got %d", len(meta.Tags), len(got.Tags))
	}
	if len(got.AllowedTools) != len(meta.AllowedTools) {
		t.Errorf("AllowedTools length: want %d, got %d", len(meta.AllowedTools), len(got.AllowedTools))
	}
	if len(got.AllowedAgents) != len(meta.AllowedAgents) {
		t.Errorf("AllowedAgents length: want %d, got %d", len(meta.AllowedAgents), len(got.AllowedAgents))
	}
}

func TestSQLiteMetadataStore_SaveUpdatesOnConflict(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	meta := sampleMeta("my_secret", ScopeGlobal)
	if err := store.SaveMeta(ctx, meta); err != nil {
		t.Fatalf("first SaveMeta: %v", err)
	}

	// Mutate and save again — should upsert, not fail.
	meta.Description = "updated description"
	meta.EnvVar = "NEW_ENV_VAR"
	meta.UpdatedAt = time.Now().UTC().Truncate(time.Second)

	if err := store.SaveMeta(ctx, meta); err != nil {
		t.Fatalf("second SaveMeta (upsert): %v", err)
	}

	got, err := store.GetMeta(ctx, meta.Name, meta.Scope)
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if got.Description != "updated description" {
		t.Errorf("Description not updated: got %q", got.Description)
	}
	if got.EnvVar != "NEW_ENV_VAR" {
		t.Errorf("EnvVar not updated: got %q", got.EnvVar)
	}
}

func TestSQLiteMetadataStore_GetNotFound(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.GetMeta(ctx, "nonexistent", ScopeGlobal)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// ListMeta with scope filtering
// ---------------------------------------------------------------------------

func TestSQLiteMetadataStore_ListMetaScopeFilter(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Insert 2 global and 1 project-scoped secrets.
	globals := []string{"global_a", "global_b"}
	for _, name := range globals {
		if err := store.SaveMeta(ctx, sampleMeta(name, ScopeGlobal)); err != nil {
			t.Fatalf("SaveMeta %s: %v", name, err)
		}
	}
	if err := store.SaveMeta(ctx, sampleMeta("project_x", ScopeProject)); err != nil {
		t.Fatalf("SaveMeta project_x: %v", err)
	}

	list, err := store.ListMeta(ctx, ScopeGlobal)
	if err != nil {
		t.Fatalf("ListMeta global: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("global count: want 2, got %d", len(list))
	}

	list, err = store.ListMeta(ctx, ScopeProject)
	if err != nil {
		t.Fatalf("ListMeta project: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("project count: want 1, got %d", len(list))
	}
	if list[0].Name != "project_x" {
		t.Errorf("project name: want project_x, got %q", list[0].Name)
	}
}

func TestSQLiteMetadataStore_ListMetaEmpty(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	list, err := store.ListMeta(ctx, ScopeGlobal)
	if err != nil {
		t.Fatalf("ListMeta on empty store: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d entries", len(list))
	}
}

func TestSQLiteMetadataStore_ListMetaOrderedByName(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	names := []string{"zebra_key", "alpha_key", "middle_key"}
	for _, n := range names {
		if err := store.SaveMeta(ctx, sampleMeta(n, ScopeGlobal)); err != nil {
			t.Fatalf("SaveMeta: %v", err)
		}
	}

	list, err := store.ListMeta(ctx, ScopeGlobal)
	if err != nil {
		t.Fatalf("ListMeta: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 results, got %d", len(list))
	}
	// Should be alphabetically ordered.
	want := []string{"alpha_key", "middle_key", "zebra_key"}
	for i, w := range want {
		if list[i].Name != w {
			t.Errorf("list[%d]: want %q, got %q", i, w, list[i].Name)
		}
	}
}

// ---------------------------------------------------------------------------
// DeleteMeta
// ---------------------------------------------------------------------------

func TestSQLiteMetadataStore_Delete(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	meta := sampleMeta("deleteme", ScopeGlobal)
	if err := store.SaveMeta(ctx, meta); err != nil {
		t.Fatalf("SaveMeta: %v", err)
	}

	if err := store.DeleteMeta(ctx, meta.Name, meta.Scope); err != nil {
		t.Fatalf("DeleteMeta: %v", err)
	}

	_, err := store.GetMeta(ctx, meta.Name, meta.Scope)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("after delete, GetMeta should return ErrNotFound, got %v", err)
	}
}

func TestSQLiteMetadataStore_DeleteNotFound(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	err := store.DeleteMeta(ctx, "nonexistent", ScopeGlobal)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Duplicate names in different scopes
// ---------------------------------------------------------------------------

func TestSQLiteMetadataStore_SameName_DifferentScopes(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	globalMeta := sampleMeta("shared_secret", ScopeGlobal)
	globalMeta.Description = "global version"

	projectMeta := sampleMeta("shared_secret", ScopeProject)
	projectMeta.Description = "project version"

	if err := store.SaveMeta(ctx, globalMeta); err != nil {
		t.Fatalf("SaveMeta global: %v", err)
	}
	if err := store.SaveMeta(ctx, projectMeta); err != nil {
		t.Fatalf("SaveMeta project: %v", err)
	}

	gotGlobal, err := store.GetMeta(ctx, "shared_secret", ScopeGlobal)
	if err != nil {
		t.Fatalf("GetMeta global: %v", err)
	}
	if gotGlobal.Description != "global version" {
		t.Errorf("global description: want %q, got %q", "global version", gotGlobal.Description)
	}

	gotProject, err := store.GetMeta(ctx, "shared_secret", ScopeProject)
	if err != nil {
		t.Fatalf("GetMeta project: %v", err)
	}
	if gotProject.Description != "project version" {
		t.Errorf("project description: want %q, got %q", "project version", gotProject.Description)
	}
}

// ---------------------------------------------------------------------------
// Zero expiry (no expiry set)
// ---------------------------------------------------------------------------

func TestSQLiteMetadataStore_ZeroExpiry(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	meta := sampleMeta("no_expiry", ScopeGlobal)
	meta.ExpiresAt = time.Time{} // explicit zero — no expiry

	if err := store.SaveMeta(ctx, meta); err != nil {
		t.Fatalf("SaveMeta: %v", err)
	}

	got, err := store.GetMeta(ctx, meta.Name, meta.Scope)
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if !got.ExpiresAt.IsZero() {
		t.Errorf("ExpiresAt should be zero, got %v", got.ExpiresAt)
	}
}

func TestSQLiteMetadataStore_WithExpiry(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	expires := time.Now().UTC().Truncate(time.Second).Add(72 * time.Hour)
	meta := sampleMeta("will_expire", ScopeGlobal)
	meta.ExpiresAt = expires

	if err := store.SaveMeta(ctx, meta); err != nil {
		t.Fatalf("SaveMeta: %v", err)
	}

	got, err := store.GetMeta(ctx, meta.Name, meta.Scope)
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if !got.ExpiresAt.Equal(expires) {
		t.Errorf("ExpiresAt: want %v, got %v", expires, got.ExpiresAt)
	}
}

// ---------------------------------------------------------------------------
// ListExpiring
// ---------------------------------------------------------------------------

func TestSQLiteMetadataStore_ListExpiring(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	now := time.Now().UTC()

	// Expiring within 5 days — should appear when querying withinDays=7.
	expiringSoon := sampleMeta("soon_key", ScopeGlobal)
	expiringSoon.ExpiresAt = now.Add(3 * 24 * time.Hour).Truncate(time.Second)
	if err := store.SaveMeta(ctx, expiringSoon); err != nil {
		t.Fatalf("SaveMeta soon_key: %v", err)
	}

	// Expiring in 30 days — should NOT appear when querying withinDays=7.
	expiringLater := sampleMeta("later_key", ScopeGlobal)
	expiringLater.ExpiresAt = now.Add(30 * 24 * time.Hour).Truncate(time.Second)
	if err := store.SaveMeta(ctx, expiringLater); err != nil {
		t.Fatalf("SaveMeta later_key: %v", err)
	}

	// No expiry — should never appear.
	noExpiry := sampleMeta("no_expiry_key", ScopeGlobal)
	noExpiry.ExpiresAt = time.Time{}
	if err := store.SaveMeta(ctx, noExpiry); err != nil {
		t.Fatalf("SaveMeta no_expiry_key: %v", err)
	}

	// Already expired — should appear (it's within 7 days relative to now).
	alreadyExpired := sampleMeta("expired_key", ScopeGlobal)
	alreadyExpired.ExpiresAt = now.Add(-1 * time.Hour).Truncate(time.Second)
	if err := store.SaveMeta(ctx, alreadyExpired); err != nil {
		t.Fatalf("SaveMeta expired_key: %v", err)
	}

	statuses, err := store.ListExpiring(ctx, 7)
	if err != nil {
		t.Fatalf("ListExpiring: %v", err)
	}

	// We expect: soon_key and expired_key (within 7 days window).
	found := map[string]bool{}
	for _, s := range statuses {
		found[s.SecretRef.Name] = true
	}

	if !found["soon_key"] {
		t.Error("expected soon_key in expiring list")
	}
	if !found["expired_key"] {
		t.Error("expected expired_key (already past expiry) in expiring list")
	}
	if found["later_key"] {
		t.Error("later_key should not appear in 7-day expiry window")
	}
	if found["no_expiry_key"] {
		t.Error("no_expiry_key should never appear in expiry list")
	}
}

func TestSQLiteMetadataStore_ListExpiring_IsExpiredFlag(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	now := time.Now().UTC()
	expired := sampleMeta("past_expiry", ScopeGlobal)
	expired.ExpiresAt = now.Add(-1 * time.Hour).Truncate(time.Second)
	if err := store.SaveMeta(ctx, expired); err != nil {
		t.Fatalf("SaveMeta: %v", err)
	}

	statuses, err := store.ListExpiring(ctx, 2)
	if err != nil {
		t.Fatalf("ListExpiring: %v", err)
	}
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
	if !statuses[0].IsExpired {
		t.Error("IsExpired should be true for past-expiry secret")
	}
}

func TestSQLiteMetadataStore_ListExpiring_Empty(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	statuses, err := store.ListExpiring(ctx, 30)
	if err != nil {
		t.Fatalf("ListExpiring on empty store: %v", err)
	}
	if len(statuses) != 0 {
		t.Errorf("expected 0 results, got %d", len(statuses))
	}
}

// ---------------------------------------------------------------------------
// AuditLog — Log + Query
// ---------------------------------------------------------------------------

func newTestAuditLog(t *testing.T) (*SQLiteAuditLog, *sql.DB) {
	t.Helper()
	db := openTestDB(t)
	// Ensure schema is created via the MetadataStore (it creates both tables).
	_, err := NewSQLiteMetadataStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteMetadataStore for audit test: %v", err)
	}
	return NewSQLiteAuditLog(db), db
}

func sampleEntry(name string, action string) AuditEntry {
	return AuditEntry{
		Timestamp: time.Now().UTC().Truncate(time.Second),
		SecretRef: SecretRef{Name: name, Scope: ScopeGlobal},
		Action:    action,
		Actor:     "agent",
		Reason:    "test reason",
		Success:   true,
		Error:     "",
	}
}

func TestSQLiteAuditLog_LogAndQuery(t *testing.T) {
	auditLog, _ := newTestAuditLog(t)
	ctx := context.Background()

	entry := sampleEntry("my_key", "read")
	if err := auditLog.Log(ctx, entry); err != nil {
		t.Fatalf("Log: %v", err)
	}

	entries, err := auditLog.Query(ctx, AuditFilter{})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	got := entries[0]
	if got.SecretRef.Name != entry.SecretRef.Name {
		t.Errorf("Name: want %q, got %q", entry.SecretRef.Name, got.SecretRef.Name)
	}
	if got.Action != entry.Action {
		t.Errorf("Action: want %q, got %q", entry.Action, got.Action)
	}
	if got.Actor != entry.Actor {
		t.Errorf("Actor: want %q, got %q", entry.Actor, got.Actor)
	}
	if got.Success != entry.Success {
		t.Errorf("Success: want %v, got %v", entry.Success, got.Success)
	}
}

func TestSQLiteAuditLog_QueryFilterByName(t *testing.T) {
	auditLog, _ := newTestAuditLog(t)
	ctx := context.Background()

	if err := auditLog.Log(ctx, sampleEntry("key_a", "read")); err != nil {
		t.Fatalf("Log key_a: %v", err)
	}
	if err := auditLog.Log(ctx, sampleEntry("key_b", "write")); err != nil {
		t.Fatalf("Log key_b: %v", err)
	}
	if err := auditLog.Log(ctx, sampleEntry("key_a", "delete")); err != nil {
		t.Fatalf("Log key_a delete: %v", err)
	}

	results, err := auditLog.Query(ctx, AuditFilter{SecretName: "key_a"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 entries for key_a, got %d", len(results))
	}
	for _, r := range results {
		if r.SecretRef.Name != "key_a" {
			t.Errorf("unexpected secret name in results: %q", r.SecretRef.Name)
		}
	}
}

func TestSQLiteAuditLog_QueryFilterByAction(t *testing.T) {
	auditLog, _ := newTestAuditLog(t)
	ctx := context.Background()

	actions := []string{"read", "write", "read", "delete"}
	for _, a := range actions {
		if err := auditLog.Log(ctx, sampleEntry("some_key", a)); err != nil {
			t.Fatalf("Log %s: %v", a, err)
		}
	}

	results, err := auditLog.Query(ctx, AuditFilter{Action: "read"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 'read' entries, got %d", len(results))
	}
}

func TestSQLiteAuditLog_QueryFilterByActor(t *testing.T) {
	auditLog, _ := newTestAuditLog(t)
	ctx := context.Background()

	e1 := sampleEntry("k", "read")
	e1.Actor = "primary"
	e2 := sampleEntry("k", "read")
	e2.Actor = "sub-agent:worker1"

	if err := auditLog.Log(ctx, e1); err != nil {
		t.Fatalf("Log e1: %v", err)
	}
	if err := auditLog.Log(ctx, e2); err != nil {
		t.Fatalf("Log e2: %v", err)
	}

	results, err := auditLog.Query(ctx, AuditFilter{Actor: "primary"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1, got %d", len(results))
	}
	if results[0].Actor != "primary" {
		t.Errorf("expected actor 'primary', got %q", results[0].Actor)
	}
}

func TestSQLiteAuditLog_QueryFilterBySince(t *testing.T) {
	auditLog, _ := newTestAuditLog(t)
	ctx := context.Background()

	now := time.Now().UTC()

	// Log an old entry with a past timestamp (2 hours ago).
	old := sampleEntry("secret", "read")
	old.Timestamp = now.Add(-2 * time.Hour).Truncate(time.Second)
	if err := auditLog.Log(ctx, old); err != nil {
		t.Fatalf("Log old: %v", err)
	}

	// Log a recent entry (just now).
	recent := sampleEntry("secret", "write")
	recent.Timestamp = now.Truncate(time.Second)
	if err := auditLog.Log(ctx, recent); err != nil {
		t.Fatalf("Log recent: %v", err)
	}

	// Filter to entries since 90 minutes ago — should only match the recent one.
	sinceTime := now.Add(-90 * time.Minute)
	results, err := auditLog.Query(ctx, AuditFilter{Since: sinceTime.Unix()})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 recent entry, got %d; entries: %+v", len(results), results)
		return // avoid panic below
	}
	if results[0].Action != "write" {
		t.Errorf("expected 'write' action, got %q", results[0].Action)
	}
}

func TestSQLiteAuditLog_QueryDefaultLimit(t *testing.T) {
	auditLog, _ := newTestAuditLog(t)
	ctx := context.Background()

	// Insert 150 entries.
	for i := 0; i < 150; i++ {
		if err := auditLog.Log(ctx, sampleEntry("k", "read")); err != nil {
			t.Fatalf("Log: %v", err)
		}
	}

	// Default limit is 100.
	results, err := auditLog.Query(ctx, AuditFilter{})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(results) != 100 {
		t.Errorf("expected 100 (default limit), got %d", len(results))
	}
}

func TestSQLiteAuditLog_QueryCustomLimit(t *testing.T) {
	auditLog, _ := newTestAuditLog(t)
	ctx := context.Background()

	for i := 0; i < 20; i++ {
		if err := auditLog.Log(ctx, sampleEntry("k", "read")); err != nil {
			t.Fatalf("Log: %v", err)
		}
	}

	results, err := auditLog.Query(ctx, AuditFilter{Limit: 5})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(results) != 5 {
		t.Errorf("expected 5, got %d", len(results))
	}
}

func TestSQLiteAuditLog_FailureEntry(t *testing.T) {
	auditLog, _ := newTestAuditLog(t)
	ctx := context.Background()

	entry := sampleEntry("locked_secret", "read")
	entry.Success = false
	entry.Error = "access denied: tool not allowed"

	if err := auditLog.Log(ctx, entry); err != nil {
		t.Fatalf("Log: %v", err)
	}

	results, err := auditLog.Query(ctx, AuditFilter{})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Success {
		t.Error("Success should be false")
	}
	if results[0].Error != entry.Error {
		t.Errorf("Error: want %q, got %q", entry.Error, results[0].Error)
	}
}

func TestSQLiteAuditLog_QueryEmpty(t *testing.T) {
	auditLog, _ := newTestAuditLog(t)
	ctx := context.Background()

	results, err := auditLog.Query(ctx, AuditFilter{})
	if err != nil {
		t.Fatalf("Query on empty log: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 entries, got %d", len(results))
	}
}

// ---------------------------------------------------------------------------
// Schema is idempotent (migrate called twice)
// ---------------------------------------------------------------------------

func TestSQLiteMetadataStore_IdempotentMigration(t *testing.T) {
	db := openTestDB(t)

	_, err := NewSQLiteMetadataStore(db)
	if err != nil {
		t.Fatalf("first NewSQLiteMetadataStore: %v", err)
	}
	_, err = NewSQLiteMetadataStore(db)
	if err != nil {
		t.Fatalf("second NewSQLiteMetadataStore (should be idempotent): %v", err)
	}
}

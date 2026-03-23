package memory

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestNewSQLiteStore(t *testing.T) {
	store := newTestStore(t)
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestStoreAndSearch(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	err := store.Store(ctx, Entry{
		ID:      "mem-1",
		Tier:    TierSemantic,
		Content: "BlackCat uses Go for fast startup",
		Score:   1.0,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	err = store.Store(ctx, Entry{
		ID:      "mem-2",
		Tier:    TierEpisodic,
		Content: "User asked about authentication patterns",
		Score:   0.8,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Search by text
	results, err := store.Search(ctx, SearchQuery{Text: "Go", Limit: 10})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected at least 1 result for 'Go'")
	}
	if results[0].Entry.ID != "mem-1" {
		t.Errorf("expected mem-1, got %q", results[0].Entry.ID)
	}
}

func TestSearchByTier(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	store.Store(ctx, Entry{ID: "s1", Tier: TierSemantic, Content: "fact about auth", Score: 1.0})
	store.Store(ctx, Entry{ID: "e1", Tier: TierEpisodic, Content: "conversation about auth", Score: 1.0})

	results, err := store.Search(ctx, SearchQuery{Text: "auth", Tier: TierSemantic, Limit: 10})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	for _, r := range results {
		if r.Entry.Tier != TierSemantic {
			t.Errorf("expected only semantic results, got %s", r.Entry.Tier)
		}
	}
}

func TestSearchByUserID(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	store.Store(ctx, Entry{ID: "u1", Tier: TierSemantic, Content: "user1 preference", UserID: "user1", Score: 1.0})
	store.Store(ctx, Entry{ID: "u2", Tier: TierSemantic, Content: "user2 preference", UserID: "user2", Score: 1.0})

	results, err := store.Search(ctx, SearchQuery{Text: "preference", UserID: "user1", Limit: 10})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	for _, r := range results {
		if r.Entry.UserID != "user1" {
			t.Errorf("expected only user1 results, got %q", r.Entry.UserID)
		}
	}
}

func TestDelete(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	store.Store(ctx, Entry{ID: "del-1", Tier: TierEpisodic, Content: "to be deleted", Score: 1.0})

	err := store.Delete(ctx, "del-1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	results, _ := store.Search(ctx, SearchQuery{Text: "deleted", Limit: 10})
	for _, r := range results {
		if r.Entry.ID == "del-1" {
			t.Error("deleted entry should not appear in search")
		}
	}
}

func TestBuildSnapshot(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		store.Store(ctx, Entry{
			ID:      fmt.Sprintf("snap-%d", i),
			Tier:    TierSemantic,
			Content: fmt.Sprintf("Important fact number %d about the project", i),
			Score:   float64(5-i) / 5.0,
		})
	}

	snap, err := store.BuildSnapshot(ctx, "project1", "")
	if err != nil {
		t.Fatalf("BuildSnapshot failed: %v", err)
	}
	if snap.EntryCount == 0 {
		t.Error("expected non-empty snapshot")
	}
	if snap.Content == "" {
		t.Error("expected non-empty snapshot content")
	}
	if snap.TokenCount <= 0 {
		t.Error("expected positive token count")
	}
}

func TestStats(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	store.Store(ctx, Entry{ID: "s1", Tier: TierSemantic, Content: "semantic", Score: 1.0})
	store.Store(ctx, Entry{ID: "e1", Tier: TierEpisodic, Content: "episodic", Score: 1.0})
	store.Store(ctx, Entry{ID: "p1", Tier: TierProcedural, Content: "procedural", Score: 1.0})

	stats, err := store.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}
	if stats.TotalEntries != 3 {
		t.Errorf("expected 3 total, got %d", stats.TotalEntries)
	}
	if stats.SemanticCount != 1 {
		t.Errorf("expected 1 semantic, got %d", stats.SemanticCount)
	}
	if stats.EpisodicCount != 1 {
		t.Errorf("expected 1 episodic, got %d", stats.EpisodicCount)
	}
	if stats.ProceduralCount != 1 {
		t.Errorf("expected 1 procedural, got %d", stats.ProceduralCount)
	}
}

func TestDecay(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Insert old episodic entry with very low score
	store.Store(ctx, Entry{
		ID: "old-1", Tier: TierEpisodic, Content: "very old memory",
		Score: 0.05, CreatedAt: 1, // very old
	})

	deleted, err := store.Decay(ctx)
	if err != nil {
		t.Fatalf("Decay failed: %v", err)
	}
	if deleted < 1 {
		t.Errorf("expected at least 1 deleted, got %d", deleted)
	}
}

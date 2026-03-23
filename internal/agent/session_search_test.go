package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// memSessionStore is an in-memory implementation of SessionStore for testing.
type memSessionStore struct {
	entries []SessionEntry
}

func (m *memSessionStore) Search(_ context.Context, query string, limit int) ([]SessionEntry, error) {
	var results []SessionEntry
	q := strings.ToLower(query)
	for _, e := range m.entries {
		if strings.Contains(strings.ToLower(e.Content), q) {
			results = append(results, e)
			if len(results) >= limit {
				break
			}
		}
	}
	return results, nil
}

func (m *memSessionStore) GetSession(_ context.Context, sessionID string) ([]SessionEntry, error) {
	var results []SessionEntry
	for _, e := range m.entries {
		if e.SessionID == sessionID {
			results = append(results, e)
		}
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("session %q not found", sessionID)
	}
	return results, nil
}

func newTestStore() *memSessionStore {
	return &memSessionStore{
		entries: []SessionEntry{
			{SessionID: "s1", Timestamp: 100, Role: "user", Content: "How do I deploy to kubernetes?"},
			{SessionID: "s1", Timestamp: 101, Role: "assistant", Content: "You can use kubectl apply to deploy manifests."},
			{SessionID: "s1", Timestamp: 102, Role: "user", Content: "What about helm charts?"},
			{SessionID: "s1", Timestamp: 103, Role: "assistant", Content: "Helm charts package kubernetes resources together."},
			{SessionID: "s2", Timestamp: 200, Role: "user", Content: "Explain docker networking."},
			{SessionID: "s2", Timestamp: 201, Role: "assistant", Content: "Docker uses bridge networks by default."},
			{SessionID: "s2", Timestamp: 202, Role: "tool", Content: "docker network ls output"},
			{SessionID: "s3", Timestamp: 300, Role: "user", Content: "How do I configure kubernetes ingress?"},
			{SessionID: "s3", Timestamp: 301, Role: "assistant", Content: "Ingress controllers route external traffic to services."},
		},
	}
}

func TestSessionSearcher_Search(t *testing.T) {
	store := newTestStore()
	searcher := NewSessionSearcher(store)
	ctx := context.Background()

	results, err := searcher.Search(ctx, "kubernetes", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}

	// Should find sessions s1 and s3 (both mention kubernetes).
	sessionIDs := make(map[string]bool)
	for _, r := range results {
		sessionIDs[r.SessionID] = true
	}
	if !sessionIDs["s1"] {
		t.Error("expected session s1 in results")
	}
	if !sessionIDs["s3"] {
		t.Error("expected session s3 in results")
	}
	if sessionIDs["s2"] {
		t.Error("did not expect session s2 in results")
	}
}

func TestSessionSearcher_Search_MaxSessions(t *testing.T) {
	store := newTestStore()
	searcher := NewSessionSearcher(store)
	ctx := context.Background()

	results, err := searcher.Search(ctx, "kubernetes", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestSessionSearcher_Search_NoResults(t *testing.T) {
	store := newTestStore()
	searcher := NewSessionSearcher(store)
	ctx := context.Background()

	results, err := searcher.Search(ctx, "nonexistentquery", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestSessionSearcher_Search_ScoreOrdering(t *testing.T) {
	store := newTestStore()
	searcher := NewSessionSearcher(store)
	ctx := context.Background()

	results, err := searcher.Search(ctx, "kubernetes", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}
	// Results should be sorted by score descending.
	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Errorf("results not sorted by score: index %d (%.2f) > index %d (%.2f)",
				i, results[i].Score, i-1, results[i-1].Score)
		}
	}
}

func TestGroupBySession(t *testing.T) {
	entries := []SessionEntry{
		{SessionID: "a", Timestamp: 1, Content: "hello"},
		{SessionID: "b", Timestamp: 2, Content: "world"},
		{SessionID: "a", Timestamp: 3, Content: "foo"},
		{SessionID: "c", Timestamp: 4, Content: "bar"},
		{SessionID: "b", Timestamp: 5, Content: "baz"},
	}

	searcher := NewSessionSearcher(nil)
	groups := searcher.GroupBySession(entries)

	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(groups))
	}
	if len(groups["a"]) != 2 {
		t.Errorf("expected 2 entries for session a, got %d", len(groups["a"]))
	}
	if len(groups["b"]) != 2 {
		t.Errorf("expected 2 entries for session b, got %d", len(groups["b"]))
	}
	if len(groups["c"]) != 1 {
		t.Errorf("expected 1 entry for session c, got %d", len(groups["c"]))
	}
}

func TestGroupBySession_Empty(t *testing.T) {
	searcher := NewSessionSearcher(nil)
	groups := searcher.GroupBySession(nil)
	if len(groups) != 0 {
		t.Errorf("expected empty map, got %d groups", len(groups))
	}
}

func TestTruncateAroundMatches(t *testing.T) {
	entries := make([]SessionEntry, 20)
	for i := range entries {
		entries[i] = SessionEntry{
			SessionID: "s1",
			Timestamp: int64(i),
			Role:      "user",
			Content:   fmt.Sprintf("entry number %d with some content", i),
		}
	}

	searcher := NewSessionSearcher(nil)

	// Match at index 10; maxChars large enough for several entries.
	result := searcher.TruncateAroundMatches(entries, []int{10}, 300)
	if len(result) == 0 {
		t.Fatal("expected non-empty result")
	}

	// The matched entry must be included.
	found := false
	for _, e := range result {
		if e.Timestamp == 10 {
			found = true
			break
		}
	}
	if !found {
		t.Error("matched entry (index 10) not found in truncated result")
	}

	// Total chars should be within limit.
	total := 0
	for _, e := range result {
		total += len(e.Content)
	}
	if total > 300 {
		t.Errorf("total chars %d exceeds maxChars 300", total)
	}
}

func TestTruncateAroundMatches_MultipleMatches(t *testing.T) {
	entries := make([]SessionEntry, 10)
	for i := range entries {
		entries[i] = SessionEntry{
			SessionID: "s1",
			Timestamp: int64(i),
			Role:      "user",
			Content:   "short",
		}
	}

	searcher := NewSessionSearcher(nil)
	result := searcher.TruncateAroundMatches(entries, []int{2, 7}, 100)
	if len(result) == 0 {
		t.Fatal("expected non-empty result")
	}

	// Both matched entries should be present.
	has2, has7 := false, false
	for _, e := range result {
		if e.Timestamp == 2 {
			has2 = true
		}
		if e.Timestamp == 7 {
			has7 = true
		}
	}
	if !has2 {
		t.Error("expected entry at index 2 in result")
	}
	if !has7 {
		t.Error("expected entry at index 7 in result")
	}
}

func TestTruncateAroundMatches_EmptyInputs(t *testing.T) {
	searcher := NewSessionSearcher(nil)

	result := searcher.TruncateAroundMatches(nil, []int{0}, 100)
	if len(result) != 0 {
		t.Errorf("expected empty result for nil entries, got %d", len(result))
	}

	result = searcher.TruncateAroundMatches([]SessionEntry{{Content: "x"}}, nil, 100)
	if len(result) != 0 {
		t.Errorf("expected empty result for nil matchIndices, got %d", len(result))
	}
}

func TestTruncateAroundMatches_ZeroMaxChars(t *testing.T) {
	entries := []SessionEntry{{SessionID: "s1", Content: "hello"}}
	searcher := NewSessionSearcher(nil)
	result := searcher.TruncateAroundMatches(entries, []int{0}, 0)
	if len(result) != 0 {
		t.Errorf("expected empty result for maxChars=0, got %d", len(result))
	}
}

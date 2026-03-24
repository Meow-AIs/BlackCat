package agent

import (
	"context"
	"sort"
	"strings"
)

// SessionEntry represents a single message in a session transcript.
type SessionEntry struct {
	SessionID string
	Timestamp int64
	Role      string // "user", "assistant", "tool"
	Content   string
}

// SessionSearchResult groups matching entries for a single session.
type SessionSearchResult struct {
	SessionID string
	Matches   []SessionEntry
	Summary   string // summarized by auxiliary model
	Score     float64
}

// SessionStore abstracts FTS5 search on session transcripts.
type SessionStore interface {
	Search(ctx context.Context, query string, limit int) ([]SessionEntry, error)
	GetSession(ctx context.Context, sessionID string) ([]SessionEntry, error)
}

// SessionSearcher provides search and grouping over session transcripts.
type SessionSearcher struct {
	store SessionStore
}

// NewSessionSearcher creates a SessionSearcher backed by the given store.
func NewSessionSearcher(store SessionStore) *SessionSearcher {
	return &SessionSearcher{store: store}
}

// Search queries the store, groups results by session, and returns the top
// maxSessions unique sessions sorted by score (match count) descending.
func (s *SessionSearcher) Search(ctx context.Context, query string, maxSessions int) ([]SessionSearchResult, error) {
	// Fetch a generous number of entries to ensure we get enough sessions.
	entries, err := s.store.Search(ctx, query, maxSessions*50)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, nil
	}

	grouped := s.GroupBySession(entries)

	results := make([]SessionSearchResult, 0, len(grouped))
	for sid, matches := range grouped {
		results = append(results, SessionSearchResult{
			SessionID: sid,
			Matches:   matches,
			Score:     float64(len(matches)),
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > maxSessions {
		results = results[:maxSessions]
	}

	return results, nil
}

// GroupBySession partitions entries by their SessionID, preserving order
// within each group.
func (s *SessionSearcher) GroupBySession(entries []SessionEntry) map[string][]SessionEntry {
	groups := make(map[string][]SessionEntry)
	for _, e := range entries {
		groups[e.SessionID] = append(groups[e.SessionID], e)
	}
	return groups
}

// TruncateAroundMatches returns a subset of entries centered around the given
// match indices, with total content length capped at maxChars. Matched entries
// are always included first, then surrounding context is added.
func (s *SessionSearcher) TruncateAroundMatches(entries []SessionEntry, matchIndices []int, maxChars int) []SessionEntry {
	if len(entries) == 0 || len(matchIndices) == 0 || maxChars <= 0 {
		return nil
	}

	n := len(entries)
	included := make([]bool, n)
	totalChars := 0

	// Phase 1: include all matched entries that fit.
	for _, idx := range matchIndices {
		if idx < 0 || idx >= n {
			continue
		}
		cost := len(entries[idx].Content)
		if totalChars+cost > maxChars {
			continue
		}
		included[idx] = true
		totalChars += cost
	}

	// Phase 2: expand around each match index, alternating before/after.
	for radius := 1; radius < n; radius++ {
		added := false
		for _, idx := range matchIndices {
			for _, offset := range []int{-radius, radius} {
				neighbor := idx + offset
				if neighbor < 0 || neighbor >= n || included[neighbor] {
					continue
				}
				cost := len(entries[neighbor].Content)
				if totalChars+cost > maxChars {
					continue
				}
				included[neighbor] = true
				totalChars += cost
				added = true
			}
		}
		if !added && totalChars > 0 {
			// Check if any more could fit.
			anyFits := false
			for i := 0; i < n; i++ {
				if !included[i] && totalChars+len(entries[i].Content) <= maxChars {
					anyFits = true
					break
				}
			}
			if !anyFits {
				break
			}
		}
	}

	result := make([]SessionEntry, 0)
	for i, e := range entries {
		if included[i] {
			result = append(result, e)
		}
	}
	return result
}

// matchIndicesForQuery returns indices of entries whose content contains the
// query (case-insensitive). Useful for combining with TruncateAroundMatches.
func (s *SessionSearcher) matchIndicesForQuery(entries []SessionEntry, query string) []int {
	q := strings.ToLower(query)
	var indices []int
	for i, e := range entries {
		if strings.Contains(strings.ToLower(e.Content), q) {
			indices = append(indices, i)
		}
	}
	return indices
}

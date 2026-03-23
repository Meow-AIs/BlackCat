//go:build cgo

package memory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

// HybridSearch performs a combined FTS5 keyword search and vector KNN search,
// then merges the results using Reciprocal Rank Fusion (RRF).
// The k parameter for RRF is fixed at 60 (standard value).
func HybridSearch(db *sql.DB, query string, embedding []float32, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 10
	}

	// Fetch more candidates than needed for fusion
	candidateLimit := limit * 3

	ftsResults, err := ftsSearch(db, query, candidateLimit)
	if err != nil {
		return nil, fmt.Errorf("fts search: %w", err)
	}

	vs := NewVectorStore(db)
	vectorMatches, err := vs.SearchKNN(embedding, candidateLimit)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}

	// Convert vector matches to a map of memoryID -> rank
	vectorRanks := make(map[string]int, len(vectorMatches))
	for i, m := range vectorMatches {
		vectorRanks[m.MemoryID] = i + 1
	}

	ftsRanks := make(map[string]int, len(ftsResults))
	for i, r := range ftsResults {
		ftsRanks[r.Entry.ID] = i + 1
	}

	// Collect all unique memory IDs
	allIDs := make(map[string]bool)
	for _, r := range ftsResults {
		allIDs[r.Entry.ID] = true
	}
	for _, m := range vectorMatches {
		allIDs[m.MemoryID] = true
	}

	// Compute RRF scores
	const k = 60
	type scored struct {
		id    string
		score float64
	}

	var candidates []scored
	for id := range allIDs {
		score := 0.0
		if rank, ok := ftsRanks[id]; ok {
			score += 1.0 / float64(k+rank)
		}
		if rank, ok := vectorRanks[id]; ok {
			score += 1.0 / float64(k+rank)
		}
		candidates = append(candidates, scored{id: id, score: score})
	}

	// Sort by RRF score descending
	for i := 1; i < len(candidates); i++ {
		for j := i; j > 0 && candidates[j].score > candidates[j-1].score; j-- {
			candidates[j], candidates[j-1] = candidates[j-1], candidates[j]
		}
	}

	if limit > len(candidates) {
		limit = len(candidates)
	}

	// Fetch full entries for top candidates
	results := make([]SearchResult, 0, limit)
	for i := 0; i < limit; i++ {
		entry, err := fetchEntry(db, candidates[i].id)
		if err != nil {
			continue
		}
		// Normalize score to [0, 1]
		maxPossible := 2.0 / float64(k+1) // best possible RRF (rank 1 in both)
		relevance := candidates[i].score / maxPossible
		relevance = math.Min(relevance, 1.0)

		results = append(results, SearchResult{
			Entry:     entry,
			Relevance: relevance,
		})
	}

	return results, nil
}

// ftsSearch performs a full-text search using the FTS5 virtual table.
func ftsSearch(db *sql.DB, query string, limit int) ([]SearchResult, error) {
	// Sanitize the query for FTS5 — wrap each word in quotes
	words := strings.Fields(query)
	if len(words) == 0 {
		return nil, nil
	}

	var ftsTerms []string
	for _, w := range words {
		ftsTerms = append(ftsTerms, `"`+w+`"`)
	}
	ftsQuery := strings.Join(ftsTerms, " OR ")

	rows, err := db.Query(`
		SELECT m.id, m.tier, m.content, m.metadata_json, m.score, m.user_id,
		       m.created_at, m.updated_at
		FROM memory_fts f
		JOIN memories m ON m.rowid = f.rowid
		WHERE memory_fts MATCH ?
		ORDER BY rank
		LIMIT ?
	`, ftsQuery, limit)
	if err != nil {
		// FTS table might be empty or query invalid — fall back to empty
		return nil, nil
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var e Entry
		var tierStr, metaJSON string
		if err := rows.Scan(&e.ID, &tierStr, &e.Content, &metaJSON, &e.Score,
			&e.UserID, &e.CreatedAt, &e.UpdatedAt); err != nil {
			continue
		}
		e.Tier = Tier(tierStr)
		parseMetadata(&e, metaJSON)
		results = append(results, SearchResult{Entry: e, Relevance: e.Score})
	}
	return results, nil
}

// fetchEntry loads a single memory entry by ID.
func fetchEntry(db *sql.DB, id string) (Entry, error) {
	var e Entry
	var tierStr, metaJSON string

	err := db.QueryRow(`
		SELECT id, tier, content, metadata_json, score, user_id, created_at, updated_at
		FROM memories WHERE id = ?
	`, id).Scan(&e.ID, &tierStr, &e.Content, &metaJSON, &e.Score,
		&e.UserID, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return Entry{}, fmt.Errorf("fetch entry %q: %w", id, err)
	}

	e.Tier = Tier(tierStr)
	parseMetadata(&e, metaJSON)
	return e, nil
}

// parseMetadata decodes a JSON string into the entry's Metadata map.
func parseMetadata(e *Entry, metaJSON string) {
	if metaJSON == "" || metaJSON == "{}" {
		return
	}
	meta := make(map[string]string)
	if err := json.Unmarshal([]byte(metaJSON), &meta); err == nil {
		e.Metadata = meta
	}
}

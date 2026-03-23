//go:build cgo

package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// FullEngine implements the Engine interface with FTS5, vector search,
// and hybrid retrieval. It requires CGo for SQLite.
type FullEngine struct {
	db       *sql.DB
	vectors  *VectorStore
	embedder Embedder
}

// NewFullEngine opens a SQLite database, creates the schema, and returns
// a fully-initialized memory engine.
func NewFullEngine(dbPath string, embedder Embedder) (*FullEngine, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := CreateSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}

	return &FullEngine{
		db:       db,
		vectors:  NewVectorStore(db),
		embedder: embedder,
	}, nil
}

// Store saves a memory entry with its vector embedding.
func (e *FullEngine) Store(ctx context.Context, entry Entry) error {
	now := time.Now().Unix()
	if entry.CreatedAt == 0 {
		entry.CreatedAt = now
	}
	entry.UpdatedAt = now

	metaJSON := "{}"
	if entry.Metadata != nil {
		data, err := json.Marshal(entry.Metadata)
		if err == nil {
			metaJSON = string(data)
		}
	}

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Insert into memories table
	_, err = tx.ExecContext(ctx,
		`INSERT OR REPLACE INTO memories (id, tier, content, metadata_json, score, user_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID, string(entry.Tier), entry.Content, metaJSON, entry.Score,
		entry.UserID, entry.CreatedAt, entry.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert memory: %w", err)
	}

	// Insert into FTS5 index
	_, err = tx.ExecContext(ctx,
		`INSERT OR REPLACE INTO memory_fts (rowid, content)
		 SELECT rowid, content FROM memories WHERE id = ?`, entry.ID)
	if err != nil {
		// FTS insert failure is non-fatal — log but continue
		_ = err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	// Generate and store vector embedding (outside tx for simplicity)
	if e.embedder != nil {
		embedding, err := e.embedder.Embed(ctx, entry.Content)
		if err == nil {
			_ = e.vectors.Insert(entry.ID, embedding)
		}
	}

	return nil
}

// Search performs hybrid retrieval combining FTS5 and vector KNN.
func (e *FullEngine) Search(ctx context.Context, query SearchQuery) ([]SearchResult, error) {
	if query.Limit <= 0 {
		query.Limit = 10
	}

	// If we have an embedder, use hybrid search
	if e.embedder != nil && query.Text != "" {
		embedding, err := e.embedder.Embed(ctx, query.Text)
		if err == nil {
			results, err := HybridSearch(e.db, query.Text, embedding, query.Limit)
			if err == nil && len(results) > 0 {
				return filterResults(results, query), nil
			}
		}
	}

	// Fall back to LIKE search
	return e.searchByLike(ctx, query)
}

// searchByLike performs a simple LIKE-based search as fallback.
func (e *FullEngine) searchByLike(ctx context.Context, query SearchQuery) ([]SearchResult, error) {
	q := `SELECT id, tier, content, metadata_json, score, user_id, created_at, updated_at
	      FROM memories WHERE 1=1`
	var args []any

	if query.Text != "" {
		q += ` AND content LIKE ?`
		args = append(args, "%"+query.Text+"%")
	}
	if query.Tier != "" {
		q += ` AND tier = ?`
		args = append(args, string(query.Tier))
	}
	if query.UserID != "" {
		q += ` AND user_id = ?`
		args = append(args, query.UserID)
	}

	q += ` ORDER BY score DESC, updated_at DESC LIMIT ?`
	args = append(args, query.Limit)

	rows, err := e.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var entry Entry
		var tierStr, metaJSON string
		if err := rows.Scan(&entry.ID, &tierStr, &entry.Content, &metaJSON,
			&entry.Score, &entry.UserID, &entry.CreatedAt, &entry.UpdatedAt); err != nil {
			continue
		}
		entry.Tier = Tier(tierStr)
		parseMetadata(&entry, metaJSON)
		results = append(results, SearchResult{Entry: entry, Relevance: entry.Score})
	}
	return results, nil
}

// filterResults applies tier and user filters to hybrid search results.
func filterResults(results []SearchResult, query SearchQuery) []SearchResult {
	if query.Tier == "" && query.UserID == "" {
		return results
	}

	filtered := make([]SearchResult, 0, len(results))
	for _, r := range results {
		if query.Tier != "" && r.Entry.Tier != query.Tier {
			continue
		}
		if query.UserID != "" && r.Entry.UserID != query.UserID {
			continue
		}
		filtered = append(filtered, r)
	}
	return filtered
}

// Delete removes a memory entry and its associated vector and FTS data.
func (e *FullEngine) Delete(ctx context.Context, id string) error {
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Delete FTS entry first (needs rowid from memories)
	_, _ = tx.ExecContext(ctx,
		`DELETE FROM memory_fts WHERE rowid = (SELECT rowid FROM memories WHERE id = ?)`, id)

	_, err = tx.ExecContext(ctx, `DELETE FROM memories WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete memory: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	// Delete vector outside tx
	_ = e.vectors.Delete(id)
	return nil
}

// BuildSnapshot creates a frozen context snapshot for session start.
func (e *FullEngine) BuildSnapshot(ctx context.Context, projectID string, userID string) (Snapshot, error) {
	rows, err := e.db.QueryContext(ctx,
		`SELECT id, tier, content, score, created_at, updated_at
		 FROM memories
		 WHERE (user_id = ? OR user_id = '')
		 ORDER BY score DESC, updated_at DESC
		 LIMIT 50`, userID)
	if err != nil {
		return Snapshot{}, fmt.Errorf("query snapshot: %w", err)
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var entry Entry
		var tierStr string
		if err := rows.Scan(&entry.ID, &tierStr, &entry.Content, &entry.Score,
			&entry.CreatedAt, &entry.UpdatedAt); err != nil {
			continue
		}
		entry.Tier = Tier(tierStr)
		entries = append(entries, entry)
	}

	return BuildSnapshotFromEntries(entries, 800), nil
}

// Decay runs time-weighted decay on episodic memories and evicts low-score entries.
func (e *FullEngine) Decay(ctx context.Context) (int, error) {
	now := time.Now().Unix()
	thirtyDaysAgo := now - 30*24*60*60

	_, err := e.db.ExecContext(ctx,
		`UPDATE memories SET score = score * 0.9, updated_at = ?
		 WHERE tier = 'episodic' AND updated_at < ?`, now, thirtyDaysAgo)
	if err != nil {
		return 0, fmt.Errorf("decay scores: %w", err)
	}

	result, err := e.db.ExecContext(ctx,
		`DELETE FROM memories WHERE tier = 'episodic' AND score < 0.1`)
	if err != nil {
		return 0, fmt.Errorf("evict low-score: %w", err)
	}
	deleted, _ := result.RowsAffected()
	return int(deleted), nil
}

// Stats returns memory usage statistics.
func (e *FullEngine) Stats(ctx context.Context) (Stats, error) {
	var stats Stats

	_ = e.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM memories`).Scan(&stats.TotalEntries)
	_ = e.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM memories WHERE tier = 'episodic'`).Scan(&stats.EpisodicCount)
	_ = e.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM memories WHERE tier = 'semantic'`).Scan(&stats.SemanticCount)
	_ = e.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM memories WHERE tier = 'procedural'`).Scan(&stats.ProceduralCount)
	_ = e.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM memory_vectors`).Scan(&stats.VectorCount)

	// Estimate DB size using page_count * page_size
	var pageCount, pageSize int64
	_ = e.db.QueryRowContext(ctx, `PRAGMA page_count`).Scan(&pageCount)
	_ = e.db.QueryRowContext(ctx, `PRAGMA page_size`).Scan(&pageSize)
	stats.DBSizeBytes = pageCount * pageSize

	return stats, nil
}

// Close cleanly shuts down the memory engine.
func (e *FullEngine) Close() error {
	return e.db.Close()
}

// Ensure unused import doesn't cause issues
var _ = strings.Join

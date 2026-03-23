package memory

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore implements the Engine interface using SQLite + FTS5.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens or creates a SQLite database at the given path.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	store := &SQLiteStore{db: db}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return store, nil
}

func (s *SQLiteStore) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS memories (
		id TEXT PRIMARY KEY,
		tier TEXT NOT NULL,
		content TEXT NOT NULL,
		metadata TEXT DEFAULT '{}',
		score REAL DEFAULT 1.0,
		user_id TEXT DEFAULT '',
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_memories_tier ON memories(tier);
	CREATE INDEX IF NOT EXISTS idx_memories_user ON memories(user_id);
	CREATE INDEX IF NOT EXISTS idx_memories_score ON memories(score DESC);

	CREATE TABLE IF NOT EXISTS schedules (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		cron TEXT NOT NULL,
		task TEXT NOT NULL,
		channel TEXT DEFAULT '',
		enabled INTEGER DEFAULT 1,
		created_at INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS schedule_history (
		id TEXT PRIMARY KEY,
		schedule_id TEXT NOT NULL,
		status TEXT NOT NULL,
		output TEXT DEFAULT '',
		started_at INTEGER NOT NULL,
		finished_at INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS skills (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT NOT NULL,
		trigger_pattern TEXT DEFAULT '',
		steps TEXT DEFAULT '[]',
		success_rate REAL DEFAULT 0.0,
		usage_count INTEGER DEFAULT 0,
		last_used_at INTEGER DEFAULT 0,
		source TEXT DEFAULT 'manual',
		created_at INTEGER NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

func (s *SQLiteStore) Store(ctx context.Context, entry Entry) error {
	now := time.Now().Unix()
	if entry.CreatedAt == 0 {
		entry.CreatedAt = now
	}
	entry.UpdatedAt = now

	metaJSON := "{}"
	if entry.Metadata != nil {
		pairs := make([]string, 0, len(entry.Metadata))
		for k, v := range entry.Metadata {
			pairs = append(pairs, fmt.Sprintf("%q:%q", k, v))
		}
		metaJSON = "{" + strings.Join(pairs, ",") + "}"
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx,
		`INSERT OR REPLACE INTO memories (id, tier, content, metadata, score, user_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID, string(entry.Tier), entry.Content, metaJSON, entry.Score,
		entry.UserID, entry.CreatedAt, entry.UpdatedAt)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *SQLiteStore) Search(ctx context.Context, query SearchQuery) ([]SearchResult, error) {
	if query.Limit <= 0 {
		query.Limit = 10
	}
	return s.searchByLike(ctx, query)
}

func (s *SQLiteStore) searchByLike(ctx context.Context, query SearchQuery) ([]SearchResult, error) {
	q := `SELECT id, tier, content, metadata, score, user_id, created_at, updated_at FROM memories WHERE 1=1`
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

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var e Entry
		var tierStr, metaJSON string
		if err := rows.Scan(&e.ID, &tierStr, &e.Content, &metaJSON, &e.Score, &e.UserID, &e.CreatedAt, &e.UpdatedAt); err != nil {
			continue
		}
		e.Tier = Tier(tierStr)
		results = append(results, SearchResult{Entry: e, Relevance: e.Score})
	}
	return results, nil
}

func (s *SQLiteStore) Delete(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `DELETE FROM memories WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLiteStore) BuildSnapshot(ctx context.Context, projectID string, userID string) (Snapshot, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT content FROM memories
		 WHERE (user_id = ? OR user_id = '')
		 ORDER BY score DESC, updated_at DESC
		 LIMIT 50`, userID)
	if err != nil {
		return Snapshot{}, err
	}
	defer rows.Close()

	var parts []string
	count := 0
	totalTokens := 0
	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			continue
		}
		tokens := len(content) / 4 // rough estimate
		if totalTokens+tokens > 800 {
			break
		}
		parts = append(parts, content)
		totalTokens += tokens
		count++
	}

	return Snapshot{
		Content:    strings.Join(parts, "\n"),
		TokenCount: totalTokens,
		EntryCount: count,
	}, nil
}

func (s *SQLiteStore) Decay(ctx context.Context) (int, error) {
	now := time.Now().Unix()
	thirtyDaysAgo := now - 30*24*60*60

	// Reduce score of old episodic memories
	_, err := s.db.ExecContext(ctx,
		`UPDATE memories SET score = score * 0.9
		 WHERE tier = 'episodic' AND updated_at < ?`, thirtyDaysAgo)
	if err != nil {
		return 0, err
	}

	// Delete episodic memories with score < 0.1
	result, err := s.db.ExecContext(ctx,
		`DELETE FROM memories WHERE tier = 'episodic' AND score < 0.1`)
	if err != nil {
		return 0, err
	}
	deleted, _ := result.RowsAffected()
	return int(deleted), nil
}

func (s *SQLiteStore) Stats(ctx context.Context) (Stats, error) {
	var stats Stats

	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM memories`).Scan(&stats.TotalEntries)
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM memories WHERE tier = 'episodic'`).Scan(&stats.EpisodicCount)
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM memories WHERE tier = 'semantic'`).Scan(&stats.SemanticCount)
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM memories WHERE tier = 'procedural'`).Scan(&stats.ProceduralCount)

	return stats, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

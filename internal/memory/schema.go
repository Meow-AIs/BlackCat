//go:build cgo

package memory

import (
	"database/sql"
	"fmt"
)

// schemaVersion is the current schema version.
const schemaVersion = 2

// CreateSchema creates all required tables for the memory engine:
// memories, memory_vectors, memory_fts (FTS5 index), and a schema_version table.
func CreateSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS schema_version (
		version INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS memories (
		id TEXT PRIMARY KEY,
		tier TEXT NOT NULL,
		content TEXT NOT NULL,
		metadata_json TEXT DEFAULT '{}',
		score REAL DEFAULT 1.0,
		user_id TEXT DEFAULT '',
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_memories_tier ON memories(tier);
	CREATE INDEX IF NOT EXISTS idx_memories_user ON memories(user_id);
	CREATE INDEX IF NOT EXISTS idx_memories_score ON memories(score DESC);

	CREATE TABLE IF NOT EXISTS memory_vectors (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		memory_id TEXT NOT NULL UNIQUE,
		embedding BLOB NOT NULL,
		FOREIGN KEY (memory_id) REFERENCES memories(id) ON DELETE CASCADE
	);

	CREATE VIRTUAL TABLE IF NOT EXISTS memory_fts USING fts5(
		content,
		content_rowid='rowid'
	);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("create schema: %w", err)
	}

	// Insert initial version if not present
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_version").Scan(&count)
	if err != nil {
		return fmt.Errorf("check schema version: %w", err)
	}
	if count == 0 {
		_, err = db.Exec("INSERT INTO schema_version (version) VALUES (?)", schemaVersion)
		if err != nil {
			return fmt.Errorf("insert schema version: %w", err)
		}
	}

	return nil
}

// MigrateSchema applies migrations to bring the database to the target version.
func MigrateSchema(db *sql.DB, version int) error {
	var currentVersion int
	err := db.QueryRow("SELECT version FROM schema_version LIMIT 1").Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}

	if currentVersion >= version {
		return nil // already at or past target
	}

	// Migration from v1 to v2: add memory_vectors and memory_fts
	if currentVersion < 2 && version >= 2 {
		migrations := `
		CREATE TABLE IF NOT EXISTS memory_vectors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			memory_id TEXT NOT NULL UNIQUE,
			embedding BLOB NOT NULL,
			FOREIGN KEY (memory_id) REFERENCES memories(id) ON DELETE CASCADE
		);

		CREATE VIRTUAL TABLE IF NOT EXISTS memory_fts USING fts5(
			content,
			content_rowid='rowid'
		);
		`
		if _, err := db.Exec(migrations); err != nil {
			return fmt.Errorf("migrate to v2: %w", err)
		}
	}

	// Update version
	_, err = db.Exec("UPDATE schema_version SET version = ?", version)
	if err != nil {
		return fmt.Errorf("update schema version: %w", err)
	}

	return nil
}

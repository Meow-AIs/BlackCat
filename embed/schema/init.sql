-- BlackCat SQLite Schema
-- Memory, scheduling, skills, sessions, and MCP server storage.

-- Memories: episodic (conversations), semantic (facts), procedural (skills)
CREATE TABLE IF NOT EXISTS memories (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    type        TEXT NOT NULL CHECK(type IN ('episodic', 'semantic', 'procedural')),
    content     TEXT NOT NULL,
    metadata    TEXT DEFAULT '{}',
    session_id  TEXT,
    importance  REAL DEFAULT 0.5,
    access_count INTEGER DEFAULT 0,
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_memories_type ON memories(type);
CREATE INDEX IF NOT EXISTS idx_memories_session ON memories(session_id);
CREATE INDEX IF NOT EXISTS idx_memories_importance ON memories(importance DESC);
CREATE INDEX IF NOT EXISTS idx_memories_created ON memories(created_at DESC);

-- FTS5 full-text search index for keyword retrieval
CREATE VIRTUAL TABLE IF NOT EXISTS memory_fts USING fts5(
    content,
    content='memories',
    content_rowid='id',
    tokenize='porter unicode61'
);

-- Triggers to keep FTS in sync with memories table
CREATE TRIGGER IF NOT EXISTS memory_fts_insert AFTER INSERT ON memories BEGIN
    INSERT INTO memory_fts(rowid, content) VALUES (new.id, new.content);
END;

CREATE TRIGGER IF NOT EXISTS memory_fts_delete AFTER DELETE ON memories BEGIN
    INSERT INTO memory_fts(memory_fts, rowid, content) VALUES ('delete', old.id, old.content);
END;

CREATE TRIGGER IF NOT EXISTS memory_fts_update AFTER UPDATE OF content ON memories BEGIN
    INSERT INTO memory_fts(memory_fts, rowid, content) VALUES ('delete', old.id, old.content);
    INSERT INTO memory_fts(rowid, content) VALUES (new.id, new.content);
END;

-- Vector embeddings stored via sqlite-vec
-- This table is created by sqlite-vec extension:
--   CREATE VIRTUAL TABLE IF NOT EXISTS memory_vectors USING vec0(
--       memory_id INTEGER PRIMARY KEY,
--       embedding float[384]
--   );
-- We create a placeholder tracking table for metadata.
CREATE TABLE IF NOT EXISTS memory_vector_meta (
    memory_id   INTEGER PRIMARY KEY REFERENCES memories(id) ON DELETE CASCADE,
    dimensions  INTEGER NOT NULL DEFAULT 384,
    quantized   INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Scheduled tasks
CREATE TABLE IF NOT EXISTS schedules (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    cron_expr   TEXT NOT NULL,
    prompt      TEXT NOT NULL,
    enabled     INTEGER NOT NULL DEFAULT 1,
    last_run    TEXT,
    next_run    TEXT,
    run_count   INTEGER NOT NULL DEFAULT 0,
    error_count INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_schedules_enabled ON schedules(enabled);
CREATE INDEX IF NOT EXISTS idx_schedules_next_run ON schedules(next_run);

-- Schedule execution history
CREATE TABLE IF NOT EXISTS schedule_history (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    schedule_id INTEGER NOT NULL REFERENCES schedules(id) ON DELETE CASCADE,
    status      TEXT NOT NULL CHECK(status IN ('success', 'error', 'timeout')),
    output      TEXT,
    error       TEXT,
    duration_ms INTEGER,
    started_at  TEXT NOT NULL DEFAULT (datetime('now')),
    finished_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_schedule_history_schedule ON schedule_history(schedule_id);
CREATE INDEX IF NOT EXISTS idx_schedule_history_started ON schedule_history(started_at DESC);

-- Learned skills (procedural memory patterns)
CREATE TABLE IF NOT EXISTS skills (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL,
    pattern     TEXT NOT NULL,
    examples    TEXT DEFAULT '[]',
    use_count   INTEGER NOT NULL DEFAULT 0,
    success_rate REAL NOT NULL DEFAULT 0.0,
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_skills_name ON skills(name);
CREATE INDEX IF NOT EXISTS idx_skills_use_count ON skills(use_count DESC);

-- Agent sessions
CREATE TABLE IF NOT EXISTS sessions (
    id          TEXT PRIMARY KEY,
    mode        TEXT NOT NULL CHECK(mode IN ('interactive', 'oneshot', 'serve', 'subagent')),
    status      TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active', 'completed', 'error')),
    metadata    TEXT DEFAULT '{}',
    message_count INTEGER NOT NULL DEFAULT 0,
    token_count INTEGER NOT NULL DEFAULT 0,
    started_at  TEXT NOT NULL DEFAULT (datetime('now')),
    ended_at    TEXT
);

CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);
CREATE INDEX IF NOT EXISTS idx_sessions_started ON sessions(started_at DESC);

-- MCP (Model Context Protocol) server configurations
CREATE TABLE IF NOT EXISTS mcp_servers (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL UNIQUE,
    command     TEXT NOT NULL,
    args        TEXT DEFAULT '[]',
    env         TEXT DEFAULT '{}',
    enabled     INTEGER NOT NULL DEFAULT 1,
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_mcp_servers_name ON mcp_servers(name);
CREATE INDEX IF NOT EXISTS idx_mcp_servers_enabled ON mcp_servers(enabled);

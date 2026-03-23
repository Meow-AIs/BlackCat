package secrets

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// SQLiteMetadataStore persists secret metadata and audit logs in SQLite.
// It shares the same database file as the memory system (~/.blackcat/memory.db)
// but uses separate tables.
type SQLiteMetadataStore struct {
	db *sql.DB
}

// NewSQLiteMetadataStore creates a metadata store backed by the given SQLite database.
// It creates the required tables if they do not exist.
func NewSQLiteMetadataStore(db *sql.DB) (*SQLiteMetadataStore, error) {
	s := &SQLiteMetadataStore{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate secrets schema: %w", err)
	}
	return s, nil
}

func (s *SQLiteMetadataStore) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS secret_metadata (
			name         TEXT NOT NULL,
			scope        TEXT NOT NULL DEFAULT 'global',
			type         TEXT NOT NULL DEFAULT 'custom',
			project_path TEXT DEFAULT '',
			description  TEXT DEFAULT '',
			env_var      TEXT DEFAULT '',
			tags         TEXT DEFAULT '[]',
			created_at   TEXT NOT NULL,
			updated_at   TEXT NOT NULL,
			expires_at   TEXT DEFAULT '',
			rotation_days INTEGER DEFAULT 0,
			allowed_tools TEXT DEFAULT '[]',
			allowed_agents TEXT DEFAULT '[]',
			imported_from TEXT DEFAULT '',
			fingerprint  TEXT DEFAULT '',
			PRIMARY KEY (name, scope)
		)`,
		`CREATE TABLE IF NOT EXISTS secret_audit_log (
			id        INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp TEXT NOT NULL,
			name      TEXT NOT NULL,
			scope     TEXT NOT NULL,
			action    TEXT NOT NULL,
			actor     TEXT NOT NULL,
			reason    TEXT DEFAULT '',
			success   INTEGER NOT NULL DEFAULT 1,
			error     TEXT DEFAULT ''
		)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_name ON secret_audit_log(name)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON secret_audit_log(timestamp)`,
	}

	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("exec %q: %w", stmt[:40], err)
		}
	}
	return nil
}

func (s *SQLiteMetadataStore) SaveMeta(ctx context.Context, meta SecretMetadata) error {
	tags, _ := json.Marshal(meta.Tags)
	allowedTools, _ := json.Marshal(meta.AllowedTools)
	allowedAgents, _ := json.Marshal(meta.AllowedAgents)

	expiresAt := ""
	if !meta.ExpiresAt.IsZero() {
		expiresAt = meta.ExpiresAt.Format(time.RFC3339)
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO secret_metadata
			(name, scope, type, project_path, description, env_var, tags,
			 created_at, updated_at, expires_at, rotation_days,
			 allowed_tools, allowed_agents, imported_from, fingerprint)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(name, scope) DO UPDATE SET
			type=excluded.type, project_path=excluded.project_path,
			description=excluded.description, env_var=excluded.env_var,
			tags=excluded.tags, updated_at=excluded.updated_at,
			expires_at=excluded.expires_at, rotation_days=excluded.rotation_days,
			allowed_tools=excluded.allowed_tools, allowed_agents=excluded.allowed_agents,
			imported_from=excluded.imported_from, fingerprint=excluded.fingerprint`,
		meta.Name, meta.Scope, meta.Type, meta.ProjectPath,
		meta.Description, meta.EnvVar, string(tags),
		meta.CreatedAt.Format(time.RFC3339), meta.UpdatedAt.Format(time.RFC3339),
		expiresAt, meta.RotationDays,
		string(allowedTools), string(allowedAgents),
		meta.ImportedFrom, meta.Fingerprint,
	)
	if err != nil {
		return fmt.Errorf("save secret metadata: %w", err)
	}
	return nil
}

func (s *SQLiteMetadataStore) GetMeta(ctx context.Context, name string, scope Scope) (SecretMetadata, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT name, scope, type, project_path, description, env_var, tags,
		       created_at, updated_at, expires_at, rotation_days,
		       allowed_tools, allowed_agents, imported_from, fingerprint
		FROM secret_metadata WHERE name = ? AND scope = ?`, name, scope)

	var meta SecretMetadata
	var tags, allowedTools, allowedAgents string
	var createdAt, updatedAt, expiresAt string

	err := row.Scan(
		&meta.Name, &meta.Scope, &meta.Type, &meta.ProjectPath,
		&meta.Description, &meta.EnvVar, &tags,
		&createdAt, &updatedAt, &expiresAt, &meta.RotationDays,
		&allowedTools, &allowedAgents, &meta.ImportedFrom, &meta.Fingerprint,
	)
	if err == sql.ErrNoRows {
		return SecretMetadata{}, ErrNotFound
	}
	if err != nil {
		return SecretMetadata{}, fmt.Errorf("get secret metadata: %w", err)
	}

	_ = json.Unmarshal([]byte(tags), &meta.Tags)
	_ = json.Unmarshal([]byte(allowedTools), &meta.AllowedTools)
	_ = json.Unmarshal([]byte(allowedAgents), &meta.AllowedAgents)
	meta.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	meta.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	if expiresAt != "" {
		meta.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
	}

	return meta, nil
}

func (s *SQLiteMetadataStore) DeleteMeta(ctx context.Context, name string, scope Scope) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM secret_metadata WHERE name = ? AND scope = ?`, name, scope)
	if err != nil {
		return fmt.Errorf("delete secret metadata: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *SQLiteMetadataStore) ListMeta(ctx context.Context, scope Scope) ([]SecretMetadata, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT name, scope, type, project_path, description, env_var, tags,
		       created_at, updated_at, expires_at, rotation_days,
		       allowed_tools, allowed_agents, imported_from, fingerprint
		FROM secret_metadata WHERE scope = ? ORDER BY name`, scope)
	if err != nil {
		return nil, fmt.Errorf("list secret metadata: %w", err)
	}
	defer rows.Close()

	var result []SecretMetadata
	for rows.Next() {
		var meta SecretMetadata
		var tags, allowedTools, allowedAgents string
		var createdAt, updatedAt, expiresAt string

		if err := rows.Scan(
			&meta.Name, &meta.Scope, &meta.Type, &meta.ProjectPath,
			&meta.Description, &meta.EnvVar, &tags,
			&createdAt, &updatedAt, &expiresAt, &meta.RotationDays,
			&allowedTools, &allowedAgents, &meta.ImportedFrom, &meta.Fingerprint,
		); err != nil {
			return nil, fmt.Errorf("scan secret metadata: %w", err)
		}

		_ = json.Unmarshal([]byte(tags), &meta.Tags)
		_ = json.Unmarshal([]byte(allowedTools), &meta.AllowedTools)
		_ = json.Unmarshal([]byte(allowedAgents), &meta.AllowedAgents)
		meta.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		meta.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		if expiresAt != "" {
			meta.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
		}

		result = append(result, meta)
	}
	return result, rows.Err()
}

func (s *SQLiteMetadataStore) ListExpiring(ctx context.Context, withinDays int) ([]ExpiryStatus, error) {
	cutoff := time.Now().AddDate(0, 0, withinDays).Format(time.RFC3339)
	now := time.Now()

	rows, err := s.db.QueryContext(ctx, `
		SELECT name, scope, expires_at, rotation_days
		FROM secret_metadata
		WHERE expires_at != '' AND expires_at <= ?
		ORDER BY expires_at`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("list expiring secrets: %w", err)
	}
	defer rows.Close()

	var result []ExpiryStatus
	for rows.Next() {
		var name string
		var scope Scope
		var expiresAtStr string
		var rotationDays int

		if err := rows.Scan(&name, &scope, &expiresAtStr, &rotationDays); err != nil {
			return nil, fmt.Errorf("scan expiring secret: %w", err)
		}

		expiresAt, _ := time.Parse(time.RFC3339, expiresAtStr)
		daysLeft := int(expiresAt.Sub(now).Hours() / 24)

		result = append(result, ExpiryStatus{
			SecretRef:     SecretRef{Name: name, Scope: scope},
			ExpiresAt:     expiresAt,
			DaysLeft:      daysLeft,
			IsExpired:     now.After(expiresAt),
			NeedsRotation: rotationDays > 0 && daysLeft <= rotationDays/4,
		})
	}
	return result, rows.Err()
}

// --- Audit log implementation ---

// SQLiteAuditLog implements AuditLog using the same SQLite database.
type SQLiteAuditLog struct {
	db *sql.DB
}

// NewSQLiteAuditLog creates an audit log backed by SQLite.
func NewSQLiteAuditLog(db *sql.DB) *SQLiteAuditLog {
	return &SQLiteAuditLog{db: db}
}

func (a *SQLiteAuditLog) Log(ctx context.Context, entry AuditEntry) error {
	_, err := a.db.ExecContext(ctx, `
		INSERT INTO secret_audit_log (timestamp, name, scope, action, actor, reason, success, error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.Timestamp.Format(time.RFC3339),
		entry.SecretRef.Name, entry.SecretRef.Scope,
		entry.Action, entry.Actor, entry.Reason,
		boolToInt(entry.Success), entry.Error,
	)
	if err != nil {
		return fmt.Errorf("log audit entry: %w", err)
	}
	return nil
}

func (a *SQLiteAuditLog) Query(ctx context.Context, filter AuditFilter) ([]AuditEntry, error) {
	query := `SELECT timestamp, name, scope, action, actor, reason, success, error
		FROM secret_audit_log WHERE 1=1`
	var args []any

	if filter.SecretName != "" {
		query += ` AND name = ?`
		args = append(args, filter.SecretName)
	}
	if filter.Action != "" {
		query += ` AND action = ?`
		args = append(args, filter.Action)
	}
	if filter.Actor != "" {
		query += ` AND actor = ?`
		args = append(args, filter.Actor)
	}
	if filter.Since > 0 {
		t := time.Unix(filter.Since, 0).UTC().Format(time.RFC3339)
		query += ` AND timestamp >= ?`
		args = append(args, t)
	}

	query += ` ORDER BY timestamp DESC`

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	query += fmt.Sprintf(` LIMIT %d`, limit)

	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query audit log: %w", err)
	}
	defer rows.Close()

	var result []AuditEntry
	for rows.Next() {
		var entry AuditEntry
		var ts string
		var success int

		if err := rows.Scan(&ts, &entry.SecretRef.Name, &entry.SecretRef.Scope,
			&entry.Action, &entry.Actor, &entry.Reason, &success, &entry.Error); err != nil {
			return nil, fmt.Errorf("scan audit entry: %w", err)
		}
		entry.Timestamp, _ = time.Parse(time.RFC3339, ts)
		entry.Success = success != 0
		result = append(result, entry)
	}
	return result, rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

package skills

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// SQLiteManager implements Manager using the shared SQLite database.
type SQLiteManager struct {
	db *sql.DB
}

// NewSQLiteManager creates a skill manager using an existing database connection.
func NewSQLiteManager(db *sql.DB) *SQLiteManager {
	return &SQLiteManager{db: db}
}

func (m *SQLiteManager) Store(ctx context.Context, skill Skill) error {
	now := time.Now().Unix()
	if skill.CreatedAt == 0 {
		skill.CreatedAt = now
	}

	stepsJSON, _ := json.Marshal(skill.Steps)

	_, err := m.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO skills (id, name, description, trigger_pattern, steps, success_rate, usage_count, last_used_at, source, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		skill.ID, skill.Name, skill.Description, skill.Trigger,
		string(stepsJSON), skill.SuccessRate, skill.UsageCount,
		skill.LastUsedAt, skill.Source, skill.CreatedAt)
	return err
}

func (m *SQLiteManager) Get(ctx context.Context, id string) (Skill, error) {
	var s Skill
	var stepsJSON string
	err := m.db.QueryRowContext(ctx,
		`SELECT id, name, description, trigger_pattern, steps, success_rate, usage_count, last_used_at, source, created_at
		 FROM skills WHERE id = ?`, id).
		Scan(&s.ID, &s.Name, &s.Description, &s.Trigger, &stepsJSON, &s.SuccessRate, &s.UsageCount, &s.LastUsedAt, &s.Source, &s.CreatedAt)
	if err != nil {
		return Skill{}, fmt.Errorf("skill not found: %w", err)
	}
	json.Unmarshal([]byte(stepsJSON), &s.Steps)
	return s, nil
}

func (m *SQLiteManager) Match(ctx context.Context, taskDescription string, limit int) ([]Skill, error) {
	if limit <= 0 {
		limit = 5
	}

	rows, err := m.db.QueryContext(ctx,
		`SELECT id, name, description, trigger_pattern, steps, success_rate, usage_count, last_used_at, source, created_at
		 FROM skills
		 WHERE description LIKE ? OR trigger_pattern LIKE ?
		 ORDER BY success_rate DESC, usage_count DESC
		 LIMIT ?`,
		"%"+taskDescription+"%", "%"+taskDescription+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSkills(rows)
}

func (m *SQLiteManager) RecordOutcome(ctx context.Context, id string, success bool) error {
	s, err := m.Get(ctx, id)
	if err != nil {
		return err
	}

	total := float64(s.UsageCount + 1)
	successes := s.SuccessRate * float64(s.UsageCount)
	if success {
		successes++
	}

	_, err = m.db.ExecContext(ctx,
		`UPDATE skills SET success_rate = ?, usage_count = usage_count + 1, last_used_at = ? WHERE id = ?`,
		successes/total, time.Now().Unix(), id)
	return err
}

func (m *SQLiteManager) List(ctx context.Context) ([]Skill, error) {
	rows, err := m.db.QueryContext(ctx,
		`SELECT id, name, description, trigger_pattern, steps, success_rate, usage_count, last_used_at, source, created_at
		 FROM skills ORDER BY usage_count DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSkills(rows)
}

func (m *SQLiteManager) Delete(ctx context.Context, id string) error {
	_, err := m.db.ExecContext(ctx, `DELETE FROM skills WHERE id = ?`, id)
	return err
}

func (m *SQLiteManager) Export(ctx context.Context) ([]byte, error) {
	skills, err := m.List(ctx)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(skills, "", "  ")
}

func (m *SQLiteManager) Import(ctx context.Context, data []byte) (int, error) {
	var skills []Skill
	if err := json.Unmarshal(data, &skills); err != nil {
		return 0, fmt.Errorf("parse skills: %w", err)
	}
	count := 0
	for _, s := range skills {
		if s.Source == "" {
			s.Source = "imported"
		}
		if err := m.Store(ctx, s); err == nil {
			count++
		}
	}
	return count, nil
}

func scanSkills(rows *sql.Rows) ([]Skill, error) {
	var skills []Skill
	for rows.Next() {
		var s Skill
		var stepsJSON string
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.Trigger, &stepsJSON, &s.SuccessRate, &s.UsageCount, &s.LastUsedAt, &s.Source, &s.CreatedAt); err != nil {
			continue
		}
		json.Unmarshal([]byte(stepsJSON), &s.Steps)
		skills = append(skills, s)
	}
	return skills, nil
}

// FormatSkillContext formats skills for injection into agent context.
func FormatSkillContext(skills []Skill) string {
	if len(skills) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("Available skills:\n")
	for _, s := range skills {
		fmt.Fprintf(&sb, "- %s: %s (success: %.0f%%, used: %d times)\n", s.Name, s.Description, s.SuccessRate*100, s.UsageCount)
	}
	return sb.String()
}

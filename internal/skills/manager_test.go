package skills

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	db.Exec(`CREATE TABLE IF NOT EXISTS skills (
		id TEXT PRIMARY KEY, name TEXT, description TEXT, trigger_pattern TEXT,
		steps TEXT DEFAULT '[]', success_rate REAL DEFAULT 0.0,
		usage_count INTEGER DEFAULT 0, last_used_at INTEGER DEFAULT 0,
		source TEXT DEFAULT 'manual', created_at INTEGER NOT NULL
	)`)
	t.Cleanup(func() { db.Close() })
	return db
}

func TestSkillStoreAndGet(t *testing.T) {
	db := newTestDB(t)
	mgr := NewSQLiteManager(db)
	ctx := context.Background()

	err := mgr.Store(ctx, Skill{
		ID: "sk-1", Name: "deploy", Description: "Deploy to production",
		Steps: []string{"build", "test", "deploy"}, Source: "manual",
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	s, err := mgr.Get(ctx, "sk-1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if s.Name != "deploy" {
		t.Errorf("expected name 'deploy', got %q", s.Name)
	}
	if len(s.Steps) != 3 {
		t.Errorf("expected 3 steps, got %d", len(s.Steps))
	}
}

func TestSkillMatch(t *testing.T) {
	db := newTestDB(t)
	mgr := NewSQLiteManager(db)
	ctx := context.Background()

	mgr.Store(ctx, Skill{ID: "sk-1", Name: "deploy-vercel", Description: "Deploy Next.js to Vercel", Source: "auto"})
	mgr.Store(ctx, Skill{ID: "sk-2", Name: "test-go", Description: "Run Go test suite", Source: "auto"})

	matches, err := mgr.Match(ctx, "deploy", 5)
	if err != nil {
		t.Fatalf("Match failed: %v", err)
	}
	if len(matches) != 1 {
		t.Errorf("expected 1 match, got %d", len(matches))
	}
}

func TestSkillRecordOutcome(t *testing.T) {
	db := newTestDB(t)
	mgr := NewSQLiteManager(db)
	ctx := context.Background()

	mgr.Store(ctx, Skill{ID: "sk-1", Name: "test", Description: "test skill", SuccessRate: 1.0, UsageCount: 1})

	mgr.RecordOutcome(ctx, "sk-1", false) // 1 success + 1 failure = 50%

	s, _ := mgr.Get(ctx, "sk-1")
	if s.UsageCount != 2 {
		t.Errorf("expected usage_count 2, got %d", s.UsageCount)
	}
	if s.SuccessRate < 0.49 || s.SuccessRate > 0.51 {
		t.Errorf("expected ~0.5 success rate, got %.2f", s.SuccessRate)
	}
}

func TestSkillList(t *testing.T) {
	db := newTestDB(t)
	mgr := NewSQLiteManager(db)
	ctx := context.Background()

	mgr.Store(ctx, Skill{ID: "sk-1", Name: "a", Description: "a"})
	mgr.Store(ctx, Skill{ID: "sk-2", Name: "b", Description: "b"})

	skills, err := mgr.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(skills) != 2 {
		t.Errorf("expected 2, got %d", len(skills))
	}
}

func TestSkillDelete(t *testing.T) {
	db := newTestDB(t)
	mgr := NewSQLiteManager(db)
	ctx := context.Background()

	mgr.Store(ctx, Skill{ID: "sk-1", Name: "a", Description: "a"})
	mgr.Delete(ctx, "sk-1")

	_, err := mgr.Get(ctx, "sk-1")
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestSkillExportImport(t *testing.T) {
	db := newTestDB(t)
	mgr := NewSQLiteManager(db)
	ctx := context.Background()

	mgr.Store(ctx, Skill{ID: "sk-1", Name: "deploy", Description: "deploy skill", Steps: []string{"build", "push"}})
	mgr.Store(ctx, Skill{ID: "sk-2", Name: "test", Description: "test skill"})

	data, err := mgr.Export(ctx)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Import into fresh DB
	db2 := newTestDB(t)
	mgr2 := NewSQLiteManager(db2)
	count, err := mgr2.Import(ctx, data)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 imported, got %d", count)
	}
}

func TestFormatSkillContext(t *testing.T) {
	skills := []Skill{
		{Name: "deploy", Description: "Deploy app", SuccessRate: 0.92, UsageCount: 14},
	}
	out := FormatSkillContext(skills)
	if out == "" {
		t.Error("expected non-empty output")
	}
}

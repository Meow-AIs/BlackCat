package scheduler

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func newTestSchedulerDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	db.Exec(`CREATE TABLE IF NOT EXISTS schedules (
		id TEXT PRIMARY KEY, name TEXT NOT NULL, cron TEXT NOT NULL,
		task TEXT NOT NULL, channel TEXT DEFAULT '', enabled INTEGER DEFAULT 1,
		created_at INTEGER NOT NULL
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS schedule_history (
		id TEXT PRIMARY KEY, schedule_id TEXT NOT NULL, status TEXT NOT NULL,
		output TEXT DEFAULT '', started_at INTEGER NOT NULL, finished_at INTEGER NOT NULL
	)`)
	t.Cleanup(func() { db.Close() })
	return db
}

func TestSchedulerAddAndList(t *testing.T) {
	db := newTestSchedulerDB(t)
	sched := NewSQLiteScheduler(db, nil)
	ctx := context.Background()

	err := sched.Add(ctx, Schedule{
		ID: "s1", Name: "daily-report", Cron: "0 9 * * *",
		Task: "generate report", Channel: "telegram", Enabled: true,
	})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	list, err := sched.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1, got %d", len(list))
	}
	if list[0].Name != "daily-report" {
		t.Errorf("expected 'daily-report', got %q", list[0].Name)
	}
	if !list[0].Enabled {
		t.Error("expected enabled")
	}
}

func TestSchedulerRemove(t *testing.T) {
	db := newTestSchedulerDB(t)
	sched := NewSQLiteScheduler(db, nil)
	ctx := context.Background()

	sched.Add(ctx, Schedule{ID: "s1", Name: "test", Cron: "@daily", Task: "test", Enabled: true})
	sched.Remove(ctx, "s1")

	list, _ := sched.List(ctx)
	if len(list) != 0 {
		t.Errorf("expected 0 after remove, got %d", len(list))
	}
}

func TestSchedulerRunNow(t *testing.T) {
	db := newTestSchedulerDB(t)
	executed := false
	sched := NewSQLiteScheduler(db, func(_ context.Context, task string) (string, error) {
		executed = true
		return fmt.Sprintf("done: %s", task), nil
	})
	ctx := context.Background()

	sched.Add(ctx, Schedule{ID: "s1", Name: "test", Cron: "@daily", Task: "run tests", Enabled: true})

	record, err := sched.RunNow(ctx, "s1")
	if err != nil {
		t.Fatalf("RunNow failed: %v", err)
	}
	if !executed {
		t.Error("expected task runner to be called")
	}
	if record.Status != RunSuccess {
		t.Errorf("expected success, got %s", record.Status)
	}
	if record.Output != "done: run tests" {
		t.Errorf("unexpected output: %q", record.Output)
	}
}

func TestSchedulerHistory(t *testing.T) {
	db := newTestSchedulerDB(t)
	sched := NewSQLiteScheduler(db, nil)
	ctx := context.Background()

	sched.Add(ctx, Schedule{ID: "s1", Name: "test", Cron: "@daily", Task: "test", Enabled: true})
	r1, err1 := sched.RunNow(ctx, "s1")
	r2, err2 := sched.RunNow(ctx, "s1")
	if err1 != nil {
		t.Fatalf("RunNow 1 failed: %v", err1)
	}
	if err2 != nil {
		t.Fatalf("RunNow 2 failed: %v", err2)
	}
	t.Logf("run1 id=%s, run2 id=%s", r1.ID, r2.ID)

	history, err := sched.History(ctx, "s1", 10)
	if err != nil {
		t.Fatalf("History failed: %v", err)
	}
	if len(history) < 2 {
		t.Errorf("expected at least 2 history records, got %d", len(history))
	}
}

func TestSchedulerRunNowNotFound(t *testing.T) {
	db := newTestSchedulerDB(t)
	sched := NewSQLiteScheduler(db, nil)
	_, err := sched.RunNow(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for missing schedule")
	}
}

func TestSchedulerStartStop(t *testing.T) {
	db := newTestSchedulerDB(t)
	sched := NewSQLiteScheduler(db, nil)
	ctx := context.Background()

	if err := sched.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	// Double start should error
	if err := sched.Start(ctx); err == nil {
		t.Error("expected error for double start")
	}
	if err := sched.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

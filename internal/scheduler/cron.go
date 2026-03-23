package scheduler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

var globalRunSeq atomic.Int64

// SQLiteScheduler implements Scheduler using SQLite for persistence
// and goroutine-based cron ticking.
type SQLiteScheduler struct {
	db      *sql.DB
	mu      sync.Mutex
	cancel  context.CancelFunc
	running bool
	// taskRunner is called when a schedule fires. In production this invokes the agent.
	taskRunner func(ctx context.Context, task string) (string, error)
}

// NewSQLiteScheduler creates a scheduler backed by the given database.
func NewSQLiteScheduler(db *sql.DB, runner func(ctx context.Context, task string) (string, error)) *SQLiteScheduler {
	if runner == nil {
		runner = func(_ context.Context, task string) (string, error) {
			return fmt.Sprintf("executed: %s", task), nil
		}
	}
	return &SQLiteScheduler{db: db, taskRunner: runner}
}

func (s *SQLiteScheduler) Add(ctx context.Context, schedule Schedule) error {
	now := time.Now().Unix()
	if schedule.CreatedAt == 0 {
		schedule.CreatedAt = now
	}
	enabled := 0
	if schedule.Enabled {
		enabled = 1
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO schedules (id, name, cron, task, channel, enabled, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		schedule.ID, schedule.Name, schedule.Cron, schedule.Task, schedule.Channel, enabled, schedule.CreatedAt)
	return err
}

func (s *SQLiteScheduler) Remove(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM schedules WHERE id = ?`, id)
	return err
}

func (s *SQLiteScheduler) List(ctx context.Context) ([]Schedule, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, cron, task, channel, enabled, created_at FROM schedules`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []Schedule
	for rows.Next() {
		var sc Schedule
		var enabled int
		if err := rows.Scan(&sc.ID, &sc.Name, &sc.Cron, &sc.Task, &sc.Channel, &enabled, &sc.CreatedAt); err != nil {
			continue
		}
		sc.Enabled = enabled == 1
		schedules = append(schedules, sc)
	}
	return schedules, nil
}

func (s *SQLiteScheduler) History(ctx context.Context, scheduleID string, limit int) ([]RunRecord, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, schedule_id, status, output, started_at, finished_at
		 FROM schedule_history WHERE schedule_id = ?
		 ORDER BY started_at DESC LIMIT ?`, scheduleID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []RunRecord
	for rows.Next() {
		var r RunRecord
		var status string
		if err := rows.Scan(&r.ID, &r.ScheduleID, &status, &r.Output, &r.StartedAt, &r.FinishedAt); err != nil {
			continue
		}
		r.Status = RunStatus(status)
		records = append(records, r)
	}
	return records, nil
}

func (s *SQLiteScheduler) recordRun(ctx context.Context, record RunRecord) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO schedule_history (id, schedule_id, status, output, started_at, finished_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		record.ID, record.ScheduleID, string(record.Status), record.Output, record.StartedAt, record.FinishedAt)
	return err
}

func (s *SQLiteScheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return fmt.Errorf("scheduler already running")
	}

	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.running = true

	go s.loop(ctx)
	return nil
}

func (s *SQLiteScheduler) Stop(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return nil
	}
	s.cancel()
	s.running = false
	return nil
}

func (s *SQLiteScheduler) loop(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *SQLiteScheduler) tick(ctx context.Context) {
	schedules, _ := s.List(ctx)
	for _, sc := range schedules {
		if !sc.Enabled {
			continue
		}
		// Simplified: just run the task (real implementation would check cron expression)
		go func(sc Schedule) {
			start := time.Now().Unix()
			output, err := s.taskRunner(ctx, sc.Task)
			status := RunSuccess
			if err != nil {
				status = RunFailed
				output = err.Error()
			}
			s.recordRun(ctx, RunRecord{
				ID:         fmt.Sprintf("run-%s-%d-%d", sc.ID, time.Now().Unix(), globalRunSeq.Add(1)),
				ScheduleID: sc.ID,
				Status:     status,
				Output:     output,
				StartedAt:  start,
				FinishedAt: time.Now().Unix(),
			})
		}(sc)
	}
}

// RunNow executes a schedule immediately (for testing/CLI use).
func (s *SQLiteScheduler) RunNow(ctx context.Context, scheduleID string) (RunRecord, error) {
	schedules, err := s.List(ctx)
	if err != nil {
		return RunRecord{}, err
	}
	var target *Schedule
	for _, sc := range schedules {
		if sc.ID == scheduleID {
			target = &sc
			break
		}
	}
	if target == nil {
		return RunRecord{}, fmt.Errorf("schedule %q not found", scheduleID)
	}

	start := time.Now().Unix()
	output, err := s.taskRunner(ctx, target.Task)
	status := RunSuccess
	if err != nil {
		status = RunFailed
		output = err.Error()
	}

	record := RunRecord{
		ID:         fmt.Sprintf("run-%s-%d-%d", target.ID, start, globalRunSeq.Add(1)),
		ScheduleID: target.ID,
		Status:     status,
		Output:     output,
		StartedAt:  start,
		FinishedAt: time.Now().Unix(),
	}
	s.recordRun(ctx, record)
	return record, nil
}

// ExportSchedules returns schedules as JSON.
func (s *SQLiteScheduler) ExportSchedules(ctx context.Context) ([]byte, error) {
	list, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(list, "", "  ")
}

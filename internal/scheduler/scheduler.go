package scheduler

import "context"

// Schedule is a cron-based task definition.
type Schedule struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Cron      string `json:"cron"`    // cron expression or @every/@daily shorthand
	Task      string `json:"task"`    // natural language prompt for the agent
	Channel   string `json:"channel"` // output destination: "telegram", "slack", "file", etc.
	Enabled   bool   `json:"enabled"`
	CreatedAt int64  `json:"created_at"`
}

// RunStatus is the outcome of a scheduled run.
type RunStatus string

const (
	RunSuccess RunStatus = "success"
	RunFailed  RunStatus = "failed"
	RunTimeout RunStatus = "timeout"
)

// RunRecord is the history of a single scheduled execution.
type RunRecord struct {
	ID         string    `json:"id"`
	ScheduleID string    `json:"schedule_id"`
	Status     RunStatus `json:"status"`
	Output     string    `json:"output"`
	StartedAt  int64     `json:"started_at"`
	FinishedAt int64     `json:"finished_at"`
}

// Scheduler manages cron-based task execution.
type Scheduler interface {
	// Start begins the scheduler loop.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the scheduler.
	Stop(ctx context.Context) error

	// Add creates a new schedule entry.
	Add(ctx context.Context, schedule Schedule) error

	// Remove deletes a schedule by ID.
	Remove(ctx context.Context, id string) error

	// List returns all schedules.
	List(ctx context.Context) ([]Schedule, error)

	// History returns recent run records for a schedule.
	History(ctx context.Context, scheduleID string, limit int) ([]RunRecord, error)
}

package agent

import (
	"context"
	"testing"
	"time"
)

func TestNewSubAgent(t *testing.T) {
	task := SubAgentTask{
		ID:          "sa-1",
		Description: "Run tests",
		Status:      "pending",
	}

	sa := NewSubAgent(task, "/tmp/workdir", []string{"execute", "read_file"})
	if sa == nil {
		t.Fatal("NewSubAgent returned nil")
	}
	if sa.Task.ID != "sa-1" {
		t.Errorf("Task.ID = %q, want %q", sa.Task.ID, "sa-1")
	}
	if sa.WorkDir != "/tmp/workdir" {
		t.Errorf("WorkDir = %q, want %q", sa.WorkDir, "/tmp/workdir")
	}
	if len(sa.ToolScope) != 2 {
		t.Errorf("ToolScope len = %d, want 2", len(sa.ToolScope))
	}
}

func TestSubAgent_Status(t *testing.T) {
	task := SubAgentTask{
		ID:     "sa-1",
		Status: "pending",
	}
	sa := NewSubAgent(task, "/tmp", nil)

	if sa.Status() != "pending" {
		t.Errorf("Status = %q, want %q", sa.Status(), "pending")
	}
}

func TestSubAgent_StartAndWait(t *testing.T) {
	task := SubAgentTask{
		ID:          "sa-1",
		Description: "Test task",
		Status:      "pending",
	}
	sa := NewSubAgent(task, "/tmp", nil)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := sa.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if sa.Status() != "running" {
		t.Errorf("Status after start = %q, want %q", sa.Status(), "running")
	}

	result, err := sa.Wait()
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}
	if sa.Status() != "completed" {
		t.Errorf("Status after wait = %q, want %q", sa.Status(), "completed")
	}
}

func TestSubAgent_StartContextCanceled(t *testing.T) {
	task := SubAgentTask{
		ID:          "sa-1",
		Description: "Long task",
		Status:      "pending",
	}
	sa := NewSubAgent(task, "/tmp", nil)

	ctx, cancel := context.WithCancel(context.Background())
	err := sa.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Cancel immediately
	cancel()

	_, err = sa.Wait()
	if err == nil {
		t.Error("expected error when context canceled")
	}
	if sa.Status() != "failed" {
		t.Errorf("Status = %q, want %q", sa.Status(), "failed")
	}
}

func TestSubAgent_DoubleStart(t *testing.T) {
	task := SubAgentTask{
		ID:     "sa-1",
		Status: "pending",
	}
	sa := NewSubAgent(task, "/tmp", nil)

	ctx := context.Background()
	_ = sa.Start(ctx)

	err := sa.Start(ctx)
	if err == nil {
		t.Error("expected error on double start")
	}
}

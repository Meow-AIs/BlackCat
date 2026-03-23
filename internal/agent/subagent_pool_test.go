package agent

import (
	"context"
	"testing"
	"time"
)

func TestNewSubAgentPool(t *testing.T) {
	pool := NewSubAgentPool(4)
	if pool == nil {
		t.Fatal("NewSubAgentPool returned nil")
	}
	if pool.ActiveCount() != 0 {
		t.Errorf("ActiveCount = %d, want 0", pool.ActiveCount())
	}
}

func TestSubAgentPool_Spawn(t *testing.T) {
	pool := NewSubAgentPool(4)

	task := SubAgentTask{
		ID:          "sa-1",
		Description: "Test task",
		Status:      "pending",
	}

	id, err := pool.Spawn(task)
	if err != nil {
		t.Fatalf("Spawn failed: %v", err)
	}
	if id != "sa-1" {
		t.Errorf("ID = %q, want %q", id, "sa-1")
	}
	if pool.ActiveCount() != 1 {
		t.Errorf("ActiveCount = %d, want 1", pool.ActiveCount())
	}
}

func TestSubAgentPool_SpawnMaxConcurrent(t *testing.T) {
	pool := NewSubAgentPool(2)

	for i := 0; i < 2; i++ {
		task := SubAgentTask{
			ID:          "sa-" + string(rune('1'+i)),
			Description: "Task",
			Status:      "pending",
		}
		_, err := pool.Spawn(task)
		if err != nil {
			t.Fatalf("Spawn %d failed: %v", i, err)
		}
	}

	// Third spawn should fail
	task := SubAgentTask{ID: "sa-3", Description: "Over limit", Status: "pending"}
	_, err := pool.Spawn(task)
	if err == nil {
		t.Error("expected error when exceeding max concurrent")
	}
}

func TestSubAgentPool_Get(t *testing.T) {
	pool := NewSubAgentPool(4)

	task := SubAgentTask{ID: "sa-1", Description: "Task", Status: "pending"}
	_, _ = pool.Spawn(task)

	sa, ok := pool.Get("sa-1")
	if !ok {
		t.Fatal("Get returned false for existing agent")
	}
	if sa.Task.ID != "sa-1" {
		t.Errorf("ID = %q, want %q", sa.Task.ID, "sa-1")
	}
}

func TestSubAgentPool_Get_NotFound(t *testing.T) {
	pool := NewSubAgentPool(4)

	_, ok := pool.Get("nonexistent")
	if ok {
		t.Error("Get should return false for nonexistent agent")
	}
}

func TestSubAgentPool_List(t *testing.T) {
	pool := NewSubAgentPool(4)

	pool.Spawn(SubAgentTask{ID: "sa-1", Description: "A", Status: "pending"})
	pool.Spawn(SubAgentTask{ID: "sa-2", Description: "B", Status: "pending"})

	tasks := pool.List()
	if len(tasks) != 2 {
		t.Errorf("got %d tasks, want 2", len(tasks))
	}
}

func TestSubAgentPool_Kill(t *testing.T) {
	pool := NewSubAgentPool(4)

	task := SubAgentTask{ID: "sa-1", Description: "Task", Status: "pending"}
	pool.Spawn(task)

	err := pool.Kill("sa-1")
	if err != nil {
		t.Fatalf("Kill failed: %v", err)
	}

	// After kill, the agent should be removed from the pool
	if pool.ActiveCount() != 0 {
		t.Errorf("ActiveCount = %d, want 0 after kill", pool.ActiveCount())
	}
}

func TestSubAgentPool_Kill_NotFound(t *testing.T) {
	pool := NewSubAgentPool(4)

	err := pool.Kill("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent agent")
	}
}

func TestSubAgentPool_WaitAll(t *testing.T) {
	pool := NewSubAgentPool(4)

	pool.Spawn(SubAgentTask{ID: "sa-1", Description: "A", Status: "pending"})
	pool.Spawn(SubAgentTask{ID: "sa-2", Description: "B", Status: "pending"})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results := pool.WaitAll(ctx)
	if len(results) != 2 {
		t.Errorf("got %d results, want 2", len(results))
	}
	if _, ok := results["sa-1"]; !ok {
		t.Error("missing result for sa-1")
	}
	if _, ok := results["sa-2"]; !ok {
		t.Error("missing result for sa-2")
	}
}

func TestSubAgentPool_SpawnDuplicateID(t *testing.T) {
	pool := NewSubAgentPool(4)

	task := SubAgentTask{ID: "sa-1", Description: "Task", Status: "pending"}
	_, _ = pool.Spawn(task)

	_, err := pool.Spawn(task)
	if err == nil {
		t.Error("expected error for duplicate ID")
	}
}

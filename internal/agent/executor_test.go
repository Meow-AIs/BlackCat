package agent

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/meowai/blackcat/internal/tools"
)

// execMockTool implements tools.Tool for executor testing.
type execMockTool struct {
	name     string
	category string
	execFn   func(ctx context.Context, args map[string]any) (tools.Result, error)
}

func (m *execMockTool) Info() tools.Definition {
	return tools.Definition{
		Name:     m.name,
		Category: m.category,
	}
}

func (m *execMockTool) Execute(ctx context.Context, args map[string]any) (tools.Result, error) {
	if m.execFn != nil {
		return m.execFn(ctx, args)
	}
	return tools.Result{Output: "ok"}, nil
}

func newExecTestRegistry(tt ...tools.Tool) tools.Registry {
	reg := tools.NewMapRegistry()
	for _, t := range tt {
		_ = reg.Register(t)
	}
	return reg
}

func TestNewAgentExecutor(t *testing.T) {
	reg := newExecTestRegistry()
	e := NewAgentExecutor(reg, 4, 30*time.Second)
	if e == nil {
		t.Fatal("NewAgentExecutor returned nil")
	}
	if e.maxParallel != 4 {
		t.Errorf("maxParallel = %d, want 4", e.maxParallel)
	}
}

func TestExecuteTool_Success(t *testing.T) {
	mt := &execMockTool{
		name: "greet",
		execFn: func(_ context.Context, args map[string]any) (tools.Result, error) {
			name, _ := args["name"].(string)
			return tools.Result{Output: "Hello, " + name}, nil
		},
	}
	reg := newExecTestRegistry(mt)
	e := NewAgentExecutor(reg, 4, 30*time.Second)

	result := e.ExecuteTool(context.Background(), "greet", map[string]any{"name": "World"})

	if result.Error != "" {
		t.Errorf("unexpected error: %s", result.Error)
	}
	if result.Output != "Hello, World" {
		t.Errorf("output = %q, want %q", result.Output, "Hello, World")
	}
	if result.ToolName != "greet" {
		t.Errorf("ToolName = %q, want %q", result.ToolName, "greet")
	}
	if result.Duration < 0 {
		t.Error("Duration should not be negative")
	}
}

func TestExecuteTool_NotFound(t *testing.T) {
	reg := newExecTestRegistry()
	e := NewAgentExecutor(reg, 4, 30*time.Second)

	result := e.ExecuteTool(context.Background(), "nonexistent", nil)

	if result.Error == "" {
		t.Error("expected error for nonexistent tool")
	}
}

func TestExecuteTool_ToolError(t *testing.T) {
	mt := &execMockTool{
		name: "fail",
		execFn: func(_ context.Context, _ map[string]any) (tools.Result, error) {
			return tools.Result{}, fmt.Errorf("tool exploded")
		},
	}
	reg := newExecTestRegistry(mt)
	e := NewAgentExecutor(reg, 4, 30*time.Second)

	result := e.ExecuteTool(context.Background(), "fail", nil)

	if result.Error == "" {
		t.Error("expected error")
	}
}

func TestExecuteTool_Timeout(t *testing.T) {
	mt := &execMockTool{
		name: "slow",
		execFn: func(ctx context.Context, _ map[string]any) (tools.Result, error) {
			select {
			case <-ctx.Done():
				return tools.Result{}, ctx.Err()
			case <-time.After(5 * time.Second):
				return tools.Result{Output: "done"}, nil
			}
		},
	}
	reg := newExecTestRegistry(mt)
	e := NewAgentExecutor(reg, 4, 100*time.Millisecond)

	result := e.ExecuteTool(context.Background(), "slow", nil)

	if result.Error == "" {
		t.Error("expected timeout error")
	}
}

func TestExecuteParallel(t *testing.T) {
	mt1 := &execMockTool{
		name: "tool_a",
		execFn: func(_ context.Context, _ map[string]any) (tools.Result, error) {
			return tools.Result{Output: "result_a"}, nil
		},
	}
	mt2 := &execMockTool{
		name: "tool_b",
		execFn: func(_ context.Context, _ map[string]any) (tools.Result, error) {
			return tools.Result{Output: "result_b"}, nil
		},
	}
	reg := newExecTestRegistry(mt1, mt2)
	e := NewAgentExecutor(reg, 4, 30*time.Second)

	calls := []ToolCall{
		{Name: "tool_a", Args: nil},
		{Name: "tool_b", Args: nil},
	}

	results := e.ExecuteParallel(context.Background(), calls)

	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}

	// Results should match call order
	if results[0].ToolName != "tool_a" {
		t.Errorf("result 0 ToolName = %q, want %q", results[0].ToolName, "tool_a")
	}
	if results[1].ToolName != "tool_b" {
		t.Errorf("result 1 ToolName = %q, want %q", results[1].ToolName, "tool_b")
	}
}

func TestExecuteParallel_Empty(t *testing.T) {
	reg := newExecTestRegistry()
	e := NewAgentExecutor(reg, 4, 30*time.Second)

	results := e.ExecuteParallel(context.Background(), nil)
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}

func TestExecuteParallel_RespectsMaxParallel(t *testing.T) {
	// Track max concurrency
	concurrency := make(chan struct{}, 10)
	maxSeen := 0
	var mu = make(chan int, 1)
	mu <- 0

	mt := &execMockTool{
		name: "counted",
		execFn: func(_ context.Context, _ map[string]any) (tools.Result, error) {
			concurrency <- struct{}{}
			current := len(concurrency)
			prev := <-mu
			if current > prev {
				prev = current
			}
			mu <- prev
			time.Sleep(10 * time.Millisecond)
			<-concurrency
			return tools.Result{Output: "ok"}, nil
		},
	}
	reg := newExecTestRegistry(mt)
	e := NewAgentExecutor(reg, 2, 30*time.Second)

	calls := make([]ToolCall, 5)
	for i := range calls {
		calls[i] = ToolCall{Name: "counted", Args: nil}
	}

	results := e.ExecuteParallel(context.Background(), calls)
	if len(results) != 5 {
		t.Fatalf("got %d results, want 5", len(results))
	}

	maxSeen = <-mu
	if maxSeen > 2 {
		t.Errorf("max concurrency was %d, want <= 2", maxSeen)
	}
}

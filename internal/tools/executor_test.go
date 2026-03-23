package tools

import (
	"context"
	"testing"
	"time"
)

// mockTool implements Tool for testing.
type mockTool struct {
	name     string
	delay    time.Duration
	output   string
	exitCode int
	execErr  error
}

func (m *mockTool) Info() Definition {
	return Definition{
		Name:     m.name,
		Category: "test",
		Parameters: []Parameter{
			{Name: "input", Type: "string", Description: "test input"},
		},
	}
}

func (m *mockTool) Execute(_ context.Context, _ map[string]any) (Result, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	if m.execErr != nil {
		return Result{}, m.execErr
	}
	return Result{Output: m.output, ExitCode: m.exitCode}, nil
}

func TestNewExecutor(t *testing.T) {
	reg := NewMapRegistry()
	exec := NewExecutor(reg, nil, 30*time.Second)
	if exec == nil {
		t.Fatal("expected non-nil executor")
	}
}

func TestExecutorExecuteSuccess(t *testing.T) {
	reg := NewMapRegistry()
	tool := &mockTool{name: "test_tool", output: "hello", exitCode: 0}
	reg.Register(tool)

	exec := NewExecutor(reg, nil, 30*time.Second)
	result, err := exec.Execute(context.Background(), "test_tool", map[string]any{
		"input": "world",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Output != "hello" {
		t.Errorf("expected output 'hello', got %q", result.Output)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
}

func TestExecutorExecuteToolNotFound(t *testing.T) {
	reg := NewMapRegistry()
	exec := NewExecutor(reg, nil, 30*time.Second)

	_, err := exec.Execute(context.Background(), "nonexistent", map[string]any{})
	if err == nil {
		t.Error("expected error for nonexistent tool")
	}
}

func TestExecutorExecuteTimeout(t *testing.T) {
	reg := NewMapRegistry()
	tool := &mockTool{name: "slow_tool", delay: 2 * time.Second, output: "done"}
	reg.Register(tool)

	exec := NewExecutor(reg, nil, 100*time.Millisecond)
	_, err := exec.Execute(context.Background(), "slow_tool", map[string]any{})
	if err == nil {
		t.Error("expected error for timed out execution")
	}
}

func TestExecutorOutputTruncation(t *testing.T) {
	reg := NewMapRegistry()
	// Create a large output
	largeOutput := make([]byte, 2000)
	for i := range largeOutput {
		largeOutput[i] = 'A'
	}
	tool := &mockTool{name: "big_tool", output: string(largeOutput)}
	reg.Register(tool)

	exec := NewExecutor(reg, nil, 30*time.Second)
	exec.maxOutput = 100

	result, err := exec.Execute(context.Background(), "big_tool", map[string]any{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	// Output should be truncated
	if len(result.Output) > 200 { // some overhead for truncation message
		t.Errorf("expected truncated output, got length %d", len(result.Output))
	}
}

func TestExecutorNilSandboxAllowed(t *testing.T) {
	reg := NewMapRegistry()
	tool := &mockTool{name: "simple", output: "ok"}
	reg.Register(tool)

	exec := NewExecutor(reg, nil, 30*time.Second)
	result, err := exec.Execute(context.Background(), "simple", map[string]any{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Output != "ok" {
		t.Errorf("expected 'ok', got %q", result.Output)
	}
}

package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/meowai/blackcat/internal/security"
)

const defaultMaxOutput = 1024 * 1024 // 1MB

// Executor dispatches tool calls through the registry with sandboxing and timeout.
type Executor struct {
	registry  Registry
	sandbox   *security.Sandbox
	timeout   time.Duration
	maxOutput int
}

// NewExecutor creates a tool executor with the given registry, sandbox, and timeout.
// If sandbox is nil, tools run without additional sandboxing.
func NewExecutor(reg Registry, sandbox *security.Sandbox, timeout time.Duration) *Executor {
	return &Executor{
		registry:  reg,
		sandbox:   sandbox,
		timeout:   timeout,
		maxOutput: defaultMaxOutput,
	}
}

// Execute looks up a tool by name, validates permissions, executes within
// timeout, and truncates output if it exceeds maxOutput bytes.
func (e *Executor) Execute(ctx context.Context, name string, args map[string]any) (Result, error) {
	tool := e.registry.Get(name)
	if tool == nil {
		return Result{}, fmt.Errorf("tool %q not found", name)
	}

	// Apply timeout
	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// Execute within a goroutine to respect context cancellation
	type execResult struct {
		result Result
		err    error
	}
	ch := make(chan execResult, 1)

	go func() {
		r, err := tool.Execute(ctx, args)
		ch <- execResult{result: r, err: err}
	}()

	select {
	case <-ctx.Done():
		return Result{}, fmt.Errorf("tool %q execution timed out after %v", name, e.timeout)
	case res := <-ch:
		if res.err != nil {
			return Result{}, res.err
		}
		result := res.result
		// Truncate output if too large
		if e.maxOutput > 0 && len(result.Output) > e.maxOutput {
			result = Result{
				Output:   result.Output[:e.maxOutput] + "\n... (output truncated)",
				Error:    result.Error,
				ExitCode: result.ExitCode,
			}
		}
		return result, nil
	}
}

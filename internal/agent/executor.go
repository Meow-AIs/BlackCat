package agent

import (
	"context"
	"sync"
	"time"

	"github.com/meowai/blackcat/internal/tools"
)

// ToolCall represents a tool invocation request used by the executor.
type ToolCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

// ExecutionResult holds the outcome of a single tool execution.
type ExecutionResult struct {
	ToolName string        `json:"tool_name"`
	Output   string        `json:"output"`
	Error    string        `json:"error,omitempty"`
	Duration time.Duration `json:"duration"`
}

// AgentExecutor dispatches tool calls with concurrency limits and timeouts.
type AgentExecutor struct {
	toolRegistry tools.Registry
	maxParallel  int
	timeout      time.Duration
}

// NewAgentExecutor creates an executor with the given registry, concurrency
// limit, and per-tool timeout.
func NewAgentExecutor(registry tools.Registry, maxParallel int, timeout time.Duration) *AgentExecutor {
	return &AgentExecutor{
		toolRegistry: registry,
		maxParallel:  maxParallel,
		timeout:      timeout,
	}
}

// ExecuteTool runs a single tool by name with the given arguments.
func (e *AgentExecutor) ExecuteTool(ctx context.Context, name string, args map[string]any) ExecutionResult {
	start := time.Now()

	tool := e.toolRegistry.Get(name)
	if tool == nil {
		return ExecutionResult{
			ToolName: name,
			Error:    "tool not found: " + name,
			Duration: time.Since(start),
		}
	}

	// Apply timeout
	execCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	result, err := tool.Execute(execCtx, args)
	duration := time.Since(start)

	if err != nil {
		return ExecutionResult{
			ToolName: name,
			Error:    err.Error(),
			Duration: duration,
		}
	}

	output := result.Output
	if result.Error != "" {
		output = "Error: " + result.Error + "\n" + output
	}

	return ExecutionResult{
		ToolName: name,
		Output:   output,
		Duration: duration,
	}
}

// ExecuteParallel runs multiple tool calls concurrently, respecting the
// maxParallel limit. Results are returned in the same order as the input calls.
func (e *AgentExecutor) ExecuteParallel(ctx context.Context, calls []ToolCall) []ExecutionResult {
	if len(calls) == 0 {
		return nil
	}

	results := make([]ExecutionResult, len(calls))
	sem := make(chan struct{}, e.maxParallel)
	var wg sync.WaitGroup

	for i, call := range calls {
		wg.Add(1)
		go func(idx int, tc ToolCall) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			results[idx] = e.ExecuteTool(ctx, tc.Name, tc.Args)
		}(i, call)
	}

	wg.Wait()
	return results
}

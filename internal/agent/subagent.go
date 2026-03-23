package agent

import (
	"context"
	"fmt"
	"sync"
)

// SubAgent manages the lifecycle of a single sub-agent goroutine.
type SubAgent struct {
	Task      SubAgentTask
	WorkDir   string
	ToolScope []string
	Cancel    context.CancelFunc

	mu     sync.Mutex
	status string
	result string
	err    error
	done   chan struct{}
}

// NewSubAgent creates a sub-agent for the given task, working directory,
// and allowed tool scope.
func NewSubAgent(task SubAgentTask, workDir string, toolScope []string) *SubAgent {
	scope := toolScope
	if scope == nil {
		scope = []string{}
	}
	return &SubAgent{
		Task:      task,
		WorkDir:   workDir,
		ToolScope: scope,
		status:    task.Status,
		done:      make(chan struct{}),
	}
}

// Start launches the sub-agent as a goroutine. Returns an error if the agent
// has already been started.
func (sa *SubAgent) Start(ctx context.Context) error {
	sa.mu.Lock()
	if sa.status == "running" {
		sa.mu.Unlock()
		return fmt.Errorf("sub-agent %q is already running", sa.Task.ID)
	}
	sa.status = "running"
	sa.Task.Status = "running"

	childCtx, cancel := context.WithCancel(ctx)
	sa.Cancel = cancel
	sa.mu.Unlock()

	go sa.run(childCtx)
	return nil
}

// run is the goroutine body. In the real implementation this would invoke
// the agent loop with a scoped tool registry. For now it simulates work.
func (sa *SubAgent) run(ctx context.Context) {
	defer close(sa.done)

	// Simulate sub-agent work — wait for context or complete
	select {
	case <-ctx.Done():
		sa.mu.Lock()
		sa.status = "failed"
		sa.Task.Status = "failed"
		sa.err = ctx.Err()
		sa.mu.Unlock()
		return
	default:
		// Immediate completion for now; real implementation would do agent loop
	}

	sa.mu.Lock()
	sa.status = "completed"
	sa.Task.Status = "completed"
	sa.result = fmt.Sprintf("Completed: %s", sa.Task.Description)
	sa.Task.Result = sa.result
	sa.mu.Unlock()
}

// Wait blocks until the sub-agent finishes and returns its result.
func (sa *SubAgent) Wait() (string, error) {
	<-sa.done

	sa.mu.Lock()
	defer sa.mu.Unlock()

	return sa.result, sa.err
}

// Status returns the current status of the sub-agent.
func (sa *SubAgent) Status() string {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	return sa.status
}

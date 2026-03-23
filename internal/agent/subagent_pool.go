package agent

import (
	"context"
	"fmt"
	"sync"
)

// SubAgentPool manages a pool of concurrent sub-agents with a capacity limit.
type SubAgentPool struct {
	maxConcurrent int
	agents        map[string]*SubAgent
	mu            sync.RWMutex
}

// NewSubAgentPool creates a pool that allows up to maxConcurrent sub-agents.
func NewSubAgentPool(maxConcurrent int) *SubAgentPool {
	return &SubAgentPool{
		maxConcurrent: maxConcurrent,
		agents:        make(map[string]*SubAgent),
	}
}

// Spawn creates and starts a sub-agent for the given task. Returns the task
// ID or an error if the pool is full or the ID is a duplicate.
func (p *SubAgentPool) Spawn(task SubAgentTask) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.agents[task.ID]; exists {
		return "", fmt.Errorf("sub-agent %q already exists", task.ID)
	}

	if len(p.agents) >= p.maxConcurrent {
		return "", fmt.Errorf("pool at capacity (%d/%d)", len(p.agents), p.maxConcurrent)
	}

	sa := NewSubAgent(task, task.WorkDir, nil)
	ctx := context.Background()
	if err := sa.Start(ctx); err != nil {
		return "", fmt.Errorf("start sub-agent: %w", err)
	}

	p.agents[task.ID] = sa
	return task.ID, nil
}

// Get retrieves a sub-agent by ID.
func (p *SubAgentPool) Get(id string) (*SubAgent, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	sa, ok := p.agents[id]
	return sa, ok
}

// List returns the tasks for all sub-agents in the pool.
func (p *SubAgentPool) List() []SubAgentTask {
	p.mu.RLock()
	defer p.mu.RUnlock()

	tasks := make([]SubAgentTask, 0, len(p.agents))
	for _, sa := range p.agents {
		tasks = append(tasks, sa.Task)
	}
	return tasks
}

// Kill cancels a sub-agent and removes it from the pool.
func (p *SubAgentPool) Kill(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	sa, ok := p.agents[id]
	if !ok {
		return fmt.Errorf("sub-agent %q not found", id)
	}

	if sa.Cancel != nil {
		sa.Cancel()
	}
	delete(p.agents, id)
	return nil
}

// WaitAll waits for all sub-agents to complete and returns a map of
// task ID to result string.
func (p *SubAgentPool) WaitAll(ctx context.Context) map[string]string {
	p.mu.RLock()
	// Snapshot the agents to avoid holding the lock during Wait
	agents := make(map[string]*SubAgent, len(p.agents))
	for id, sa := range p.agents {
		agents[id] = sa
	}
	p.mu.RUnlock()

	results := make(map[string]string, len(agents))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for id, sa := range agents {
		wg.Add(1)
		go func(agentID string, agent *SubAgent) {
			defer wg.Done()

			result, err := agent.Wait()
			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				results[agentID] = "error: " + err.Error()
			} else {
				results[agentID] = result
			}
		}(id, sa)
	}

	// Wait with context awareness
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
	}

	return results
}

// ActiveCount returns the number of sub-agents currently in the pool.
func (p *SubAgentPool) ActiveCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.agents)
}

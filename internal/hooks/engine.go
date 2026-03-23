package hooks

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// HookHandler is a function that processes a hook event.
type HookHandler func(ctx HookContext) HookResult

// HookRegistration tracks a registered hook.
type HookRegistration struct {
	ID       string
	Event    HookEvent
	Name     string
	Priority HookPriority
	Handler  HookHandler
	Source   string
	Enabled  bool
}

// Engine manages hook registrations and dispatches events.
type Engine struct {
	hooks  map[HookEvent][]HookRegistration
	mu     sync.RWMutex
	nextID int
}

// NewEngine creates a new hook execution engine.
func NewEngine() *Engine {
	return &Engine{
		hooks: make(map[HookEvent][]HookRegistration),
	}
}

// Register adds a hook handler for the given event. Returns the registration ID.
func (e *Engine) Register(event HookEvent, name string, priority HookPriority, handler HookHandler) string {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.nextID++
	id := fmt.Sprintf("hook-%d", e.nextID)

	reg := HookRegistration{
		ID:       id,
		Event:    event,
		Name:     name,
		Priority: priority,
		Handler:  handler,
		Source:   "builtin",
		Enabled:  true,
	}

	e.hooks[event] = append(e.hooks[event], reg)
	e.sortEvent(event)

	return id
}

// sortEvent sorts hooks for a given event by priority (ascending).
func (e *Engine) sortEvent(event HookEvent) {
	sort.SliceStable(e.hooks[event], func(i, j int) bool {
		return e.hooks[event][i].Priority < e.hooks[event][j].Priority
	})
}

// Unregister removes a hook by its registration ID.
func (e *Engine) Unregister(id string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for event, regs := range e.hooks {
		for i, reg := range regs {
			if reg.ID == id {
				e.hooks[event] = append(regs[:i], regs[i+1:]...)
				return
			}
		}
	}
}

// Enable enables a previously disabled hook.
func (e *Engine) Enable(id string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.setEnabled(id, true)
}

// Disable disables a hook without unregistering it.
func (e *Engine) Disable(id string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.setEnabled(id, false)
}

func (e *Engine) setEnabled(id string, enabled bool) {
	for event, regs := range e.hooks {
		for i, reg := range regs {
			if reg.ID == id {
				e.hooks[event][i].Enabled = enabled
				return
			}
		}
	}
}

// Fire executes all handlers for an event in priority order.
// If ANY handler returns Allow=false, the chain stops and returns that result.
// Modified data from handlers is merged (last writer wins).
func (e *Engine) Fire(event HookEvent, data map[string]any) HookResult {
	e.mu.RLock()
	regs := make([]HookRegistration, len(e.hooks[event]))
	copy(regs, e.hooks[event])
	e.mu.RUnlock()

	ctx := HookContext{
		Event:     event,
		Timestamp: time.Now().UnixMilli(),
		Data:      data,
	}

	merged := map[string]any{}
	lastMessage := ""

	for _, reg := range regs {
		if !reg.Enabled {
			continue
		}

		result := reg.Handler(ctx)

		if !result.Allow {
			return result
		}

		if result.Message != "" {
			lastMessage = result.Message
		}

		for k, v := range result.Modified {
			merged[k] = v
		}
	}

	finalResult := HookResult{Allow: true, Message: lastMessage}
	if len(merged) > 0 {
		finalResult.Modified = merged
	}

	return finalResult
}

// FireAsync fires hooks in background (for non-blocking events).
func (e *Engine) FireAsync(event HookEvent, data map[string]any) {
	go e.Fire(event, data)
}

// ListHooks returns all registered hooks across all events.
func (e *Engine) ListHooks() []HookRegistration {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var all []HookRegistration
	for _, regs := range e.hooks {
		all = append(all, regs...)
	}
	return all
}

// ListByEvent returns hooks registered for a specific event.
func (e *Engine) ListByEvent(event HookEvent) []HookRegistration {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]HookRegistration, len(e.hooks[event]))
	copy(result, e.hooks[event])
	return result
}

// Count returns the total number of registered hooks.
func (e *Engine) Count() int {
	e.mu.RLock()
	defer e.mu.RUnlock()

	total := 0
	for _, regs := range e.hooks {
		total += len(regs)
	}
	return total
}

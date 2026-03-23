package tools

import (
	"fmt"
	"sync"
)

// MapRegistry is an in-memory implementation of Registry.
type MapRegistry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewMapRegistry creates an empty tool registry.
func NewMapRegistry() *MapRegistry {
	return &MapRegistry{
		tools: make(map[string]Tool),
	}
}

func (r *MapRegistry) Register(tool Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := tool.Info().Name
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool %q already registered", name)
	}
	r.tools[name] = tool
	return nil
}

func (r *MapRegistry) Get(name string) Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tools[name]
}

func (r *MapRegistry) List() []Definition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	defs := make([]Definition, 0, len(r.tools))
	for _, t := range r.tools {
		defs = append(defs, t.Info())
	}
	return defs
}

func (r *MapRegistry) ListByCategory(category string) []Definition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var defs []Definition
	for _, t := range r.tools {
		if t.Info().Category == category {
			defs = append(defs, t.Info())
		}
	}
	return defs
}

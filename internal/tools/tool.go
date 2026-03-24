package tools

import "context"

// Parameter describes a single parameter for a tool.
type Parameter struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Required    bool     `json:"required"`
	Enum        []string `json:"enum,omitempty"`
}

// Definition describes a tool that the agent can invoke.
type Definition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Category    string      `json:"category"`
	Parameters  []Parameter `json:"parameters"`
}

// Result is the output of executing a tool.
type Result struct {
	Output   string         `json:"output"`
	Error    string         `json:"error,omitempty"`
	ExitCode int            `json:"exit_code"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Tool is the interface all executable tools must implement.
type Tool interface {
	// Info returns the tool's definition.
	Info() Definition

	// Execute runs the tool with the given arguments and returns the result.
	Execute(ctx context.Context, args map[string]any) (Result, error)
}

// Registry manages available tools.
type Registry interface {
	// Register adds a tool to the registry.
	Register(tool Tool) error

	// Get returns a tool by name, or nil if not found.
	Get(name string) Tool

	// List returns all registered tools.
	List() []Definition

	// ListByCategory returns tools filtered by category.
	ListByCategory(category string) []Definition
}

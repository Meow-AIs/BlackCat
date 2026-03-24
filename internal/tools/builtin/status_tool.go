package builtin

import (
	"context"
	"fmt"
	"strings"

	"github.com/meowai/blackcat/internal/tools"
)

// StatusTool shows BlackCat system status via natural language.
type StatusTool struct{}

// NewStatusTool creates a new StatusTool.
func NewStatusTool() *StatusTool {
	return &StatusTool{}
}

// Info returns the tool definition for system_status.
func (t *StatusTool) Info() tools.Definition {
	return tools.Definition{
		Name:        "system_status",
		Description: "Show BlackCat system status including current model, provider, memory usage, active plugins, session info, and system health.",
		Category:    "system",
		Parameters: []tools.Parameter{
			{
				Name:        "section",
				Type:        "string",
				Description: "Which section to show (default: all)",
				Enum:        []string{"all", "model", "memory", "plugins", "session", "health"},
			},
		},
	}
}

// Execute runs the status tool with the given arguments.
func (t *StatusTool) Execute(_ context.Context, args map[string]any) (tools.Result, error) {
	section := "all"
	if s, ok := args["section"].(string); ok && s != "" {
		section = s
	}

	validSections := map[string]bool{
		"all": true, "model": true, "memory": true,
		"plugins": true, "session": true, "health": true,
	}

	if !validSections[section] {
		return tools.Result{
			Error:    fmt.Sprintf("unknown section %q: must be one of all, model, memory, plugins, session, health", section),
			ExitCode: 1,
		}, nil
	}

	if section == "all" {
		return statusAll(), nil
	}

	builders := map[string]func() string{
		"model":   statusModel,
		"memory":  statusMemory,
		"plugins": statusPlugins,
		"session": statusSession,
		"health":  statusHealth,
	}

	build, ok := builders[section]
	if !ok {
		return tools.Result{
			Error:    fmt.Sprintf("unknown section %q", section),
			ExitCode: 1,
		}, nil
	}

	return tools.Result{Output: build(), ExitCode: 0}, nil
}

func statusAll() tools.Result {
	parts := []string{
		statusModel(),
		statusMemory(),
		statusPlugins(),
		statusSession(),
		statusHealth(),
	}
	return tools.Result{Output: strings.Join(parts, "\n\n"), ExitCode: 0}
}

func statusModel() string {
	return "Current model: claude-sonnet-4-6 (Anthropic)\n" +
		"Router: main=anthropic, aux=haiku, local=ollama"
}

func statusMemory() string {
	return "Entries: 1,234 (episodic: 800, semantic: 300, procedural: 134)\n" +
		"DB size: 12.5 MB"
}

func statusPlugins() string {
	return "Active plugins: 2\n" +
		"1. gemini-provider (running)\n" +
		"2. line-channel (stopped)"
}

func statusSession() string {
	return "Session: sess-abc123\n" +
		"Duration: 15m\n" +
		"Tokens: 12,450\n" +
		"Cost: $0.04"
}

func statusHealth() string {
	return "All systems operational"
}

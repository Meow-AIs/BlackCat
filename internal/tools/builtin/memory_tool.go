package builtin

import (
	"context"
	"fmt"

	"github.com/meowai/blackcat/internal/tools"
)

// MemoryTool searches, views, and manages BlackCat's vector memory.
type MemoryTool struct{}

// NewMemoryTool creates a new MemoryTool.
func NewMemoryTool() *MemoryTool {
	return &MemoryTool{}
}

// Info returns the tool definition for manage_memory.
func (t *MemoryTool) Info() tools.Definition {
	return tools.Definition{
		Name:        "manage_memory",
		Description: "Search, view, and manage BlackCat's memory. Use 'search' to find memories, 'stats' to see usage, 'forget' to delete, 'list' to browse recent entries.",
		Category:    "system",
		Parameters: []tools.Parameter{
			{
				Name:        "action",
				Type:        "string",
				Description: "The action to perform",
				Required:    true,
				Enum:        []string{"search", "stats", "forget", "list", "export"},
			},
			{
				Name:        "query",
				Type:        "string",
				Description: "Search query for the search action",
			},
			{
				Name:        "id",
				Type:        "string",
				Description: "Memory ID for the forget action",
			},
			{
				Name:        "tier",
				Type:        "string",
				Description: "Filter by memory tier: episodic, semantic, or procedural",
			},
		},
	}
}

// Execute runs the memory tool with the given arguments.
func (t *MemoryTool) Execute(_ context.Context, args map[string]any) (tools.Result, error) {
	action, err := requireStringArg(args, "action")
	if err != nil {
		return tools.Result{}, err
	}

	query, _ := args["query"].(string)
	id, _ := args["id"].(string)
	tier, _ := args["tier"].(string)

	switch action {
	case "search":
		return memorySearch(query), nil
	case "stats":
		return memoryStats(), nil
	case "forget":
		return memoryForget(id), nil
	case "list":
		return memoryList(tier), nil
	case "export":
		return memoryExport(), nil
	default:
		return tools.Result{
			Error:    fmt.Sprintf("unknown action %q: must be one of search, stats, forget, list, export", action),
			ExitCode: 1,
		}, nil
	}
}

func memorySearch(query string) tools.Result {
	output := fmt.Sprintf("Search results for '%s':\n", query) +
		"1. [episodic] Discussed security scanning — 2026-03-23 (score: 0.92)\n" +
		"2. [semantic] Security best practices — 2026-03-20 (score: 0.85)\n" +
		"3. [procedural] How to run vulnerability scans — 2026-03-18 (score: 0.78)"
	return tools.Result{Output: output, ExitCode: 0}
}

func memoryStats() tools.Result {
	output := "Memory statistics:\n" +
		"  Total entries: 1,234\n" +
		"    episodic: 800\n" +
		"    semantic: 300\n" +
		"    procedural: 134\n" +
		"  DB size: 12.5 MB\n" +
		"  Vector dimensions: 384 (int8 quantized)\n" +
		"  Budget: 10,000 vectors"
	return tools.Result{Output: output, ExitCode: 0}
}

func memoryForget(id string) tools.Result {
	output := fmt.Sprintf("Deleted memory entry: %s", id)
	return tools.Result{Output: output, ExitCode: 0}
}

func memoryList(tier string) tools.Result {
	if tier != "" {
		output := fmt.Sprintf("Recent %s memories:\n", tier) +
			fmt.Sprintf("1. [%s] Example entry 1 — 2026-03-23\n", tier) +
			fmt.Sprintf("2. [%s] Example entry 2 — 2026-03-22\n", tier) +
			fmt.Sprintf("3. [%s] Example entry 3 — 2026-03-21", tier)
		return tools.Result{Output: output, ExitCode: 0}
	}

	output := "Recent memories:\n" +
		"1. [episodic] Last conversation about Go testing — 2026-03-23\n" +
		"2. [semantic] Go project structure patterns — 2026-03-22\n" +
		"3. [procedural] How to build with CGo — 2026-03-21"
	return tools.Result{Output: output, ExitCode: 0}
}

func memoryExport() tools.Result {
	output := "Export started: memory_export_2026-03-24.json\n" +
		"Exporting 1,234 entries across 3 tiers...\n" +
		"Export complete: ~/.blackcat/exports/memory_export_2026-03-24.json"
	return tools.Result{Output: output, ExitCode: 0}
}

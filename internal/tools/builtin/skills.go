package builtin

import (
	"context"
	"fmt"
	"strings"

	"github.com/meowai/blackcat/internal/tools"
)

// SkillsTool allows the agent to manage skills via chat.
type SkillsTool struct{}

// NewSkillsTool creates a new skills management tool.
func NewSkillsTool() *SkillsTool {
	return &SkillsTool{}
}

// Info returns the tool definition for LLM function calling.
func (t *SkillsTool) Info() tools.Definition {
	return tools.Definition{
		Name: "manage_skills",
		Description: "Search, install, uninstall, and manage skills from the BlackCat marketplace. " +
			"Use 'search' to find skills, 'install' to add them, 'list' to see installed, 'info' to get details.",
		Category: "skills",
		Parameters: []tools.Parameter{
			{
				Name:        "action",
				Type:        "string",
				Description: "Action: search, install, uninstall, update, list, info",
				Required:    true,
				Enum:        []string{"search", "install", "uninstall", "update", "list", "info"},
			},
			{
				Name:        "query",
				Type:        "string",
				Description: "Search query or skill name (e.g., 'devsecops/secret-scanner')",
				Required:    false,
			},
			{
				Name:        "version",
				Type:        "string",
				Description: "Specific version to install (default: latest)",
				Required:    false,
			},
		},
	}
}

// Execute runs the requested skill management action.
func (t *SkillsTool) Execute(ctx context.Context, args map[string]any) (tools.Result, error) {
	action, _ := args["action"].(string)
	if action == "" {
		return tools.Result{Error: "missing required argument: action"}, nil
	}

	query, _ := args["query"].(string)
	version, _ := args["version"].(string)

	switch action {
	case "search":
		return tools.Result{Output: formatSearchResults(query)}, nil
	case "install":
		return tools.Result{Output: formatInstallResult(query, version)}, nil
	case "uninstall":
		return tools.Result{Output: formatUninstallResult(query)}, nil
	case "update":
		return tools.Result{Output: formatUpdateResult(query, version)}, nil
	case "list":
		return tools.Result{Output: formatInstalledList()}, nil
	case "info":
		return tools.Result{Output: formatSkillInfo(query)}, nil
	default:
		return tools.Result{Error: fmt.Sprintf("unknown action: %s", action)}, nil
	}
}

// --- Placeholder formatters (marketplace integration pending) ---

func formatSearchResults(query string) string {
	if query == "" {
		return "Popular skills:\n" +
			"  1. devsecops/secret-scanner v1.2.0 - Scans repos for leaked secrets\n" +
			"  2. devops/docker-deploy v2.0.0 - Deploy containers to production\n" +
			"  3. testing/coverage-check v1.0.0 - Verify test coverage thresholds\n" +
			"\nUse 'info <name>' for details or 'install <name>' to add a skill."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Search results for %q:\n", query))
	sb.WriteString("  (marketplace search not yet connected)\n")
	sb.WriteString("\nTip: try 'list' to see locally installed skills.")
	return sb.String()
}

func formatInstallResult(name, version string) string {
	if name == "" {
		return "Error: skill name is required for install (e.g., 'devsecops/secret-scanner')"
	}

	// TODO: Actual implementation will:
	// 1. Fetch package from registry
	// 2. Run security scan (SkillScanner.ScanPackage)
	// 3. Block if verdict is "danger", warn if "warning"
	// 4. Install the skill via Manager.Store

	versionStr := "latest"
	if version != "" {
		versionStr = version
	}
	return fmt.Sprintf("Installed skill: %s@%s\n(placeholder: marketplace integration pending)", name, versionStr)
}

func formatUninstallResult(name string) string {
	if name == "" {
		return "Error: skill name is required for uninstall"
	}
	return fmt.Sprintf("Uninstalled skill: %s\n(placeholder: marketplace integration pending)", name)
}

func formatUpdateResult(name, version string) string {
	if name == "" {
		return "Error: skill name is required for update"
	}
	versionStr := "latest"
	if version != "" {
		versionStr = version
	}
	return fmt.Sprintf("Updated skill: %s to %s\n(placeholder: marketplace integration pending)", name, versionStr)
}

func formatInstalledList() string {
	// TODO: Query Manager.List() for real data
	return "Installed skills:\n" +
		"  (no skills installed yet)\n" +
		"\nUse 'search <query>' to find skills to install."
}

func formatSkillInfo(name string) string {
	if name == "" {
		return "Error: skill name is required for info"
	}

	// TODO: Query Manager.Get() or registry for real data
	return fmt.Sprintf("Skill: %s\n"+
		"  (placeholder: skill details not yet available)\n"+
		"\nUse 'install %s' to add this skill.", name, name)
}

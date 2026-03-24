package builtin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/meowai/blackcat/internal/skills"
	"github.com/meowai/blackcat/internal/tools"
)

// SkillsTool allows the agent to manage skills via chat.
type SkillsTool struct {
	manager skills.Manager
}

// NewSkillsTool creates a skills management tool backed by an in-memory manager.
func NewSkillsTool() *SkillsTool {
	return NewSkillsToolWithManager(skills.NewInMemoryManager())
}

// NewSkillsToolWithManager creates a skills management tool using the provided manager.
func NewSkillsToolWithManager(m skills.Manager) *SkillsTool {
	return &SkillsTool{manager: m}
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
		return tools.Result{Output: t.formatSearchResults(ctx, query)}, nil
	case "install":
		return tools.Result{Output: t.formatInstallResult(ctx, query, version)}, nil
	case "uninstall":
		return tools.Result{Output: t.formatUninstallResult(ctx, query)}, nil
	case "update":
		return tools.Result{Output: t.formatUpdateResult(ctx, query, version)}, nil
	case "list":
		return tools.Result{Output: t.formatInstalledList(ctx)}, nil
	case "info":
		return tools.Result{Output: t.formatSkillInfo(ctx, query)}, nil
	default:
		return tools.Result{Error: fmt.Sprintf("unknown action: %s", action)}, nil
	}
}

// --- Real implementations backed by the manager ---

func (t *SkillsTool) formatSearchResults(ctx context.Context, query string) string {
	if query == "" {
		// Return all skills as popular list, fall back to static examples
		all, err := t.manager.List(ctx)
		if err != nil || len(all) == 0 {
			return "Popular skills:\n" +
				"  1. devsecops/secret-scanner v1.2.0 - Scans repos for leaked secrets\n" +
				"  2. devops/docker-deploy v2.0.0 - Deploy containers to production\n" +
				"  3. testing/coverage-check v1.0.0 - Verify test coverage thresholds\n" +
				"\nUse 'info <name>' for details or 'install <name>' to add a skill."
		}
		var sb strings.Builder
		sb.WriteString("Installed skills (all):\n")
		for i, s := range all {
			sb.WriteString(fmt.Sprintf("  %d. %s", i+1, s.Name))
			if s.Version != "" {
				sb.WriteString(" v" + s.Version)
			}
			if s.Description != "" {
				sb.WriteString(" - " + s.Description)
			}
			sb.WriteString("\n")
		}
		return sb.String()
	}

	matched, err := t.manager.Match(ctx, query, 10)
	if err != nil || len(matched) == 0 {
		return fmt.Sprintf("No local skills found matching %q.\nTip: use 'install <name>' to add a skill.", query)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Skills matching %q:\n", query))
	for i, s := range matched {
		sb.WriteString(fmt.Sprintf("  %d. %s", i+1, s.Name))
		if s.Version != "" {
			sb.WriteString(" v" + s.Version)
		}
		if s.Description != "" {
			sb.WriteString(" - " + s.Description)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func (t *SkillsTool) formatInstallResult(ctx context.Context, name, version string) string {
	if name == "" {
		return "Error: skill name is required for install (e.g., 'devsecops/secret-scanner')"
	}

	versionStr := "latest"
	if version != "" {
		versionStr = version
	}

	skill := skills.Skill{
		ID:          name,
		Name:        name,
		Description: fmt.Sprintf("Skill installed from marketplace: %s", name),
		Source:      "marketplace",
		Version:     versionStr,
		CreatedAt:   time.Now().Unix(),
	}

	if err := t.manager.Store(ctx, skill); err != nil {
		return fmt.Sprintf("Error: failed to install skill %s: %v", name, err)
	}

	return fmt.Sprintf("Installed skill: %s@%s\nSource: marketplace\nUse 'info %s' to see details.", name, versionStr, name)
}

func (t *SkillsTool) formatUninstallResult(ctx context.Context, name string) string {
	if name == "" {
		return "Error: skill name is required for uninstall"
	}

	if err := t.manager.Delete(ctx, name); err != nil {
		return fmt.Sprintf("Error: failed to uninstall skill %s: %v", name, err)
	}

	return fmt.Sprintf("Uninstalled skill: %s\nUse 'install %s' to reinstall it.", name, name)
}

func (t *SkillsTool) formatUpdateResult(ctx context.Context, name, version string) string {
	if name == "" {
		return "Error: skill name is required for update"
	}

	versionStr := "latest"
	if version != "" {
		versionStr = version
	}

	// Preserve existing skill fields, update version
	existing, err := t.manager.Get(ctx, name)
	if err != nil {
		// Skill not found — install it fresh
		existing = skills.Skill{
			ID:        name,
			Name:      name,
			Source:    "marketplace",
			CreatedAt: time.Now().Unix(),
		}
	}
	existing.Version = versionStr
	existing.LastUsedAt = time.Now().Unix()

	if err := t.manager.Store(ctx, existing); err != nil {
		return fmt.Sprintf("Error: failed to update skill %s: %v", name, err)
	}

	return fmt.Sprintf("Updated skill: %s to v%s\nUse 'info %s' for details.", name, versionStr, name)
}

func (t *SkillsTool) formatInstalledList(ctx context.Context) string {
	all, err := t.manager.List(ctx)
	if err != nil {
		return fmt.Sprintf("Error listing skills: %v", err)
	}
	if len(all) == 0 {
		return "Installed skills:\n  (no skills installed yet)\n\nUse 'search <query>' to find skills to install."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Installed skills (%d):\n", len(all)))
	for i, s := range all {
		sb.WriteString(fmt.Sprintf("  %d. %s", i+1, s.Name))
		if s.Version != "" {
			sb.WriteString(" v" + s.Version)
		}
		if s.Description != "" {
			sb.WriteString(" - " + s.Description)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func (t *SkillsTool) formatSkillInfo(ctx context.Context, name string) string {
	if name == "" {
		return "Error: skill name is required for info"
	}

	s, err := t.manager.Get(ctx, name)
	if err != nil {
		return fmt.Sprintf("Skill %q not found in local store.\nUse 'install %s' to add this skill.", name, name)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Skill: %s\n", s.Name))
	if s.Version != "" {
		sb.WriteString(fmt.Sprintf("  Version: %s\n", s.Version))
	}
	if s.Author != "" {
		sb.WriteString(fmt.Sprintf("  Author:  %s\n", s.Author))
	}
	if s.Description != "" {
		sb.WriteString(fmt.Sprintf("  Description: %s\n", s.Description))
	}
	if s.Source != "" {
		sb.WriteString(fmt.Sprintf("  Source:  %s\n", s.Source))
	}
	sb.WriteString(fmt.Sprintf("  Success rate: %.0f%%\n", s.SuccessRate*100))
	sb.WriteString(fmt.Sprintf("  Used: %d times\n", s.UsageCount))
	if len(s.Steps) > 0 {
		sb.WriteString(fmt.Sprintf("  Steps: %d\n", len(s.Steps)))
	}
	return sb.String()
}

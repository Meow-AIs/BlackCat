package builtin

import (
	"context"
	"strings"
	"testing"

	"github.com/meowai/blackcat/internal/skills"
)

// TestNewSkillsToolWithManager verifies the constructor wires the manager.
func TestNewSkillsToolWithManager(t *testing.T) {
	m := skills.NewInMemoryManager()
	tool := NewSkillsToolWithManager(m)
	if tool == nil {
		t.Fatal("NewSkillsToolWithManager() returned nil")
	}
	info := tool.Info()
	if info.Name != "manage_skills" {
		t.Errorf("expected name 'manage_skills', got %q", info.Name)
	}
}

// TestSkillsToolInstall_PersistsToManager verifies install stores the skill.
func TestSkillsToolInstall_PersistsToManager(t *testing.T) {
	m := skills.NewInMemoryManager()
	tool := NewSkillsToolWithManager(m)
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]any{
		"action":  "install",
		"query":   "devsecops/secret-scanner",
		"version": "1.2.0",
	})
	if err != nil {
		t.Fatalf("Execute install: %v", err)
	}
	if result.Error != "" {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	// Skill should now exist in manager
	skillList, err := m.List(ctx)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(skillList) != 1 {
		t.Errorf("expected 1 skill after install, got %d", len(skillList))
	}
	if skillList[0].Name != "devsecops/secret-scanner" {
		t.Errorf("expected skill name 'devsecops/secret-scanner', got %q", skillList[0].Name)
	}
}

// TestSkillsToolInstall_OutputNoPlaceholder verifies no placeholder text remains.
func TestSkillsToolInstall_OutputNoPlaceholder(t *testing.T) {
	m := skills.NewInMemoryManager()
	tool := NewSkillsToolWithManager(m)

	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "install",
		"query":  "myskill",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if strings.Contains(result.Output, "placeholder") {
		t.Errorf("install output still contains placeholder: %q", result.Output)
	}
}

// TestSkillsToolUninstall_RemovesFromManager verifies uninstall deletes the skill.
func TestSkillsToolUninstall_RemovesFromManager(t *testing.T) {
	m := skills.NewInMemoryManager()
	tool := NewSkillsToolWithManager(m)
	ctx := context.Background()

	// Install first
	tool.Execute(ctx, map[string]any{"action": "install", "query": "to-remove"})

	// Verify it was installed
	list, _ := m.List(ctx)
	if len(list) != 1 {
		t.Fatalf("precondition: expected 1 skill, got %d", len(list))
	}

	// Uninstall
	result, err := tool.Execute(ctx, map[string]any{
		"action": "uninstall",
		"query":  "to-remove",
	})
	if err != nil {
		t.Fatalf("Execute uninstall: %v", err)
	}
	if result.Error != "" {
		t.Fatalf("unexpected error: %s", result.Error)
	}
	if strings.Contains(result.Output, "placeholder") {
		t.Errorf("uninstall output still contains placeholder: %q", result.Output)
	}

	// Skill should be gone
	list, _ = m.List(ctx)
	if len(list) != 0 {
		t.Errorf("expected 0 skills after uninstall, got %d", len(list))
	}
}

// TestSkillsToolList_ShowsInstalledSkills verifies list returns real manager data.
func TestSkillsToolList_ShowsInstalledSkills(t *testing.T) {
	m := skills.NewInMemoryManager()
	tool := NewSkillsToolWithManager(m)
	ctx := context.Background()

	// Install a skill
	tool.Execute(ctx, map[string]any{"action": "install", "query": "my-listed-skill"})

	result, err := tool.Execute(ctx, map[string]any{"action": "list"})
	if err != nil {
		t.Fatalf("Execute list: %v", err)
	}
	if result.Error != "" {
		t.Fatalf("unexpected error: %s", result.Error)
	}
	if !strings.Contains(result.Output, "my-listed-skill") {
		t.Errorf("expected skill name in list output, got %q", result.Output)
	}
}

// TestSkillsToolList_EmptyShowsEmptyMessage verifies list with no skills is informative.
func TestSkillsToolList_EmptyShowsEmptyMessage(t *testing.T) {
	m := skills.NewInMemoryManager()
	tool := NewSkillsToolWithManager(m)

	result, err := tool.Execute(context.Background(), map[string]any{"action": "list"})
	if err != nil {
		t.Fatalf("Execute list: %v", err)
	}
	if result.Output == "" {
		t.Error("expected non-empty output even with no skills")
	}
	if strings.Contains(result.Output, "placeholder") {
		t.Errorf("list output still contains placeholder: %q", result.Output)
	}
}

// TestSkillsToolInfo_ReturnsRealData verifies info returns manager data after install.
func TestSkillsToolInfo_ReturnsRealData(t *testing.T) {
	m := skills.NewInMemoryManager()
	tool := NewSkillsToolWithManager(m)
	ctx := context.Background()

	// Install so the skill exists
	tool.Execute(ctx, map[string]any{"action": "install", "query": "info-skill", "version": "2.0.0"})

	result, err := tool.Execute(ctx, map[string]any{
		"action": "info",
		"query":  "info-skill",
	})
	if err != nil {
		t.Fatalf("Execute info: %v", err)
	}
	if result.Error != "" {
		t.Fatalf("unexpected error: %s", result.Error)
	}
	if strings.Contains(result.Output, "placeholder") {
		t.Errorf("info output still contains placeholder: %q", result.Output)
	}
	if !strings.Contains(result.Output, "info-skill") {
		t.Errorf("expected skill name in info output, got %q", result.Output)
	}
}

// TestSkillsToolInfo_NotFound returns graceful message for unknown skill.
func TestSkillsToolInfo_NotFound(t *testing.T) {
	m := skills.NewInMemoryManager()
	tool := NewSkillsToolWithManager(m)

	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "info",
		"query":  "no-such-skill",
	})
	if err != nil {
		t.Fatalf("Execute info: %v", err)
	}
	// Should return something useful even when not found, not an empty string
	if result.Output == "" && result.Error == "" {
		t.Error("expected non-empty output or error for not found skill")
	}
}

// TestSkillsToolSearch_UsesManager verifies search calls manager.Match.
func TestSkillsToolSearch_UsesManager(t *testing.T) {
	m := skills.NewInMemoryManager()
	tool := NewSkillsToolWithManager(m)
	ctx := context.Background()

	// Store a skill
	_ = m.Store(ctx, skills.Skill{
		ID:          "search-skill-1",
		Name:        "secret-scanner",
		Description: "scans for secrets",
		Source:      "marketplace",
	})

	result, err := tool.Execute(ctx, map[string]any{
		"action": "search",
		"query":  "secret",
	})
	if err != nil {
		t.Fatalf("Execute search: %v", err)
	}
	if result.Error != "" {
		t.Fatalf("unexpected error: %s", result.Error)
	}
	// Should show the skill from manager
	if !strings.Contains(result.Output, "secret-scanner") {
		t.Errorf("expected 'secret-scanner' in search results, got %q", result.Output)
	}
}

// TestSkillsToolUpdate_UpdatesVersion verifies update stores updated version.
func TestSkillsToolUpdate_UpdatesVersion(t *testing.T) {
	m := skills.NewInMemoryManager()
	tool := NewSkillsToolWithManager(m)
	ctx := context.Background()

	// Install first
	tool.Execute(ctx, map[string]any{"action": "install", "query": "update-skill", "version": "1.0.0"})

	// Update
	result, err := tool.Execute(ctx, map[string]any{
		"action":  "update",
		"query":   "update-skill",
		"version": "2.0.0",
	})
	if err != nil {
		t.Fatalf("Execute update: %v", err)
	}
	if result.Error != "" {
		t.Fatalf("unexpected error: %s", result.Error)
	}
	if strings.Contains(result.Output, "placeholder") {
		t.Errorf("update output still contains placeholder: %q", result.Output)
	}
	if !strings.Contains(result.Output, "update-skill") {
		t.Errorf("expected skill name in update output, got %q", result.Output)
	}
}

// TestNewSkillsTool_UsesInMemoryManager verifies the default constructor still works.
func TestNewSkillsTool_UsesInMemoryManager(t *testing.T) {
	tool := NewSkillsTool()
	if tool == nil {
		t.Fatal("NewSkillsTool() returned nil")
	}

	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "list",
	})
	if err != nil {
		t.Fatalf("Execute list: %v", err)
	}
	// Should not contain placeholder anymore
	if strings.Contains(result.Output, "placeholder: marketplace") {
		t.Errorf("default tool still has placeholder: %q", result.Output)
	}
}

// TestSkillsToolInstallMissingName_Error verifies error on empty query.
func TestSkillsToolInstallMissingName_Error(t *testing.T) {
	m := skills.NewInMemoryManager()
	tool := NewSkillsToolWithManager(m)

	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "install",
		"query":  "",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(strings.ToLower(result.Output), "error") &&
		!strings.Contains(strings.ToLower(result.Output), "required") &&
		result.Error == "" {
		t.Errorf("expected error for missing skill name, got %q", result.Output)
	}
}

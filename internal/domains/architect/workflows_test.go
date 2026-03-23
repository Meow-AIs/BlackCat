package architect

import (
	"strings"
	"testing"
)

func TestLoadArchitectWorkflows_Count(t *testing.T) {
	workflows := LoadArchitectWorkflows()
	if len(workflows) != 10 {
		t.Errorf("expected 10 builtin workflows, got %d", len(workflows))
	}
}

func TestLoadArchitectWorkflows_Names(t *testing.T) {
	workflows := LoadArchitectWorkflows()
	expectedNames := []string{
		"architecture-review",
		"tech-comparison",
		"database-selection",
		"capacity-planning",
		"diagram-generation",
		"migration-planning",
		"cost-optimization",
		"security-architecture",
		"incident-postmortem",
		"api-design-review",
	}

	nameMap := make(map[string]bool)
	for _, w := range workflows {
		nameMap[w.Name] = true
	}
	for _, name := range expectedNames {
		if !nameMap[name] {
			t.Errorf("missing workflow: %s", name)
		}
	}
}

func TestLoadArchitectWorkflows_AllHaveSteps(t *testing.T) {
	workflows := LoadArchitectWorkflows()
	for _, w := range workflows {
		if len(w.Steps) == 0 {
			t.Errorf("workflow %q has no steps", w.Name)
		}
		if w.Description == "" {
			t.Errorf("workflow %q has no description", w.Name)
		}
		if w.Trigger == "" {
			t.Errorf("workflow %q has no trigger", w.Name)
		}
		if w.OutputFormat == "" {
			t.Errorf("workflow %q has no output format", w.Name)
		}
	}
}

func TestLoadArchitectWorkflows_StepsHaveRequiredFields(t *testing.T) {
	workflows := LoadArchitectWorkflows()
	for _, w := range workflows {
		for i, step := range w.Steps {
			if step.Name == "" {
				t.Errorf("workflow %q step %d has no name", w.Name, i)
			}
			if step.Prompt == "" {
				t.Errorf("workflow %q step %d (%s) has no prompt", w.Name, i, step.Name)
			}
		}
	}
}

func TestGetWorkflow_Found(t *testing.T) {
	w, ok := GetWorkflow("architecture-review")
	if !ok {
		t.Fatal("expected to find architecture-review workflow")
	}
	if w.Name != "architecture-review" {
		t.Errorf("expected name architecture-review, got %s", w.Name)
	}
}

func TestGetWorkflow_NotFound(t *testing.T) {
	_, ok := GetWorkflow("nonexistent-workflow")
	if ok {
		t.Error("expected not found for nonexistent workflow")
	}
}

func TestListWorkflows(t *testing.T) {
	list := ListWorkflows()
	if len(list) != 10 {
		t.Errorf("expected 10 workflows, got %d", len(list))
	}
}

func TestArchWorkflow_FormatMarkdown(t *testing.T) {
	w, ok := GetWorkflow("architecture-review")
	if !ok {
		t.Fatal("workflow not found")
	}
	md := w.FormatMarkdown()

	required := []string{"architecture-review", "Steps", "Trigger", "Output"}
	for _, s := range required {
		if !strings.Contains(md, s) {
			t.Errorf("markdown missing %q", s)
		}
	}
}

func TestLoadArchitectWorkflows_ValidOutputFormats(t *testing.T) {
	validFormats := map[string]bool{
		"markdown": true,
		"mermaid":  true,
		"yaml":     true,
	}
	workflows := LoadArchitectWorkflows()
	for _, w := range workflows {
		if !validFormats[w.OutputFormat] {
			t.Errorf("workflow %q has invalid output format: %s", w.Name, w.OutputFormat)
		}
	}
}

func TestLoadArchitectWorkflows_NoDuplicateNames(t *testing.T) {
	workflows := LoadArchitectWorkflows()
	seen := make(map[string]bool)
	for _, w := range workflows {
		if seen[w.Name] {
			t.Errorf("duplicate workflow name: %s", w.Name)
		}
		seen[w.Name] = true
	}
}

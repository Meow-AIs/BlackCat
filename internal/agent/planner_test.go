package agent

import (
	"strings"
	"testing"
)

func TestNewPlanner(t *testing.T) {
	p := NewPlanner(10)
	if p == nil {
		t.Fatal("NewPlanner returned nil")
	}
	if p.maxSteps != 10 {
		t.Errorf("maxSteps = %d, want 10", p.maxSteps)
	}
}

func TestParsePlan_ValidJSON(t *testing.T) {
	p := NewPlanner(10)

	llmOutput := `{
		"steps": [
			{"id": "1", "description": "Read the file", "tool": "read_file", "args": {"path": "main.go"}, "depends_on": []},
			{"id": "2", "description": "Edit the file", "tool": "write_file", "args": {"path": "main.go", "content": "new"}, "depends_on": ["1"]},
			{"id": "3", "description": "Run tests", "tool": "execute", "args": {"command": "go test"}, "depends_on": ["2"]}
		]
	}`

	plan, err := p.ParsePlan(llmOutput)
	if err != nil {
		t.Fatalf("ParsePlan failed: %v", err)
	}

	if len(plan.Steps) != 3 {
		t.Fatalf("got %d steps, want 3", len(plan.Steps))
	}

	if plan.Steps[0].ID != "1" {
		t.Errorf("step 0 ID = %q, want %q", plan.Steps[0].ID, "1")
	}
	if plan.Steps[0].ToolName != "read_file" {
		t.Errorf("step 0 ToolName = %q, want %q", plan.Steps[0].ToolName, "read_file")
	}
	if plan.Steps[0].Status != "pending" {
		t.Errorf("step 0 Status = %q, want %q", plan.Steps[0].Status, "pending")
	}

	if len(plan.Steps[1].DependsOn) != 1 || plan.Steps[1].DependsOn[0] != "1" {
		t.Errorf("step 1 DependsOn = %v, want [1]", plan.Steps[1].DependsOn)
	}
}

func TestParsePlan_ExtractsFromMarkdown(t *testing.T) {
	p := NewPlanner(10)

	llmOutput := `Here is my plan:
` + "```json" + `
{
	"steps": [
		{"id": "1", "description": "Do something", "tool": "read_file", "args": {}, "depends_on": []}
	]
}
` + "```" + `
That should work.`

	plan, err := p.ParsePlan(llmOutput)
	if err != nil {
		t.Fatalf("ParsePlan failed: %v", err)
	}
	if len(plan.Steps) != 1 {
		t.Fatalf("got %d steps, want 1", len(plan.Steps))
	}
}

func TestParsePlan_InvalidJSON(t *testing.T) {
	p := NewPlanner(10)
	_, err := p.ParsePlan("this is not json at all")
	if err == nil {
		t.Fatal("expected error for invalid input")
	}
}

func TestParsePlan_TooManySteps(t *testing.T) {
	p := NewPlanner(2)

	llmOutput := `{
		"steps": [
			{"id": "1", "description": "Step 1", "tool": "", "args": {}, "depends_on": []},
			{"id": "2", "description": "Step 2", "tool": "", "args": {}, "depends_on": []},
			{"id": "3", "description": "Step 3", "tool": "", "args": {}, "depends_on": []}
		]
	}`

	_, err := p.ParsePlan(llmOutput)
	if err == nil {
		t.Fatal("expected error for too many steps")
	}
}

func TestCanParallelize_IndependentSteps(t *testing.T) {
	p := NewPlanner(10)

	plan := Plan{
		Steps: []PlanStep{
			{ID: "1", Description: "Read A", DependsOn: nil},
			{ID: "2", Description: "Read B", DependsOn: nil},
			{ID: "3", Description: "Merge", DependsOn: []string{"1", "2"}},
		},
	}

	groups := p.CanParallelize(plan)
	if len(groups) != 2 {
		t.Fatalf("got %d groups, want 2", len(groups))
	}

	// First group: steps 1 and 2 (independent)
	if len(groups[0]) != 2 {
		t.Errorf("group 0 has %d steps, want 2", len(groups[0]))
	}

	// Second group: step 3 (depends on 1 and 2)
	if len(groups[1]) != 1 {
		t.Errorf("group 1 has %d steps, want 1", len(groups[1]))
	}
}

func TestCanParallelize_AllSequential(t *testing.T) {
	p := NewPlanner(10)

	plan := Plan{
		Steps: []PlanStep{
			{ID: "1", Description: "Step A", DependsOn: nil},
			{ID: "2", Description: "Step B", DependsOn: []string{"1"}},
			{ID: "3", Description: "Step C", DependsOn: []string{"2"}},
		},
	}

	groups := p.CanParallelize(plan)
	if len(groups) != 3 {
		t.Fatalf("got %d groups, want 3", len(groups))
	}
	for i, g := range groups {
		if len(g) != 1 {
			t.Errorf("group %d has %d steps, want 1", i, len(g))
		}
	}
}

func TestCanParallelize_Empty(t *testing.T) {
	p := NewPlanner(10)
	groups := p.CanParallelize(Plan{})
	if len(groups) != 0 {
		t.Errorf("got %d groups, want 0", len(groups))
	}
}

func TestFormatForPrompt(t *testing.T) {
	p := NewPlanner(10)

	plan := Plan{
		Steps: []PlanStep{
			{ID: "1", Description: "Read file", ToolName: "read_file", Status: "pending"},
			{ID: "2", Description: "Write file", ToolName: "write_file", Status: "completed", DependsOn: []string{"1"}},
		},
	}

	output := p.FormatForPrompt(plan)

	if !strings.Contains(output, "Read file") {
		t.Error("output should contain step description")
	}
	if !strings.Contains(output, "read_file") {
		t.Error("output should contain tool name")
	}
	if !strings.Contains(output, "pending") {
		t.Error("output should contain status")
	}
	if !strings.Contains(output, "completed") {
		t.Error("output should contain completed status")
	}
}

package agent

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// PlanToolSequence tests
// ---------------------------------------------------------------------------

func TestPlanToolSequence_ReadFile(t *testing.T) {
	tp := NewToolPlanner(2, 5)
	available := []string{"read_file", "write_file", "execute", "search_content"}

	plan := tp.PlanToolSequence("read the config file", available)

	if len(plan.Steps) == 0 {
		t.Fatal("expected at least one step")
	}
	if plan.Steps[0].Tool != "read_file" {
		t.Errorf("first step tool = %q, want read_file", plan.Steps[0].Tool)
	}
	if plan.Score <= 0 || plan.Score > 1 {
		t.Errorf("score = %f, want (0, 1]", plan.Score)
	}
	if plan.Reasoning == "" {
		t.Error("expected non-empty reasoning")
	}
}

func TestPlanToolSequence_SearchAndRead(t *testing.T) {
	tp := NewToolPlanner(2, 5)
	available := []string{"read_file", "write_file", "execute", "search_content"}

	plan := tp.PlanToolSequence("search for the login function", available)

	if len(plan.Steps) < 2 {
		t.Fatalf("expected at least 2 steps, got %d", len(plan.Steps))
	}
	if plan.Steps[0].Tool != "search_content" {
		t.Errorf("first step = %q, want search_content", plan.Steps[0].Tool)
	}
	if plan.Steps[1].Tool != "read_file" {
		t.Errorf("second step = %q, want read_file", plan.Steps[1].Tool)
	}
}

func TestPlanToolSequence_FixBug(t *testing.T) {
	tp := NewToolPlanner(2, 5)
	available := []string{"read_file", "write_file", "execute", "search_content"}

	plan := tp.PlanToolSequence("fix the bug in parser", available)

	if len(plan.Steps) < 3 {
		t.Fatalf("expected at least 3 steps for fix bug, got %d", len(plan.Steps))
	}
	// Should include search, read, and write
	tools := make(map[string]bool)
	for _, s := range plan.Steps {
		tools[s.Tool] = true
	}
	for _, expected := range []string{"search_content", "read_file", "write_file"} {
		if !tools[expected] {
			t.Errorf("expected %q in plan steps", expected)
		}
	}
}

func TestPlanToolSequence_RunTests(t *testing.T) {
	tp := NewToolPlanner(2, 5)
	available := []string{"read_file", "write_file", "execute"}

	plan := tp.PlanToolSequence("run the tests", available)

	if len(plan.Steps) == 0 {
		t.Fatal("expected at least one step")
	}
	if plan.Steps[0].Tool != "execute" {
		t.Errorf("first step = %q, want execute", plan.Steps[0].Tool)
	}
}

func TestPlanToolSequence_EmptyTask(t *testing.T) {
	tp := NewToolPlanner(2, 5)

	plan := tp.PlanToolSequence("", []string{"read_file"})

	if len(plan.Steps) != 0 {
		t.Errorf("expected empty plan for empty task, got %d steps", len(plan.Steps))
	}
	if plan.Score != 0 {
		t.Errorf("score = %f, want 0 for empty task", plan.Score)
	}
}

func TestPlanToolSequence_MaxStepsEnforced(t *testing.T) {
	tp := NewToolPlanner(2, 3)
	available := []string{"read_file", "write_file", "execute", "search_content", "git_status", "git_commit"}

	plan := tp.PlanToolSequence("fix bug and commit", available)

	if len(plan.Steps) > 3 {
		t.Errorf("got %d steps, want <= 3 (maxSteps)", len(plan.Steps))
	}
}

func TestPlanToolSequence_Alternatives(t *testing.T) {
	tp := NewToolPlanner(2, 5)
	available := []string{"read_file", "write_file", "execute", "search_content", "search_files"}

	plan := tp.PlanToolSequence("search for function definition", available)

	// At least one step should have alternatives
	hasAlternatives := false
	for _, s := range plan.Steps {
		if len(s.Alternatives) > 0 {
			hasAlternatives = true
		}
	}
	if !hasAlternatives {
		t.Error("expected at least one step with alternatives")
	}
}

func TestPlanToolSequence_GitCommit(t *testing.T) {
	tp := NewToolPlanner(2, 5)
	available := []string{"read_file", "write_file", "execute", "git_status", "git_commit"}

	plan := tp.PlanToolSequence("git commit the changes", available)

	if len(plan.Steps) < 2 {
		t.Fatalf("expected at least 2 steps, got %d", len(plan.Steps))
	}
	if plan.Steps[0].Tool != "git_status" {
		t.Errorf("first step = %q, want git_status", plan.Steps[0].Tool)
	}
	if plan.Steps[1].Tool != "git_commit" {
		t.Errorf("second step = %q, want git_commit", plan.Steps[1].Tool)
	}
}

// ---------------------------------------------------------------------------
// ValidateToolCall tests
// ---------------------------------------------------------------------------

func TestValidateToolCall_Valid(t *testing.T) {
	tp := NewToolPlanner(2, 5)

	errs := tp.ValidateToolCall("read_file", map[string]any{"path": "/tmp/foo.go"})

	errorCount := 0
	for _, e := range errs {
		if e.Severity == "error" {
			errorCount++
		}
	}
	if errorCount > 0 {
		t.Errorf("expected no errors, got %d: %v", errorCount, errs)
	}
}

func TestValidateToolCall_EmptyToolName(t *testing.T) {
	tp := NewToolPlanner(2, 5)

	errs := tp.ValidateToolCall("", map[string]any{"path": "/tmp"})

	hasError := false
	for _, e := range errs {
		if e.Field == "tool_name" && e.Severity == "error" {
			hasError = true
		}
	}
	if !hasError {
		t.Error("expected error for empty tool name")
	}
}

func TestValidateToolCall_NilArgs(t *testing.T) {
	tp := NewToolPlanner(2, 5)

	errs := tp.ValidateToolCall("read_file", nil)

	hasError := false
	for _, e := range errs {
		if e.Field == "args" && e.Severity == "error" {
			hasError = true
		}
	}
	if !hasError {
		t.Error("expected error for nil args")
	}
}

func TestValidateToolCall_MissingRequiredPath(t *testing.T) {
	tp := NewToolPlanner(2, 5)

	errs := tp.ValidateToolCall("read_file", map[string]any{"other": "value"})

	hasError := false
	for _, e := range errs {
		if e.Field == "path" {
			hasError = true
		}
	}
	if !hasError {
		t.Error("expected error for missing path in read_file")
	}
}

func TestValidateToolCall_EmptyStringArg(t *testing.T) {
	tp := NewToolPlanner(2, 5)

	errs := tp.ValidateToolCall("read_file", map[string]any{"path": ""})

	hasError := false
	for _, e := range errs {
		if e.Field == "path" {
			hasError = true
		}
	}
	if !hasError {
		t.Error("expected error for empty path string")
	}
}

func TestValidateToolCall_PathTraversal(t *testing.T) {
	tp := NewToolPlanner(2, 5)

	errs := tp.ValidateToolCall("read_file", map[string]any{"path": "/etc/../../../etc/passwd"})

	hasTraversal := false
	for _, e := range errs {
		if strings.Contains(strings.ToLower(e.Message), "traversal") {
			hasTraversal = true
		}
	}
	if !hasTraversal {
		t.Error("expected path traversal error")
	}
}

func TestValidateToolCall_NegativeNumericArg(t *testing.T) {
	tp := NewToolPlanner(2, 5)

	errs := tp.ValidateToolCall("some_tool", map[string]any{"count": -5.0})

	hasWarning := false
	for _, e := range errs {
		if e.Field == "count" {
			hasWarning = true
		}
	}
	if !hasWarning {
		t.Error("expected warning for negative numeric arg")
	}
}

// ---------------------------------------------------------------------------
// ScoreStep tests
// ---------------------------------------------------------------------------

func TestScoreStep_ToolExists(t *testing.T) {
	tp := NewToolPlanner(2, 5)
	step := PlannedStep{
		Tool:            "read_file",
		Args:            map[string]any{"path": "/foo"},
		ExpectedOutcome: "file contents of foo",
		Confidence:      0.9,
	}

	score := tp.ScoreStep(step, []string{"read_file", "write_file"})

	if score <= 0 {
		t.Error("expected positive score for existing tool")
	}
	if score > 1 {
		t.Errorf("score = %f, want <= 1.0", score)
	}
}

func TestScoreStep_ToolNotExists(t *testing.T) {
	tp := NewToolPlanner(2, 5)
	step := PlannedStep{
		Tool:            "nonexistent",
		Args:            map[string]any{"path": "/foo"},
		ExpectedOutcome: "something",
	}

	score := tp.ScoreStep(step, []string{"read_file", "write_file"})

	if score != 0 {
		t.Errorf("score = %f, want 0 for nonexistent tool", score)
	}
}

func TestScoreStep_NoArgs(t *testing.T) {
	tp := NewToolPlanner(2, 5)
	step := PlannedStep{
		Tool:            "read_file",
		Args:            nil,
		ExpectedOutcome: "something",
	}

	scoreNoArgs := tp.ScoreStep(step, []string{"read_file"})

	stepWithArgs := PlannedStep{
		Tool:            "read_file",
		Args:            map[string]any{"path": "/foo"},
		ExpectedOutcome: "something",
	}
	scoreWithArgs := tp.ScoreStep(stepWithArgs, []string{"read_file"})

	if scoreNoArgs >= scoreWithArgs {
		t.Errorf("score without args (%f) should be less than with args (%f)", scoreNoArgs, scoreWithArgs)
	}
}

func TestScoreStep_NoExpectedOutcome(t *testing.T) {
	tp := NewToolPlanner(2, 5)
	stepNoOutcome := PlannedStep{
		Tool: "read_file",
		Args: map[string]any{"path": "/foo"},
	}
	stepWithOutcome := PlannedStep{
		Tool:            "read_file",
		Args:            map[string]any{"path": "/foo"},
		ExpectedOutcome: "configuration values",
	}

	scoreNo := tp.ScoreStep(stepNoOutcome, []string{"read_file"})
	scoreWith := tp.ScoreStep(stepWithOutcome, []string{"read_file"})

	if scoreNo >= scoreWith {
		t.Errorf("score without outcome (%f) should be less than with outcome (%f)", scoreNo, scoreWith)
	}
}

// ---------------------------------------------------------------------------
// TaskToTools tests
// ---------------------------------------------------------------------------

func TestTaskToTools_ReadFile(t *testing.T) {
	tools := TaskToTools("read file")
	assertContains(t, tools, "read_file")
}

func TestTaskToTools_SearchFunction(t *testing.T) {
	tools := TaskToTools("search for function")
	assertContains(t, tools, "search_content")
	assertContains(t, tools, "read_file")
}

func TestTaskToTools_FixBug(t *testing.T) {
	tools := TaskToTools("fix bug")
	assertContains(t, tools, "search_content")
	assertContains(t, tools, "read_file")
	assertContains(t, tools, "write_file")
}

func TestTaskToTools_RunTests(t *testing.T) {
	tools := TaskToTools("run tests")
	assertContains(t, tools, "execute")
}

func TestTaskToTools_SecurityScan(t *testing.T) {
	tools := TaskToTools("security scan")
	assertContains(t, tools, "scan_secrets")
	assertContains(t, tools, "scan_dependencies")
}

func TestTaskToTools_GitCommit(t *testing.T) {
	tools := TaskToTools("git commit")
	assertContains(t, tools, "git_status")
	assertContains(t, tools, "git_commit")
}

func TestTaskToTools_Deploy(t *testing.T) {
	tools := TaskToTools("deploy")
	assertContains(t, tools, "execute")
	assertContains(t, tools, "git_push")
}

func TestTaskToTools_EmptyTask(t *testing.T) {
	tools := TaskToTools("")
	if len(tools) != 0 {
		t.Errorf("expected empty tools for empty task, got %v", tools)
	}
}

// ---------------------------------------------------------------------------
// NewToolPlanner tests
// ---------------------------------------------------------------------------

func TestNewToolPlanner_Defaults(t *testing.T) {
	tp := NewToolPlanner(0, 0)
	if tp.maxAlternatives != 2 {
		t.Errorf("maxAlternatives = %d, want 2 (default)", tp.maxAlternatives)
	}
	if tp.maxSteps != 5 {
		t.Errorf("maxSteps = %d, want 5 (default)", tp.maxSteps)
	}
}

func TestNewToolPlanner_Custom(t *testing.T) {
	tp := NewToolPlanner(3, 10)
	if tp.maxAlternatives != 3 {
		t.Errorf("maxAlternatives = %d, want 3", tp.maxAlternatives)
	}
	if tp.maxSteps != 10 {
		t.Errorf("maxSteps = %d, want 10", tp.maxSteps)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func assertContains(t *testing.T, tools []string, expected string) {
	t.Helper()
	for _, tool := range tools {
		if tool == expected {
			return
		}
	}
	t.Errorf("expected tools %v to contain %q", tools, expected)
}

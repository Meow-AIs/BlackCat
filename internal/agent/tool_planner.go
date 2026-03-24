package agent

import (
	"strings"
)

// PlannedStep represents a single step in a tool execution plan.
type PlannedStep struct {
	Tool            string         // tool to invoke
	Args            map[string]any // arguments to pass
	ExpectedOutcome string         // what we expect from this step
	Alternatives    []PlannedStep  // backup options if this step fails
	Confidence      float64        // 0-1 how confident this is correct
}

// ToolPlan holds a sequence of planned tool invocations.
type ToolPlan struct {
	Steps     []PlannedStep
	Score     float64 // overall plan quality 0-1
	Reasoning string  // why this plan was chosen
}

// ValidationError describes a problem with a tool call.
type ValidationError struct {
	Field    string
	Message  string
	Severity string // "error" or "warning"
}

// ToolPlanner plans and validates tool call sequences.
type ToolPlanner struct {
	maxAlternatives int
	maxSteps        int
}

// NewToolPlanner creates a ToolPlanner with the given limits.
// Zero values are replaced with defaults (2 alternatives, 5 steps).
func NewToolPlanner(maxAlternatives, maxSteps int) *ToolPlanner {
	if maxAlternatives <= 0 {
		maxAlternatives = 2
	}
	if maxSteps <= 0 {
		maxSteps = 5
	}
	return &ToolPlanner{
		maxAlternatives: maxAlternatives,
		maxSteps:        maxSteps,
	}
}

// ---------------------------------------------------------------------------
// Task-to-tool mapping
// ---------------------------------------------------------------------------

// taskPattern maps keyword patterns to tool sequences.
type taskPattern struct {
	keywords     []string            // all must match
	tools        []string            // ordered tool sequence
	alternatives map[string][]string // tool -> alternative tool names
}

// knownPatterns defines task-to-tool mappings. Patterns are checked in order
// so more specific patterns must come first.
var knownPatterns = []taskPattern{
	{
		keywords:     []string{"security", "scan"},
		tools:        []string{"scan_secrets", "scan_dependencies"},
		alternatives: nil,
	},
	{
		keywords:     []string{"git", "commit"},
		tools:        []string{"git_status", "git_commit"},
		alternatives: nil,
	},
	{
		keywords: []string{"deploy"},
		tools:    []string{"execute", "git_push"},
	},
	{
		keywords: []string{"fix", "bug"},
		tools:    []string{"search_content", "read_file", "write_file"},
		alternatives: map[string][]string{
			"search_content": {"search_files", "grep"},
		},
	},
	{
		keywords: []string{"search"},
		tools:    []string{"search_content", "read_file"},
		alternatives: map[string][]string{
			"search_content": {"search_files", "grep"},
		},
	},
	{
		keywords: []string{"find"},
		tools:    []string{"search_content", "read_file"},
		alternatives: map[string][]string{
			"search_content": {"search_files", "grep"},
		},
	},
	{
		keywords:     []string{"read", "file"},
		tools:        []string{"read_file"},
		alternatives: nil,
	},
	{
		keywords:     []string{"run", "test"},
		tools:        []string{"execute"},
		alternatives: nil,
	},
	{
		keywords:     []string{"test"},
		tools:        []string{"execute"},
		alternatives: nil,
	},
	{
		keywords:     []string{"execute"},
		tools:        []string{"execute"},
		alternatives: nil,
	},
	{
		keywords: []string{"write"},
		tools:    []string{"read_file", "write_file"},
	},
}

// TaskToTools returns the recommended tool sequence for a task description.
func TaskToTools(task string) []string {
	if strings.TrimSpace(task) == "" {
		return nil
	}

	lower := strings.ToLower(task)
	for _, p := range knownPatterns {
		if matchesAllKeywords(lower, p.keywords) {
			return p.tools
		}
	}

	return nil
}

// matchesAllKeywords returns true if the text contains all of the keywords.
func matchesAllKeywords(text string, keywords []string) bool {
	for _, kw := range keywords {
		if !strings.Contains(text, kw) {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// PlanToolSequence
// ---------------------------------------------------------------------------

// PlanToolSequence creates a ToolPlan for the given task using available tools.
func (tp *ToolPlanner) PlanToolSequence(task string, available []string) ToolPlan {
	if strings.TrimSpace(task) == "" {
		return ToolPlan{}
	}

	lower := strings.ToLower(task)
	availSet := toStringSet(available)

	// Find the best matching pattern.
	var bestPattern *taskPattern
	for i := range knownPatterns {
		if matchesAllKeywords(lower, knownPatterns[i].keywords) {
			bestPattern = &knownPatterns[i]
			break
		}
	}

	if bestPattern == nil {
		return ToolPlan{
			Score:     0,
			Reasoning: "no matching pattern found for task",
		}
	}

	var steps []PlannedStep
	for _, tool := range bestPattern.tools {
		if len(steps) >= tp.maxSteps {
			break
		}
		if !availSet[tool] {
			continue
		}

		step := PlannedStep{
			Tool:            tool,
			Args:            buildDefaultArgs(tool),
			ExpectedOutcome: expectedOutcomeFor(tool),
			Confidence:      baseConfidenceFor(tool),
		}

		// Populate alternatives as PlannedStep structs.
		if bestPattern.alternatives != nil {
			if alts, ok := bestPattern.alternatives[tool]; ok {
				for _, alt := range alts {
					if !availSet[alt] || len(step.Alternatives) >= tp.maxAlternatives {
						continue
					}
					step.Alternatives = append(step.Alternatives, PlannedStep{
						Tool:            alt,
						Args:            buildDefaultArgs(alt),
						ExpectedOutcome: expectedOutcomeFor(alt),
						Confidence:      baseConfidenceFor(alt) * 0.8,
					})
				}
			}
		}

		steps = append(steps, step)
	}

	if len(steps) == 0 {
		return ToolPlan{
			Score:     0,
			Reasoning: "no available tools match the task pattern",
		}
	}

	score := tp.scorePlan(steps, available)

	return ToolPlan{
		Steps:     steps,
		Score:     score,
		Reasoning: buildPlanReasoning(task, steps),
	}
}

// ---------------------------------------------------------------------------
// ValidateToolCall
// ---------------------------------------------------------------------------

// ValidateToolCall checks a tool invocation for common issues.
func (tp *ToolPlanner) ValidateToolCall(toolName string, args map[string]any) []ValidationError {
	var errs []ValidationError

	if toolName == "" {
		errs = append(errs, ValidationError{
			Field:    "tool_name",
			Message:  "tool name is empty",
			Severity: "error",
		})
	}

	if args == nil {
		errs = append(errs, ValidationError{
			Field:    "args",
			Message:  "arguments are nil",
			Severity: "error",
		})
		return errs
	}

	// Check required keys for known tools.
	required := requiredKeysForTool(toolName)
	for _, key := range required {
		val, ok := args[key]
		if !ok {
			errs = append(errs, ValidationError{
				Field:    key,
				Message:  "missing required argument: " + key,
				Severity: "error",
			})
			continue
		}
		if s, isStr := val.(string); isStr && s == "" {
			errs = append(errs, ValidationError{
				Field:    key,
				Message:  key + " must not be empty",
				Severity: "error",
			})
		}
	}

	// Check all args for type-specific issues.
	for key, val := range args {
		switch v := val.(type) {
		case string:
			if hasPathTraversal(v) {
				errs = append(errs, ValidationError{
					Field:    key,
					Message:  "path traversal detected in " + key,
					Severity: "error",
				})
			}
		case float64:
			if v < 0 {
				errs = append(errs, ValidationError{
					Field:    key,
					Message:  key + " should not be negative",
					Severity: "warning",
				})
			}
		case int:
			if v < 0 {
				errs = append(errs, ValidationError{
					Field:    key,
					Message:  key + " should not be negative",
					Severity: "warning",
				})
			}
		}
	}

	return errs
}

// ---------------------------------------------------------------------------
// ScoreStep
// ---------------------------------------------------------------------------

// ScoreStep evaluates how likely a planned step will succeed. Returns 0-1.
func (tp *ToolPlanner) ScoreStep(step PlannedStep, available []string) float64 {
	availSet := toStringSet(available)

	if !availSet[step.Tool] {
		return 0
	}

	score := 0.4 // base score for existing tool

	if step.Args != nil && len(step.Args) > 0 {
		score += 0.3
	}

	if step.ExpectedOutcome != "" {
		score += 0.3
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func requiredKeysForTool(toolName string) []string {
	switch toolName {
	case "read_file", "write_file", "delete_file":
		return []string{"path"}
	case "execute", "bash":
		return []string{"command"}
	case "search_content", "search_files", "grep":
		return []string{"query"}
	case "web_fetch":
		return []string{"url"}
	default:
		return nil
	}
}

func buildDefaultArgs(tool string) map[string]any {
	switch tool {
	case "read_file":
		return map[string]any{"path": ""}
	case "write_file":
		return map[string]any{"path": "", "content": ""}
	case "execute":
		return map[string]any{"command": ""}
	case "search_content", "search_files":
		return map[string]any{"query": ""}
	default:
		return map[string]any{}
	}
}

func expectedOutcomeFor(tool string) string {
	switch tool {
	case "read_file":
		return "file contents"
	case "write_file":
		return "file written successfully"
	case "execute":
		return "command output"
	case "search_content", "search_files":
		return "matching results"
	case "git_status":
		return "repository status"
	case "git_commit":
		return "commit created"
	case "git_push":
		return "changes pushed"
	case "scan_secrets":
		return "secret scan results"
	case "scan_dependencies":
		return "dependency scan results"
	default:
		return "tool output"
	}
}

func baseConfidenceFor(tool string) float64 {
	switch tool {
	case "read_file", "list_dir":
		return 0.9
	case "search_content", "search_files":
		return 0.8
	case "execute":
		return 0.7
	case "write_file":
		return 0.85
	case "git_status":
		return 0.95
	case "git_commit", "git_push":
		return 0.8
	default:
		return 0.7
	}
}

func (tp *ToolPlanner) scorePlan(steps []PlannedStep, available []string) float64 {
	if len(steps) == 0 {
		return 0
	}
	total := 0.0
	for _, step := range steps {
		total += tp.ScoreStep(step, available)
	}
	return total / float64(len(steps))
}

func hasPathTraversal(s string) bool {
	return strings.Contains(s, "/../") || strings.HasSuffix(s, "/..")
}

func buildPlanReasoning(task string, steps []PlannedStep) string {
	var b strings.Builder
	b.WriteString("Plan for: ")
	b.WriteString(task)
	b.WriteString(". Steps: ")
	for i, s := range steps {
		if i > 0 {
			b.WriteString(" -> ")
		}
		b.WriteString(s.Tool)
	}
	return b.String()
}

func toStringSet(items []string) map[string]bool {
	set := make(map[string]bool, len(items))
	for _, item := range items {
		set[item] = true
	}
	return set
}

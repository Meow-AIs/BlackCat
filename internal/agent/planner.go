package agent

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Plan represents a decomposed task with ordered steps.
type Plan struct {
	Steps    []PlanStep `json:"steps"`
	Parallel bool       `json:"parallel"`
}

// PlanStep is a single step in a plan.
type PlanStep struct {
	ID          string         `json:"id"`
	Description string         `json:"description"`
	ToolName    string         `json:"tool"`
	Args        map[string]any `json:"args"`
	DependsOn   []string       `json:"depends_on"`
	Status      string         `json:"status"`
}

// Planner parses and manages task plans from LLM output.
type Planner struct {
	maxSteps int
}

// NewPlanner creates a planner with the given step limit.
func NewPlanner(maxSteps int) *Planner {
	return &Planner{maxSteps: maxSteps}
}

// rawPlan is used for JSON unmarshalling of LLM output.
type rawPlan struct {
	Steps []rawStep `json:"steps"`
}

type rawStep struct {
	ID          string         `json:"id"`
	Description string         `json:"description"`
	Tool        string         `json:"tool"`
	Args        map[string]any `json:"args"`
	DependsOn   []string       `json:"depends_on"`
}

// ParsePlan parses a structured plan from LLM output. The output may contain
// a raw JSON object or a JSON block wrapped in markdown code fences.
func (p *Planner) ParsePlan(llmOutput string) (Plan, error) {
	jsonStr := extractJSON(llmOutput)

	var raw rawPlan
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return Plan{}, fmt.Errorf("parse plan JSON: %w", err)
	}

	if len(raw.Steps) > p.maxSteps {
		return Plan{}, fmt.Errorf("plan has %d steps, max is %d", len(raw.Steps), p.maxSteps)
	}

	steps := make([]PlanStep, len(raw.Steps))
	for i, rs := range raw.Steps {
		deps := rs.DependsOn
		if deps == nil {
			deps = []string{}
		}
		args := rs.Args
		if args == nil {
			args = map[string]any{}
		}
		steps[i] = PlanStep{
			ID:          rs.ID,
			Description: rs.Description,
			ToolName:    rs.Tool,
			Args:        args,
			DependsOn:   deps,
			Status:      "pending",
		}
	}

	plan := Plan{Steps: steps}
	groups := p.CanParallelize(plan)
	plan.Parallel = len(groups) > 0 && len(groups) < len(steps)

	return plan, nil
}

// extractJSON finds the first JSON object in the text, handling markdown fences.
func extractJSON(text string) string {
	// Try to find JSON in markdown code block
	if idx := strings.Index(text, "```json"); idx != -1 {
		start := idx + len("```json")
		end := strings.Index(text[start:], "```")
		if end != -1 {
			return strings.TrimSpace(text[start : start+end])
		}
	}
	if idx := strings.Index(text, "```"); idx != -1 {
		start := idx + len("```")
		end := strings.Index(text[start:], "```")
		if end != -1 {
			candidate := strings.TrimSpace(text[start : start+end])
			if strings.HasPrefix(candidate, "{") {
				return candidate
			}
		}
	}

	// Try the raw text as JSON
	trimmed := strings.TrimSpace(text)
	if strings.HasPrefix(trimmed, "{") {
		return trimmed
	}

	return text
}

// CanParallelize groups independent steps into execution batches.
// Steps in the same group can run in parallel; groups run sequentially.
func (p *Planner) CanParallelize(plan Plan) [][]PlanStep {
	if len(plan.Steps) == 0 {
		return nil
	}

	// Build a set of completed step IDs per round
	completed := make(map[string]bool)
	remaining := make([]PlanStep, len(plan.Steps))
	copy(remaining, plan.Steps)

	var groups [][]PlanStep

	for len(remaining) > 0 {
		var ready []PlanStep
		var notReady []PlanStep

		for _, step := range remaining {
			if allDepsCompleted(step.DependsOn, completed) {
				ready = append(ready, step)
			} else {
				notReady = append(notReady, step)
			}
		}

		if len(ready) == 0 {
			// Circular dependency or invalid graph — add all remaining
			groups = append(groups, notReady)
			break
		}

		groups = append(groups, ready)
		for _, s := range ready {
			completed[s.ID] = true
		}
		remaining = notReady
	}

	return groups
}

func allDepsCompleted(deps []string, completed map[string]bool) bool {
	for _, d := range deps {
		if !completed[d] {
			return false
		}
	}
	return true
}

// FormatForPrompt renders a plan as a human-readable string for inclusion
// in a system prompt.
func (p *Planner) FormatForPrompt(plan Plan) string {
	if len(plan.Steps) == 0 {
		return "No steps in plan."
	}

	var b strings.Builder
	b.WriteString("## Plan\n\n")

	for i, step := range plan.Steps {
		b.WriteString(fmt.Sprintf("%d. [%s] %s", i+1, step.Status, step.Description))
		if step.ToolName != "" {
			b.WriteString(fmt.Sprintf(" (tool: %s)", step.ToolName))
		}
		if len(step.DependsOn) > 0 {
			b.WriteString(fmt.Sprintf(" [depends: %s]", strings.Join(step.DependsOn, ", ")))
		}
		b.WriteString("\n")
	}

	return b.String()
}

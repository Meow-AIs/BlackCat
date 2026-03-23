package agent

import (
	"fmt"
	"strconv"
	"strings"
)

// DefaultConfidenceThreshold is the minimum confidence for a critique to pass.
const DefaultConfidenceThreshold = 0.7

// DefaultMaxRetries is the default maximum number of self-critique retries.
const DefaultMaxRetries = 2

// CritiqueResult holds the outcome of a self-critique evaluation.
type CritiqueResult struct {
	Passed     bool     // true if confidence >= threshold
	Confidence float64  // 0.0-1.0
	Issues     []string // list of identified issues
	Suggestion string   // suggested improvement
}

// ParseCritiqueOutput parses confidence, issues, and suggestion tags from
// LLM self-critique output. Issues are split on semicolons. Confidence
// below DefaultConfidenceThreshold sets Passed to false.
func ParseCritiqueOutput(text string) CritiqueResult {
	confidence := 0.0
	if raw := extractTag(text, "confidence"); raw != "" {
		if parsed, err := strconv.ParseFloat(raw, 64); err == nil {
			confidence = parsed
		}
	}

	var issues []string
	if raw := extractTag(text, "issues"); raw != "" {
		issues = splitAndTrim(raw)
	}

	suggestion := extractTag(text, "suggestion")

	return CritiqueResult{
		Passed:     confidence >= DefaultConfidenceThreshold,
		Confidence: confidence,
		Issues:     issues,
		Suggestion: suggestion,
	}
}

// splitAndTrim splits a string on semicolons and trims whitespace from
// each element. Empty elements are excluded.
func splitAndTrim(s string) []string {
	parts := strings.Split(s, ";")
	var result []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// ShouldRetry returns true if the critique confidence is below the
// threshold and there are retries remaining.
func ShouldRetry(result CritiqueResult, maxRetries, currentRetry int) bool {
	return result.Confidence < DefaultConfidenceThreshold &&
		currentRetry < maxRetries
}

// BuildCritiquePrompt generates a self-critique prompt asking the LLM to
// evaluate an observation against the original task.
func BuildCritiquePrompt(observation, originalTask string) string {
	var b strings.Builder

	b.WriteString("Evaluate the following result against the original task.\n\n")
	fmt.Fprintf(&b, "Original task: %s\n\n", originalTask)
	fmt.Fprintf(&b, "Result/Observation:\n%s\n\n", observation)
	b.WriteString("Assess whether the result adequately addresses the task. ")
	b.WriteString("Identify any issues or gaps.\n\n")
	b.WriteString("Respond with:\n")
	b.WriteString("<confidence>0.0-1.0 (how well the result addresses the task)</confidence>\n")
	b.WriteString("<issues>issue1; issue2; ...</issues>\n")
	b.WriteString("<suggestion>what to do next to improve</suggestion>\n")

	return b.String()
}

// BuildStepBackPrompt generates a "step back" prompt that asks the LLM
// to identify the higher-level goal behind the current task.
func BuildStepBackPrompt(originalTask string) string {
	var b strings.Builder

	b.WriteString("Take a step back and consider the bigger picture.\n\n")
	fmt.Fprintf(&b, "Current task: %s\n\n", originalTask)
	b.WriteString("What is the higher-level goal behind this task? ")
	b.WriteString("What broader objective are we trying to achieve?\n\n")
	b.WriteString("Consider:\n")
	b.WriteString("1. Why was this task requested in the first place?\n")
	b.WriteString("2. What would success look like from the user's perspective?\n")
	b.WriteString("3. Are there alternative approaches that better serve the higher-level goal?\n\n")
	b.WriteString("Respond with your analysis of the higher-level goal and any revised approach.\n")

	return b.String()
}

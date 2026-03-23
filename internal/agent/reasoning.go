package agent

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ReasoningStep represents a single step in the ReAct reasoning chain.
type ReasoningStep struct {
	Thought     string  // What the agent is thinking
	Action      string  // What tool to use
	Observation string  // Tool result
	Critique    string  // Self-evaluation
	Confidence  float64 // 0.0-1.0
}

// ReasoningChain holds an immutable sequence of reasoning steps.
type ReasoningChain struct {
	steps []ReasoningStep
}

// reasoningTags are the XML tags used in the ReAct scratchpad pattern.
var reasoningTags = []string{
	"thinking",
	"action",
	"observation",
	"critique",
	"confidence",
}

// tagPattern builds a regexp that matches <tag>content</tag> including newlines.
func tagPattern(tag string) *regexp.Regexp {
	return regexp.MustCompile(`(?s)<` + tag + `>(.*?)</` + tag + `>`)
}

// extractTag extracts the content between the given XML-style tags.
// Returns empty string if the tag pair is not found or malformed.
func extractTag(text, tag string) string {
	re := tagPattern(tag)
	matches := re.FindStringSubmatch(text)
	if len(matches) < 2 {
		return ""
	}
	return strings.TrimSpace(matches[1])
}

// ParseReasoningOutput parses structured reasoning tags from LLM output
// and returns a ReasoningStep. Unclosed or malformed tags yield empty fields.
func ParseReasoningOutput(text string) ReasoningStep {
	confidence := 0.0
	if raw := extractTag(text, "confidence"); raw != "" {
		if parsed, err := strconv.ParseFloat(raw, 64); err == nil {
			confidence = parsed
		}
	}

	return ReasoningStep{
		Thought:     extractTag(text, "thinking"),
		Action:      extractTag(text, "action"),
		Observation: extractTag(text, "observation"),
		Critique:    extractTag(text, "critique"),
		Confidence:  confidence,
	}
}

// IsComplete returns true if all required fields of a ReasoningStep are populated.
// Required: Thought, Action, Observation, Critique, and Confidence > 0.
func IsComplete(step ReasoningStep) bool {
	return step.Thought != "" &&
		step.Action != "" &&
		step.Observation != "" &&
		step.Critique != "" &&
		step.Confidence > 0.0
}

// DetectIncompleteScratchpad returns true if any reasoning tag is opened
// but not closed in the given text (Hermes pattern detection).
func DetectIncompleteScratchpad(text string) bool {
	for _, tag := range reasoningTags {
		openTag := "<" + tag + ">"
		closeTag := "</" + tag + ">"

		openCount := strings.Count(text, openTag)
		closeCount := strings.Count(text, closeTag)

		if openCount > closeCount {
			return true
		}
	}
	return false
}

// BuildReActPrompt generates the follow-up prompt for the next ReAct iteration
// based on the previous reasoning step.
func BuildReActPrompt(step ReasoningStep) string {
	var b strings.Builder

	b.WriteString("Based on the previous reasoning step:\n\n")

	if step.Thought != "" {
		fmt.Fprintf(&b, "Thought: %s\n", step.Thought)
	}
	if step.Action != "" {
		fmt.Fprintf(&b, "Action: %s\n", step.Action)
	}
	if step.Observation != "" {
		fmt.Fprintf(&b, "Observation: %s\n", step.Observation)
	}
	if step.Critique != "" {
		fmt.Fprintf(&b, "Critique: %s\n", step.Critique)
	}

	b.WriteString("\nContinue reasoning. Respond with:\n")
	b.WriteString("<thinking>your next thought</thinking>\n")
	b.WriteString("<action>next tool or action</action>\n")
	b.WriteString("<observation>result of the action</observation>\n")
	b.WriteString("<critique>self-evaluation of progress</critique>\n")
	b.WriteString("<confidence>0.0-1.0</confidence>\n")

	return b.String()
}

// NewReasoningChain creates an empty, immutable reasoning chain.
func NewReasoningChain() ReasoningChain {
	return ReasoningChain{steps: nil}
}

// AddStep returns a new ReasoningChain with the given step appended.
// The original chain is not modified (immutable).
func (c ReasoningChain) AddStep(step ReasoningStep) ReasoningChain {
	newSteps := make([]ReasoningStep, len(c.steps), len(c.steps)+1)
	copy(newSteps, c.steps)
	newSteps = append(newSteps, step)
	return ReasoningChain{steps: newSteps}
}

// Steps returns a defensive copy of all steps in the chain.
func (c ReasoningChain) Steps() []ReasoningStep {
	if len(c.steps) == 0 {
		return nil
	}
	out := make([]ReasoningStep, len(c.steps))
	copy(out, c.steps)
	return out
}

// LastStep returns the most recent step and true, or a zero value and false
// if the chain is empty.
func (c ReasoningChain) LastStep() (ReasoningStep, bool) {
	if len(c.steps) == 0 {
		return ReasoningStep{}, false
	}
	return c.steps[len(c.steps)-1], true
}

// AverageConfidence returns the mean confidence across all steps,
// or 0.0 if the chain is empty.
func (c ReasoningChain) AverageConfidence() float64 {
	if len(c.steps) == 0 {
		return 0.0
	}
	var sum float64
	for _, s := range c.steps {
		sum += s.Confidence
	}
	return sum / float64(len(c.steps))
}

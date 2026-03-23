package agent

import (
	"testing"
)

// --- ParseReasoningOutput Tests ---

func TestParseReasoningOutput_FullInput(t *testing.T) {
	input := `<thinking>I need to search for files</thinking>
<action>grep -r "pattern" .</action>
<observation>Found 3 matches</observation>
<critique>The search was too broad</critique>
<confidence>0.85</confidence>`

	step := ParseReasoningOutput(input)

	if step.Thought != "I need to search for files" {
		t.Errorf("Thought = %q, want %q", step.Thought, "I need to search for files")
	}
	if step.Action != `grep -r "pattern" .` {
		t.Errorf("Action = %q, want %q", step.Action, `grep -r "pattern" .`)
	}
	if step.Observation != "Found 3 matches" {
		t.Errorf("Observation = %q, want %q", step.Observation, "Found 3 matches")
	}
	if step.Critique != "The search was too broad" {
		t.Errorf("Critique = %q, want %q", step.Critique, "The search was too broad")
	}
	if step.Confidence != 0.85 {
		t.Errorf("Confidence = %f, want %f", step.Confidence, 0.85)
	}
}

func TestParseReasoningOutput_PartialInput(t *testing.T) {
	input := `<thinking>Just thinking</thinking>
<action>do something</action>`

	step := ParseReasoningOutput(input)

	if step.Thought != "Just thinking" {
		t.Errorf("Thought = %q, want %q", step.Thought, "Just thinking")
	}
	if step.Action != "do something" {
		t.Errorf("Action = %q, want %q", step.Action, "do something")
	}
	if step.Observation != "" {
		t.Errorf("Observation = %q, want empty", step.Observation)
	}
	if step.Critique != "" {
		t.Errorf("Critique = %q, want empty", step.Critique)
	}
	if step.Confidence != 0.0 {
		t.Errorf("Confidence = %f, want 0.0", step.Confidence)
	}
}

func TestParseReasoningOutput_EmptyInput(t *testing.T) {
	step := ParseReasoningOutput("")

	if step.Thought != "" || step.Action != "" || step.Observation != "" {
		t.Errorf("Expected all empty fields for empty input, got %+v", step)
	}
	if step.Confidence != 0.0 {
		t.Errorf("Confidence = %f, want 0.0", step.Confidence)
	}
}

func TestParseReasoningOutput_MalformedTags(t *testing.T) {
	input := `<thinking>unclosed tag
<action>valid action</action>
<confidence>not_a_number</confidence>`

	step := ParseReasoningOutput(input)

	// Unclosed thinking tag should not parse
	if step.Thought != "" {
		t.Errorf("Thought = %q, want empty for unclosed tag", step.Thought)
	}
	if step.Action != "valid action" {
		t.Errorf("Action = %q, want %q", step.Action, "valid action")
	}
	// Invalid confidence should default to 0
	if step.Confidence != 0.0 {
		t.Errorf("Confidence = %f, want 0.0 for non-numeric", step.Confidence)
	}
}

func TestParseReasoningOutput_MultilineContent(t *testing.T) {
	input := `<thinking>Line one
Line two
Line three</thinking>
<action>multi
line action</action>`

	step := ParseReasoningOutput(input)

	if step.Thought != "Line one\nLine two\nLine three" {
		t.Errorf("Thought = %q, want multiline content", step.Thought)
	}
	if step.Action != "multi\nline action" {
		t.Errorf("Action = %q, want multiline content", step.Action)
	}
}

func TestParseReasoningOutput_WhitespaceHandling(t *testing.T) {
	input := `<thinking>  spaced thought  </thinking>
<confidence>  0.9  </confidence>`

	step := ParseReasoningOutput(input)

	if step.Thought != "spaced thought" {
		t.Errorf("Thought = %q, want trimmed", step.Thought)
	}
	if step.Confidence != 0.9 {
		t.Errorf("Confidence = %f, want 0.9", step.Confidence)
	}
}

// --- IsComplete Tests ---

func TestReasoningStepIsComplete_AllFields(t *testing.T) {
	step := ReasoningStep{
		Thought:     "think",
		Action:      "act",
		Observation: "obs",
		Critique:    "crit",
		Confidence:  0.8,
	}
	if !IsComplete(step) {
		t.Error("Expected complete step to return true")
	}
}

func TestReasoningStepIsComplete_MissingThought(t *testing.T) {
	step := ReasoningStep{
		Action:      "act",
		Observation: "obs",
		Critique:    "crit",
		Confidence:  0.8,
	}
	if IsComplete(step) {
		t.Error("Expected incomplete step (missing Thought) to return false")
	}
}

func TestReasoningStepIsComplete_MissingAction(t *testing.T) {
	step := ReasoningStep{
		Thought:     "think",
		Observation: "obs",
		Critique:    "crit",
		Confidence:  0.8,
	}
	if IsComplete(step) {
		t.Error("Expected incomplete step (missing Action) to return false")
	}
}

func TestReasoningStepIsComplete_ZeroConfidence(t *testing.T) {
	step := ReasoningStep{
		Thought:     "think",
		Action:      "act",
		Observation: "obs",
		Critique:    "crit",
		Confidence:  0.0,
	}
	if IsComplete(step) {
		t.Error("Expected incomplete step (zero Confidence) to return false")
	}
}

// --- DetectIncompleteScratchpad Tests ---

func TestDetectIncompleteScratchpad_UnclosedThinking(t *testing.T) {
	input := `<thinking>I am still reasoning about this`
	if !DetectIncompleteScratchpad(input) {
		t.Error("Expected true for unclosed <thinking> tag")
	}
}

func TestDetectIncompleteScratchpad_UnclosedAction(t *testing.T) {
	input := `<thinking>done</thinking><action>still going`
	if !DetectIncompleteScratchpad(input) {
		t.Error("Expected true for unclosed <action> tag")
	}
}

func TestDetectIncompleteScratchpad_AllClosed(t *testing.T) {
	input := `<thinking>done</thinking><action>done</action>`
	if DetectIncompleteScratchpad(input) {
		t.Error("Expected false when all tags are closed")
	}
}

func TestDetectIncompleteScratchpad_EmptyInput(t *testing.T) {
	if DetectIncompleteScratchpad("") {
		t.Error("Expected false for empty input")
	}
}

func TestDetectIncompleteScratchpad_NoTags(t *testing.T) {
	if DetectIncompleteScratchpad("just plain text") {
		t.Error("Expected false for text without tags")
	}
}

func TestDetectIncompleteScratchpad_UnclosedObservation(t *testing.T) {
	input := `<observation>partial result`
	if !DetectIncompleteScratchpad(input) {
		t.Error("Expected true for unclosed <observation> tag")
	}
}

func TestDetectIncompleteScratchpad_UnclosedCritique(t *testing.T) {
	input := `<critique>hmm this is`
	if !DetectIncompleteScratchpad(input) {
		t.Error("Expected true for unclosed <critique> tag")
	}
}

func TestDetectIncompleteScratchpad_UnclosedConfidence(t *testing.T) {
	input := `<confidence>0.`
	if !DetectIncompleteScratchpad(input) {
		t.Error("Expected true for unclosed <confidence> tag")
	}
}

// --- BuildReActPrompt Tests ---

func TestBuildReActPrompt_WithObservation(t *testing.T) {
	step := ReasoningStep{
		Thought:     "I searched for files",
		Action:      "grep pattern",
		Observation: "Found matches in foo.go",
	}
	prompt := BuildReActPrompt(step)

	if prompt == "" {
		t.Fatal("Expected non-empty prompt")
	}
	// Should reference the observation
	if !containsSubstring(prompt, "Found matches in foo.go") {
		t.Error("Prompt should contain the observation")
	}
	// Should ask for next step
	if !containsSubstring(prompt, "<thinking>") {
		t.Error("Prompt should ask for next thinking step")
	}
}

func TestBuildReActPrompt_EmptyStep(t *testing.T) {
	step := ReasoningStep{}
	prompt := BuildReActPrompt(step)
	if prompt == "" {
		t.Fatal("Expected non-empty prompt even for empty step")
	}
}

// --- ReasoningChain Tests ---

func TestChainAddStep(t *testing.T) {
	chain := NewReasoningChain()
	step := ReasoningStep{
		Thought:    "first thought",
		Action:     "first action",
		Confidence: 0.8,
	}

	updated := chain.AddStep(step)

	if len(updated.Steps()) != 1 {
		t.Errorf("Steps count = %d, want 1", len(updated.Steps()))
	}
	// Original chain should be unchanged (immutability)
	if len(chain.Steps()) != 0 {
		t.Errorf("Original chain modified: Steps count = %d, want 0", len(chain.Steps()))
	}
}

func TestChainSteps_ReturnsDefensiveCopy(t *testing.T) {
	chain := NewReasoningChain()
	step := ReasoningStep{Thought: "t", Action: "a", Confidence: 0.5}
	chain = chain.AddStep(step)

	steps := chain.Steps()
	steps[0].Thought = "mutated"

	// Internal state should not be affected
	if chain.Steps()[0].Thought != "t" {
		t.Error("Steps() returned a reference, not a copy — mutation leaked")
	}
}

func TestChainLastStep_Empty(t *testing.T) {
	chain := NewReasoningChain()
	_, ok := chain.LastStep()
	if ok {
		t.Error("Expected ok=false for empty chain")
	}
}

func TestChainLastStep_NonEmpty(t *testing.T) {
	chain := NewReasoningChain()
	chain = chain.AddStep(ReasoningStep{Thought: "first", Confidence: 0.5})
	chain = chain.AddStep(ReasoningStep{Thought: "second", Confidence: 0.9})

	last, ok := chain.LastStep()
	if !ok {
		t.Fatal("Expected ok=true for non-empty chain")
	}
	if last.Thought != "second" {
		t.Errorf("LastStep().Thought = %q, want %q", last.Thought, "second")
	}
}

func TestChainAverageConfidence_Empty(t *testing.T) {
	chain := NewReasoningChain()
	avg := chain.AverageConfidence()
	if avg != 0.0 {
		t.Errorf("AverageConfidence = %f, want 0.0 for empty chain", avg)
	}
}

func TestChainAverageConfidence_SingleStep(t *testing.T) {
	chain := NewReasoningChain()
	chain = chain.AddStep(ReasoningStep{Confidence: 0.8})
	avg := chain.AverageConfidence()
	if avg != 0.8 {
		t.Errorf("AverageConfidence = %f, want 0.8", avg)
	}
}

func TestChainAverageConfidence_MultipleSteps(t *testing.T) {
	chain := NewReasoningChain()
	chain = chain.AddStep(ReasoningStep{Confidence: 0.6})
	chain = chain.AddStep(ReasoningStep{Confidence: 0.8})
	chain = chain.AddStep(ReasoningStep{Confidence: 1.0})

	avg := chain.AverageConfidence()
	expected := 0.8
	if avg < expected-0.001 || avg > expected+0.001 {
		t.Errorf("AverageConfidence = %f, want ~%f", avg, expected)
	}
}

// helper
func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || findSubstring(s, sub))
}

func findSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

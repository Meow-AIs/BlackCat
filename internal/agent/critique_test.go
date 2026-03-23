package agent

import (
	"testing"
)

// --- ParseCritiqueOutput Tests ---

func TestParseCritiqueOutput_FullInput(t *testing.T) {
	input := `<confidence>0.85</confidence>
<issues>Missing error handling; No input validation</issues>
<suggestion>Add error handling for the file read operation</suggestion>`

	result := ParseCritiqueOutput(input)

	if !result.Passed {
		t.Error("Expected Passed=true for confidence >= 0.7")
	}
	if result.Confidence != 0.85 {
		t.Errorf("Confidence = %f, want 0.85", result.Confidence)
	}
	if len(result.Issues) != 2 {
		t.Errorf("Issues count = %d, want 2", len(result.Issues))
	}
	if result.Issues[0] != "Missing error handling" {
		t.Errorf("Issues[0] = %q, want %q", result.Issues[0], "Missing error handling")
	}
	if result.Issues[1] != "No input validation" {
		t.Errorf("Issues[1] = %q, want %q", result.Issues[1], "No input validation")
	}
	if result.Suggestion != "Add error handling for the file read operation" {
		t.Errorf("Suggestion = %q, want %q", result.Suggestion, "Add error handling for the file read operation")
	}
}

func TestParseCritiqueOutput_LowConfidence(t *testing.T) {
	input := `<confidence>0.3</confidence>
<issues>Completely wrong approach</issues>
<suggestion>Start over with a different strategy</suggestion>`

	result := ParseCritiqueOutput(input)

	if result.Passed {
		t.Error("Expected Passed=false for confidence < 0.7")
	}
	if result.Confidence != 0.3 {
		t.Errorf("Confidence = %f, want 0.3", result.Confidence)
	}
}

func TestParseCritiqueOutput_ExactThreshold(t *testing.T) {
	input := `<confidence>0.7</confidence>`

	result := ParseCritiqueOutput(input)

	if !result.Passed {
		t.Error("Expected Passed=true for confidence == 0.7 (threshold)")
	}
}

func TestParseCritiqueOutput_EmptyInput(t *testing.T) {
	result := ParseCritiqueOutput("")

	if result.Passed {
		t.Error("Expected Passed=false for empty input (zero confidence)")
	}
	if result.Confidence != 0.0 {
		t.Errorf("Confidence = %f, want 0.0", result.Confidence)
	}
	if len(result.Issues) != 0 {
		t.Errorf("Issues count = %d, want 0", len(result.Issues))
	}
	if result.Suggestion != "" {
		t.Errorf("Suggestion = %q, want empty", result.Suggestion)
	}
}

func TestParseCritiqueOutput_MalformedConfidence(t *testing.T) {
	input := `<confidence>abc</confidence>
<issues>Something</issues>`

	result := ParseCritiqueOutput(input)

	if result.Confidence != 0.0 {
		t.Errorf("Confidence = %f, want 0.0 for malformed input", result.Confidence)
	}
	if result.Passed {
		t.Error("Expected Passed=false for malformed confidence")
	}
}

func TestParseCritiqueOutput_NoIssues(t *testing.T) {
	input := `<confidence>0.95</confidence>
<suggestion>Looks good</suggestion>`

	result := ParseCritiqueOutput(input)

	if !result.Passed {
		t.Error("Expected Passed=true")
	}
	if len(result.Issues) != 0 {
		t.Errorf("Issues count = %d, want 0", len(result.Issues))
	}
}

func TestParseCritiqueOutput_SingleIssue(t *testing.T) {
	input := `<confidence>0.6</confidence>
<issues>One problem</issues>`

	result := ParseCritiqueOutput(input)

	if len(result.Issues) != 1 {
		t.Errorf("Issues count = %d, want 1", len(result.Issues))
	}
	if result.Issues[0] != "One problem" {
		t.Errorf("Issues[0] = %q, want %q", result.Issues[0], "One problem")
	}
}

func TestParseCritiqueOutput_WhitespaceInIssues(t *testing.T) {
	input := `<issues>  first issue  ;  second issue  </issues>`

	result := ParseCritiqueOutput(input)

	if len(result.Issues) != 2 {
		t.Fatalf("Issues count = %d, want 2", len(result.Issues))
	}
	if result.Issues[0] != "first issue" {
		t.Errorf("Issues[0] = %q, want trimmed", result.Issues[0])
	}
	if result.Issues[1] != "second issue" {
		t.Errorf("Issues[1] = %q, want trimmed", result.Issues[1])
	}
}

// --- ShouldRetry Tests ---

func TestShouldRetry_LowConfidenceWithRetriesLeft(t *testing.T) {
	result := CritiqueResult{Passed: false, Confidence: 0.5}
	if !ShouldRetry(result, 2, 0) {
		t.Error("Expected true: low confidence and retries remaining")
	}
}

func TestShouldRetry_LowConfidenceNoRetriesLeft(t *testing.T) {
	result := CritiqueResult{Passed: false, Confidence: 0.5}
	if ShouldRetry(result, 2, 2) {
		t.Error("Expected false: no retries remaining")
	}
}

func TestShouldRetry_HighConfidence(t *testing.T) {
	result := CritiqueResult{Passed: true, Confidence: 0.9}
	if ShouldRetry(result, 2, 0) {
		t.Error("Expected false: confidence >= 0.7")
	}
}

func TestShouldRetry_ExactThresholdConfidence(t *testing.T) {
	result := CritiqueResult{Passed: true, Confidence: 0.7}
	if ShouldRetry(result, 2, 0) {
		t.Error("Expected false: confidence == 0.7 meets threshold")
	}
}

func TestShouldRetry_JustBelowThreshold(t *testing.T) {
	result := CritiqueResult{Passed: false, Confidence: 0.69}
	if !ShouldRetry(result, 2, 1) {
		t.Error("Expected true: confidence < 0.7 and retries remaining")
	}
}

func TestShouldRetry_ZeroMaxRetries(t *testing.T) {
	result := CritiqueResult{Passed: false, Confidence: 0.1}
	if ShouldRetry(result, 0, 0) {
		t.Error("Expected false: maxRetries is 0")
	}
}

// --- BuildCritiquePrompt Tests ---

func TestBuildCritiquePrompt_ContainsObservation(t *testing.T) {
	prompt := BuildCritiquePrompt("file has 3 errors", "fix all lint errors")

	if prompt == "" {
		t.Fatal("Expected non-empty prompt")
	}
	if !containsSubstring(prompt, "file has 3 errors") {
		t.Error("Prompt should contain the observation")
	}
	if !containsSubstring(prompt, "fix all lint errors") {
		t.Error("Prompt should contain the original task")
	}
}

func TestBuildCritiquePrompt_ContainsResponseFormat(t *testing.T) {
	prompt := BuildCritiquePrompt("some result", "some task")

	if !containsSubstring(prompt, "<confidence>") {
		t.Error("Prompt should specify expected confidence tag format")
	}
	if !containsSubstring(prompt, "<issues>") {
		t.Error("Prompt should specify expected issues tag format")
	}
}

func TestBuildCritiquePrompt_EmptyInputs(t *testing.T) {
	prompt := BuildCritiquePrompt("", "")
	if prompt == "" {
		t.Fatal("Expected non-empty prompt even for empty inputs")
	}
}

// --- BuildStepBackPrompt Tests ---

func TestBuildStepBackPrompt_ContainsTask(t *testing.T) {
	prompt := BuildStepBackPrompt("fix the login bug")

	if prompt == "" {
		t.Fatal("Expected non-empty prompt")
	}
	if !containsSubstring(prompt, "fix the login bug") {
		t.Error("Prompt should contain the original task")
	}
}

func TestBuildStepBackPrompt_AsksHigherLevel(t *testing.T) {
	prompt := BuildStepBackPrompt("some task")

	if !containsSubstring(prompt, "higher-level") || !containsSubstring(prompt, "goal") {
		t.Error("Prompt should ask about higher-level goal")
	}
}

func TestBuildStepBackPrompt_EmptyTask(t *testing.T) {
	prompt := BuildStepBackPrompt("")
	if prompt == "" {
		t.Fatal("Expected non-empty prompt even for empty task")
	}
}

package tools

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// LevenshteinDistance tests
// ---------------------------------------------------------------------------

func TestLevenshteinDistanceIdentical(t *testing.T) {
	if d := LevenshteinDistance("abc", "abc"); d != 0 {
		t.Errorf("expected 0, got %d", d)
	}
}

func TestLevenshteinDistanceEmptyLeft(t *testing.T) {
	if d := LevenshteinDistance("", "abc"); d != 3 {
		t.Errorf("expected 3, got %d", d)
	}
}

func TestLevenshteinDistanceEmptyRight(t *testing.T) {
	if d := LevenshteinDistance("abc", ""); d != 3 {
		t.Errorf("expected 3, got %d", d)
	}
}

func TestLevenshteinDistanceBothEmpty(t *testing.T) {
	if d := LevenshteinDistance("", ""); d != 0 {
		t.Errorf("expected 0, got %d", d)
	}
}

func TestLevenshteinDistanceKittenSitting(t *testing.T) {
	// Classic example: kitten -> sitting = 3
	if d := LevenshteinDistance("kitten", "sitting"); d != 3 {
		t.Errorf("expected 3, got %d", d)
	}
}

func TestLevenshteinDistanceSingleInsertion(t *testing.T) {
	if d := LevenshteinDistance("cat", "cats"); d != 1 {
		t.Errorf("expected 1, got %d", d)
	}
}

func TestLevenshteinDistanceSingleDeletion(t *testing.T) {
	if d := LevenshteinDistance("cats", "cat"); d != 1 {
		t.Errorf("expected 1, got %d", d)
	}
}

func TestLevenshteinDistanceSingleSubstitution(t *testing.T) {
	if d := LevenshteinDistance("cat", "bat"); d != 1 {
		t.Errorf("expected 1, got %d", d)
	}
}

func TestLevenshteinDistanceCompletelyDifferent(t *testing.T) {
	// "abc" -> "xyz" requires 3 substitutions
	if d := LevenshteinDistance("abc", "xyz"); d != 3 {
		t.Errorf("expected 3, got %d", d)
	}
}

// ---------------------------------------------------------------------------
// FuzzyScore tests
// ---------------------------------------------------------------------------

func TestFuzzyScoreIdentical(t *testing.T) {
	score := FuzzyScore("read_file", "read_file")
	if score != 1.0 {
		t.Errorf("expected 1.0, got %f", score)
	}
}

func TestFuzzyScoreBothEmpty(t *testing.T) {
	score := FuzzyScore("", "")
	if score != 1.0 {
		t.Errorf("expected 1.0 for both empty, got %f", score)
	}
}

func TestFuzzyScoreOneEmpty(t *testing.T) {
	score := FuzzyScore("", "abc")
	if score != 0.0 {
		t.Errorf("expected 0.0, got %f", score)
	}
}

func TestFuzzyScoreNearMatch(t *testing.T) {
	// "read_fil" vs "read_file" — one deletion, length 9 => score = 1 - 1/9 ≈ 0.888
	score := FuzzyScore("read_fil", "read_file")
	if score <= 0.7 {
		t.Errorf("expected score > 0.7, got %f", score)
	}
}

func TestFuzzyScoreCompletelyDifferent(t *testing.T) {
	score := FuzzyScore("xyzabc", "read_file")
	// max_len = 9, distance will be large -> score should be low
	if score >= 0.5 {
		t.Errorf("expected score < 0.5, got %f", score)
	}
}

func TestFuzzyScoreSymmetric(t *testing.T) {
	a := FuzzyScore("read_file", "read_fil")
	b := FuzzyScore("read_fil", "read_file")
	if math.Abs(a-b) > 1e-9 {
		t.Errorf("expected symmetric scores, got %f vs %f", a, b)
	}
}

// ---------------------------------------------------------------------------
// RepairToolName tests
// ---------------------------------------------------------------------------

func TestRepairExactMatch(t *testing.T) {
	available := []string{"read_file", "write_file", "shell_command"}
	result := RepairToolName("read_file", available)

	if result.Strategy != "exact" {
		t.Errorf("expected strategy 'exact', got %q", result.Strategy)
	}
	if result.Repaired != "read_file" {
		t.Errorf("expected repaired 'read_file', got %q", result.Repaired)
	}
	if result.Confidence != 1.0 {
		t.Errorf("expected confidence 1.0, got %f", result.Confidence)
	}
	if result.Original != "read_file" {
		t.Errorf("expected original 'read_file', got %q", result.Original)
	}
}

func TestRepairLowercaseMatch(t *testing.T) {
	available := []string{"readfile", "writefile"}
	result := RepairToolName("ReadFile", available)

	if result.Strategy != "lowercase" {
		t.Errorf("expected strategy 'lowercase', got %q", result.Strategy)
	}
	if result.Repaired != "readfile" {
		t.Errorf("expected repaired 'readfile', got %q", result.Repaired)
	}
	if result.Confidence != 1.0 {
		t.Errorf("expected confidence 1.0, got %f", result.Confidence)
	}
}

func TestRepairNormalizeHyphen(t *testing.T) {
	available := []string{"read_file", "write_file"}
	result := RepairToolName("read-file", available)

	if result.Strategy != "normalize" {
		t.Errorf("expected strategy 'normalize', got %q", result.Strategy)
	}
	if result.Repaired != "read_file" {
		t.Errorf("expected repaired 'read_file', got %q", result.Repaired)
	}
}

func TestRepairNormalizeSpace(t *testing.T) {
	available := []string{"shell_command"}
	result := RepairToolName("shell command", available)

	if result.Strategy != "normalize" {
		t.Errorf("expected strategy 'normalize', got %q", result.Strategy)
	}
	if result.Repaired != "shell_command" {
		t.Errorf("expected repaired 'shell_command', got %q", result.Repaired)
	}
}

func TestRepairFuzzyNearMatch(t *testing.T) {
	available := []string{"read_file", "write_file", "shell_command"}
	result := RepairToolName("read_fil", available)

	if result.Strategy != "fuzzy" {
		t.Errorf("expected strategy 'fuzzy', got %q", result.Strategy)
	}
	if result.Repaired != "read_file" {
		t.Errorf("expected repaired 'read_file', got %q", result.Repaired)
	}
	if result.Confidence <= 0.7 {
		t.Errorf("expected confidence > 0.7, got %f", result.Confidence)
	}
}

func TestRepairFuzzyNoMatchTooLow(t *testing.T) {
	available := []string{"read_file", "write_file", "shell_command"}
	result := RepairToolName("xyzabc", available)

	if result.Strategy != "none" {
		t.Errorf("expected strategy 'none', got %q", result.Strategy)
	}
	if result.Repaired != "" {
		t.Errorf("expected empty repaired, got %q", result.Repaired)
	}
}

func TestRepairEmptyName(t *testing.T) {
	available := []string{"read_file", "write_file"}
	result := RepairToolName("", available)

	if result.Strategy != "none" {
		t.Errorf("expected strategy 'none', got %q", result.Strategy)
	}
	if result.Repaired != "" {
		t.Errorf("expected empty repaired, got %q", result.Repaired)
	}
}

func TestRepairEmptyAvailable(t *testing.T) {
	result := RepairToolName("read_file", []string{})

	if result.Strategy != "none" {
		t.Errorf("expected strategy 'none', got %q", result.Strategy)
	}
	if result.Repaired != "" {
		t.Errorf("expected empty repaired, got %q", result.Repaired)
	}
}

func TestRepairNilAvailable(t *testing.T) {
	result := RepairToolName("read_file", nil)

	if result.Strategy != "none" {
		t.Errorf("expected strategy 'none', got %q", result.Strategy)
	}
	if result.Repaired != "" {
		t.Errorf("expected empty repaired, got %q", result.Repaired)
	}
}

func TestRepairBestFuzzyMatchChosen(t *testing.T) {
	// "write_fil" is closer to "write_file" than "read_file"
	available := []string{"read_file", "write_file"}
	result := RepairToolName("write_fil", available)

	if result.Repaired != "write_file" {
		t.Errorf("expected best match 'write_file', got %q", result.Repaired)
	}
}

func TestRepairOriginalPreserved(t *testing.T) {
	available := []string{"read_file"}
	result := RepairToolName("Read-File", available)
	if result.Original != "Read-File" {
		t.Errorf("expected original 'Read-File' preserved, got %q", result.Original)
	}
}

func TestRepairNoneConfidenceIsZero(t *testing.T) {
	result := RepairToolName("zzzzzzzzz", []string{"read_file"})
	if result.Strategy != "none" {
		t.Errorf("expected strategy 'none', got %q", result.Strategy)
	}
	if result.Confidence != 0.0 {
		t.Errorf("expected confidence 0.0 for no match, got %f", result.Confidence)
	}
}

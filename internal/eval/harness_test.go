package eval

import (
	"strings"
	"testing"
	"time"
)

func TestNewHarness(t *testing.T) {
	h := NewHarness()
	if h == nil {
		t.Fatal("NewHarness returned nil")
	}
	if h.Count() != 0 {
		t.Errorf("expected 0 cases, got %d", h.Count())
	}
}

func TestAddCase(t *testing.T) {
	h := NewHarness()
	tc := TestCase{
		ID:       "test-001",
		Name:     "sample test",
		Category: CatCoding,
	}
	h.AddCase(tc)
	if h.Count() != 1 {
		t.Errorf("expected 1 case, got %d", h.Count())
	}
}

func TestAddCases(t *testing.T) {
	h := NewHarness()
	cases := []TestCase{
		{ID: "t1", Name: "first", Category: CatCoding},
		{ID: "t2", Name: "second", Category: CatSecurity},
		{ID: "t3", Name: "third", Category: CatArchitecture},
	}
	h.AddCases(cases)
	if h.Count() != 3 {
		t.Errorf("expected 3 cases, got %d", h.Count())
	}
}

func TestByCategory(t *testing.T) {
	h := NewHarness()
	h.AddCases([]TestCase{
		{ID: "t1", Category: CatCoding},
		{ID: "t2", Category: CatSecurity},
		{ID: "t3", Category: CatCoding},
		{ID: "t4", Category: CatDevSecOps},
	})

	coding := h.ByCategory(CatCoding)
	if len(coding) != 2 {
		t.Errorf("expected 2 coding cases, got %d", len(coding))
	}

	security := h.ByCategory(CatSecurity)
	if len(security) != 1 {
		t.Errorf("expected 1 security case, got %d", len(security))
	}

	arch := h.ByCategory(CatArchitecture)
	if len(arch) != 0 {
		t.Errorf("expected 0 architecture cases, got %d", len(arch))
	}
}

func TestByDifficulty(t *testing.T) {
	h := NewHarness()
	h.AddCases([]TestCase{
		{ID: "t1", Difficulty: Easy},
		{ID: "t2", Difficulty: Medium},
		{ID: "t3", Difficulty: Hard},
		{ID: "t4", Difficulty: Easy},
	})

	easy := h.ByDifficulty(Easy)
	if len(easy) != 2 {
		t.Errorf("expected 2 easy cases, got %d", len(easy))
	}

	hard := h.ByDifficulty(Hard)
	if len(hard) != 1 {
		t.Errorf("expected 1 hard case, got %d", len(hard))
	}
}

func TestRunCaseAllExpectedMatch(t *testing.T) {
	h := NewHarness()
	tc := TestCase{
		ID:       "tc-match-all",
		Name:     "all expected match",
		Category: CatCoding,
		Expected: []string{"hello", "world"},
		Timeout:  5 * time.Second,
	}

	result := h.RunCase(tc, "hello world")
	if !result.Passed {
		t.Error("expected test to pass")
	}
	if result.Score != 1.0 {
		t.Errorf("expected score 1.0, got %f", result.Score)
	}
	if len(result.MatchedExpected) != 2 {
		t.Errorf("expected 2 matched, got %d", len(result.MatchedExpected))
	}
}

func TestRunCasePartialExpectedMatch(t *testing.T) {
	h := NewHarness()
	tc := TestCase{
		ID:       "tc-partial",
		Name:     "partial match",
		Category: CatCoding,
		Expected: []string{"hello", "world", "foo"},
		Timeout:  5 * time.Second,
	}

	result := h.RunCase(tc, "hello world bar")
	// 2/3 expected matched = 0.6667, below 0.7 threshold
	if result.Passed {
		t.Error("expected test to fail (score < 0.7)")
	}
	if result.Score < 0.66 || result.Score > 0.67 {
		t.Errorf("expected score ~0.6667, got %f", result.Score)
	}
}

func TestRunCaseForbiddenMatched(t *testing.T) {
	h := NewHarness()
	tc := TestCase{
		ID:        "tc-forbidden",
		Name:      "forbidden found",
		Category:  CatSecurity,
		Expected:  []string{"safe output"},
		Forbidden: []string{"SECRET_KEY"},
		Timeout:   5 * time.Second,
	}

	result := h.RunCase(tc, "safe output with SECRET_KEY leaked")
	if result.Passed {
		t.Error("expected test to fail due to forbidden match")
	}
	if len(result.MatchedForbidden) != 1 {
		t.Errorf("expected 1 forbidden match, got %d", len(result.MatchedForbidden))
	}
}

func TestRunCaseForbiddenReducesScore(t *testing.T) {
	h := NewHarness()
	tc := TestCase{
		ID:        "tc-forbidden-score",
		Name:      "forbidden reduces score",
		Category:  CatSecurity,
		Expected:  []string{"result"},
		Forbidden: []string{"bad1", "bad2"},
		Timeout:   5 * time.Second,
	}

	// All expected match, 1 of 2 forbidden matched
	result := h.RunCase(tc, "result with bad1 present")
	// score = (1/1) * (1 - 1/2) = 0.5
	if result.Score != 0.5 {
		t.Errorf("expected score 0.5, got %f", result.Score)
	}
	if result.Passed {
		t.Error("expected test to fail (forbidden matched)")
	}
}

func TestRunCaseNoExpectedNoForbidden(t *testing.T) {
	h := NewHarness()
	tc := TestCase{
		ID:       "tc-empty",
		Name:     "no patterns",
		Category: CatCoding,
		Timeout:  5 * time.Second,
	}

	result := h.RunCase(tc, "any output")
	if !result.Passed {
		t.Error("expected pass when no expected/forbidden patterns")
	}
	if result.Score != 1.0 {
		t.Errorf("expected score 1.0 for empty patterns, got %f", result.Score)
	}
}

func TestRunCaseCaseInsensitive(t *testing.T) {
	h := NewHarness()
	tc := TestCase{
		ID:       "tc-case",
		Name:     "case insensitive matching",
		Category: CatCoding,
		Expected: []string{"Hello", "WORLD"},
		Timeout:  5 * time.Second,
	}

	result := h.RunCase(tc, "hello world")
	if !result.Passed {
		t.Error("expected case-insensitive match to pass")
	}
}

func TestRunAll(t *testing.T) {
	h := NewHarness()
	h.AddCases([]TestCase{
		{
			ID:         "t1",
			Name:       "pass test",
			Category:   CatCoding,
			Difficulty: Easy,
			Expected:   []string{"correct"},
		},
		{
			ID:         "t2",
			Name:       "fail test",
			Category:   CatSecurity,
			Difficulty: Hard,
			Expected:   []string{"missing pattern"},
		},
		{
			ID:         "t3",
			Name:       "another pass",
			Category:   CatCoding,
			Difficulty: Easy,
			Expected:   []string{"good"},
		},
	})

	outputs := map[string]string{
		"t1": "correct answer",
		"t2": "wrong answer",
		"t3": "good answer",
	}

	report := h.RunAll(outputs)

	if report.TotalTests != 3 {
		t.Errorf("expected 3 total, got %d", report.TotalTests)
	}
	if report.Passed != 2 {
		t.Errorf("expected 2 passed, got %d", report.Passed)
	}
	if report.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", report.Failed)
	}

	codingReport := report.ByCategory[CatCoding]
	if codingReport.Total != 2 {
		t.Errorf("expected 2 coding tests, got %d", codingReport.Total)
	}
	if codingReport.Passed != 2 {
		t.Errorf("expected 2 coding passed, got %d", codingReport.Passed)
	}

	easyReport := report.ByDifficulty[Easy]
	if easyReport.Total != 2 {
		t.Errorf("expected 2 easy tests, got %d", easyReport.Total)
	}
}

func TestRunAllMissingOutput(t *testing.T) {
	h := NewHarness()
	h.AddCase(TestCase{
		ID:       "t1",
		Name:     "test with no output",
		Category: CatCoding,
		Expected: []string{"something"},
	})

	report := h.RunAll(map[string]string{})

	if report.TotalTests != 1 {
		t.Errorf("expected 1 total, got %d", report.TotalTests)
	}
	if report.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", report.Failed)
	}
}

func TestFormatMarkdown(t *testing.T) {
	h := NewHarness()
	h.AddCases([]TestCase{
		{ID: "t1", Category: CatCoding, Difficulty: Easy, Expected: []string{"yes"}},
		{ID: "t2", Category: CatSecurity, Difficulty: Hard, Expected: []string{"no"}},
	})

	report := h.RunAll(map[string]string{
		"t1": "yes",
		"t2": "no",
	})

	md := report.FormatMarkdown()
	if !strings.Contains(md, "Evaluation Report") {
		t.Error("markdown should contain 'Evaluation Report'")
	}
	if !strings.Contains(md, "coding") {
		t.Error("markdown should contain category name")
	}
	if !strings.Contains(md, "2") {
		t.Error("markdown should contain test counts")
	}
}

func TestScoreCalculationEdgeCases(t *testing.T) {
	h := NewHarness()

	tests := []struct {
		name      string
		tc        TestCase
		output    string
		wantScore float64
		wantPass  bool
	}{
		{
			name: "all expected, no forbidden list",
			tc: TestCase{
				ID: "e1", Expected: []string{"a", "b"}, Forbidden: nil,
			},
			output:    "a b",
			wantScore: 1.0,
			wantPass:  true,
		},
		{
			name: "no expected, forbidden not matched",
			tc: TestCase{
				ID: "e2", Expected: nil, Forbidden: []string{"bad"},
			},
			output:    "good output",
			wantScore: 1.0,
			wantPass:  true,
		},
		{
			name: "no expected, forbidden matched",
			tc: TestCase{
				ID: "e3", Expected: nil, Forbidden: []string{"bad"},
			},
			output:    "bad output",
			wantScore: 0.0,
			wantPass:  false,
		},
		{
			name: "3 of 4 expected, 0 forbidden matched",
			tc: TestCase{
				ID: "e4", Expected: []string{"a", "b", "c", "d"}, Forbidden: []string{"x"},
			},
			output:    "a b c",
			wantScore: 0.75,
			wantPass:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.RunCase(tt.tc, tt.output)
			if result.Passed != tt.wantPass {
				t.Errorf("passed: got %v, want %v (score=%f)", result.Passed, tt.wantPass, result.Score)
			}
			if diff := result.Score - tt.wantScore; diff > 0.01 || diff < -0.01 {
				t.Errorf("score: got %f, want %f", result.Score, tt.wantScore)
			}
		})
	}
}

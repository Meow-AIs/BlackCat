package eval

import (
	"fmt"
	"strings"
	"time"
)

// Difficulty represents the difficulty level of a test case.
type Difficulty string

const (
	Easy   Difficulty = "easy"
	Medium Difficulty = "medium"
	Hard   Difficulty = "hard"
)

// EvalCategory represents the evaluation category.
type EvalCategory string

const (
	CatDevSecOps    EvalCategory = "devsecops"
	CatArchitecture EvalCategory = "architecture"
	CatCoding       EvalCategory = "coding"
	CatSecurity     EvalCategory = "security"
)

// TestCase defines a single evaluation scenario.
type TestCase struct {
	ID         string
	Name       string
	Category   EvalCategory
	Difficulty Difficulty
	Input      string   // prompt/scenario
	Expected   []string // acceptable patterns in output
	Forbidden  []string // patterns that should NOT appear
	Tags       []string
	Timeout    time.Duration
}

// TestResult holds the outcome of evaluating a single test case.
type TestResult struct {
	TestCase         TestCase
	Passed           bool
	Output           string
	Score            float64 // 0.0-1.0 partial credit
	Duration         time.Duration
	MatchedExpected  []string // which expected patterns matched
	MatchedForbidden []string // which forbidden patterns were found
	Error            string
}

// EvalReport aggregates results across all test cases.
type EvalReport struct {
	TotalTests   int
	Passed       int
	Failed       int
	AvgScore     float64
	ByCategory   map[EvalCategory]CategoryReport
	ByDifficulty map[Difficulty]CategoryReport
	Duration     time.Duration
}

// CategoryReport holds aggregated results for a single category or difficulty.
type CategoryReport struct {
	Total    int
	Passed   int
	AvgScore float64
}

// Harness manages test cases and runs evaluations.
type Harness struct {
	Cases []TestCase
}

// NewHarness creates a new empty evaluation harness.
func NewHarness() *Harness {
	return &Harness{
		Cases: []TestCase{},
	}
}

// AddCase appends a single test case to the harness.
func (h *Harness) AddCase(tc TestCase) {
	h.Cases = append(h.Cases, tc)
}

// AddCases appends multiple test cases to the harness.
func (h *Harness) AddCases(cases []TestCase) {
	h.Cases = append(h.Cases, cases...)
}

// Count returns the number of test cases in the harness.
func (h *Harness) Count() int {
	return len(h.Cases)
}

// ByCategory returns all test cases matching the given category.
func (h *Harness) ByCategory(cat EvalCategory) []TestCase {
	var result []TestCase
	for _, tc := range h.Cases {
		if tc.Category == cat {
			result = append(result, tc)
		}
	}
	return result
}

// ByDifficulty returns all test cases matching the given difficulty.
func (h *Harness) ByDifficulty(d Difficulty) []TestCase {
	var result []TestCase
	for _, tc := range h.Cases {
		if tc.Difficulty == d {
			result = append(result, tc)
		}
	}
	return result
}

// RunCase evaluates an output string against a single test case and returns the result.
func (h *Harness) RunCase(tc TestCase, output string) TestResult {
	start := time.Now()
	lowerOutput := strings.ToLower(output)

	// Find matched expected patterns
	var matchedExpected []string
	for _, pattern := range tc.Expected {
		if strings.Contains(lowerOutput, strings.ToLower(pattern)) {
			matchedExpected = append(matchedExpected, pattern)
		}
	}

	// Find matched forbidden patterns
	var matchedForbidden []string
	for _, pattern := range tc.Forbidden {
		if strings.Contains(lowerOutput, strings.ToLower(pattern)) {
			matchedForbidden = append(matchedForbidden, pattern)
		}
	}

	// Calculate score
	score := calculateScore(
		len(matchedExpected), len(tc.Expected),
		len(matchedForbidden), len(tc.Forbidden),
	)

	// Determine pass/fail
	passed := score >= 0.7 && len(matchedForbidden) == 0

	duration := time.Since(start)

	return TestResult{
		TestCase:         tc,
		Passed:           passed,
		Output:           output,
		Score:            score,
		Duration:         duration,
		MatchedExpected:  matchedExpected,
		MatchedForbidden: matchedForbidden,
	}
}

// calculateScore computes the evaluation score.
// Score = (matched_expected / total_expected) * (1 - matched_forbidden / max(1, total_forbidden))
func calculateScore(matchedExp, totalExp, matchedForbid, totalForbid int) float64 {
	var expectedRatio float64
	if totalExp == 0 {
		expectedRatio = 1.0
	} else {
		expectedRatio = float64(matchedExp) / float64(totalExp)
	}

	var forbiddenPenalty float64
	if totalForbid == 0 {
		if matchedForbid > 0 {
			forbiddenPenalty = 1.0
		} else {
			forbiddenPenalty = 0.0
		}
	} else {
		forbiddenPenalty = float64(matchedForbid) / float64(totalForbid)
	}

	score := expectedRatio * (1.0 - forbiddenPenalty)
	if score < 0 {
		return 0.0
	}
	return score
}

// RunAll evaluates all test cases in the harness using the provided outputs map.
// The map key is the test case ID and the value is the output string.
func (h *Harness) RunAll(outputs map[string]string) EvalReport {
	start := time.Now()

	catResults := make(map[EvalCategory]*categoryAccum)
	diffResults := make(map[Difficulty]*categoryAccum)

	var totalScore float64
	passed := 0
	failed := 0

	for _, tc := range h.Cases {
		output, ok := outputs[tc.ID]
		var result TestResult
		if !ok {
			result = TestResult{
				TestCase: tc,
				Passed:   false,
				Score:    0.0,
				Error:    "no output provided",
			}
		} else {
			result = h.RunCase(tc, output)
		}

		totalScore += result.Score
		if result.Passed {
			passed++
		} else {
			failed++
		}

		// Accumulate by category
		acc := getOrCreateAccum(catResults, tc.Category)
		acc.total++
		acc.scoreSum += result.Score
		if result.Passed {
			acc.passed++
		}

		// Accumulate by difficulty
		dacc := getOrCreateDiffAccum(diffResults, tc.Difficulty)
		dacc.total++
		dacc.scoreSum += result.Score
		if result.Passed {
			dacc.passed++
		}
	}

	total := len(h.Cases)
	var avgScore float64
	if total > 0 {
		avgScore = totalScore / float64(total)
	}

	byCategory := make(map[EvalCategory]CategoryReport, len(catResults))
	for cat, acc := range catResults {
		byCategory[cat] = acc.toReport()
	}

	byDifficulty := make(map[Difficulty]CategoryReport, len(diffResults))
	for diff, acc := range diffResults {
		byDifficulty[diff] = acc.toReport()
	}

	return EvalReport{
		TotalTests:   total,
		Passed:       passed,
		Failed:       failed,
		AvgScore:     avgScore,
		ByCategory:   byCategory,
		ByDifficulty: byDifficulty,
		Duration:     time.Since(start),
	}
}

type categoryAccum struct {
	total    int
	passed   int
	scoreSum float64
}

func (a *categoryAccum) toReport() CategoryReport {
	var avg float64
	if a.total > 0 {
		avg = a.scoreSum / float64(a.total)
	}
	return CategoryReport{
		Total:    a.total,
		Passed:   a.passed,
		AvgScore: avg,
	}
}

func getOrCreateAccum[K comparable](m map[K]*categoryAccum, key K) *categoryAccum {
	if acc, ok := m[key]; ok {
		return acc
	}
	acc := &categoryAccum{}
	m[key] = acc
	return acc
}

func getOrCreateDiffAccum(m map[Difficulty]*categoryAccum, key Difficulty) *categoryAccum {
	return getOrCreateAccum(m, key)
}

// FormatMarkdown generates a markdown-formatted evaluation report.
func (r EvalReport) FormatMarkdown() string {
	var sb strings.Builder

	sb.WriteString("# Evaluation Report\n\n")
	sb.WriteString(fmt.Sprintf("**Total Tests:** %d | **Passed:** %d | **Failed:** %d | **Avg Score:** %.2f\n\n",
		r.TotalTests, r.Passed, r.Failed, r.AvgScore))
	sb.WriteString(fmt.Sprintf("**Duration:** %s\n\n", r.Duration.Round(time.Millisecond)))

	if len(r.ByCategory) > 0 {
		sb.WriteString("## By Category\n\n")
		sb.WriteString("| Category | Total | Passed | Avg Score |\n")
		sb.WriteString("|----------|-------|--------|-----------|\n")
		for cat, cr := range r.ByCategory {
			sb.WriteString(fmt.Sprintf("| %s | %d | %d | %.2f |\n", cat, cr.Total, cr.Passed, cr.AvgScore))
		}
		sb.WriteString("\n")
	}

	if len(r.ByDifficulty) > 0 {
		sb.WriteString("## By Difficulty\n\n")
		sb.WriteString("| Difficulty | Total | Passed | Avg Score |\n")
		sb.WriteString("|------------|-------|--------|-----------|\n")
		for diff, dr := range r.ByDifficulty {
			sb.WriteString(fmt.Sprintf("| %s | %d | %d | %.2f |\n", diff, dr.Total, dr.Passed, dr.AvgScore))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

package architect

import (
	"strings"
	"testing"
)

func TestComparisonMatrix_Empty(t *testing.T) {
	m := NewComparisonMatrix()
	results := m.Evaluate()
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestComparisonMatrix_SingleOption(t *testing.T) {
	m := NewComparisonMatrix()
	m.AddCriterion("performance", 0.5)
	m.AddCriterion("ease_of_use", 0.5)
	m.AddOption("Go", map[string]float64{"performance": 9, "ease_of_use": 7})

	results := m.Evaluate()
	if len(results) != 1 {
		t.Fatal("expected 1 result")
	}
	// 9*0.5 + 7*0.5 = 8.0
	if results[0].WeightedScore != 8.0 {
		t.Errorf("expected 8.0, got %.2f", results[0].WeightedScore)
	}
	if results[0].Rank != 1 {
		t.Errorf("expected rank 1, got %d", results[0].Rank)
	}
}

func TestComparisonMatrix_MultipleOptions_Ranked(t *testing.T) {
	m := NewComparisonMatrix()
	m.AddCriterion("performance", 0.6)
	m.AddCriterion("ecosystem", 0.4)

	m.AddOption("Go", map[string]float64{"performance": 9, "ecosystem": 7})
	m.AddOption("Python", map[string]float64{"performance": 5, "ecosystem": 10})
	m.AddOption("Rust", map[string]float64{"performance": 10, "ecosystem": 5})

	results := m.Evaluate()

	// Go: 9*0.6 + 7*0.4 = 5.4 + 2.8 = 8.2
	// Python: 5*0.6 + 10*0.4 = 3.0 + 4.0 = 7.0
	// Rust: 10*0.6 + 5*0.4 = 6.0 + 2.0 = 8.0
	if results[0].Name != "Go" {
		t.Errorf("expected Go first, got %q", results[0].Name)
	}
	if results[1].Name != "Rust" {
		t.Errorf("expected Rust second, got %q", results[1].Name)
	}
	if results[2].Name != "Python" {
		t.Errorf("expected Python third, got %q", results[2].Name)
	}

	if results[0].Rank != 1 || results[1].Rank != 2 || results[2].Rank != 3 {
		t.Error("ranks should be 1, 2, 3")
	}
}

func TestComparisonMatrix_MissingScore_TreatedAsZero(t *testing.T) {
	m := NewComparisonMatrix()
	m.AddCriterion("speed", 1.0)
	m.AddOption("A", map[string]float64{}) // no scores

	results := m.Evaluate()
	if results[0].WeightedScore != 0 {
		t.Errorf("expected 0 for missing scores, got %.2f", results[0].WeightedScore)
	}
}

func TestComparisonMatrix_NormalizeWeights(t *testing.T) {
	m := NewComparisonMatrix()
	m.AddCriterion("a", 3)
	m.AddCriterion("b", 7)
	m.NormalizeWeights()

	if m.Criteria[0].Weight < 0.29 || m.Criteria[0].Weight > 0.31 {
		t.Errorf("expected ~0.30 for 'a', got %.4f", m.Criteria[0].Weight)
	}
	if m.Criteria[1].Weight < 0.69 || m.Criteria[1].Weight > 0.71 {
		t.Errorf("expected ~0.70 for 'b', got %.4f", m.Criteria[1].Weight)
	}
}

func TestComparisonMatrix_NormalizeWeights_ZeroSum(t *testing.T) {
	m := NewComparisonMatrix()
	m.AddCriterion("a", 0)
	m.AddCriterion("b", 0)
	m.NormalizeWeights() // should not panic
}

func TestComparisonMatrix_FormatMarkdown(t *testing.T) {
	m := NewComparisonMatrix()
	m.AddCriterion("speed", 0.5)
	m.AddCriterion("safety", 0.5)
	m.AddOption("Go", map[string]float64{"speed": 8, "safety": 7})
	m.AddOption("Rust", map[string]float64{"speed": 9, "safety": 10})

	md := m.FormatMarkdown()

	if !strings.Contains(md, "Technology") {
		t.Error("expected header row")
	}
	if !strings.Contains(md, "Go") {
		t.Error("expected Go in table")
	}
	if !strings.Contains(md, "Rust") {
		t.Error("expected Rust in table")
	}
	if !strings.Contains(md, "#1") {
		t.Error("expected rank #1")
	}
}

package architect

import (
	"fmt"
	"sort"
	"strings"
)

// Criterion is a weighted evaluation criterion.
type Criterion struct {
	Name   string  `json:"name"`
	Weight float64 `json:"weight"` // 0.0-1.0, weights should sum to 1.0
}

// TechOption represents a technology being evaluated.
type TechOption struct {
	Name   string             `json:"name"`
	Scores map[string]float64 `json:"scores"` // criterion name -> score (0-10)
}

// ComparisonResult holds the evaluation of one technology.
type ComparisonResult struct {
	Name           string  `json:"name"`
	WeightedScore  float64 `json:"weighted_score"`
	Rank           int     `json:"rank"`
}

// ComparisonMatrix evaluates technologies against weighted criteria.
type ComparisonMatrix struct {
	Criteria []Criterion
	Options  []TechOption
}

// NewComparisonMatrix creates an empty matrix.
func NewComparisonMatrix() *ComparisonMatrix {
	return &ComparisonMatrix{}
}

// AddCriterion adds an evaluation criterion.
func (m *ComparisonMatrix) AddCriterion(name string, weight float64) {
	m.Criteria = append(m.Criteria, Criterion{Name: name, Weight: weight})
}

// AddOption adds a technology option with its scores.
func (m *ComparisonMatrix) AddOption(name string, scores map[string]float64) {
	m.Options = append(m.Options, TechOption{Name: name, Scores: scores})
}

// Evaluate computes weighted scores and ranks all options.
func (m *ComparisonMatrix) Evaluate() []ComparisonResult {
	results := make([]ComparisonResult, len(m.Options))

	for i, opt := range m.Options {
		var total float64
		for _, crit := range m.Criteria {
			score, ok := opt.Scores[crit.Name]
			if !ok {
				score = 0
			}
			total += score * crit.Weight
		}
		results[i] = ComparisonResult{
			Name:          opt.Name,
			WeightedScore: total,
		}
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].WeightedScore > results[j].WeightedScore
	})

	// Assign ranks
	for i := range results {
		results[i].Rank = i + 1
	}

	return results
}

// NormalizeWeights adjusts weights so they sum to 1.0.
func (m *ComparisonMatrix) NormalizeWeights() {
	var sum float64
	for _, c := range m.Criteria {
		sum += c.Weight
	}
	if sum == 0 {
		return
	}
	for i := range m.Criteria {
		m.Criteria[i].Weight /= sum
	}
}

// FormatMarkdown renders the comparison as a markdown table.
func (m *ComparisonMatrix) FormatMarkdown() string {
	results := m.Evaluate()
	var b strings.Builder

	// Header
	b.WriteString("| Technology |")
	for _, c := range m.Criteria {
		b.WriteString(fmt.Sprintf(" %s (%.0f%%) |", c.Name, c.Weight*100))
	}
	b.WriteString(" **Score** | **Rank** |\n")

	// Separator
	b.WriteString("|------------|")
	for range m.Criteria {
		b.WriteString("--------|")
	}
	b.WriteString("---------|----------|\n")

	// Rows
	for _, r := range results {
		b.WriteString(fmt.Sprintf("| %s |", r.Name))
		for _, c := range m.Criteria {
			score := 0.0
			for _, opt := range m.Options {
				if opt.Name == r.Name {
					score = opt.Scores[c.Name]
					break
				}
			}
			b.WriteString(fmt.Sprintf(" %.1f |", score))
		}
		b.WriteString(fmt.Sprintf(" **%.2f** | #%d |\n", r.WeightedScore, r.Rank))
	}

	return b.String()
}

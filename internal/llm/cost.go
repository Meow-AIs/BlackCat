package llm

import "sync"

// CostTracker tracks token usage and estimated cost per session.
type CostTracker struct {
	mu       sync.Mutex
	entries  []CostEntry
	budget   float64 // max USD per session, 0 = unlimited
	warnAt   float64
}

// CostEntry records a single API call's cost.
type CostEntry struct {
	Model            string  `json:"model"`
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	Cost             float64 `json:"cost"`
}

// CostSummary is the aggregate cost for a session.
type CostSummary struct {
	TotalCost        float64 `json:"total_cost"`
	TotalPrompt      int     `json:"total_prompt_tokens"`
	TotalCompletion  int     `json:"total_completion_tokens"`
	Entries          int     `json:"entries"`
	BudgetRemaining  float64 `json:"budget_remaining"`
	OverBudget       bool    `json:"over_budget"`
}

// NewCostTracker creates a tracker with optional budget.
func NewCostTracker(budget, warnAt float64) *CostTracker {
	return &CostTracker{budget: budget, warnAt: warnAt}
}

// Record adds a usage entry and calculates cost.
func (ct *CostTracker) Record(model string, usage Usage, modelInfo *ModelInfo) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	cost := 0.0
	if modelInfo != nil {
		cost = float64(usage.PromptTokens)/1_000_000*modelInfo.InputCost +
			float64(usage.CompletionTokens)/1_000_000*modelInfo.OutputCost
	}

	ct.entries = append(ct.entries, CostEntry{
		Model:            model,
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		Cost:             cost,
	})
}

// Summary returns the aggregate cost info.
func (ct *CostTracker) Summary() CostSummary {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	var s CostSummary
	for _, e := range ct.entries {
		s.TotalCost += e.Cost
		s.TotalPrompt += e.PromptTokens
		s.TotalCompletion += e.CompletionTokens
	}
	s.Entries = len(ct.entries)
	if ct.budget > 0 {
		s.BudgetRemaining = ct.budget - s.TotalCost
		s.OverBudget = s.TotalCost > ct.budget
	}
	return s
}

// ShouldWarn returns true if cost exceeds the warning threshold.
func (ct *CostTracker) ShouldWarn() bool {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	if ct.warnAt <= 0 {
		return false
	}
	total := 0.0
	for _, e := range ct.entries {
		total += e.Cost
	}
	return total >= ct.warnAt
}

// IsOverBudget returns true if total cost exceeds budget.
func (ct *CostTracker) IsOverBudget() bool {
	return ct.Summary().OverBudget
}

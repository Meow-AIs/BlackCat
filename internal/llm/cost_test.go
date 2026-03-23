package llm

import "testing"

func TestCostTrackerRecord(t *testing.T) {
	ct := NewCostTracker(2.0, 1.5)

	model := &ModelInfo{ID: "gpt-4.1", InputCost: 2.0, OutputCost: 8.0}
	ct.Record("gpt-4.1", Usage{PromptTokens: 1000, CompletionTokens: 500}, model)

	s := ct.Summary()
	if s.Entries != 1 {
		t.Errorf("expected 1 entry, got %d", s.Entries)
	}
	if s.TotalPrompt != 1000 {
		t.Errorf("expected 1000 prompt tokens, got %d", s.TotalPrompt)
	}
	// Cost: 1000/1M * 2.0 + 500/1M * 8.0 = 0.002 + 0.004 = 0.006
	expectedCost := 0.006
	if s.TotalCost < expectedCost-0.001 || s.TotalCost > expectedCost+0.001 {
		t.Errorf("expected cost ~%.4f, got %.4f", expectedCost, s.TotalCost)
	}
}

func TestCostTrackerBudget(t *testing.T) {
	ct := NewCostTracker(0.01, 0.005)
	model := &ModelInfo{ID: "gpt-4.1", InputCost: 2.0, OutputCost: 8.0}

	// First call: ~0.006
	ct.Record("gpt-4.1", Usage{PromptTokens: 1000, CompletionTokens: 500}, model)
	if ct.IsOverBudget() {
		t.Error("should not be over budget yet")
	}
	if !ct.ShouldWarn() {
		t.Error("should warn (cost >= 0.005)")
	}

	// Second call: pushes over 0.01
	ct.Record("gpt-4.1", Usage{PromptTokens: 1000, CompletionTokens: 500}, model)
	if !ct.IsOverBudget() {
		t.Error("should be over budget")
	}
}

func TestCostTrackerNoBudget(t *testing.T) {
	ct := NewCostTracker(0, 0)
	model := &ModelInfo{ID: "gpt-4.1", InputCost: 2.0, OutputCost: 8.0}
	ct.Record("gpt-4.1", Usage{PromptTokens: 100000, CompletionTokens: 50000}, model)

	if ct.IsOverBudget() {
		t.Error("should never be over budget with 0 budget")
	}
	if ct.ShouldWarn() {
		t.Error("should never warn with 0 warn threshold")
	}
}

func TestCostTrackerNilModelInfo(t *testing.T) {
	ct := NewCostTracker(0, 0)
	ct.Record("unknown", Usage{PromptTokens: 100, CompletionTokens: 50}, nil)

	s := ct.Summary()
	if s.TotalCost != 0 {
		t.Errorf("expected 0 cost with nil model info, got %f", s.TotalCost)
	}
	if s.TotalPrompt != 100 {
		t.Errorf("expected 100 prompt tokens, got %d", s.TotalPrompt)
	}
}

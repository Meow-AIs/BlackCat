package architect

import (
	"strings"
	"testing"
)

func TestLoadReliabilityPatterns(t *testing.T) {
	patterns := LoadReliabilityPatterns()
	if len(patterns) < 15 {
		t.Errorf("expected at least 15 patterns, got %d", len(patterns))
	}
	for _, p := range patterns {
		if p.Name == "" {
			t.Error("pattern has empty name")
		}
		if p.Category == "" {
			t.Errorf("pattern %q has empty category", p.Name)
		}
		if p.Description == "" {
			t.Errorf("pattern %q has empty description", p.Name)
		}
	}
}

func TestReliabilityPatternCategories(t *testing.T) {
	patterns := LoadReliabilityPatterns()
	cats := make(map[string]bool)
	for _, p := range patterns {
		cats[p.Category] = true
	}
	required := []string{"resilience", "observability", "deployment", "data", "chaos"}
	for _, c := range required {
		if !cats[c] {
			t.Errorf("missing category %q", c)
		}
	}
}

func TestSearchReliabilityPatterns(t *testing.T) {
	results := SearchReliabilityPatterns("circuit")
	if len(results) == 0 {
		t.Error("expected at least one result for 'circuit'")
	}

	results = SearchReliabilityPatterns("xyznonexistent")
	if len(results) != 0 {
		t.Errorf("expected no results, got %d", len(results))
	}
}

func TestGetReliabilityPattern(t *testing.T) {
	p, ok := GetReliabilityPattern("Circuit Breaker")
	if !ok {
		t.Fatal("expected to find Circuit Breaker")
	}
	if p.Category != "resilience" {
		t.Errorf("unexpected category: %q", p.Category)
	}

	_, ok = GetReliabilityPattern("nonexistent")
	if ok {
		t.Error("expected not found")
	}
}

func TestDefaultSLIs(t *testing.T) {
	slis := DefaultSLIs()
	if len(slis) < 4 {
		t.Errorf("expected at least 4 SLIs, got %d", len(slis))
	}
	metrics := make(map[string]bool)
	for _, s := range slis {
		if s.Name == "" {
			t.Error("SLI has empty name")
		}
		metrics[s.Metric] = true
	}
	for _, m := range []string{"latency_p99", "error_rate", "availability", "throughput"} {
		if !metrics[m] {
			t.Errorf("missing SLI metric %q", m)
		}
	}
}

func TestCalculateErrorBudget(t *testing.T) {
	// 99.9% over 30 days = 0.1% of 30 days = 43.2 minutes
	budget := CalculateErrorBudget(99.9, 30, 0)
	if budget < 40 || budget > 50 {
		t.Errorf("expected ~43 minutes of error budget, got %f", budget)
	}

	// No downtime means full budget remaining.
	budget = CalculateErrorBudget(99.9, 30, 0)
	if budget <= 0 {
		t.Error("expected positive error budget with no downtime")
	}
}

func TestCalculateErrorBudgetFullyConsumed(t *testing.T) {
	// If downtime equals the full budget, remaining should be ~0.
	budget := CalculateErrorBudget(99.9, 30, 43.2)
	if budget < -1 || budget > 1 {
		t.Errorf("expected ~0 remaining budget, got %f", budget)
	}
}

func TestCalculateBurnRate(t *testing.T) {
	// 1x burn rate = consuming budget at exactly the expected rate.
	rate := CalculateBurnRate(99.9, 30, 43.2)
	if rate < 0.9 || rate > 1.1 {
		t.Errorf("expected burn rate ~1.0, got %f", rate)
	}

	// Double the downtime = 2x burn rate.
	rate = CalculateBurnRate(99.9, 30, 86.4)
	if rate < 1.8 || rate > 2.2 {
		t.Errorf("expected burn rate ~2.0, got %f", rate)
	}

	// No downtime = 0 burn rate.
	rate = CalculateBurnRate(99.9, 30, 0)
	if rate != 0 {
		t.Errorf("expected burn rate 0, got %f", rate)
	}
}

func TestGenerateRunbook(t *testing.T) {
	rb := GenerateRunbook("api-gateway", "HighLatency", "SEV2")
	if rb.Title == "" {
		t.Error("runbook has empty title")
	}
	if rb.Service != "api-gateway" {
		t.Errorf("unexpected service: %q", rb.Service)
	}
	if len(rb.Steps) == 0 {
		t.Error("runbook has no steps")
	}
	for i, step := range rb.Steps {
		if step.Action == "" {
			t.Errorf("step %d has empty action", i)
		}
		if step.Order != i+1 {
			t.Errorf("step %d has order %d", i, step.Order)
		}
	}
}

func TestRunbookFormatMarkdown(t *testing.T) {
	rb := GenerateRunbook("payments", "HighErrorRate", "SEV1")
	md := rb.FormatMarkdown()
	if !strings.Contains(md, "Runbook") {
		t.Error("markdown missing header")
	}
	if !strings.Contains(md, "payments") {
		t.Error("markdown missing service name")
	}
	if !strings.Contains(md, "Escalation") {
		t.Error("markdown missing escalation section")
	}
}

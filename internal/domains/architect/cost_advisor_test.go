package architect

import (
	"strings"
	"testing"
)

func TestNewCostAdvisor(t *testing.T) {
	ca := NewCostAdvisor()
	if ca == nil {
		t.Fatal("expected non-nil CostAdvisor")
	}
}

func TestIdentifyWasteIdleCompute(t *testing.T) {
	ca := NewCostAdvisor()
	resources := []ResourceDescription{
		{Type: "ec2", Size: "t3.xlarge", Usage: 0.1, Region: "us-east-1"},
	}
	wastes := ca.IdentifyWaste(resources)
	if len(wastes) == 0 {
		t.Fatal("expected waste for idle compute")
	}
	found := false
	for _, w := range wastes {
		if w.Category == "idle_compute" || w.Category == "oversized" {
			found = true
		}
	}
	if !found {
		t.Error("expected idle_compute or oversized waste category")
	}
}

func TestIdentifyWasteHighUtilization(t *testing.T) {
	ca := NewCostAdvisor()
	resources := []ResourceDescription{
		{Type: "ec2", Size: "t3.xlarge", Usage: 0.9, Region: "us-east-1"},
	}
	wastes := ca.IdentifyWaste(resources)
	found := false
	for _, w := range wastes {
		if w.Category == "missing_reserved" {
			found = true
		}
	}
	if !found {
		t.Error("expected missing_reserved recommendation for high utilization")
	}
}

func TestIdentifyWasteUnusedStorage(t *testing.T) {
	ca := NewCostAdvisor()
	resources := []ResourceDescription{
		{Type: "ebs", Size: "gp3", Usage: 0, Region: "us-east-1"},
	}
	wastes := ca.IdentifyWaste(resources)
	if len(wastes) == 0 {
		t.Fatal("expected waste for unused storage")
	}
	found := false
	for _, w := range wastes {
		if w.Category == "unused_storage" || w.Category == "unattached_resources" {
			found = true
		}
	}
	if !found {
		t.Error("expected unused_storage or unattached_resources category")
	}
}

func TestIdentifyWasteEmpty(t *testing.T) {
	ca := NewCostAdvisor()
	wastes := ca.IdentifyWaste(nil)
	if len(wastes) != 0 {
		t.Errorf("expected no waste for nil resources, got %d", len(wastes))
	}
}

func TestIdentifyWasteOptimized(t *testing.T) {
	ca := NewCostAdvisor()
	resources := []ResourceDescription{
		{Type: "ec2", Size: "t3.small", Usage: 0.55, Region: "us-east-1",
			Tags: map[string]string{"reserved": "true"}},
	}
	wastes := ca.IdentifyWaste(resources)
	// Well-utilized reserved instance should produce minimal or no waste.
	for _, w := range wastes {
		if w.Category == "idle_compute" || w.Category == "oversized" {
			t.Error("unexpected waste for well-utilized reserved instance")
		}
	}
}

func TestOptimizationRules(t *testing.T) {
	rules := OptimizationRules()
	if len(rules) < 10 {
		t.Errorf("expected at least 10 rules, got %d", len(rules))
	}
}

func TestEstimateSavings(t *testing.T) {
	ca := NewCostAdvisor()
	wastes := []CostWaste{
		{Savings: 100},
		{Savings: 250},
		{Savings: 50},
	}
	total := ca.EstimateSavings(wastes)
	if total != 400 {
		t.Errorf("expected 400 total savings, got %f", total)
	}
}

func TestEstimateSavingsEmpty(t *testing.T) {
	ca := NewCostAdvisor()
	total := ca.EstimateSavings(nil)
	if total != 0 {
		t.Errorf("expected 0 savings for nil wastes, got %f", total)
	}
}

func TestFormatReport(t *testing.T) {
	ca := NewCostAdvisor()
	wastes := []CostWaste{
		{Category: "idle_compute", Resource: "i-abc123", CurrentCost: 500, OptimalCost: 100, Savings: 400, Action: "Downsize", Effort: "easy", Risk: "low"},
	}
	report := ca.FormatReport(wastes)
	if !strings.Contains(report, "Cost Optimization") {
		t.Error("report missing header")
	}
	if !strings.Contains(report, "i-abc123") {
		t.Error("report missing resource")
	}
	if !strings.Contains(report, "Downsize") {
		t.Error("report missing action")
	}
}

func TestFormatReportEmpty(t *testing.T) {
	ca := NewCostAdvisor()
	report := ca.FormatReport(nil)
	if !strings.Contains(report, "No waste") || !strings.Contains(report, "no waste") {
		// Accept either casing.
		if !strings.Contains(strings.ToLower(report), "no waste") && !strings.Contains(strings.ToLower(report), "optimized") {
			t.Error("empty report should indicate no waste found")
		}
	}
}

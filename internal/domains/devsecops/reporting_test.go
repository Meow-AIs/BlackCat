package devsecops

import (
	"strings"
	"testing"
)

func TestGenerateSecurityPostureReport(t *testing.T) {
	findings := []Finding{
		{ID: "F1", Severity: SeverityCritical, Title: "SQL Injection", Scanner: "sast"},
		{ID: "F2", Severity: SeverityHigh, Title: "XSS", Scanner: "sast"},
		{ID: "F3", Severity: SeverityMedium, Title: "Missing HSTS", Scanner: "headers"},
		{ID: "F4", Severity: SeverityLow, Title: "Verbose errors", Scanner: "sast"},
	}
	report := GenerateSecurityPostureReport("myapp", findings, nil)
	if report.ProjectName != "myapp" {
		t.Errorf("unexpected project name: %q", report.ProjectName)
	}
	if report.OverallScore < 0 || report.OverallScore > 100 {
		t.Errorf("score out of range: %f", report.OverallScore)
	}
	if report.RiskLevel == "" {
		t.Error("risk level is empty")
	}
}

func TestSecurityPostureReportAllCritical(t *testing.T) {
	findings := []Finding{
		{ID: "F1", Severity: SeverityCritical, Title: "Crit1", Scanner: "s"},
		{ID: "F2", Severity: SeverityCritical, Title: "Crit2", Scanner: "s"},
		{ID: "F3", Severity: SeverityCritical, Title: "Crit3", Scanner: "s"},
	}
	report := GenerateSecurityPostureReport("proj", findings, nil)
	if report.RiskLevel != "critical" {
		t.Errorf("expected critical risk level, got %q", report.RiskLevel)
	}
	if report.OverallScore > 30 {
		t.Errorf("expected low score for all-critical findings, got %f", report.OverallScore)
	}
}

func TestSecurityPostureReportAllLow(t *testing.T) {
	findings := []Finding{
		{ID: "F1", Severity: SeverityLow, Title: "Low1", Scanner: "s"},
		{ID: "F2", Severity: SeverityLow, Title: "Low2", Scanner: "s"},
	}
	report := GenerateSecurityPostureReport("proj", findings, nil)
	if report.RiskLevel != "low" {
		t.Errorf("expected low risk level, got %q", report.RiskLevel)
	}
	if report.OverallScore < 70 {
		t.Errorf("expected high score for all-low findings, got %f", report.OverallScore)
	}
}

func TestSecurityPostureReportEmpty(t *testing.T) {
	report := GenerateSecurityPostureReport("clean", nil, nil)
	if report.OverallScore != 100 {
		t.Errorf("expected score 100 for no findings, got %f", report.OverallScore)
	}
	if report.RiskLevel != "low" {
		t.Errorf("expected low risk level for no findings, got %q", report.RiskLevel)
	}
}

func TestSecurityPostureFormatMarkdown(t *testing.T) {
	findings := []Finding{
		{ID: "F1", Severity: SeverityHigh, Title: "XSS", Scanner: "sast"},
	}
	report := GenerateSecurityPostureReport("app", findings, nil)
	md := report.FormatMarkdown()
	if !strings.Contains(md, "Security Posture") {
		t.Error("markdown missing header")
	}
	if !strings.Contains(md, "app") {
		t.Error("markdown missing project name")
	}
}

func TestSecurityPostureFormatExecutiveSummary(t *testing.T) {
	report := GenerateSecurityPostureReport("app", nil, nil)
	summary := report.FormatExecutiveSummary()
	if !strings.Contains(summary, "app") {
		t.Error("executive summary missing project name")
	}
}

func TestGenerateArchReviewReport(t *testing.T) {
	scores := map[string]float64{
		"security":               4.0,
		"reliability":            3.5,
		"performance":            4.2,
		"cost_optimization":      3.0,
		"operational_excellence": 3.8,
		"sustainability":         2.5,
	}
	report := GenerateArchReviewReport("myservice", scores)
	if report.ProjectName != "myservice" {
		t.Errorf("unexpected project name: %q", report.ProjectName)
	}
	if report.OverallScore <= 0 {
		t.Error("expected positive overall score")
	}
	if len(report.Strengths) == 0 {
		t.Error("expected at least one strength")
	}
	if len(report.Weaknesses) == 0 {
		t.Error("expected at least one weakness")
	}
}

func TestArchReviewFormatMarkdown(t *testing.T) {
	scores := map[string]float64{"security": 4.0, "reliability": 3.0}
	report := GenerateArchReviewReport("svc", scores)
	md := report.FormatMarkdown()
	if !strings.Contains(md, "Architecture Review") {
		t.Error("markdown missing header")
	}
}

func TestGenerateCostOptReport(t *testing.T) {
	costs := []CostRecommendation{
		{Category: "compute", Description: "Downsize instance", CurrentCost: 500, OptimizedCost: 200, Savings: 300, Effort: "easy", Risk: "low"},
		{Category: "storage", Description: "Move to cold tier", CurrentCost: 100, OptimizedCost: 20, Savings: 80, Effort: "medium", Risk: "low"},
	}
	report := GenerateCostOptReport(costs)
	if report.CurrentMonthly != 600 {
		t.Errorf("expected current monthly 600, got %f", report.CurrentMonthly)
	}
	if report.ProjectedSaving != 380 {
		t.Errorf("expected projected saving 380, got %f", report.ProjectedSaving)
	}
	if len(report.Recommendations) != 2 {
		t.Errorf("expected 2 recommendations, got %d", len(report.Recommendations))
	}
}

func TestCostOptReportEmpty(t *testing.T) {
	report := GenerateCostOptReport(nil)
	if report.CurrentMonthly != 0 {
		t.Error("expected 0 for empty costs")
	}
	if report.ProjectedSaving != 0 {
		t.Error("expected 0 savings for empty costs")
	}
}

func TestCostOptFormatMarkdown(t *testing.T) {
	costs := []CostRecommendation{
		{Category: "compute", Description: "Right-size", CurrentCost: 100, OptimizedCost: 50, Savings: 50, Effort: "easy", Risk: "low"},
	}
	report := GenerateCostOptReport(costs)
	md := report.FormatMarkdown()
	if !strings.Contains(md, "Cost Optimization") {
		t.Error("markdown missing header")
	}
	if !strings.Contains(md, "Right-size") {
		t.Error("markdown missing recommendation description")
	}
}

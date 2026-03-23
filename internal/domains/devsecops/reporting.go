package devsecops

import (
	"fmt"
	"strings"
	"time"
)

// SecurityPostureReport summarizes the overall security posture of a project.
type SecurityPostureReport struct {
	ProjectName     string
	ScanDate        string
	OverallScore    float64 // 0-100
	RiskLevel       string  // "critical", "high", "medium", "low"
	Sections        []ReportSection
	Recommendations []string
}

// ReportSection represents a scored section within a security report.
type ReportSection struct {
	Title    string
	Score    float64
	Findings int
	Critical int
	High     int
	Medium   int
	Low      int
	Details  string
}

// GenerateSecurityPostureReport builds a posture report from findings.
func GenerateSecurityPostureReport(projectName string, findings []Finding, compliance *ComplianceGapReport) *SecurityPostureReport {
	report := &SecurityPostureReport{
		ProjectName: projectName,
		ScanDate:    time.Now().Format("2006-01-02"),
	}

	if len(findings) == 0 {
		report.OverallScore = 100
		report.RiskLevel = "low"
		report.Recommendations = []string{"Continue regular scanning to maintain clean posture"}
		return report
	}

	var critCount, highCount, medCount, lowCount int
	for _, f := range findings {
		switch f.Severity {
		case SeverityCritical:
			critCount++
		case SeverityHigh:
			highCount++
		case SeverityMedium:
			medCount++
		default:
			lowCount++
		}
	}

	section := ReportSection{
		Title:    "Security Findings",
		Findings: len(findings),
		Critical: critCount,
		High:     highCount,
		Medium:   medCount,
		Low:      lowCount,
	}

	// Score: start at 100, deduct per severity.
	deduction := float64(critCount)*25 + float64(highCount)*10 + float64(medCount)*3 + float64(lowCount)*1
	score := 100 - deduction
	if score < 0 {
		score = 0
	}
	section.Score = score
	report.Sections = append(report.Sections, section)
	report.OverallScore = score

	// Risk level based on score and critical findings.
	switch {
	case critCount > 0 || score < 30:
		report.RiskLevel = "critical"
	case highCount > 0 || score < 50:
		report.RiskLevel = "high"
	case medCount > 0 || score < 70:
		report.RiskLevel = "medium"
	default:
		report.RiskLevel = "low"
	}

	// Recommendations.
	if critCount > 0 {
		report.Recommendations = append(report.Recommendations, fmt.Sprintf("Fix %d critical findings immediately", critCount))
	}
	if highCount > 0 {
		report.Recommendations = append(report.Recommendations, fmt.Sprintf("Address %d high-severity findings within this sprint", highCount))
	}
	if medCount > 0 {
		report.Recommendations = append(report.Recommendations, fmt.Sprintf("Plan remediation for %d medium findings", medCount))
	}

	if compliance != nil && compliance.Score < 80 {
		report.Recommendations = append(report.Recommendations,
			fmt.Sprintf("Compliance gap: %s score is %.0f%% — review gaps", string(compliance.Framework), compliance.Score))
	}

	return report
}

// FormatMarkdown renders the security posture report as Markdown.
func (r *SecurityPostureReport) FormatMarkdown() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# Security Posture Report: %s\n\n", r.ProjectName))
	b.WriteString(fmt.Sprintf("**Date:** %s | **Score:** %.0f/100 | **Risk:** %s\n\n", r.ScanDate, r.OverallScore, r.RiskLevel))

	for _, s := range r.Sections {
		b.WriteString(fmt.Sprintf("## %s\n\n", s.Title))
		b.WriteString(fmt.Sprintf("- **Total findings:** %d\n", s.Findings))
		b.WriteString(fmt.Sprintf("- Critical: %d | High: %d | Medium: %d | Low: %d\n", s.Critical, s.High, s.Medium, s.Low))
		b.WriteString(fmt.Sprintf("- **Section Score:** %.0f/100\n\n", s.Score))
	}

	if len(r.Recommendations) > 0 {
		b.WriteString("## Recommendations\n\n")
		for _, rec := range r.Recommendations {
			b.WriteString(fmt.Sprintf("- %s\n", rec))
		}
	}

	return b.String()
}

// FormatExecutiveSummary returns a brief executive summary.
func (r *SecurityPostureReport) FormatExecutiveSummary() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Executive Summary: %s\n\n", r.ProjectName))
	b.WriteString(fmt.Sprintf("**Security Score:** %.0f/100 (%s risk)\n\n", r.OverallScore, r.RiskLevel))

	total := 0
	for _, s := range r.Sections {
		total += s.Findings
	}
	b.WriteString(fmt.Sprintf("**Total Findings:** %d\n", total))

	if len(r.Recommendations) > 0 {
		b.WriteString(fmt.Sprintf("\n**Top Priority:** %s\n", r.Recommendations[0]))
	}
	return b.String()
}

// ArchitectureReviewReport summarizes a WAF-based architecture review.
type ArchitectureReviewReport struct {
	ProjectName     string
	ReviewDate      string
	Reviewer        string
	WAFScores       map[string]float64 // pillar -> score
	OverallScore    float64
	Strengths       []string
	Weaknesses      []string
	Recommendations []string
	RiskAreas       []string
}

// GenerateArchReviewReport builds an architecture review from WAF pillar scores.
func GenerateArchReviewReport(projectName string, wafScores map[string]float64) *ArchitectureReviewReport {
	report := &ArchitectureReviewReport{
		ProjectName: projectName,
		ReviewDate:  time.Now().Format("2006-01-02"),
		WAFScores:   wafScores,
	}

	var total float64
	count := 0
	for pillar, score := range wafScores {
		total += score
		count++
		if score >= 4.0 {
			report.Strengths = append(report.Strengths, fmt.Sprintf("%s (%.1f/5.0)", pillar, score))
		}
		if score < 3.0 {
			report.Weaknesses = append(report.Weaknesses, fmt.Sprintf("%s (%.1f/5.0)", pillar, score))
			report.Recommendations = append(report.Recommendations, fmt.Sprintf("Improve %s — currently %.1f/5.0", pillar, score))
			report.RiskAreas = append(report.RiskAreas, pillar)
		}
	}

	if count > 0 {
		report.OverallScore = total / float64(count)
	}

	return report
}

// FormatMarkdown renders the architecture review as Markdown.
func (r *ArchitectureReviewReport) FormatMarkdown() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# Architecture Review: %s\n\n", r.ProjectName))
	b.WriteString(fmt.Sprintf("**Date:** %s | **Overall Score:** %.1f/5.0\n\n", r.ReviewDate, r.OverallScore))

	b.WriteString("## WAF Pillar Scores\n\n")
	for pillar, score := range r.WAFScores {
		b.WriteString(fmt.Sprintf("- **%s:** %.1f/5.0\n", pillar, score))
	}
	b.WriteString("\n")

	if len(r.Strengths) > 0 {
		b.WriteString("## Strengths\n\n")
		for _, s := range r.Strengths {
			b.WriteString(fmt.Sprintf("- %s\n", s))
		}
		b.WriteString("\n")
	}

	if len(r.Weaknesses) > 0 {
		b.WriteString("## Weaknesses\n\n")
		for _, w := range r.Weaknesses {
			b.WriteString(fmt.Sprintf("- %s\n", w))
		}
		b.WriteString("\n")
	}

	if len(r.Recommendations) > 0 {
		b.WriteString("## Recommendations\n\n")
		for _, rec := range r.Recommendations {
			b.WriteString(fmt.Sprintf("- %s\n", rec))
		}
	}

	return b.String()
}

// CostOptReport summarizes cost optimization opportunities.
type CostOptReport struct {
	CurrentMonthly  float64
	ProjectedSaving float64
	Recommendations []CostRecommendation
}

// CostRecommendation describes a single cost optimization opportunity.
type CostRecommendation struct {
	Category      string  // "compute", "storage", "transfer", "database", "unused"
	Description   string
	CurrentCost   float64
	OptimizedCost float64
	Savings       float64
	Effort        string // "easy", "medium", "hard"
	Risk          string // "low", "medium", "high"
}

// GenerateCostOptReport builds a cost optimization report from recommendations.
func GenerateCostOptReport(costs []CostRecommendation) *CostOptReport {
	report := &CostOptReport{Recommendations: costs}
	for _, c := range costs {
		report.CurrentMonthly += c.CurrentCost
		report.ProjectedSaving += c.Savings
	}
	return report
}

// FormatMarkdown renders the cost optimization report as Markdown.
func (r *CostOptReport) FormatMarkdown() string {
	var b strings.Builder

	b.WriteString("# Cost Optimization Report\n\n")
	b.WriteString(fmt.Sprintf("**Current Monthly:** $%.2f | **Projected Savings:** $%.2f\n\n", r.CurrentMonthly, r.ProjectedSaving))

	if len(r.Recommendations) == 0 {
		b.WriteString("No cost optimization opportunities identified.\n")
		return b.String()
	}

	b.WriteString("## Recommendations\n\n")
	b.WriteString("| Category | Description | Current | Optimized | Savings | Effort | Risk |\n")
	b.WriteString("|----------|-------------|---------|-----------|---------|--------|------|\n")
	for _, c := range r.Recommendations {
		b.WriteString(fmt.Sprintf("| %s | %s | $%.2f | $%.2f | $%.2f | %s | %s |\n",
			c.Category, c.Description, c.CurrentCost, c.OptimizedCost, c.Savings, c.Effort, c.Risk))
	}

	return b.String()
}

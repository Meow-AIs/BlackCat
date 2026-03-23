package devsecops

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// ListFrameworks
// ---------------------------------------------------------------------------

func TestListFrameworks_ReturnsAllFrameworks(t *testing.T) {
	frameworks := ListFrameworks()
	if len(frameworks) < 6 {
		t.Errorf("expected at least 6 frameworks, got %d", len(frameworks))
	}
	expected := []ComplianceFramework{
		FrameworkSOC2, FrameworkISO27001, FrameworkPCIDSS,
		FrameworkCIS, FrameworkNIST, FrameworkHIPAA,
	}
	fwSet := map[ComplianceFramework]bool{}
	for _, f := range frameworks {
		fwSet[f] = true
	}
	for _, e := range expected {
		if !fwSet[e] {
			t.Errorf("expected framework %q in list", e)
		}
	}
}

// ---------------------------------------------------------------------------
// LoadComplianceMappings
// ---------------------------------------------------------------------------

func TestLoadComplianceMappings_HasMappings(t *testing.T) {
	mappings := LoadComplianceMappings()
	if len(mappings) < 10 {
		t.Errorf("expected at least 10 mappings, got %d", len(mappings))
	}
}

func TestLoadComplianceMappings_AllHaveCategory(t *testing.T) {
	for _, m := range LoadComplianceMappings() {
		if m.FindingCategory == "" {
			t.Error("found mapping with empty FindingCategory")
		}
		if len(m.Controls) == 0 {
			t.Errorf("mapping %q has no controls", m.FindingCategory)
		}
	}
}

func TestLoadComplianceMappings_ControlsHaveRequiredFields(t *testing.T) {
	for _, m := range LoadComplianceMappings() {
		for _, c := range m.Controls {
			if c.ID == "" {
				t.Errorf("control in mapping %q has empty ID", m.FindingCategory)
			}
			if c.Framework == "" {
				t.Errorf("control %q has empty Framework", c.ID)
			}
			if c.Title == "" {
				t.Errorf("control %q has empty Title", c.ID)
			}
			if c.Category == "" {
				t.Errorf("control %q has empty Category", c.ID)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// MapFindingToControls
// ---------------------------------------------------------------------------

func TestMapFindingToControls_HardcodedSecret(t *testing.T) {
	controls := MapFindingToControls("hardcoded-secret")
	if len(controls) == 0 {
		t.Fatal("expected controls for 'hardcoded-secret'")
	}
	// Should map to multiple frameworks
	frameworks := map[ComplianceFramework]bool{}
	for _, c := range controls {
		frameworks[c.Framework] = true
	}
	if len(frameworks) < 2 {
		t.Errorf("expected controls from at least 2 frameworks, got %d", len(frameworks))
	}
}

func TestMapFindingToControls_NoEncryption(t *testing.T) {
	controls := MapFindingToControls("no-encryption")
	if len(controls) == 0 {
		t.Fatal("expected controls for 'no-encryption'")
	}
}

func TestMapFindingToControls_UnknownCategory(t *testing.T) {
	controls := MapFindingToControls("nonexistent-category-xyz")
	if len(controls) != 0 {
		t.Errorf("expected 0 controls for unknown category, got %d", len(controls))
	}
}

func TestMapFindingToControls_WeakAuth(t *testing.T) {
	controls := MapFindingToControls("weak-auth")
	if len(controls) == 0 {
		t.Fatal("expected controls for 'weak-auth'")
	}
	// Verify at least one access-control category
	hasAccessControl := false
	for _, c := range controls {
		if c.Category == "access-control" {
			hasAccessControl = true
			break
		}
	}
	if !hasAccessControl {
		t.Error("expected at least one control with category 'access-control'")
	}
}

// ---------------------------------------------------------------------------
// GenerateGapReport
// ---------------------------------------------------------------------------

func TestGenerateGapReport_SOC2_WithFindings(t *testing.T) {
	findings := []Finding{
		{RuleID: "hardcoded-secret", Severity: SeverityCritical, Metadata: map[string]string{"category": "hardcoded-secret"}},
		{RuleID: "no-encryption", Severity: SeverityHigh, Metadata: map[string]string{"category": "no-encryption"}},
	}

	report := GenerateGapReport(FrameworkSOC2, findings)
	if report.Framework != FrameworkSOC2 {
		t.Errorf("expected framework SOC2, got %q", report.Framework)
	}
	if report.TotalControls == 0 {
		t.Error("expected non-zero total controls")
	}
	if report.Score < 0 || report.Score > 100 {
		t.Errorf("expected score 0-100, got %f", report.Score)
	}
	if report.Covered == 0 {
		t.Error("expected some controls to be covered by findings")
	}
}

func TestGenerateGapReport_NoFindings(t *testing.T) {
	report := GenerateGapReport(FrameworkSOC2, nil)
	if report.Covered != 0 {
		t.Errorf("expected 0 covered controls with no findings, got %d", report.Covered)
	}
	if report.Score != 0 {
		t.Errorf("expected score 0 with no findings, got %f", report.Score)
	}
	if len(report.Gaps) != report.TotalControls {
		t.Errorf("all controls should be gaps when no findings, gaps=%d total=%d",
			len(report.Gaps), report.TotalControls)
	}
}

func TestGenerateGapReport_PCIDSS(t *testing.T) {
	findings := []Finding{
		{Metadata: map[string]string{"category": "no-encryption"}},
		{Metadata: map[string]string{"category": "no-logging"}},
		{Metadata: map[string]string{"category": "weak-auth"}},
	}
	report := GenerateGapReport(FrameworkPCIDSS, findings)
	if report.Framework != FrameworkPCIDSS {
		t.Errorf("expected PCI-DSS framework, got %q", report.Framework)
	}
	if report.TotalControls == 0 {
		t.Error("expected non-zero total controls for PCI-DSS")
	}
}

func TestGenerateGapReport_AllFrameworks(t *testing.T) {
	findings := []Finding{
		{Metadata: map[string]string{"category": "hardcoded-secret"}},
	}
	for _, fw := range ListFrameworks() {
		report := GenerateGapReport(fw, findings)
		if report.TotalControls == 0 {
			t.Errorf("framework %q has 0 total controls", fw)
		}
	}
}

// ---------------------------------------------------------------------------
// FormatMarkdown
// ---------------------------------------------------------------------------

func TestComplianceGapReport_FormatMarkdown(t *testing.T) {
	findings := []Finding{
		{Metadata: map[string]string{"category": "hardcoded-secret"}},
	}
	report := GenerateGapReport(FrameworkSOC2, findings)
	md := report.FormatMarkdown()

	if md == "" {
		t.Fatal("expected non-empty markdown output")
	}
	if !strings.Contains(md, "SOC2") {
		t.Error("expected markdown to contain framework name")
	}
	if !strings.Contains(md, "Score") {
		t.Error("expected markdown to contain 'Score'")
	}
}

func TestComplianceGapReport_FormatMarkdown_WithGaps(t *testing.T) {
	report := GenerateGapReport(FrameworkNIST, nil)
	md := report.FormatMarkdown()

	if !strings.Contains(md, "Gap") {
		t.Error("expected markdown to mention gaps when no findings provided")
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestGenerateGapReport_DuplicateFindings(t *testing.T) {
	findings := []Finding{
		{Metadata: map[string]string{"category": "hardcoded-secret"}},
		{Metadata: map[string]string{"category": "hardcoded-secret"}},
		{Metadata: map[string]string{"category": "hardcoded-secret"}},
	}
	report := GenerateGapReport(FrameworkSOC2, findings)
	// Duplicate findings should not double-count coverage
	singleReport := GenerateGapReport(FrameworkSOC2, findings[:1])
	if report.Covered != singleReport.Covered {
		t.Errorf("duplicate findings should not increase coverage: got %d vs %d",
			report.Covered, singleReport.Covered)
	}
}

package devsecops

import (
	"strings"
	"testing"
	"time"
)

func TestNewFindingsAggregator(t *testing.T) {
	agg := NewFindingsAggregator()
	if agg == nil {
		t.Fatal("NewFindingsAggregator returned nil")
	}
	if agg.OpenCount() != 0 {
		t.Errorf("expected 0 open findings, got %d", agg.OpenCount())
	}
}

func TestAddFinding(t *testing.T) {
	agg := NewFindingsAggregator()
	now := time.Now()

	agg.AddFinding(AggregatedFinding{
		ID:        "f1",
		CVEID:     "CVE-2024-1234",
		Title:     "SQL Injection",
		Severity:  "critical",
		Source:    "semgrep",
		FirstSeen: now,
		LastSeen:  now,
		Count:     1,
		Status:    "open",
		FilePath:  "app/db.go",
		Compliance: []string{"SOC2", "PCI-DSS"},
	})

	if agg.OpenCount() != 1 {
		t.Errorf("expected 1 open finding, got %d", agg.OpenCount())
	}
}

func TestAddFinding_Dedup(t *testing.T) {
	agg := NewFindingsAggregator()
	now := time.Now()

	f := AggregatedFinding{
		ID:        "f1",
		CVEID:     "CVE-2024-1234",
		Title:     "SQL Injection",
		Severity:  "critical",
		Source:    "semgrep",
		FirstSeen: now,
		LastSeen:  now,
		Count:     1,
		Status:    "open",
		FilePath:  "app/db.go",
	}

	agg.AddFinding(f)
	agg.AddFinding(f) // same CVEID+FilePath

	if agg.OpenCount() != 1 {
		t.Errorf("expected 1 after dedup, got %d", agg.OpenCount())
	}

	findings := agg.BySeverity("critical")
	if len(findings) != 1 {
		t.Fatalf("expected 1 critical finding, got %d", len(findings))
	}
	if findings[0].Count != 2 {
		t.Errorf("expected count 2 after dedup, got %d", findings[0].Count)
	}
}

func TestAddFinding_DifferentFiles(t *testing.T) {
	agg := NewFindingsAggregator()
	now := time.Now()

	agg.AddFinding(AggregatedFinding{
		CVEID: "CVE-2024-1234", FilePath: "a.go", Severity: "high",
		Status: "open", FirstSeen: now, LastSeen: now, Count: 1,
	})
	agg.AddFinding(AggregatedFinding{
		CVEID: "CVE-2024-1234", FilePath: "b.go", Severity: "high",
		Status: "open", FirstSeen: now, LastSeen: now, Count: 1,
	})

	if agg.OpenCount() != 2 {
		t.Errorf("expected 2 open findings for different files, got %d", agg.OpenCount())
	}
}

func TestResolve(t *testing.T) {
	agg := NewFindingsAggregator()
	now := time.Now()

	agg.AddFinding(AggregatedFinding{
		ID: "f1", CVEID: "CVE-2024-1234", FilePath: "a.go",
		Severity: "high", Status: "open", FirstSeen: now, LastSeen: now, Count: 1,
	})

	agg.Resolve("f1")
	if agg.OpenCount() != 0 {
		t.Errorf("expected 0 open after resolve, got %d", agg.OpenCount())
	}
}

func TestMarkFalsePositive(t *testing.T) {
	agg := NewFindingsAggregator()
	now := time.Now()

	agg.AddFinding(AggregatedFinding{
		ID: "f1", CVEID: "CVE-2024-1234", FilePath: "a.go",
		Severity: "medium", Status: "open", FirstSeen: now, LastSeen: now, Count: 1,
	})

	agg.MarkFalsePositive("f1")
	if agg.OpenCount() != 0 {
		t.Errorf("expected 0 open after false positive, got %d", agg.OpenCount())
	}
}

func TestBySeverity(t *testing.T) {
	agg := NewFindingsAggregator()
	now := time.Now()

	agg.AddFinding(AggregatedFinding{
		ID: "f1", CVEID: "CVE-2024-1", FilePath: "a.go",
		Severity: "critical", Status: "open", FirstSeen: now, LastSeen: now, Count: 1,
	})
	agg.AddFinding(AggregatedFinding{
		ID: "f2", CVEID: "CVE-2024-2", FilePath: "b.go",
		Severity: "high", Status: "open", FirstSeen: now, LastSeen: now, Count: 1,
	})
	agg.AddFinding(AggregatedFinding{
		ID: "f3", CVEID: "CVE-2024-3", FilePath: "c.go",
		Severity: "critical", Status: "open", FirstSeen: now, LastSeen: now, Count: 1,
	})

	criticals := agg.BySeverity("critical")
	if len(criticals) != 2 {
		t.Errorf("expected 2 critical findings, got %d", len(criticals))
	}

	lows := agg.BySeverity("low")
	if len(lows) != 0 {
		t.Errorf("expected 0 low findings, got %d", len(lows))
	}
}

func TestSummary(t *testing.T) {
	agg := NewFindingsAggregator()
	now := time.Now()

	agg.AddFinding(AggregatedFinding{
		ID: "f1", CVEID: "CVE-2024-1", FilePath: "a.go",
		Severity: "critical", Status: "open", FirstSeen: now, LastSeen: now, Count: 1,
	})
	agg.AddFinding(AggregatedFinding{
		ID: "f2", CVEID: "CVE-2024-2", FilePath: "b.go",
		Severity: "high", Status: "open", FirstSeen: now, LastSeen: now, Count: 1,
	})
	agg.AddFinding(AggregatedFinding{
		ID: "f3", CVEID: "CVE-2024-3", FilePath: "c.go",
		Severity: "high", Status: "open", FirstSeen: now, LastSeen: now, Count: 1,
	})

	summary := agg.Summary()
	if summary["critical"] != 1 {
		t.Errorf("expected 1 critical, got %d", summary["critical"])
	}
	if summary["high"] != 2 {
		t.Errorf("expected 2 high, got %d", summary["high"])
	}
}

func TestFormatMarkdown(t *testing.T) {
	agg := NewFindingsAggregator()
	now := time.Now()

	agg.AddFinding(AggregatedFinding{
		ID: "f1", CVEID: "CVE-2024-1234", Title: "SQL Injection",
		FilePath: "app/db.go", LineNumber: 42, Severity: "critical",
		Source: "semgrep", Status: "open", FirstSeen: now, LastSeen: now, Count: 1,
	})

	md := agg.FormatMarkdown()
	if !strings.Contains(md, "CVE-2024-1234") {
		t.Error("markdown should contain CVE ID")
	}
	if !strings.Contains(md, "SQL Injection") {
		t.Error("markdown should contain title")
	}
	if !strings.Contains(md, "critical") {
		t.Error("markdown should contain severity")
	}
}

func TestMapToCompliance(t *testing.T) {
	agg := NewFindingsAggregator()
	now := time.Now()

	agg.AddFinding(AggregatedFinding{
		ID: "f1", CVEID: "CVE-2024-1", FilePath: "a.go",
		Severity: "high", Status: "open", FirstSeen: now, LastSeen: now, Count: 1,
		Compliance: []string{"SOC2", "PCI-DSS"},
	})
	agg.AddFinding(AggregatedFinding{
		ID: "f2", CVEID: "CVE-2024-2", FilePath: "b.go",
		Severity: "medium", Status: "open", FirstSeen: now, LastSeen: now, Count: 1,
		Compliance: []string{"ISO27001"},
	})

	soc2 := agg.MapToCompliance("SOC2")
	if len(soc2) != 1 {
		t.Errorf("expected 1 SOC2 finding, got %d", len(soc2))
	}

	iso := agg.MapToCompliance("ISO27001")
	if len(iso) != 1 {
		t.Errorf("expected 1 ISO27001 finding, got %d", len(iso))
	}

	none := agg.MapToCompliance("HIPAA")
	if len(none) != 0 {
		t.Errorf("expected 0 HIPAA findings, got %d", len(none))
	}
}

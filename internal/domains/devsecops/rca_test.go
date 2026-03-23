package devsecops

import (
	"strings"
	"testing"
)

func TestNewRCAReport(t *testing.T) {
	r := NewRCAReport("DB Outage", "SEV1")
	if r.Title != "DB Outage" {
		t.Errorf("expected title 'DB Outage', got %q", r.Title)
	}
	if r.Severity != "SEV1" {
		t.Errorf("expected severity SEV1, got %q", r.Severity)
	}
	if r.Status != "investigating" {
		t.Errorf("expected initial status 'investigating', got %q", r.Status)
	}
}

func TestAddTimelineEntry(t *testing.T) {
	r := NewRCAReport("Test", "SEV2")
	r.AddTimelineEntry("10:00", "Alert fired", "PagerDuty", "degraded")
	r.AddTimelineEntry("10:15", "Team assembled", "oncall", "degraded")

	if len(r.Timeline) != 2 {
		t.Fatalf("expected 2 timeline entries, got %d", len(r.Timeline))
	}
	if r.Timeline[0].Event != "Alert fired" {
		t.Errorf("unexpected event: %q", r.Timeline[0].Event)
	}
	if r.Timeline[1].Actor != "oncall" {
		t.Errorf("unexpected actor: %q", r.Timeline[1].Actor)
	}
}

func TestAddFiveWhy(t *testing.T) {
	r := NewRCAReport("Test", "SEV2")
	r.AddFiveWhy("Why did the service go down?", "Database connection pool exhausted")
	r.AddFiveWhy("Why was the pool exhausted?", "Queries were not closing connections")

	if len(r.FiveWhys) != 2 {
		t.Fatalf("expected 2 five-why steps, got %d", len(r.FiveWhys))
	}
	if r.FiveWhys[0].Why != "Why did the service go down?" {
		t.Error("unexpected first why question")
	}
}

func TestAddActionItem(t *testing.T) {
	r := NewRCAReport("Test", "SEV2")
	r.AddActionItem("Add connection pool monitoring", "eng-team", "P0", "2026-04-01", "detect")
	r.AddActionItem("Implement connection timeout", "eng-team", "P1", "2026-04-15", "prevent")

	if len(r.ActionItems) != 2 {
		t.Fatalf("expected 2 action items, got %d", len(r.ActionItems))
	}
	if r.ActionItems[0].Priority != "P0" {
		t.Errorf("unexpected priority: %q", r.ActionItems[0].Priority)
	}
	if r.ActionItems[0].ID == "" {
		t.Error("expected auto-generated ID")
	}
	if r.ActionItems[1].Type != "prevent" {
		t.Errorf("unexpected type: %q", r.ActionItems[1].Type)
	}
}

func TestAddLesson(t *testing.T) {
	r := NewRCAReport("Test", "SEV2")
	r.AddLesson("Always monitor connection pools")
	if len(r.Lessons) != 1 {
		t.Fatal("expected 1 lesson")
	}
}

func TestSetResolved(t *testing.T) {
	r := NewRCAReport("Test", "SEV2")
	r.SetResolved("2026-03-23", "2h30m", "Connection pool leak")

	if r.Status != "resolved" {
		t.Errorf("expected status 'resolved', got %q", r.Status)
	}
	if r.RootCause != "Connection pool leak" {
		t.Errorf("unexpected root cause: %q", r.RootCause)
	}
	if r.Duration != "2h30m" {
		t.Errorf("unexpected duration: %q", r.Duration)
	}
}

func TestRCAFormatMarkdown(t *testing.T) {
	r := NewRCAReport("Database Outage", "SEV1")
	r.AddTimelineEntry("10:00", "Alert fired", "PagerDuty", "outage")
	r.AddFiveWhy("Why did requests fail?", "Database was unreachable")
	r.AddActionItem("Add failover", "infra", "P0", "2026-04-01", "prevent")
	r.AddLesson("Need automated failover")
	r.SetResolved("2026-03-23", "45m", "Single point of failure in DB")

	md := r.FormatMarkdown()

	required := []string{
		"Database Outage",
		"SEV1",
		"Timeline",
		"Five Whys",
		"Root Cause",
		"Action Items",
		"Lessons Learned",
		"Alert fired",
		"Single point of failure",
	}
	for _, s := range required {
		if !strings.Contains(md, s) {
			t.Errorf("markdown missing %q", s)
		}
	}
}

func TestRCAFormatExecutiveSummary(t *testing.T) {
	r := NewRCAReport("API Latency Spike", "SEV2")
	r.SetResolved("2026-03-23", "1h", "Cache eviction storm")
	r.AddActionItem("Tune cache TTL", "platform", "P1", "2026-04-01", "prevent")

	summary := r.FormatExecutiveSummary()
	if !strings.Contains(summary, "API Latency Spike") {
		t.Error("executive summary missing title")
	}
	if !strings.Contains(summary, "SEV2") {
		t.Error("executive summary missing severity")
	}
}

func TestRCATemplate(t *testing.T) {
	for _, sev := range []string{"SEV1", "SEV2", "SEV3", "SEV4"} {
		r := RCATemplate(sev)
		if r.Severity != sev {
			t.Errorf("template severity mismatch: expected %q, got %q", sev, r.Severity)
		}
		if r.Title == "" {
			t.Errorf("template for %s has empty title", sev)
		}
	}
}

func TestSampleRCA(t *testing.T) {
	r := SampleRCA()
	if r.Title == "" {
		t.Error("sample RCA has empty title")
	}
	if len(r.Timeline) == 0 {
		t.Error("sample RCA has no timeline entries")
	}
	if len(r.FiveWhys) == 0 {
		t.Error("sample RCA has no five-why steps")
	}
	if len(r.ActionItems) == 0 {
		t.Error("sample RCA has no action items")
	}
	if r.RootCause == "" {
		t.Error("sample RCA has no root cause")
	}
}

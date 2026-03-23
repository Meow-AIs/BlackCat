package devsecops

import (
	"fmt"
	"strings"
)

// RCAReport represents a Root Cause Analysis postmortem report.
type RCAReport struct {
	Title         string
	Severity      string // "SEV1", "SEV2", "SEV3", "SEV4"
	Status        string // "investigating", "identified", "resolved", "monitoring"
	IncidentDate  string
	ResolvedDate  string
	Duration      string
	ImpactSummary string
	Timeline      []TimelineEntry
	FiveWhys      []FiveWhyStep
	RootCause     string
	Contributing  []string
	ActionItems   []ActionItem
	Lessons       []string
	Participants  []string
}

// TimelineEntry records a single event during an incident.
type TimelineEntry struct {
	Time   string
	Event  string
	Actor  string // who did what
	Impact string // "none", "degraded", "outage"
}

// FiveWhyStep represents one iteration of the Five Whys technique.
type FiveWhyStep struct {
	Why    string
	Answer string
}

// ActionItem tracks a follow-up task from the RCA.
type ActionItem struct {
	ID          string
	Description string
	Owner       string
	Priority    string // "P0", "P1", "P2"
	DueDate     string
	Status      string // "open", "in_progress", "done"
	Type        string // "prevent", "detect", "mitigate"
}

// NewRCAReport creates a new RCA report in investigating status.
func NewRCAReport(title, severity string) *RCAReport {
	return &RCAReport{
		Title:    title,
		Severity: severity,
		Status:   "investigating",
	}
}

// AddTimelineEntry appends a timeline event.
func (r *RCAReport) AddTimelineEntry(time, event, actor, impact string) {
	r.Timeline = append(r.Timeline, TimelineEntry{
		Time:   time,
		Event:  event,
		Actor:  actor,
		Impact: impact,
	})
}

// AddFiveWhy appends a Five Whys iteration.
func (r *RCAReport) AddFiveWhy(why, answer string) {
	r.FiveWhys = append(r.FiveWhys, FiveWhyStep{
		Why:    why,
		Answer: answer,
	})
}

// AddActionItem appends a follow-up action with an auto-generated ID.
func (r *RCAReport) AddActionItem(desc, owner, priority, dueDate, actionType string) {
	id := fmt.Sprintf("AI-%d", len(r.ActionItems)+1)
	r.ActionItems = append(r.ActionItems, ActionItem{
		ID:          id,
		Description: desc,
		Owner:       owner,
		Priority:    priority,
		DueDate:     dueDate,
		Status:      "open",
		Type:        actionType,
	})
}

// AddLesson appends a lessons-learned entry.
func (r *RCAReport) AddLesson(lesson string) {
	r.Lessons = append(r.Lessons, lesson)
}

// SetResolved marks the incident as resolved with root cause.
func (r *RCAReport) SetResolved(date, duration, rootCause string) {
	r.Status = "resolved"
	r.ResolvedDate = date
	r.Duration = duration
	r.RootCause = rootCause
}

// FormatMarkdown returns a full postmortem in Markdown format.
func (r *RCAReport) FormatMarkdown() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# RCA: %s\n\n", r.Title))
	b.WriteString(fmt.Sprintf("**Severity:** %s | **Status:** %s\n\n", r.Severity, r.Status))

	if r.IncidentDate != "" {
		b.WriteString(fmt.Sprintf("**Incident Date:** %s\n", r.IncidentDate))
	}
	if r.ResolvedDate != "" {
		b.WriteString(fmt.Sprintf("**Resolved Date:** %s\n", r.ResolvedDate))
	}
	if r.Duration != "" {
		b.WriteString(fmt.Sprintf("**Duration:** %s\n", r.Duration))
	}
	if r.ImpactSummary != "" {
		b.WriteString(fmt.Sprintf("\n## Impact Summary\n\n%s\n", r.ImpactSummary))
	}
	b.WriteString("\n")

	// Timeline
	b.WriteString("## Timeline\n\n")
	if len(r.Timeline) == 0 {
		b.WriteString("_No timeline entries._\n")
	} else {
		b.WriteString("| Time | Event | Actor | Impact |\n|------|-------|-------|--------|\n")
		for _, e := range r.Timeline {
			b.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", e.Time, e.Event, e.Actor, e.Impact))
		}
	}
	b.WriteString("\n")

	// Five Whys
	b.WriteString("## Five Whys\n\n")
	for i, fw := range r.FiveWhys {
		b.WriteString(fmt.Sprintf("%d. **%s**\n   %s\n", i+1, fw.Why, fw.Answer))
	}
	b.WriteString("\n")

	// Root Cause
	b.WriteString("## Root Cause\n\n")
	if r.RootCause != "" {
		b.WriteString(r.RootCause + "\n")
	} else {
		b.WriteString("_Under investigation._\n")
	}
	b.WriteString("\n")

	if len(r.Contributing) > 0 {
		b.WriteString("### Contributing Factors\n\n")
		for _, c := range r.Contributing {
			b.WriteString(fmt.Sprintf("- %s\n", c))
		}
		b.WriteString("\n")
	}

	// Action Items
	b.WriteString("## Action Items\n\n")
	if len(r.ActionItems) > 0 {
		b.WriteString("| ID | Description | Owner | Priority | Due | Type | Status |\n")
		b.WriteString("|-----|-------------|-------|----------|-----|------|--------|\n")
		for _, a := range r.ActionItems {
			b.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s |\n",
				a.ID, a.Description, a.Owner, a.Priority, a.DueDate, a.Type, a.Status))
		}
	}
	b.WriteString("\n")

	// Lessons Learned
	b.WriteString("## Lessons Learned\n\n")
	for _, l := range r.Lessons {
		b.WriteString(fmt.Sprintf("- %s\n", l))
	}
	b.WriteString("\n")

	// Participants
	if len(r.Participants) > 0 {
		b.WriteString("## Participants\n\n")
		for _, p := range r.Participants {
			b.WriteString(fmt.Sprintf("- %s\n", p))
		}
	}

	return b.String()
}

// FormatExecutiveSummary returns a concise one-page summary.
func (r *RCAReport) FormatExecutiveSummary() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# Executive Summary: %s\n\n", r.Title))
	b.WriteString(fmt.Sprintf("**Severity:** %s | **Status:** %s | **Duration:** %s\n\n", r.Severity, r.Status, r.Duration))

	if r.RootCause != "" {
		b.WriteString(fmt.Sprintf("**Root Cause:** %s\n\n", r.RootCause))
	}

	if len(r.ActionItems) > 0 {
		p0Count := 0
		for _, a := range r.ActionItems {
			if a.Priority == "P0" {
				p0Count++
			}
		}
		b.WriteString(fmt.Sprintf("**Action Items:** %d total (%d P0)\n", len(r.ActionItems), p0Count))
	}

	return b.String()
}

// RCATemplate returns a pre-filled template based on severity level.
func RCATemplate(severity string) *RCAReport {
	titles := map[string]string{
		"SEV1": "[SEV1] Critical Service Outage",
		"SEV2": "[SEV2] Major Service Degradation",
		"SEV3": "[SEV3] Minor Service Impact",
		"SEV4": "[SEV4] Low-Impact Issue",
	}
	title, ok := titles[severity]
	if !ok {
		title = "[" + severity + "] Incident Report"
	}
	return NewRCAReport(title, severity)
}

// SampleRCA returns a fully filled-out sample RCA for reference.
func SampleRCA() *RCAReport {
	r := NewRCAReport("Production Database Outage - Connection Pool Exhaustion", "SEV1")
	r.IncidentDate = "2026-03-20"
	r.ImpactSummary = "All API requests returned 500 errors for 45 minutes. Approximately 12,000 users affected."
	r.Participants = []string{"oncall-engineer", "db-team-lead", "platform-manager"}

	r.AddTimelineEntry("14:00", "Monitoring alert: API error rate > 50%", "PagerDuty", "degraded")
	r.AddTimelineEntry("14:05", "Oncall engineer acknowledges alert", "oncall-engineer", "degraded")
	r.AddTimelineEntry("14:10", "Identified DB connection pool exhaustion", "oncall-engineer", "outage")
	r.AddTimelineEntry("14:25", "Restarted application pods", "oncall-engineer", "degraded")
	r.AddTimelineEntry("14:45", "Service fully recovered", "oncall-engineer", "none")

	r.AddFiveWhy("Why did API requests fail?", "Database connections timed out")
	r.AddFiveWhy("Why did connections time out?", "Connection pool was exhausted")
	r.AddFiveWhy("Why was the pool exhausted?", "Long-running queries held connections open")
	r.AddFiveWhy("Why were queries long-running?", "Missing index on frequently-queried column")
	r.AddFiveWhy("Why was the index missing?", "No query performance review in deployment process")

	r.AddActionItem("Add missing database index", "db-team", "P0", "2026-03-21", "prevent")
	r.AddActionItem("Add connection pool monitoring alarm", "platform", "P0", "2026-03-22", "detect")
	r.AddActionItem("Implement query timeout at application level", "backend", "P1", "2026-03-28", "mitigate")

	r.AddLesson("Database schema changes need query performance review")
	r.AddLesson("Connection pool metrics should be in standard dashboards")

	r.SetResolved("2026-03-20", "45m", "Missing database index caused long-running queries that exhausted the connection pool")

	return r
}

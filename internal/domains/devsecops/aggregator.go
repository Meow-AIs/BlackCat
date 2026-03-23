package devsecops

import (
	"fmt"
	"strings"
	"time"
)

// AggregatedFinding represents a deduplicated security finding with tracking metadata.
type AggregatedFinding struct {
	ID         string    `json:"id"`
	CVEID      string    `json:"cve_id"`
	Title      string    `json:"title"`
	Severity   string    `json:"severity"` // critical, high, medium, low, info
	Source     string    `json:"source"`   // scanner name
	FirstSeen  time.Time `json:"first_seen"`
	LastSeen   time.Time `json:"last_seen"`
	Count      int       `json:"count"` // how many times detected
	Status     string    `json:"status"` // open, resolved, false_positive, accepted
	FilePath   string    `json:"file_path"`
	LineNumber int       `json:"line_number"`
	Compliance []string  `json:"compliance"` // SOC2, ISO27001, PCI-DSS controls
}

// FindingsAggregator deduplicates and manages security findings.
type FindingsAggregator struct {
	findings map[string]*AggregatedFinding // dedup key -> finding
}

// NewFindingsAggregator creates a new empty findings aggregator.
func NewFindingsAggregator() *FindingsAggregator {
	return &FindingsAggregator{
		findings: make(map[string]*AggregatedFinding),
	}
}

// dedupKey generates a deduplication key from CVEID and FilePath.
func dedupKey(f AggregatedFinding) string {
	return f.CVEID + "|" + f.FilePath
}

// AddFinding adds a finding, deduplicating by CVEID+FilePath.
// If a duplicate exists, it increments Count and updates LastSeen.
func (a *FindingsAggregator) AddFinding(f AggregatedFinding) {
	key := dedupKey(f)
	if existing, ok := a.findings[key]; ok {
		updated := *existing
		updated.Count++
		if f.LastSeen.After(updated.LastSeen) {
			updated.LastSeen = f.LastSeen
		}
		a.findings[key] = &updated
		return
	}
	copy := f
	a.findings[key] = &copy
}

// Resolve marks a finding as resolved by its ID.
func (a *FindingsAggregator) Resolve(id string) {
	for key, f := range a.findings {
		if f.ID == id {
			updated := *f
			updated.Status = "resolved"
			a.findings[key] = &updated
			return
		}
	}
}

// MarkFalsePositive marks a finding as a false positive by its ID.
func (a *FindingsAggregator) MarkFalsePositive(id string) {
	for key, f := range a.findings {
		if f.ID == id {
			updated := *f
			updated.Status = "false_positive"
			a.findings[key] = &updated
			return
		}
	}
}

// BySeverity returns all findings matching the given severity level.
func (a *FindingsAggregator) BySeverity(severity string) []AggregatedFinding {
	var result []AggregatedFinding
	for _, f := range a.findings {
		if f.Severity == severity {
			result = append(result, *f)
		}
	}
	return result
}

// OpenCount returns the number of findings with "open" status.
func (a *FindingsAggregator) OpenCount() int {
	count := 0
	for _, f := range a.findings {
		if f.Status == "open" {
			count++
		}
	}
	return count
}

// Summary returns a map of severity to count of open findings.
func (a *FindingsAggregator) Summary() map[string]int {
	summary := make(map[string]int)
	for _, f := range a.findings {
		if f.Status == "open" {
			summary[f.Severity]++
		}
	}
	return summary
}

// FormatMarkdown renders all findings as a Markdown report.
func (a *FindingsAggregator) FormatMarkdown() string {
	var b strings.Builder
	b.WriteString("# Security Findings Report\n\n")

	summary := a.Summary()
	b.WriteString("## Summary\n\n")
	b.WriteString(fmt.Sprintf("| Severity | Count |\n"))
	b.WriteString("| --- | --- |\n")
	for _, sev := range []string{"critical", "high", "medium", "low", "info"} {
		if c, ok := summary[sev]; ok {
			b.WriteString(fmt.Sprintf("| %s | %d |\n", sev, c))
		}
	}
	b.WriteString("\n")

	b.WriteString("## Findings\n\n")
	for _, f := range a.findings {
		b.WriteString(fmt.Sprintf("### %s\n\n", f.Title))
		b.WriteString(fmt.Sprintf("- **CVE**: %s\n", f.CVEID))
		b.WriteString(fmt.Sprintf("- **Severity**: %s\n", f.Severity))
		b.WriteString(fmt.Sprintf("- **Source**: %s\n", f.Source))
		b.WriteString(fmt.Sprintf("- **Status**: %s\n", f.Status))
		if f.FilePath != "" {
			b.WriteString(fmt.Sprintf("- **File**: %s", f.FilePath))
			if f.LineNumber > 0 {
				b.WriteString(fmt.Sprintf(":%d", f.LineNumber))
			}
			b.WriteString("\n")
		}
		b.WriteString(fmt.Sprintf("- **Count**: %d\n", f.Count))
		if len(f.Compliance) > 0 {
			b.WriteString(fmt.Sprintf("- **Compliance**: %s\n", strings.Join(f.Compliance, ", ")))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// MapToCompliance returns all findings that match the given compliance framework tag.
func (a *FindingsAggregator) MapToCompliance(framework string) []AggregatedFinding {
	var result []AggregatedFinding
	for _, f := range a.findings {
		for _, c := range f.Compliance {
			if c == framework {
				result = append(result, *f)
				break
			}
		}
	}
	return result
}

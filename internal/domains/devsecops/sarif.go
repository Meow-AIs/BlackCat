package devsecops

import (
	"encoding/json"
	"fmt"
)

// SARIF 2.1.0 structures (simplified).

// SARIFReport is the top-level SARIF document.
type SARIFReport struct {
	Schema  string     `json:"$schema,omitempty"`
	Version string     `json:"version"`
	Runs    []SARIFRun `json:"runs"`
}

// SARIFRun represents a single analysis run.
type SARIFRun struct {
	Tool    SARIFTool     `json:"tool"`
	Results []SARIFResult `json:"results"`
}

// SARIFTool identifies the analysis tool.
type SARIFTool struct {
	Driver SARIFDriver `json:"driver"`
}

// SARIFDriver provides tool metadata.
type SARIFDriver struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// SARIFResult represents a single finding in SARIF format.
type SARIFResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"` // error, warning, note
	Message   SARIFMessage    `json:"message"`
	Locations []SARIFLocation `json:"locations,omitempty"`
}

// SARIFMessage holds the finding description.
type SARIFMessage struct {
	Text string `json:"text"`
}

// SARIFLocation points to where the finding occurs.
type SARIFLocation struct {
	PhysicalLocation SARIFPhysicalLocation `json:"physicalLocation"`
}

// SARIFPhysicalLocation specifies file and position.
type SARIFPhysicalLocation struct {
	ArtifactLocation SARIFArtifactLocation `json:"artifactLocation"`
	Region           *SARIFRegion          `json:"region,omitempty"`
}

// SARIFArtifactLocation identifies the file.
type SARIFArtifactLocation struct {
	URI string `json:"uri"`
}

// SARIFRegion identifies the position within a file.
type SARIFRegion struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn,omitempty"`
}

// ParseSARIF deserializes a SARIF 2.1.0 JSON document.
func ParseSARIF(data []byte) (*SARIFReport, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("sarif: empty input")
	}
	var report SARIFReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("sarif: %w", err)
	}
	return &report, nil
}

// TotalResults returns the count of results across all runs.
func (r *SARIFReport) TotalResults() int {
	total := 0
	for _, run := range r.Runs {
		total += len(run.Results)
	}
	return total
}

// ByLevel filters results across all runs by level (error, warning, note).
func (r *SARIFReport) ByLevel(level string) []SARIFResult {
	var matched []SARIFResult
	for _, run := range r.Runs {
		for _, result := range run.Results {
			if result.Level == level {
				matched = append(matched, result)
			}
		}
	}
	return matched
}

// SARIFToFindings converts a SARIF report to internal Finding objects.
func SARIFToFindings(report *SARIFReport) []Finding {
	var findings []Finding
	for _, run := range report.Runs {
		scannerName := run.Tool.Driver.Name
		for _, result := range run.Results {
			f := Finding{
				ID:          fmt.Sprintf("sarif:%s:%s", scannerName, result.RuleID),
				Scanner:     scannerName,
				Severity:    sarifLevelToSeverity(result.Level),
				Title:       result.Message.Text,
				Description: result.Message.Text,
				RuleID:      result.RuleID,
				Confidence:  0.8,
			}

			if len(result.Locations) > 0 {
				loc := result.Locations[0].PhysicalLocation
				f.FilePath = loc.ArtifactLocation.URI
				if loc.Region != nil {
					f.Line = loc.Region.StartLine
				}
			}

			findings = append(findings, f)
		}
	}
	return findings
}

// sarifLevelToSeverity maps SARIF level strings to internal Severity.
func sarifLevelToSeverity(level string) Severity {
	switch level {
	case "error":
		return SeverityCritical
	case "warning":
		return SeverityMedium
	case "note":
		return SeverityLow
	default:
		return SeverityInfo
	}
}

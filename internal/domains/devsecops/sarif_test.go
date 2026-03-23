package devsecops

import (
	"encoding/json"
	"testing"
)

// ---------------------------------------------------------------------------
// Sample SARIF JSON fixtures
// ---------------------------------------------------------------------------

const sampleSemgrepSARIF = `{
  "$schema": "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
  "version": "2.1.0",
  "runs": [
    {
      "tool": {
        "driver": {
          "name": "semgrep",
          "version": "1.50.0"
        }
      },
      "results": [
        {
          "ruleId": "python.lang.security.audit.eval-detected",
          "level": "error",
          "message": { "text": "Detected use of eval(). This can allow arbitrary code execution." },
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": { "uri": "src/app.py" },
                "region": { "startLine": 42, "startColumn": 5 }
              }
            }
          ]
        },
        {
          "ruleId": "python.lang.security.audit.exec-detected",
          "level": "warning",
          "message": { "text": "Detected use of exec()." },
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": { "uri": "src/utils.py" },
                "region": { "startLine": 10 }
              }
            }
          ]
        }
      ]
    }
  ]
}`

const sampleTrivySARIF = `{
  "$schema": "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
  "version": "2.1.0",
  "runs": [
    {
      "tool": {
        "driver": {
          "name": "trivy",
          "version": "0.48.0"
        }
      },
      "results": [
        {
          "ruleId": "CVE-2023-44487",
          "level": "error",
          "message": { "text": "HTTP/2 rapid reset vulnerability in golang.org/x/net" },
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": { "uri": "go.sum" },
                "region": { "startLine": 1 }
              }
            }
          ]
        },
        {
          "ruleId": "CVE-2023-39325",
          "level": "error",
          "message": { "text": "Resource consumption in golang.org/x/net" },
          "locations": []
        },
        {
          "ruleId": "CVE-2023-12345",
          "level": "note",
          "message": { "text": "Low severity info disclosure" }
        }
      ]
    }
  ]
}`

const emptySARIF = `{
  "$schema": "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
  "version": "2.1.0",
  "runs": [
    {
      "tool": { "driver": { "name": "semgrep", "version": "1.50.0" } },
      "results": []
    }
  ]
}`

const multiRunSARIF = `{
  "version": "2.1.0",
  "runs": [
    {
      "tool": { "driver": { "name": "scanner-a", "version": "1.0" } },
      "results": [
        { "ruleId": "rule-a1", "level": "error", "message": { "text": "issue a1" } }
      ]
    },
    {
      "tool": { "driver": { "name": "scanner-b", "version": "2.0" } },
      "results": [
        { "ruleId": "rule-b1", "level": "warning", "message": { "text": "issue b1" } },
        { "ruleId": "rule-b2", "level": "note", "message": { "text": "issue b2" } }
      ]
    }
  ]
}`

// ---------------------------------------------------------------------------
// ParseSARIF
// ---------------------------------------------------------------------------

func TestParseSARIF_SemgrepReport(t *testing.T) {
	report, err := ParseSARIF([]byte(sampleSemgrepSARIF))
	if err != nil {
		t.Fatalf("ParseSARIF: %v", err)
	}
	if report.Version != "2.1.0" {
		t.Errorf("expected version 2.1.0, got %q", report.Version)
	}
	if len(report.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(report.Runs))
	}
	if report.Runs[0].Tool.Driver.Name != "semgrep" {
		t.Errorf("expected tool name 'semgrep', got %q", report.Runs[0].Tool.Driver.Name)
	}
	if len(report.Runs[0].Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(report.Runs[0].Results))
	}
}

func TestParseSARIF_TrivyReport(t *testing.T) {
	report, err := ParseSARIF([]byte(sampleTrivySARIF))
	if err != nil {
		t.Fatalf("ParseSARIF: %v", err)
	}
	if len(report.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(report.Runs))
	}
	if report.Runs[0].Tool.Driver.Name != "trivy" {
		t.Errorf("expected tool name 'trivy', got %q", report.Runs[0].Tool.Driver.Name)
	}
	if len(report.Runs[0].Results) != 3 {
		t.Errorf("expected 3 results, got %d", len(report.Runs[0].Results))
	}
}

func TestParseSARIF_EmptyResults(t *testing.T) {
	report, err := ParseSARIF([]byte(emptySARIF))
	if err != nil {
		t.Fatalf("ParseSARIF: %v", err)
	}
	if report.TotalResults() != 0 {
		t.Errorf("expected 0 total results, got %d", report.TotalResults())
	}
}

func TestParseSARIF_InvalidJSON(t *testing.T) {
	_, err := ParseSARIF([]byte(`{invalid json`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseSARIF_EmptyInput(t *testing.T) {
	_, err := ParseSARIF([]byte{})
	if err == nil {
		t.Error("expected error for empty input")
	}
}

// ---------------------------------------------------------------------------
// TotalResults
// ---------------------------------------------------------------------------

func TestSARIFReport_TotalResults_MultiRun(t *testing.T) {
	report, err := ParseSARIF([]byte(multiRunSARIF))
	if err != nil {
		t.Fatalf("ParseSARIF: %v", err)
	}
	if report.TotalResults() != 3 {
		t.Errorf("expected 3 total results across runs, got %d", report.TotalResults())
	}
}

func TestSARIFReport_TotalResults_SingleRun(t *testing.T) {
	report, err := ParseSARIF([]byte(sampleSemgrepSARIF))
	if err != nil {
		t.Fatalf("ParseSARIF: %v", err)
	}
	if report.TotalResults() != 2 {
		t.Errorf("expected 2 total results, got %d", report.TotalResults())
	}
}

// ---------------------------------------------------------------------------
// ByLevel
// ---------------------------------------------------------------------------

func TestSARIFReport_ByLevel_Error(t *testing.T) {
	report, err := ParseSARIF([]byte(sampleTrivySARIF))
	if err != nil {
		t.Fatalf("ParseSARIF: %v", err)
	}
	errors := report.ByLevel("error")
	if len(errors) != 2 {
		t.Errorf("expected 2 error-level results, got %d", len(errors))
	}
}

func TestSARIFReport_ByLevel_Note(t *testing.T) {
	report, err := ParseSARIF([]byte(sampleTrivySARIF))
	if err != nil {
		t.Fatalf("ParseSARIF: %v", err)
	}
	notes := report.ByLevel("note")
	if len(notes) != 1 {
		t.Errorf("expected 1 note-level result, got %d", len(notes))
	}
}

func TestSARIFReport_ByLevel_NonExistent(t *testing.T) {
	report, err := ParseSARIF([]byte(sampleSemgrepSARIF))
	if err != nil {
		t.Fatalf("ParseSARIF: %v", err)
	}
	none := report.ByLevel("note")
	if len(none) != 0 {
		t.Errorf("expected 0 results for 'note' level, got %d", len(none))
	}
}

// ---------------------------------------------------------------------------
// SARIFToFindings
// ---------------------------------------------------------------------------

func TestSARIFToFindings_SemgrepConversion(t *testing.T) {
	report, err := ParseSARIF([]byte(sampleSemgrepSARIF))
	if err != nil {
		t.Fatalf("ParseSARIF: %v", err)
	}

	findings := SARIFToFindings(report)
	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(findings))
	}

	f := findings[0]
	if f.RuleID != "python.lang.security.audit.eval-detected" {
		t.Errorf("expected ruleId 'python.lang.security.audit.eval-detected', got %q", f.RuleID)
	}
	if f.Severity != SeverityCritical {
		t.Errorf("expected critical severity for error level, got %q", f.Severity)
	}
	if f.FilePath != "src/app.py" {
		t.Errorf("expected file path 'src/app.py', got %q", f.FilePath)
	}
	if f.Line != 42 {
		t.Errorf("expected line 42, got %d", f.Line)
	}
	if f.Scanner != "semgrep" {
		t.Errorf("expected scanner 'semgrep', got %q", f.Scanner)
	}
}

func TestSARIFToFindings_TrivyConversion(t *testing.T) {
	report, err := ParseSARIF([]byte(sampleTrivySARIF))
	if err != nil {
		t.Fatalf("ParseSARIF: %v", err)
	}

	findings := SARIFToFindings(report)
	if len(findings) != 3 {
		t.Fatalf("expected 3 findings, got %d", len(findings))
	}

	// Check severity mapping for "note" level
	noteFinding := findings[2]
	if noteFinding.Severity != SeverityLow {
		t.Errorf("expected low severity for note level, got %q", noteFinding.Severity)
	}
}

func TestSARIFToFindings_EmptyReport(t *testing.T) {
	report, err := ParseSARIF([]byte(emptySARIF))
	if err != nil {
		t.Fatalf("ParseSARIF: %v", err)
	}
	findings := SARIFToFindings(report)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(findings))
	}
}

func TestSARIFToFindings_MultiRunMerge(t *testing.T) {
	report, err := ParseSARIF([]byte(multiRunSARIF))
	if err != nil {
		t.Fatalf("ParseSARIF: %v", err)
	}
	findings := SARIFToFindings(report)
	if len(findings) != 3 {
		t.Errorf("expected 3 findings from 2 runs, got %d", len(findings))
	}

	// Verify scanners are different
	scanners := map[string]bool{}
	for _, f := range findings {
		scanners[f.Scanner] = true
	}
	if len(scanners) != 2 {
		t.Errorf("expected 2 different scanners, got %d", len(scanners))
	}
}

// ---------------------------------------------------------------------------
// Location handling
// ---------------------------------------------------------------------------

func TestSARIFToFindings_NoLocations(t *testing.T) {
	report, err := ParseSARIF([]byte(sampleTrivySARIF))
	if err != nil {
		t.Fatalf("ParseSARIF: %v", err)
	}
	findings := SARIFToFindings(report)

	// Third result has no locations in the fixture
	lastFinding := findings[2]
	if lastFinding.FilePath != "" {
		t.Errorf("expected empty file path for result without locations, got %q", lastFinding.FilePath)
	}
}

// ---------------------------------------------------------------------------
// Roundtrip: SARIF structs marshal correctly
// ---------------------------------------------------------------------------

func TestSARIFStructs_JSONRoundtrip(t *testing.T) {
	report, err := ParseSARIF([]byte(sampleSemgrepSARIF))
	if err != nil {
		t.Fatalf("ParseSARIF: %v", err)
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	report2, err := ParseSARIF(data)
	if err != nil {
		t.Fatalf("re-ParseSARIF: %v", err)
	}
	if report.TotalResults() != report2.TotalResults() {
		t.Errorf("roundtrip mismatch: %d vs %d", report.TotalResults(), report2.TotalResults())
	}
}

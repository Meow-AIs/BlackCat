package agent

import (
	"testing"
)

func TestNewFactChecker(t *testing.T) {
	fc := NewFactChecker()
	if fc == nil {
		t.Fatal("NewFactChecker returned nil")
	}
}

func TestCheckAgainstToolOutputMatch(t *testing.T) {
	fc := NewFactChecker()
	response := "The command output shows `error: permission denied` when running"
	toolOutput := "error: permission denied for user root"

	facts := fc.CheckAgainstToolOutput(response, "bash", toolOutput)
	if len(facts) == 0 {
		t.Fatal("expected at least one checked fact")
	}

	hasVerified := false
	for _, f := range facts {
		if f.Status == FactVerified {
			hasVerified = true
			break
		}
	}
	if !hasVerified {
		t.Errorf("expected at least one verified fact, got: %+v", facts)
	}
}

func TestCheckAgainstToolOutputMismatch(t *testing.T) {
	fc := NewFactChecker()
	response := "The output shows `success: deployment complete`"
	toolOutput := "error: deployment failed with exit code 1"

	facts := fc.CheckAgainstToolOutput(response, "bash", toolOutput)
	if len(facts) == 0 {
		t.Fatal("expected at least one checked fact")
	}

	hasContradicted := false
	for _, f := range facts {
		if f.Status == FactContradicted {
			hasContradicted = true
			break
		}
	}
	if !hasContradicted {
		t.Errorf("expected contradicted fact, got: %+v", facts)
	}
}

func TestCheckAgainstToolOutputEmpty(t *testing.T) {
	fc := NewFactChecker()
	facts := fc.CheckAgainstToolOutput("no quoted content here", "bash", "some output")
	if len(facts) != 0 {
		t.Errorf("expected no facts for response without quoted content, got %d", len(facts))
	}
}

func TestCheckCodeReferencesKnownFiles(t *testing.T) {
	fc := NewFactChecker()
	response := "Look at `main.go` and `utils.go` for the implementation"
	knownFiles := []string{"main.go", "handler.go"}
	knownFunctions := []string{}

	facts := fc.CheckCodeReferences(response, knownFiles, knownFunctions)
	if len(facts) == 0 {
		t.Fatal("expected at least one fact")
	}

	var mainVerified, utilsContradicted bool
	for _, f := range facts {
		if f.Claim == "main.go" && f.Status == FactVerified {
			mainVerified = true
		}
		if f.Claim == "utils.go" && f.Status == FactContradicted {
			utilsContradicted = true
		}
	}
	if !mainVerified {
		t.Error("expected main.go to be verified")
	}
	if !utilsContradicted {
		t.Error("expected utils.go to be contradicted")
	}
}

func TestCheckCodeReferencesKnownFunctions(t *testing.T) {
	fc := NewFactChecker()
	response := "The function `HandleRequest` processes incoming data"
	knownFiles := []string{}
	knownFunctions := []string{"HandleRequest", "ProcessData"}

	facts := fc.CheckCodeReferences(response, knownFiles, knownFunctions)

	hasVerified := false
	for _, f := range facts {
		if f.Claim == "HandleRequest" && f.Status == FactVerified {
			hasVerified = true
		}
	}
	if !hasVerified {
		t.Error("expected HandleRequest to be verified")
	}
}

func TestCheckCodeReferencesEmpty(t *testing.T) {
	fc := NewFactChecker()
	facts := fc.CheckCodeReferences("no references here", nil, nil)
	if len(facts) != 0 {
		t.Errorf("expected no facts, got %d", len(facts))
	}
}

func TestSummarizeAllVerified(t *testing.T) {
	fc := NewFactChecker()
	facts := []CheckedFact{
		{Claim: "a", Category: "code_reference", Status: FactVerified, Evidence: "found"},
		{Claim: "b", Category: "tool_output", Status: FactVerified, Evidence: "found"},
	}
	result := fc.Summarize(facts)

	if result.TotalFacts != 2 {
		t.Errorf("expected 2 total, got %d", result.TotalFacts)
	}
	if result.Verified != 2 {
		t.Errorf("expected 2 verified, got %d", result.Verified)
	}
	if result.Reliability != 1.0 {
		t.Errorf("expected reliability 1.0, got %f", result.Reliability)
	}
}

func TestSummarizeMixed(t *testing.T) {
	fc := NewFactChecker()
	facts := []CheckedFact{
		{Claim: "a", Status: FactVerified},
		{Claim: "b", Status: FactContradicted},
		{Claim: "c", Status: FactUnverifiable},
	}
	result := fc.Summarize(facts)

	if result.TotalFacts != 3 {
		t.Errorf("expected 3 total, got %d", result.TotalFacts)
	}
	if result.Verified != 1 {
		t.Errorf("expected 1 verified, got %d", result.Verified)
	}
	if result.Contradicted != 1 {
		t.Errorf("expected 1 contradicted, got %d", result.Contradicted)
	}
	if result.Unverifiable != 1 {
		t.Errorf("expected 1 unverifiable, got %d", result.Unverifiable)
	}
	// reliability = verified / (verified + contradicted) = 1/2 = 0.5
	if result.Reliability < 0.49 || result.Reliability > 0.51 {
		t.Errorf("expected reliability ~0.5, got %f", result.Reliability)
	}
}

func TestSummarizeEmpty(t *testing.T) {
	fc := NewFactChecker()
	result := fc.Summarize(nil)

	if result.TotalFacts != 0 {
		t.Errorf("expected 0 total, got %d", result.TotalFacts)
	}
	if result.Reliability != 1.0 {
		t.Errorf("expected reliability 1.0 for empty, got %f", result.Reliability)
	}
}

func TestExtractCodeReferences(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		wants []string
	}{
		{
			"backtick_refs",
			"Check `main.go` and `utils.go`",
			[]string{"main.go", "utils.go"},
		},
		{
			"function_refs",
			"The function `HandleAuth` processes auth",
			[]string{"HandleAuth"},
		},
		{
			"no_refs",
			"This is plain text with no code references",
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs := ExtractCodeReferences(tt.text)
			for _, want := range tt.wants {
				found := false
				for _, ref := range refs {
					if ref == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected to find reference %q in %v", want, refs)
				}
			}
		})
	}
}

func TestExtractQuotedContent(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		wants []string
	}{
		{
			"single_backtick",
			"The value is `hello world` in the output",
			[]string{"hello world"},
		},
		{
			"multiple_backticks",
			"Use `foo` and `bar` together",
			[]string{"foo", "bar"},
		},
		{
			"no_backticks",
			"Plain text with nothing quoted",
			nil,
		},
		{
			"empty_backticks",
			"An empty `` backtick pair",
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quoted := ExtractQuotedContent(tt.text)
			for _, want := range tt.wants {
				found := false
				for _, q := range quoted {
					if q == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected to find %q in %v", want, quoted)
				}
			}
			if tt.wants == nil && len(quoted) != 0 {
				t.Errorf("expected no quoted content, got %v", quoted)
			}
		})
	}
}

func TestSummarizeFactsPreserved(t *testing.T) {
	fc := NewFactChecker()
	facts := []CheckedFact{
		{Claim: "x", Status: FactVerified},
	}
	result := fc.Summarize(facts)
	if len(result.Facts) != 1 {
		t.Errorf("expected 1 fact preserved, got %d", len(result.Facts))
	}
}

package agent

import (
	"fmt"
	"regexp"
	"strings"
)

// FactStatus represents the verification status of a fact.
type FactStatus string

const (
	FactVerified     FactStatus = "verified"
	FactContradicted FactStatus = "contradicted"
	FactUnverifiable FactStatus = "unverifiable"
)

// CheckedFact represents a single fact that has been checked.
type CheckedFact struct {
	Claim    string     // the asserted fact
	Category string    // "code_reference", "tool_output", "general_assertion"
	Status   FactStatus // verification result
	Evidence string     // supporting or contradicting evidence
}

// FactCheckResult aggregates all checked facts.
type FactCheckResult struct {
	TotalFacts   int
	Verified     int
	Contradicted int
	Unverifiable int
	Facts        []CheckedFact
	Reliability  float64 // 0-1 (verified / (verified + contradicted)), 1.0 if none
}

// FactChecker verifies LLM output claims against actual data.
type FactChecker struct{}

// Precompiled patterns for extraction.
var (
	// Matches content inside single backticks (not triple backticks).
	singleBacktickRe = regexp.MustCompile("(?:^|[^`])`([^`]+)`(?:[^`]|$)")

	// Matches file-like references: word.ext patterns.
	fileRefRe = regexp.MustCompile(`\b([a-zA-Z0-9_/-]+\.[a-zA-Z]{1,10})\b`)

	// Matches function-like references: PascalCase or camelCase identifiers.
	funcRefRe = regexp.MustCompile(`\b([A-Z][a-zA-Z0-9_]{2,})\b`)
)

// NewFactChecker creates a new FactChecker.
func NewFactChecker() *FactChecker {
	return &FactChecker{}
}

// CheckAgainstToolOutput verifies if quoted content in the response matches
// the actual tool output. Each backtick-quoted phrase is checked for presence
// in the tool output. Returns nil if no quoted content is found.
func (fc *FactChecker) CheckAgainstToolOutput(response, toolName, toolOutput string) []CheckedFact {
	quoted := ExtractQuotedContent(response)
	if len(quoted) == 0 {
		return nil
	}

	var facts []CheckedFact
	lowerOutput := strings.ToLower(toolOutput)

	for _, q := range quoted {
		lowerQ := strings.ToLower(q)
		fact := CheckedFact{
			Claim:    q,
			Category: "tool_output",
		}

		if strings.Contains(lowerOutput, lowerQ) {
			fact.Status = FactVerified
			fact.Evidence = fmt.Sprintf("found in %s output", toolName)
		} else {
			fact.Status = FactContradicted
			fact.Evidence = fmt.Sprintf("not found in %s output", toolName)
		}

		facts = append(facts, fact)
	}

	return facts
}

// CheckCodeReferences verifies backtick-quoted code references against
// known files and functions. Returns nil if no references are found.
func (fc *FactChecker) CheckCodeReferences(response string, knownFiles, knownFunctions []string) []CheckedFact {
	refs := ExtractCodeReferences(response)
	if len(refs) == 0 {
		return nil
	}

	fileSet := toSet(knownFiles)
	funcSet := toSet(knownFunctions)

	var facts []CheckedFact
	for _, ref := range refs {
		fact := CheckedFact{
			Claim:    ref,
			Category: "code_reference",
		}

		if isFileLike(ref) {
			if fileSet[ref] {
				fact.Status = FactVerified
				fact.Evidence = "file exists in known files"
			} else {
				fact.Status = FactContradicted
				fact.Evidence = "file not found in known files"
			}
		} else if funcSet[ref] {
			fact.Status = FactVerified
			fact.Evidence = "function exists in known functions"
		} else if len(knownFiles) > 0 || len(knownFunctions) > 0 {
			fact.Status = FactUnverifiable
			fact.Evidence = "reference not matched to known files or functions"
		} else {
			fact.Status = FactUnverifiable
			fact.Evidence = "no known references to check against"
		}

		facts = append(facts, fact)
	}

	return facts
}

// Summarize aggregates checked facts into a FactCheckResult.
// Reliability = verified / (verified + contradicted). Returns 1.0 if empty.
func (fc *FactChecker) Summarize(facts []CheckedFact) FactCheckResult {
	if len(facts) == 0 {
		return FactCheckResult{Reliability: 1.0}
	}

	// Copy facts for immutability.
	copied := make([]CheckedFact, len(facts))
	copy(copied, facts)

	var verified, contradicted, unverifiable int
	for _, f := range copied {
		switch f.Status {
		case FactVerified:
			verified++
		case FactContradicted:
			contradicted++
		case FactUnverifiable:
			unverifiable++
		}
	}

	reliability := 1.0
	denominator := verified + contradicted
	if denominator > 0 {
		reliability = float64(verified) / float64(denominator)
	}

	return FactCheckResult{
		TotalFacts:   len(copied),
		Verified:     verified,
		Contradicted: contradicted,
		Unverifiable: unverifiable,
		Facts:        copied,
		Reliability:  reliability,
	}
}

// ExtractCodeReferences pulls out file and function references from
// backtick-quoted content in the text.
func ExtractCodeReferences(text string) []string {
	quoted := ExtractQuotedContent(text)
	if len(quoted) == 0 {
		return nil
	}

	seen := make(map[string]bool)
	var refs []string

	for _, q := range quoted {
		// Clean trailing parentheses for function calls.
		cleaned := strings.TrimSuffix(q, "()")
		cleaned = strings.TrimSuffix(cleaned, "(")
		cleaned = strings.TrimSpace(cleaned)

		if cleaned == "" {
			continue
		}
		if seen[cleaned] {
			continue
		}

		// Accept if it looks like a file path or an identifier.
		if isFileLike(cleaned) || isIdentifier(cleaned) {
			refs = append(refs, cleaned)
			seen[cleaned] = true
		}
	}

	return refs
}

// ExtractQuotedContent pulls out content inside single backticks.
// Excludes empty matches and triple-backtick code blocks.
func ExtractQuotedContent(text string) []string {
	matches := singleBacktickRe.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}

	var results []string
	for _, m := range matches {
		content := strings.TrimSpace(m[1])
		if content != "" {
			results = append(results, content)
		}
	}

	return results
}

// isFileLike returns true if the string looks like a file path.
func isFileLike(s string) bool {
	return strings.Contains(s, ".") && !strings.HasPrefix(s, ".")
}

// isIdentifier returns true if the string looks like a code identifier.
func isIdentifier(s string) bool {
	if len(s) < 2 || len(s) > 100 {
		return false
	}
	// Must start with a letter and contain only valid identifier chars.
	for i, c := range s {
		if i == 0 && !isLetter(c) {
			return false
		}
		if !isLetter(c) && !isDigit(c) && c != '_' {
			return false
		}
	}
	return true
}

func isLetter(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isDigit(c rune) bool {
	return c >= '0' && c <= '9'
}

// toSet converts a string slice to a set for O(1) lookup.
func toSet(items []string) map[string]bool {
	set := make(map[string]bool, len(items))
	for _, item := range items {
		set[item] = true
	}
	return set
}

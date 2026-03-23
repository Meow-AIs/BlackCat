package agent

import (
	"fmt"
	"regexp"
	"strings"
)

// ClaimType categorizes a grounding claim extracted from LLM output.
type ClaimType string

const (
	ClaimFilePath     ClaimType = "file_path"
	ClaimLineNumber   ClaimType = "line_number"
	ClaimFunctionName ClaimType = "function_name"
	ClaimErrorMessage ClaimType = "error_message"
	ClaimPackageName  ClaimType = "package_name"
)

// GroundingClaim represents a verifiable claim found in an LLM response.
type GroundingClaim struct {
	Type     ClaimType // category of claim
	Value    string    // the claimed value
	Source   string    // where in response this was found
	Verified bool      // whether verification passed
	Evidence string    // what we found when checking
}

// GroundingResult aggregates verification results for all claims.
type GroundingResult struct {
	TotalClaims      int
	VerifiedClaims   int
	FalseClaims      int
	UnverifiedClaims int
	Claims           []GroundingClaim
	Score            float64 // verified/total ratio (1.0 if no claims)
}

// GroundingVerifier extracts and verifies factual claims from LLM output.
type GroundingVerifier struct {
	workDir string
}

// Precompiled extraction patterns.
var (
	// File paths: backtick-wrapped paths like `path/to/file.go` or bare paths.
	filePathBacktickRe = regexp.MustCompile("`([a-zA-Z0-9_./-]+\\.[a-zA-Z]{1,10})`")

	// Line numbers: "line 42", "L42", ":42".
	lineNumberLineRe  = regexp.MustCompile(`\bline\s+(\d+)\b`)
	lineNumberColonRe = regexp.MustCompile(`[:.](\d+)\b`)
	lineNumberLRe     = regexp.MustCompile(`\bL(\d+)\b`)

	// Function names: `FuncName()` in backticks.
	funcNameRe = regexp.MustCompile("`([A-Z][a-zA-Z0-9_]*(?:\\([^)]*\\))?)`")

	// Package names: "package X".
	packageNameRe = regexp.MustCompile(`\bpackage\s+([a-z][a-z0-9_]*)\b`)
)

// NewGroundingVerifier creates a verifier rooted at the given work directory.
func NewGroundingVerifier(workDir string) *GroundingVerifier {
	return &GroundingVerifier{workDir: workDir}
}

// ExtractClaims scans an LLM response for verifiable claims and returns them.
func (v *GroundingVerifier) ExtractClaims(response string) []GroundingClaim {
	var claims []GroundingClaim
	seen := make(map[string]bool)

	// File paths in backticks.
	for _, match := range filePathBacktickRe.FindAllStringSubmatch(response, -1) {
		val := match[1]
		key := string(ClaimFilePath) + ":" + val
		if seen[key] {
			continue
		}
		// Only include if it looks like a path (has a slash or extension).
		if strings.Contains(val, "/") || strings.Contains(val, ".") {
			// Skip if it looks like a function call (ends with parentheses).
			if strings.HasSuffix(val, ")") {
				continue
			}
			claims = append(claims, GroundingClaim{
				Type:   ClaimFilePath,
				Value:  val,
				Source: match[0],
			})
			seen[key] = true
		}
	}

	// Line numbers: "line N".
	for _, match := range lineNumberLineRe.FindAllStringSubmatch(response, -1) {
		val := match[1]
		key := string(ClaimLineNumber) + ":" + val
		if !seen[key] {
			claims = append(claims, GroundingClaim{
				Type:   ClaimLineNumber,
				Value:  val,
				Source: match[0],
			})
			seen[key] = true
		}
	}

	// Line numbers: ":N" (after a filename-like context).
	for _, match := range lineNumberColonRe.FindAllStringSubmatchIndex(response, -1) {
		// Only match ":N" if preceded by a file extension.
		prefix := response[:match[0]]
		if len(prefix) >= 3 && prefix[len(prefix)-3:len(prefix)-1] == ".g" ||
			strings.HasSuffix(prefix, ".go") ||
			strings.HasSuffix(prefix, ".js") ||
			strings.HasSuffix(prefix, ".py") ||
			strings.HasSuffix(prefix, ".ts") {
			val := response[match[2]:match[3]]
			key := string(ClaimLineNumber) + ":" + val
			if !seen[key] {
				claims = append(claims, GroundingClaim{
					Type:   ClaimLineNumber,
					Value:  val,
					Source: response[match[0]:match[1]],
				})
				seen[key] = true
			}
		}
	}

	// Line numbers: "LN".
	for _, match := range lineNumberLRe.FindAllStringSubmatch(response, -1) {
		val := match[1]
		key := string(ClaimLineNumber) + ":" + val
		if !seen[key] {
			claims = append(claims, GroundingClaim{
				Type:   ClaimLineNumber,
				Value:  val,
				Source: match[0],
			})
			seen[key] = true
		}
	}

	// Function names in backticks (PascalCase with optional parens).
	for _, match := range funcNameRe.FindAllStringSubmatch(response, -1) {
		val := match[1]
		// Strip trailing () for the value.
		cleaned := strings.TrimSuffix(strings.TrimSuffix(val, "()"), "()")
		// Skip if already captured as a file path.
		key := string(ClaimFunctionName) + ":" + cleaned
		if seen[key] || seen[string(ClaimFilePath)+":"+val] {
			continue
		}
		// Must not contain slashes or dots (those are file paths).
		if strings.Contains(cleaned, "/") || strings.Contains(cleaned, ".") {
			continue
		}
		claims = append(claims, GroundingClaim{
			Type:   ClaimFunctionName,
			Value:  cleaned,
			Source: match[0],
		})
		seen[key] = true
	}

	// Package names: "package X".
	for _, match := range packageNameRe.FindAllStringSubmatch(response, -1) {
		val := match[1]
		key := string(ClaimPackageName) + ":" + val
		if !seen[key] {
			claims = append(claims, GroundingClaim{
				Type:   ClaimPackageName,
				Value:  val,
				Source: match[0],
			})
			seen[key] = true
		}
	}

	return claims
}

// VerifyClaims checks extracted claims against tool outputs and returns
// an aggregated result. Claims of type ClaimErrorMessage are checked by
// substring matching against tool outputs. Other claim types are marked
// as unverified (filesystem checks would require I/O).
func (v *GroundingVerifier) VerifyClaims(claims []GroundingClaim, toolOutputs []string) GroundingResult {
	if len(claims) == 0 {
		return GroundingResult{Score: 1.0}
	}

	verified := make([]GroundingClaim, 0, len(claims))
	for _, c := range claims {
		checked := v.verifySingleClaim(c, toolOutputs)
		verified = append(verified, checked)
	}

	var verifiedCount, falseCount, unverifiedCount int
	for _, c := range verified {
		switch {
		case c.Verified:
			verifiedCount++
		case c.Evidence == "not found in tool output":
			falseCount++
		default:
			unverifiedCount++
		}
	}

	score := float64(verifiedCount) / float64(len(verified))

	return GroundingResult{
		TotalClaims:      len(verified),
		VerifiedClaims:   verifiedCount,
		FalseClaims:      falseCount,
		UnverifiedClaims: unverifiedCount,
		Claims:           verified,
		Score:            score,
	}
}

// verifySingleClaim checks one claim against available tool outputs.
func (v *GroundingVerifier) verifySingleClaim(claim GroundingClaim, toolOutputs []string) GroundingClaim {
	result := GroundingClaim{
		Type:   claim.Type,
		Value:  claim.Value,
		Source: claim.Source,
	}

	switch claim.Type {
	case ClaimErrorMessage:
		for _, output := range toolOutputs {
			if strings.Contains(strings.ToLower(output), strings.ToLower(claim.Value)) {
				result.Verified = true
				result.Evidence = "matched in tool output"
				return result
			}
		}
		result.Evidence = "not found in tool output"

	default:
		// For file paths, line numbers, function names, and package names,
		// check if they appear in any tool output as a lightweight verification.
		for _, output := range toolOutputs {
			if strings.Contains(output, claim.Value) {
				result.Verified = true
				result.Evidence = "found in tool output"
				return result
			}
		}
		result.Evidence = "could not verify without filesystem access"
	}

	return result
}

// AnnotateResponse inserts [verified] or [unverified] markers next to
// claims in the response text.
func (v *GroundingVerifier) AnnotateResponse(response string, result GroundingResult) string {
	annotated := response
	for _, claim := range result.Claims {
		if claim.Value == "" {
			continue
		}
		marker := "[unverified]"
		if claim.Verified {
			marker = "[verified]"
		}
		tag := fmt.Sprintf("%s %s", claim.Value, marker)
		annotated = strings.Replace(annotated, claim.Value, tag, 1)
	}
	return annotated
}

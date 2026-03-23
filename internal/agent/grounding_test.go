package agent

import (
	"strings"
	"testing"
)

func TestNewGroundingVerifier(t *testing.T) {
	v := NewGroundingVerifier("/tmp/project")
	if v == nil {
		t.Fatal("NewGroundingVerifier returned nil")
	}
	if v.workDir != "/tmp/project" {
		t.Errorf("expected workDir /tmp/project, got %s", v.workDir)
	}
}

func TestExtractClaimsFilePaths(t *testing.T) {
	v := NewGroundingVerifier("/tmp")
	response := "The bug is in `internal/agent/core.go` and also in path/to/file.go"
	claims := v.ExtractClaims(response)

	found := false
	for _, c := range claims {
		if c.Type == ClaimFilePath && strings.Contains(c.Value, "core.go") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find file path claim for core.go")
	}
}

func TestExtractClaimsLineNumbers(t *testing.T) {
	v := NewGroundingVerifier("/tmp")
	response := "The error is at line 42 in the file"
	claims := v.ExtractClaims(response)

	found := false
	for _, c := range claims {
		if c.Type == ClaimLineNumber && c.Value == "42" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected line number claim for 42, got claims: %+v", claims)
	}
}

func TestExtractClaimsFunctionNames(t *testing.T) {
	v := NewGroundingVerifier("/tmp")
	response := "The function `HandleAuth()` is responsible"
	claims := v.ExtractClaims(response)

	found := false
	for _, c := range claims {
		if c.Type == ClaimFunctionName && strings.Contains(c.Value, "HandleAuth") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected function name claim for HandleAuth, got: %+v", claims)
	}
}

func TestExtractClaimsPackageNames(t *testing.T) {
	v := NewGroundingVerifier("/tmp")
	response := "This is in package agent which handles the logic"
	claims := v.ExtractClaims(response)

	found := false
	for _, c := range claims {
		if c.Type == ClaimPackageName && c.Value == "agent" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected package name claim for agent, got: %+v", claims)
	}
}

func TestExtractClaimsNoClaims(t *testing.T) {
	v := NewGroundingVerifier("/tmp")
	response := "I think this is a good approach to solving the problem."
	claims := v.ExtractClaims(response)

	if len(claims) != 0 {
		t.Errorf("expected no claims, got %d: %+v", len(claims), claims)
	}
}

func TestVerifyClaimsAllVerified(t *testing.T) {
	v := NewGroundingVerifier("/tmp")
	claims := []GroundingClaim{
		{Type: ClaimErrorMessage, Value: "nil pointer dereference", Source: "response"},
	}
	toolOutputs := []string{"panic: runtime error: nil pointer dereference"}

	result := v.VerifyClaims(claims, toolOutputs)
	if result.VerifiedClaims != 1 {
		t.Errorf("expected 1 verified claim, got %d", result.VerifiedClaims)
	}
	if result.Score != 1.0 {
		t.Errorf("expected score 1.0, got %f", result.Score)
	}
}

func TestVerifyClaimsMixed(t *testing.T) {
	v := NewGroundingVerifier("/tmp")
	claims := []GroundingClaim{
		{Type: ClaimErrorMessage, Value: "nil pointer", Source: "resp"},
		{Type: ClaimErrorMessage, Value: "stack overflow", Source: "resp"},
	}
	toolOutputs := []string{"panic: nil pointer dereference at main.go:10"}

	result := v.VerifyClaims(claims, toolOutputs)
	if result.VerifiedClaims != 1 {
		t.Errorf("expected 1 verified, got %d", result.VerifiedClaims)
	}
	if result.TotalClaims != 2 {
		t.Errorf("expected 2 total, got %d", result.TotalClaims)
	}
}

func TestVerifyClaimsNoClaims(t *testing.T) {
	v := NewGroundingVerifier("/tmp")
	result := v.VerifyClaims(nil, nil)
	if result.TotalClaims != 0 {
		t.Errorf("expected 0 total claims, got %d", result.TotalClaims)
	}
	if result.Score != 1.0 {
		t.Errorf("expected score 1.0 for no claims, got %f", result.Score)
	}
}

func TestAnnotateResponse(t *testing.T) {
	v := NewGroundingVerifier("/tmp")
	response := "The error is nil pointer"
	result := GroundingResult{
		TotalClaims:    1,
		VerifiedClaims: 1,
		Claims: []GroundingClaim{
			{Type: ClaimErrorMessage, Value: "nil pointer", Verified: true, Evidence: "found in output"},
		},
		Score: 1.0,
	}

	annotated := v.AnnotateResponse(response, result)
	if !strings.Contains(annotated, "[verified]") {
		t.Error("expected [verified] marker in annotated response")
	}
}

func TestAnnotateResponseUnverified(t *testing.T) {
	v := NewGroundingVerifier("/tmp")
	response := "The error is stack overflow"
	result := GroundingResult{
		TotalClaims:  1,
		FalseClaims:  1,
		Claims: []GroundingClaim{
			{Type: ClaimErrorMessage, Value: "stack overflow", Verified: false, Evidence: "not found"},
		},
		Score: 0.0,
	}

	annotated := v.AnnotateResponse(response, result)
	if !strings.Contains(annotated, "[unverified]") {
		t.Error("expected [unverified] marker in annotated response")
	}
}

func TestExtractClaimsLineNumberFormats(t *testing.T) {
	v := NewGroundingVerifier("/tmp")

	tests := []struct {
		name     string
		input    string
		wantLine string
	}{
		{"line_N", "error at line 99", "99"},
		{"colon_N", "in file.go:55 we see", "55"},
		{"L_N", "check L123 for the bug", "123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := v.ExtractClaims(tt.input)
			found := false
			for _, c := range claims {
				if c.Type == ClaimLineNumber && c.Value == tt.wantLine {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected line number %s, got claims: %+v", tt.wantLine, claims)
			}
		})
	}
}

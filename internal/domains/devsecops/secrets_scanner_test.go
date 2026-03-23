package devsecops

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// ---------------------------------------------------------------------------
// DefaultSecretRules
// ---------------------------------------------------------------------------

func TestDefaultSecretRules_HasRules(t *testing.T) {
	rules := DefaultSecretRules()
	if len(rules) < 10 {
		t.Errorf("expected at least 10 rules, got %d", len(rules))
	}
}

func TestDefaultSecretRules_AllHaveRequiredFields(t *testing.T) {
	for _, r := range DefaultSecretRules() {
		if r.ID == "" {
			t.Error("rule has empty ID")
		}
		if r.Description == "" {
			t.Errorf("rule %q has empty Description", r.ID)
		}
		if r.Pattern == nil {
			t.Errorf("rule %q has nil Pattern", r.ID)
		}
		if r.Severity == "" {
			t.Errorf("rule %q has empty Severity", r.ID)
		}
	}
}

// ---------------------------------------------------------------------------
// SecretsScanner — interface compliance
// ---------------------------------------------------------------------------

func TestSecretsScanner_Name(t *testing.T) {
	s := NewSecretsScanner()
	if s.Name() != "scan_secrets" {
		t.Errorf("expected name 'scan_secrets', got %q", s.Name())
	}
}

func TestSecretsScanner_Description(t *testing.T) {
	s := NewSecretsScanner()
	if s.Description() == "" {
		t.Error("expected non-empty description")
	}
}

// ---------------------------------------------------------------------------
// Pattern detection — AWS
// ---------------------------------------------------------------------------

func TestSecretsScanner_DetectsAWSAccessKey(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.go", `const accessKey = "AKIAIOSFODNN7EXAMPLE"`)

	findings := scanDir(t, dir)
	assertFindingWithRule(t, findings, "aws-access-key")
}

func TestSecretsScanner_DetectsAWSSecretKey(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.go", `aws_secret_access_key = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"`)

	findings := scanDir(t, dir)
	assertFindingWithRule(t, findings, "aws-secret-key")
}

// ---------------------------------------------------------------------------
// Pattern detection — GitHub
// ---------------------------------------------------------------------------

func TestSecretsScanner_DetectsGitHubPAT(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "ci.yml", `token: ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij`)

	findings := scanDir(t, dir)
	assertFindingWithRule(t, findings, "github-pat")
}

// ---------------------------------------------------------------------------
// Pattern detection — Private Key
// ---------------------------------------------------------------------------

func TestSecretsScanner_DetectsPrivateKey(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "key.pem", `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEA0Z3VS5JJcds3xf...
-----END RSA PRIVATE KEY-----`)

	findings := scanDir(t, dir)
	assertFindingWithRule(t, findings, "private-key")
}

// ---------------------------------------------------------------------------
// Pattern detection — Generic API Key
// ---------------------------------------------------------------------------

func TestSecretsScanner_DetectsGenericAPIKey(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.py", `api_key = "sk_live_1234567890abcdefghij"`)

	findings := scanDir(t, dir)
	assertFindingWithRule(t, findings, "generic-api-key")
}

// ---------------------------------------------------------------------------
// Pattern detection — Connection String
// ---------------------------------------------------------------------------

func TestSecretsScanner_DetectsConnectionString(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.go", `dsn := "postgres://user:pass@localhost:5432/db"`)

	findings := scanDir(t, dir)
	assertFindingWithRule(t, findings, "connection-string")
}

// ---------------------------------------------------------------------------
// Pattern detection — Stripe
// ---------------------------------------------------------------------------

func TestSecretsScanner_DetectsStripeKey(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "billing.go", `key := "sk_test_4eC39HqLyjWDarjtT1zdp7dc123456"`)

	findings := scanDir(t, dir)
	assertFindingWithRule(t, findings, "stripe-key")
}

// ---------------------------------------------------------------------------
// Pattern detection — Anthropic Key
// ---------------------------------------------------------------------------

func TestSecretsScanner_DetectsAnthropicKey(t *testing.T) {
	dir := t.TempDir()
	fakeKey := "sk-ant-" + repeatChar('a', 80)
	writeFile(t, dir, "llm.go", `key := "`+fakeKey+`"`)

	findings := scanDir(t, dir)
	assertFindingWithRule(t, findings, "anthropic-key")
}

// ---------------------------------------------------------------------------
// Clean files — no false positives
// ---------------------------------------------------------------------------

func TestSecretsScanner_CleanGoFile_NoFindings(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.go", `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`)
	findings := scanDir(t, dir)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for clean file, got %d: %v", len(findings), findingIDs(findings))
	}
}

// ---------------------------------------------------------------------------
// Exclusions
// ---------------------------------------------------------------------------

func TestSecretsScanner_ExcludesGitDir(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	_ = os.MkdirAll(gitDir, 0755)
	writeFile(t, gitDir, "config", `password = "AKIAIOSFODNN7EXAMPLE"`)

	findings := scanDir(t, dir)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings (inside .git), got %d", len(findings))
	}
}

func TestSecretsScanner_ExcludesBinaryFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "app.exe", `AKIAIOSFODNN7EXAMPLE`)

	findings := scanDir(t, dir)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for binary, got %d", len(findings))
	}
}

func TestSecretsScanner_IncludeExtFilter(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.go", `key := "AKIAIOSFODNN7EXAMPLE"`)
	writeFile(t, dir, "config.py", `key = "AKIAIOSFODNN7EXAMPLE"`)

	s := NewSecretsScanner()
	result, err := s.Scan(context.Background(), ScanRequest{
		Path:       dir,
		Recursive:  true,
		IncludeExt: []string{".go"},
	})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	// Only .go file should be scanned
	if result.Scanned != 1 {
		t.Errorf("expected 1 file scanned, got %d", result.Scanned)
	}
}

// ---------------------------------------------------------------------------
// Finding metadata
// ---------------------------------------------------------------------------

func TestSecretsScanner_FindingHasFilePath(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "secret.go", `key := "AKIAIOSFODNN7EXAMPLE"`)

	findings := scanDir(t, dir)
	if len(findings) == 0 {
		t.Fatal("expected findings")
	}
	if findings[0].FilePath == "" {
		t.Error("expected non-empty FilePath")
	}
}

func TestSecretsScanner_FindingHasLineNumber(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "secret.go", "package main\n\nvar x = \"AKIAIOSFODNN7EXAMPLE\"\n")

	findings := scanDir(t, dir)
	if len(findings) == 0 {
		t.Fatal("expected findings")
	}
	if findings[0].Line != 3 {
		t.Errorf("expected line 3, got %d", findings[0].Line)
	}
}

func TestSecretsScanner_FindingHasSeverity(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "secret.go", `key := "AKIAIOSFODNN7EXAMPLE"`)

	findings := scanDir(t, dir)
	if len(findings) == 0 {
		t.Fatal("expected findings")
	}
	if findings[0].Severity != SeverityCritical {
		t.Errorf("expected critical severity for AWS key, got %q", findings[0].Severity)
	}
}

// ---------------------------------------------------------------------------
// AddRule
// ---------------------------------------------------------------------------

func TestSecretsScanner_AddRule(t *testing.T) {
	s := NewSecretsScanner()
	initialCount := len(s.Rules())

	s.AddRule(SecretRule{
		ID:          "custom-token",
		Description: "Custom Token",
		Pattern:     compilePattern(`CUSTOM_[A-Z]{20}`),
		Severity:    SeverityHigh,
	})

	if len(s.Rules()) != initialCount+1 {
		t.Errorf("expected %d rules after AddRule, got %d", initialCount+1, len(s.Rules()))
	}
}

// ---------------------------------------------------------------------------
// Context cancellation
// ---------------------------------------------------------------------------

func TestSecretsScanner_ContextCancelled(t *testing.T) {
	dir := t.TempDir()
	// Create many files
	for i := 0; i < 100; i++ {
		writeFile(t, dir, filepath.Base(filepath.Join(dir, "file"+string(rune('a'+i%26))+".go")), `key := "AKIAIOSFODNN7EXAMPLE"`)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	s := NewSecretsScanner()
	result, _ := s.Scan(ctx, ScanRequest{Path: dir, Recursive: true})
	// Should have scanned fewer files than total due to cancellation
	_ = result // just ensure no panic
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("writeFile: %v", err)
	}
}

func scanDir(t *testing.T, dir string) []Finding {
	t.Helper()
	s := NewSecretsScanner()
	result, err := s.Scan(context.Background(), ScanRequest{Path: dir, Recursive: true})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	return result.Findings
}

func assertFindingWithRule(t *testing.T, findings []Finding, ruleID string) {
	t.Helper()
	for _, f := range findings {
		if f.RuleID == ruleID {
			return
		}
	}
	t.Errorf("expected finding with rule %q, got: %v", ruleID, findingIDs(findings))
}

func findingIDs(findings []Finding) []string {
	ids := make([]string, len(findings))
	for i, f := range findings {
		ids[i] = f.RuleID
	}
	return ids
}

func repeatChar(ch byte, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = ch
	}
	return string(b)
}

func compilePattern(pattern string) *regexp.Regexp {
	return regexp.MustCompile(pattern)
}

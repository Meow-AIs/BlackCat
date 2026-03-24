package devsecops

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SecretRule defines a regex pattern for detecting hardcoded secrets.
type SecretRule struct {
	ID          string
	Description string
	Pattern     *regexp.Regexp
	Severity    Severity
}

// DefaultSecretRules returns Gitleaks-compatible rules for common secret types.
func DefaultSecretRules() []SecretRule {
	return []SecretRule{
		{ID: "aws-access-key", Description: "AWS Access Key ID", Pattern: regexp.MustCompile(`(?i)(AKIA[0-9A-Z]{16})`), Severity: SeverityCritical},
		{ID: "aws-secret-key", Description: "AWS Secret Access Key", Pattern: regexp.MustCompile(`(?i)aws_secret_access_key\s*[=:]\s*["']?([A-Za-z0-9/+=]{40})["']?`), Severity: SeverityCritical},
		{ID: "github-pat", Description: "GitHub Personal Access Token", Pattern: regexp.MustCompile(`gh[pousr]_[A-Za-z0-9_]{36,255}`), Severity: SeverityCritical},
		{ID: "github-fine-grained", Description: "GitHub Fine-Grained PAT", Pattern: regexp.MustCompile(`github_pat_[A-Za-z0-9_]{82,}`), Severity: SeverityCritical},
		{ID: "generic-api-key", Description: "Generic API Key Assignment", Pattern: regexp.MustCompile(`(?i)(api[_-]?key|apikey)\s*[=:]\s*["']([A-Za-z0-9_\-]{20,})["']`), Severity: SeverityHigh},
		{ID: "generic-secret", Description: "Generic Secret Assignment", Pattern: regexp.MustCompile(`(?i)(secret|password|passwd|pwd)\s*[=:]\s*["']([^\s"']{8,})["']`), Severity: SeverityHigh},
		{ID: "private-key", Description: "Private Key Header", Pattern: regexp.MustCompile(`-----BEGIN\s+(RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`), Severity: SeverityCritical},
		{ID: "slack-token", Description: "Slack Token", Pattern: regexp.MustCompile(`xox[baprs]-[0-9]{10,13}-[0-9]{10,13}-[a-zA-Z0-9]{24,34}`), Severity: SeverityHigh},
		{ID: "slack-webhook", Description: "Slack Webhook URL", Pattern: regexp.MustCompile(`https://hooks\.slack\.com/services/T[A-Z0-9]+/B[A-Z0-9]+/[a-zA-Z0-9]+`), Severity: SeverityHigh},
		{ID: "stripe-key", Description: "Stripe API Key", Pattern: regexp.MustCompile(`(sk|pk)_(test|live)_[0-9a-zA-Z]{24,}`), Severity: SeverityCritical},
		{ID: "google-api-key", Description: "Google API Key", Pattern: regexp.MustCompile(`AIza[0-9A-Za-z\-_]{35}`), Severity: SeverityHigh},
		{ID: "jwt-token", Description: "JSON Web Token", Pattern: regexp.MustCompile(`eyJ[A-Za-z0-9_-]{10,}\.eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_\-]+`), Severity: SeverityMedium},
		{ID: "connection-string", Description: "Database Connection String", Pattern: regexp.MustCompile(`(?i)(postgres|mysql|mongodb|redis)://[^\s"']+@[^\s"']+`), Severity: SeverityCritical},
		{ID: "bearer-token", Description: "Bearer Token in Code", Pattern: regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9_\-.]{20,}`), Severity: SeverityHigh},
		{ID: "anthropic-key", Description: "Anthropic API Key", Pattern: regexp.MustCompile(`sk-ant-[A-Za-z0-9_\-]{80,}`), Severity: SeverityCritical},
		{ID: "openai-key", Description: "OpenAI API Key", Pattern: regexp.MustCompile(`sk-[A-Za-z0-9]{32,}`), Severity: SeverityCritical},
	}
}

// SecretsScanner scans files for hardcoded secrets using regex rules.
type SecretsScanner struct {
	rules       []SecretRule
	excludeDirs map[string]bool
	excludeExts map[string]bool
}

// NewSecretsScanner creates a scanner with the default rules.
func NewSecretsScanner() *SecretsScanner {
	return &SecretsScanner{
		rules: DefaultSecretRules(),
		excludeDirs: map[string]bool{
			".git": true, "node_modules": true, "vendor": true,
			"__pycache__": true, ".venv": true, "dist": true,
			"build": true, ".next": true, "target": true,
		},
		excludeExts: map[string]bool{
			".exe": true, ".dll": true, ".so": true, ".dylib": true,
			".png": true, ".jpg": true, ".gif": true, ".ico": true,
			".woff": true, ".woff2": true, ".ttf": true, ".eot": true,
			".zip": true, ".tar": true, ".gz": true, ".onnx": true,
			".pdf": true, ".mp4": true, ".mp3": true, ".wav": true,
		},
	}
}

func (s *SecretsScanner) Name() string { return "scan_secrets" }
func (s *SecretsScanner) Description() string {
	return "Detect hardcoded secrets in files using Gitleaks-compatible rules"
}

func (s *SecretsScanner) Scan(ctx context.Context, req ScanRequest) (ScanResult, error) {
	result := ScanResult{Scanner: s.Name()}

	// Merge exclude dirs
	excludeDirs := make(map[string]bool)
	for k, v := range s.excludeDirs {
		excludeDirs[k] = v
	}
	for _, d := range req.ExcludeDirs {
		excludeDirs[d] = true
	}

	// Walk files
	err := filepath.WalkDir(req.Path, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable paths
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if d.IsDir() {
			if excludeDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(d.Name()))
		if s.excludeExts[ext] {
			return nil
		}

		// Apply include filter
		if len(req.IncludeExt) > 0 {
			matched := false
			for _, ie := range req.IncludeExt {
				if strings.EqualFold(ext, ie) {
					matched = true
					break
				}
			}
			if !matched {
				return nil
			}
		}

		findings := s.scanFile(path)
		result.Findings = append(result.Findings, findings...)
		result.Scanned++
		return nil
	})

	if err != nil {
		result.Errors = append(result.Errors, err.Error())
	}
	return result, nil
}

func (s *SecretsScanner) scanFile(path string) []Finding {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	// Skip large files (> 1MB)
	info, err := f.Stat()
	if err != nil || info.Size() > 1<<20 {
		return nil
	}

	var findings []Finding
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		for _, rule := range s.rules {
			if rule.Pattern.MatchString(line) {
				findings = append(findings, Finding{
					ID:          fmt.Sprintf("%s:%s:%d", rule.ID, filepath.Base(path), lineNum),
					Scanner:     "scan_secrets",
					Severity:    rule.Severity,
					Title:       rule.Description,
					Description: fmt.Sprintf("Potential %s found in %s at line %d", rule.Description, filepath.Base(path), lineNum),
					FilePath:    path,
					Line:        lineNum,
					RuleID:      rule.ID,
					Confidence:  0.9,
				})
			}
		}
	}
	return findings
}

// AddRule adds a custom rule to the scanner.
func (s *SecretsScanner) AddRule(rule SecretRule) {
	s.rules = append(s.rules, rule)
}

// Rules returns the current set of rules.
func (s *SecretsScanner) Rules() []SecretRule {
	result := make([]SecretRule, len(s.rules))
	copy(result, s.rules)
	return result
}

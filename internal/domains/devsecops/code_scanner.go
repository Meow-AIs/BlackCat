package devsecops

import (
	"fmt"
	"regexp"
	"strings"
)

// CodeSecurityRule defines a regex-based code security check.
type CodeSecurityRule struct {
	ID          string
	Language    string // "go", "python", "javascript", "any"
	Pattern     *regexp.Regexp
	Severity    string
	Description string
	CWE         string // CWE-xxx reference
}

// CodeScanner performs SAST-lite scanning using regex rules.
type CodeScanner struct {
	rules []CodeSecurityRule
}

// NewCodeScanner creates a scanner pre-loaded with rules for Go, Python, and JS.
func NewCodeScanner() *CodeScanner {
	s := &CodeScanner{}
	s.LoadGoRules()
	s.LoadPythonRules()
	s.LoadJSRules()
	return s
}

// LoadGoRules adds Go-specific security rules (OWASP Top 10 coverage).
func (s *CodeScanner) LoadGoRules() {
	s.rules = append(s.rules, []CodeSecurityRule{
		{ID: "go-sql-injection", Language: "go", Pattern: regexp.MustCompile(`(?:\.Query|\.Exec|\.QueryRow)\s*\([^)]*\+\s*`), Severity: "critical", Description: "SQL injection via string concatenation", CWE: "CWE-89"},
		{ID: "go-sql-fmt-sprintf", Language: "go", Pattern: regexp.MustCompile(`(?:\.Query|\.Exec|\.QueryRow)\s*\(\s*fmt\.Sprintf`), Severity: "critical", Description: "SQL injection via fmt.Sprintf", CWE: "CWE-89"},
		{ID: "go-cmd-injection", Language: "go", Pattern: regexp.MustCompile(`exec\.Command\s*\(\s*"(?:sh|bash|cmd)".*,\s*"-c"`), Severity: "critical", Description: "OS command injection via shell execution", CWE: "CWE-78"},
		{ID: "go-cmd-injection-var", Language: "go", Pattern: regexp.MustCompile(`exec\.Command\s*\([^")\n]+\)`), Severity: "high", Description: "OS command with variable arguments", CWE: "CWE-78"},
		{ID: "go-weak-md5", Language: "go", Pattern: regexp.MustCompile(`(?:crypto/md5|md5\.New|md5\.Sum)`), Severity: "medium", Description: "Use of weak MD5 hash algorithm", CWE: "CWE-328"},
		{ID: "go-weak-sha1", Language: "go", Pattern: regexp.MustCompile(`(?:crypto/sha1|sha1\.New|sha1\.Sum)`), Severity: "medium", Description: "Use of weak SHA1 hash algorithm", CWE: "CWE-328"},
		{ID: "go-weak-des", Language: "go", Pattern: regexp.MustCompile(`(?:crypto/des|des\.NewCipher)`), Severity: "high", Description: "Use of weak DES cipher", CWE: "CWE-328"},
		{ID: "go-hardcoded-password", Language: "go", Pattern: regexp.MustCompile(`(?i)(?:password|passwd|pwd)\s*(?::=|=)\s*"[^"]{4,}"`), Severity: "high", Description: "Hardcoded password in source code", CWE: "CWE-798"},
		{ID: "go-tls-insecure", Language: "go", Pattern: regexp.MustCompile(`InsecureSkipVerify\s*:\s*true`), Severity: "high", Description: "TLS certificate verification disabled", CWE: "CWE-295"},
		{ID: "go-ssrf-http-var", Language: "go", Pattern: regexp.MustCompile(`http\.(?:Get|Post|Head)\s*\([^")\n]+\)`), Severity: "medium", Description: "HTTP request with variable URL (potential SSRF)", CWE: "CWE-918"},
		{ID: "go-unescaped-template", Language: "go", Pattern: regexp.MustCompile(`template\.HTML\s*\(`), Severity: "medium", Description: "Unescaped HTML in template (potential XSS)", CWE: "CWE-79"},
		{ID: "go-path-traversal", Language: "go", Pattern: regexp.MustCompile(`filepath\.Join\s*\([^)]*\+`), Severity: "medium", Description: "Path traversal via string concatenation in filepath.Join", CWE: "CWE-22"},
	}...)
}

// LoadPythonRules adds Python-specific security rules (OWASP Top 10 coverage).
func (s *CodeScanner) LoadPythonRules() {
	s.rules = append(s.rules, []CodeSecurityRule{
		{ID: "py-eval", Language: "python", Pattern: regexp.MustCompile(`\beval\s*\(`), Severity: "critical", Description: "Use of eval() allows arbitrary code execution", CWE: "CWE-95"},
		{ID: "py-exec", Language: "python", Pattern: regexp.MustCompile(`\bexec\s*\(`), Severity: "critical", Description: "Use of exec() allows arbitrary code execution", CWE: "CWE-95"},
		{ID: "py-pickle-loads", Language: "python", Pattern: regexp.MustCompile(`pickle\.loads?\s*\(`), Severity: "high", Description: "Unsafe deserialization via pickle", CWE: "CWE-502"},
		{ID: "py-yaml-load", Language: "python", Pattern: regexp.MustCompile(`yaml\.load\s*\(`), Severity: "high", Description: "Unsafe YAML deserialization (use yaml.safe_load)", CWE: "CWE-502"},
		{ID: "py-sql-injection", Language: "python", Pattern: regexp.MustCompile(`(?:execute|executemany)\s*\(\s*(?:f"|f'|"[^"]*"\s*%|"[^"]*"\s*\.\s*format|'[^']*'\s*%|'[^']*'\s*\.\s*format|"[^"]*"\s*\+|'[^']*'\s*\+)`), Severity: "critical", Description: "SQL injection via string formatting", CWE: "CWE-89"},
		{ID: "py-os-system", Language: "python", Pattern: regexp.MustCompile(`os\.system\s*\(`), Severity: "high", Description: "OS command execution via os.system()", CWE: "CWE-78"},
		{ID: "py-subprocess-shell", Language: "python", Pattern: regexp.MustCompile(`subprocess\.(?:call|run|Popen)\s*\([^)]*shell\s*=\s*True`), Severity: "high", Description: "Shell injection via subprocess with shell=True", CWE: "CWE-78"},
		{ID: "py-hardcoded-password", Language: "python", Pattern: regexp.MustCompile(`(?i)(?:password|passwd|pwd|secret)\s*=\s*(?:"[^"]{4,}"|'[^']{4,}')`), Severity: "high", Description: "Hardcoded password in source code", CWE: "CWE-798"},
		{ID: "py-weak-hash", Language: "python", Pattern: regexp.MustCompile(`hashlib\.(?:md5|sha1)\s*\(`), Severity: "medium", Description: "Use of weak hash algorithm (MD5/SHA1)", CWE: "CWE-328"},
		{ID: "py-flask-debug", Language: "python", Pattern: regexp.MustCompile(`\.run\s*\([^)]*debug\s*=\s*True`), Severity: "high", Description: "Flask debug mode enabled in production", CWE: "CWE-215"},
		{ID: "py-ssrf", Language: "python", Pattern: regexp.MustCompile(`requests\.(?:get|post|put|delete|head)\s*\(\s*[a-zA-Z_]`), Severity: "medium", Description: "HTTP request with variable URL (potential SSRF)", CWE: "CWE-918"},
		{ID: "py-tempfile-insecure", Language: "python", Pattern: regexp.MustCompile(`(?:tempfile\.mktemp|os\.tmpnam)\s*\(`), Severity: "medium", Description: "Insecure temporary file creation (race condition)", CWE: "CWE-377"},
	}...)
}

// LoadJSRules adds JavaScript/TypeScript security rules (OWASP Top 10 coverage).
func (s *CodeScanner) LoadJSRules() {
	s.rules = append(s.rules, []CodeSecurityRule{
		{ID: "js-eval", Language: "javascript", Pattern: regexp.MustCompile(`\beval\s*\(`), Severity: "critical", Description: "Use of eval() allows arbitrary code execution", CWE: "CWE-95"},
		{ID: "js-innerhtml", Language: "javascript", Pattern: regexp.MustCompile(`\.innerHTML\s*=`), Severity: "high", Description: "Direct innerHTML assignment (XSS risk)", CWE: "CWE-79"},
		{ID: "js-document-write", Language: "javascript", Pattern: regexp.MustCompile(`document\.write\s*\(`), Severity: "high", Description: "document.write() usage (XSS risk)", CWE: "CWE-79"},
		{ID: "js-outerhtml", Language: "javascript", Pattern: regexp.MustCompile(`\.outerHTML\s*=`), Severity: "high", Description: "Direct outerHTML assignment (XSS risk)", CWE: "CWE-79"},
		{ID: "js-prototype-pollution", Language: "javascript", Pattern: regexp.MustCompile(`\[["']__proto__["']\]`), Severity: "high", Description: "Prototype pollution via __proto__ access", CWE: "CWE-1321"},
		{ID: "js-sql-concat", Language: "javascript", Pattern: regexp.MustCompile(`(?:query|execute)\s*\(\s*(?:["'` + "`" + `][^"'` + "`" + `]*["'` + "`" + `]\s*\+|` + "`" + `[^` + "`" + `]*\$\{)`), Severity: "critical", Description: "SQL injection via string concatenation or template literal", CWE: "CWE-89"},
		{ID: "js-exec-child", Language: "javascript", Pattern: regexp.MustCompile(`(?:child_process\.exec|execSync)\s*\(`), Severity: "high", Description: "OS command execution via child_process", CWE: "CWE-78"},
		{ID: "js-hardcoded-secret", Language: "javascript", Pattern: regexp.MustCompile(`(?i)(?:password|secret|api_?key|token)\s*(?:=|:)\s*(?:"[^"]{4,}"|'[^']{4,}'|` + "`" + `[^` + "`" + `]{4,}` + "`" + `)`), Severity: "high", Description: "Hardcoded secret in source code", CWE: "CWE-798"},
		{ID: "js-dangerously-set", Language: "javascript", Pattern: regexp.MustCompile(`dangerouslySetInnerHTML`), Severity: "medium", Description: "React dangerouslySetInnerHTML usage (XSS risk)", CWE: "CWE-79"},
		{ID: "js-no-csrf", Language: "javascript", Pattern: regexp.MustCompile(`(?:cors|CORS)\s*\(\s*\{[^}]*origin\s*:\s*(?:true|["']\*["'])`), Severity: "medium", Description: "Overly permissive CORS configuration", CWE: "CWE-346"},
		{ID: "js-open-redirect", Language: "javascript", Pattern: regexp.MustCompile(`(?:window\.location|location\.href|location\.assign)\s*=\s*[a-zA-Z_]`), Severity: "medium", Description: "Open redirect via variable URL assignment", CWE: "CWE-601"},
		{ID: "js-new-function", Language: "javascript", Pattern: regexp.MustCompile(`new\s+Function\s*\(`), Severity: "high", Description: "Dynamic code execution via new Function()", CWE: "CWE-95"},
	}...)
}

// ScanContent scans file content line-by-line against matching rules.
// If language is empty, it is auto-detected from the filename.
func (s *CodeScanner) ScanContent(filename, content, language string) []Finding {
	if language == "" {
		language = s.DetectLanguage(filename)
	}
	if language == "" {
		return nil
	}

	var findings []Finding
	lines := strings.Split(content, "\n")
	for lineNum, line := range lines {
		for _, rule := range s.rules {
			if rule.Language != language && rule.Language != "any" {
				continue
			}
			if rule.Pattern.MatchString(line) {
				findings = append(findings, Finding{
					ID:          fmt.Sprintf("cs:%s:%s:%d", rule.ID, filename, lineNum+1),
					Scanner:     "code-scanner",
					Severity:    Severity(rule.Severity),
					Title:       rule.Description,
					Description: fmt.Sprintf("[%s] %s in %s at line %d", rule.CWE, rule.Description, filename, lineNum+1),
					FilePath:    filename,
					Line:        lineNum + 1,
					RuleID:      rule.ID,
					Confidence:  0.7,
					Metadata:    map[string]string{"cwe": rule.CWE, "language": language},
				})
			}
		}
	}
	return findings
}

// DetectLanguage infers the programming language from a filename extension.
func (s *CodeScanner) DetectLanguage(filename string) string {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".go"):
		return "go"
	case strings.HasSuffix(lower, ".py"):
		return "python"
	case strings.HasSuffix(lower, ".js"),
		strings.HasSuffix(lower, ".jsx"),
		strings.HasSuffix(lower, ".ts"),
		strings.HasSuffix(lower, ".tsx"),
		strings.HasSuffix(lower, ".mjs"),
		strings.HasSuffix(lower, ".cjs"):
		return "javascript"
	default:
		return ""
	}
}

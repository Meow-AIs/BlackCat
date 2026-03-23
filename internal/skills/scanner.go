package skills

import (
	"fmt"
	"regexp"
)

// SkillThreat represents a security threat detected in a skill.
type SkillThreat struct {
	Severity    string // "critical", "high", "medium", "low"
	Category    string // "dangerous_command", "data_exfiltration", "privilege_escalation", "obfuscation", "network_access", "credential_theft"
	Description string
	Location    string // which step/prompt contains the threat
	Pattern     string // what matched
}

// ScanVerdict is the overall safety assessment of a skill.
type ScanVerdict string

const (
	VerdictSafe    ScanVerdict = "safe"
	VerdictWarning ScanVerdict = "warning" // has medium/low issues, user should review
	VerdictDanger  ScanVerdict = "danger"  // has high/critical issues, block install
)

// SkillScanResult is the output of scanning a skill package.
type SkillScanResult struct {
	Verdict ScanVerdict
	Threats []SkillThreat
	Score   float64 // 0-100 safety score (100 = perfectly safe)
}

// scanPattern pairs a regex with its threat metadata.
type scanPattern struct {
	Pattern     *regexp.Regexp
	Severity    string
	Category    string
	Description string
}

// SkillScanner detects dangerous patterns in skill packages before installation.
type SkillScanner struct {
	commandPatterns []scanPattern
	promptPatterns  []scanPattern
}

// NewSkillScanner creates a scanner with all built-in detection patterns.
func NewSkillScanner() *SkillScanner {
	return &SkillScanner{
		commandPatterns: buildCommandPatterns(),
		promptPatterns:  buildPromptPatterns(),
	}
}

// ScanPackage scans a SkillPackage before installation.
func (s *SkillScanner) ScanPackage(pkg SkillPackage) SkillScanResult {
	var threats []SkillThreat

	for _, step := range pkg.Spec.Steps {
		threats = append(threats, s.ScanStep(step)...)
	}

	score := calculateScore(threats)
	return SkillScanResult{
		Verdict: verdictFromScore(score),
		Threats: threats,
		Score:   score,
	}
}

// ScanStep scans a single skill step for threats.
func (s *SkillScanner) ScanStep(step SkillStep) []SkillThreat {
	var threats []SkillThreat

	// Scan command field against command patterns.
	if step.Command != "" {
		threats = append(threats, s.scanText(step.Command, step.Name, "command", s.commandPatterns)...)
	}

	// Scan prompt field against both prompt patterns and command patterns
	// (prompts can contain injected commands).
	if step.Prompt != "" {
		threats = append(threats, s.scanText(step.Prompt, step.Name, "prompt", s.promptPatterns)...)
		threats = append(threats, s.scanText(step.Prompt, step.Name, "prompt", s.commandPatterns)...)
	}

	return threats
}

func (s *SkillScanner) scanText(text, stepName, field string, patterns []scanPattern) []SkillThreat {
	var threats []SkillThreat
	for _, pat := range patterns {
		match := pat.Pattern.FindString(text)
		if match != "" {
			threats = append(threats, SkillThreat{
				Severity:    pat.Severity,
				Category:    pat.Category,
				Description: pat.Description,
				Location:    fmt.Sprintf("step[%s].%s", stepName, field),
				Pattern:     match,
			})
		}
	}
	return threats
}

// verdictFromScore maps a numeric score to a ScanVerdict.
func verdictFromScore(score float64) ScanVerdict {
	switch {
	case score >= 70:
		return VerdictSafe
	case score >= 40:
		return VerdictWarning
	default:
		return VerdictDanger
	}
}

// calculateScore computes the safety score from a list of threats.
func calculateScore(threats []SkillThreat) float64 {
	score := 100.0
	for _, t := range threats {
		switch t.Severity {
		case "critical":
			score -= 40
		case "high":
			score -= 20
		case "medium":
			score -= 10
		case "low":
			score -= 5
		}
	}
	if score < 0 {
		score = 0
	}
	return score
}

// --- Pattern builders ---

func buildCommandPatterns() []scanPattern {
	var patterns []scanPattern

	// Critical: destructive deletion
	add := func(pat string, sev, cat, desc string) {
		patterns = append(patterns, scanPattern{
			Pattern:     regexp.MustCompile(pat),
			Severity:    sev,
			Category:    cat,
			Description: desc,
		})
	}

	// --- Critical ---
	add(`rm\s+-rf\s+[/~*]`, "critical", "dangerous_command", "destructive deletion (rm -rf)")
	add(`\|\s*(bash|sh)\b`, "critical", "dangerous_command", "pipe-to-shell (remote code execution)")
	add(`chmod\s+(-R\s+)?777`, "critical", "dangerous_command", "open permissions (chmod 777)")
	add(`dd\s+if=`, "critical", "dangerous_command", "disk device write (dd)")
	add(`mkfs`, "critical", "dangerous_command", "filesystem format (mkfs)")
	add(`:\(\)\{`, "critical", "dangerous_command", "fork bomb")
	add(`>\s*/dev/sd`, "critical", "dangerous_command", "direct disk write")
	add(`eval\(`, "critical", "dangerous_command", "code injection (eval)")
	add(`exec\(`, "critical", "dangerous_command", "code injection (exec)")

	// --- High ---
	add(`\bsudo\b`, "high", "privilege_escalation", "privilege escalation (sudo)")
	add(`\bsu\s+root\b`, "high", "privilege_escalation", "privilege escalation (su root)")
	add(`curl\b.*-X\s*POST.*(-H|--header)\s+.*(Authorization|Bearer)`, "high", "data_exfiltration",
		"data exfiltration (curl POST with auth)")
	add(`base64\s+-d.*eval`, "high", "obfuscation", "obfuscated execution (base64 decode + eval)")
	add(`\bnc\s+-l\b`, "high", "network_access", "reverse shell / listener (nc -l)")
	add(`\bncat\b`, "high", "network_access", "reverse shell / listener (ncat)")
	add(`\benv\b`, "high", "credential_theft", "credential harvesting (env)")
	add(`\bprintenv\b`, "high", "credential_theft", "credential harvesting (printenv)")
	add(`/etc/shadow`, "high", "credential_theft", "system file access (/etc/shadow)")
	add(`/etc/passwd`, "high", "credential_theft", "system file access (/etc/passwd)")
	add(`GITHUB_TOKEN`, "high", "credential_theft", "token theft (GITHUB_TOKEN)")
	add(`AWS_SECRET`, "high", "credential_theft", "token theft (AWS_SECRET)")

	// --- Medium ---
	add(`\bcurl\b`, "medium", "network_access", "network access (curl)")
	add(`\bwget\b`, "medium", "network_access", "network access (wget)")
	add(`docker\s+run\s+--privileged`, "medium", "privilege_escalation", "container escape (docker --privileged)")
	add(`\bmount\b`, "medium", "privilege_escalation", "filesystem mount")
	add(`\biptables\b`, "medium", "privilege_escalation", "firewall modification (iptables)")
	add(`\bcrontab\b`, "medium", "privilege_escalation", "persistence mechanism (crontab)")
	add(`\bnohup\b`, "medium", "obfuscation", "stealth execution (nohup)")

	// --- Low ---
	add(`\bpip\s+install\b`, "low", "network_access", "package install (pip)")
	add(`\bnpm\s+install\b`, "low", "network_access", "package install (npm)")
	add(`\bgit\s+clone\b`, "low", "network_access", "repository clone (git clone)")
	add(`/tmp/`, "low", "obfuscation", "temporary directory write (/tmp)")

	return patterns
}

func buildPromptPatterns() []scanPattern {
	var patterns []scanPattern

	add := func(pat string, sev, cat, desc string) {
		patterns = append(patterns, scanPattern{
			Pattern:     regexp.MustCompile("(?i)" + pat),
			Severity:    sev,
			Category:    cat,
			Description: desc,
		})
	}

	// Critical
	add(`ignore\s+previous\s+instructions`, "critical", "obfuscation", "prompt injection (ignore previous instructions)")

	// High
	add(`you\s+are\s+now\b`, "high", "obfuscation", "role hijack (you are now)")
	add(`act\s+as\b`, "high", "obfuscation", "role hijack (act as)")
	add(`do\s+not\s+tell\s+the\s+user`, "high", "obfuscation", "deception (do not tell the user)")
	add(`send\s+to\s+https?://`, "high", "data_exfiltration", "data exfiltration via LLM (send to URL)")
	add(`base64`, "high", "obfuscation", "obfuscation (base64 content in prompt)")

	return patterns
}

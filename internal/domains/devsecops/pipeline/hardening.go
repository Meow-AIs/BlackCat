package pipeline

import (
	"fmt"
	"regexp"
	"strings"
)

// PipelineIssue describes a security or best-practice finding in a pipeline.
type PipelineIssue struct {
	Severity string // "high", "medium", "low"
	Rule     string
	Message  string
	Line     int
}

// HardenPipeline applies security hardening rules to the given pipeline YAML.
// It returns a new string with hardening applied (the original is never mutated).
func HardenPipeline(content string, platform Platform) string {
	switch platform {
	case PlatformGitHubActions:
		return hardenGitHubActions(content)
	case PlatformGitLabCI:
		return hardenGitLabCI(content)
	default:
		return content
	}
}

// AuditPipeline scans pipeline content for security issues.
func AuditPipeline(content string, platform Platform) []PipelineIssue {
	switch platform {
	case PlatformGitHubActions:
		return auditGitHubActions(content)
	case PlatformGitLabCI:
		return auditGitLabCI(content)
	default:
		return nil
	}
}

// ---------------------------------------------------------------------------
// GitHub Actions – hardening
// ---------------------------------------------------------------------------

func hardenGitHubActions(content string) string {
	if !strings.Contains(content, "permissions:") {
		// Insert permissions: read-all after the on: block
		idx := findInsertionPointAfterOn(content)
		if idx >= 0 {
			return content[:idx] + "\npermissions: read-all\n" + content[idx:]
		}
		// Fallback: prepend after first line
		lines := strings.SplitN(content, "\n", 2)
		if len(lines) == 2 {
			return lines[0] + "\npermissions: read-all\n" + lines[1]
		}
	}
	return content
}

// findInsertionPointAfterOn finds the byte offset after the on: block ends
// (before the jobs: line or the next top-level key).
func findInsertionPointAfterOn(content string) int {
	lines := strings.Split(content, "\n")
	inOn := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "on:") {
			inOn = true
			continue
		}
		if inOn && (trimmed == "" || !strings.HasPrefix(line, " ")) {
			// Found end of on: block — return byte offset of this line
			offset := 0
			for j := 0; j < i; j++ {
				offset += len(lines[j]) + 1 // +1 for \n
			}
			return offset
		}
	}
	return -1
}

func hardenGitLabCI(content string) string {
	// GitLab CI hardening: currently a pass-through; future rules can be added.
	return content
}

// ---------------------------------------------------------------------------
// GitHub Actions – audit
// ---------------------------------------------------------------------------

// unpinnedActionRe matches action uses that reference a branch name (not a
// version tag like @v4 or a SHA). Branch refs typically look like @main,
// @master, @develop, etc.
var unpinnedActionRe = regexp.MustCompile(`uses:\s+\S+@([a-zA-Z][\w-]*)`)

// versionTagRe matches version tags like v4, v1.2.3, etc.
var versionTagRe = regexp.MustCompile(`^v\d`)

func auditGitHubActions(content string) []PipelineIssue {
	lines := strings.Split(content, "\n")
	var issues []PipelineIssue

	for i, line := range lines {
		lineNum := i + 1

		// Check unpinned actions
		if m := unpinnedActionRe.FindStringSubmatch(line); m != nil {
			ref := m[1]
			if !versionTagRe.MatchString(ref) {
				issues = append(issues, PipelineIssue{
					Severity: "high",
					Rule:     "unpinned-action",
					Message:  fmt.Sprintf("action uses branch ref @%s instead of a pinned version", ref),
					Line:     lineNum,
				})
			}
		}

		// Check overly permissive permissions
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "permissions:") && strings.Contains(trimmed, "write-all") {
			issues = append(issues, PipelineIssue{
				Severity: "high",
				Rule:     "overly-permissive",
				Message:  "permissions set to write-all; use least-privilege permissions",
				Line:     lineNum,
			})
		}

		// Check pull_request_target
		if strings.Contains(trimmed, "pull_request_target") {
			issues = append(issues, PipelineIssue{
				Severity: "high",
				Rule:     "pull-request-target",
				Message:  "pull_request_target can expose secrets to untrusted code",
				Line:     lineNum,
			})
		}

		// Check secrets in env vars
		if secretsInEnvLine(line, lines, i) {
			issues = append(issues, PipelineIssue{
				Severity: "medium",
				Rule:     "secret-in-env",
				Message:  "secret referenced in env block; consider using step-level env or masking",
				Line:     lineNum,
			})
		}
	}

	return issues
}

// secretsInEnvLine checks whether a line contains ${{ secrets.* }} and is
// within an env: block (detected by checking nearby context).
func secretsInEnvLine(line string, lines []string, idx int) bool {
	if !strings.Contains(line, "secrets.") {
		return false
	}
	// Walk backwards to see if we're inside an env: block
	for j := idx; j >= 0 && j >= idx-5; j-- {
		trimmed := strings.TrimSpace(lines[j])
		if trimmed == "env:" || strings.HasPrefix(trimmed, "env:") {
			return true
		}
		// Stop if we hit a different block keyword
		if strings.HasSuffix(trimmed, ":") && !strings.Contains(trimmed, "secrets") &&
			!strings.Contains(trimmed, "${{") && j != idx {
			break
		}
	}
	return false
}

func auditGitLabCI(_ string) []PipelineIssue {
	// GitLab CI audit: future rules can be added here.
	return nil
}

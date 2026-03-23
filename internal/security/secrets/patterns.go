// Package secrets provides output sanitization for the BlackCat agent — scanning
// tool output, file reads, and memory entries for secrets before they reach an
// LLM provider or external channel.
package secrets

import (
	"regexp"
	"sync"
)

// PatternMatch represents a single secret detected in text.
type PatternMatch struct {
	// PatternName is the human-readable name of the matched pattern (e.g. "aws_access_key").
	PatternName string
	// Value is the exact substring that was matched.
	Value string
	// Start is the byte offset of the first character of the match in the original text.
	Start int
	// End is the byte offset one past the last character of the match.
	End int
}

// compiledPattern pairs a name with a pre-compiled regular expression.
type compiledPattern struct {
	name string
	re   *regexp.Regexp
}

// patternRegistry holds all compiled detection patterns.
// Patterns are compiled once at package init time.
var (
	patternOnce     sync.Once
	patternRegistry []compiledPattern
)

// initPatterns compiles all detection regexps exactly once.
func initPatterns() {
	patternOnce.Do(func() {
		defs := []struct {
			name    string
			pattern string
		}{
			// AWS access key IDs: AKIA followed by exactly 16 uppercase letters/digits.
			// No trailing \b so keys embedded inside larger strings are also caught.
			{
				name:    "aws_access_key",
				pattern: `\bAKIA[A-Z0-9]{16}`,
			},
			// GitHub tokens: ghp_, gho_, ghs_, ghr_ followed by >= 30 alphanumeric/underscore chars.
			{
				name:    "github_token",
				pattern: `\bgh[posh]_[A-Za-z0-9_]{30,}\b`,
			},
			// GitHub refresh token: ghr_ prefix.
			{
				name:    "github_refresh_token",
				pattern: `\bghr_[A-Za-z0-9_]{30,}\b`,
			},
			// GitHub fine-grained PAT: github_pat_ prefix.
			{
				name:    "github_fine_grained_pat",
				pattern: `\bgithub_pat_[A-Za-z0-9_]{36,}\b`,
			},
			// Anthropic API keys: sk-ant- followed by >= 20 chars.
			{
				name:    "anthropic_api_key",
				pattern: `\bsk-ant-[A-Za-z0-9\-_]{20,}\b`,
			},
			// OpenAI project keys: sk-proj- followed by >= 20 chars.
			{
				name:    "openai_project_key",
				pattern: `\bsk-proj-[A-Za-z0-9\-_]{20,}\b`,
			},
			// Legacy OpenAI keys: sk- followed by >= 40 alphanumeric chars.
			// We rely on the length requirement (40+) to avoid colliding with
			// the shorter ant-/proj- variants above; Go regexp has no lookahead.
			{
				name:    "openai_legacy_key",
				pattern: `\bsk-[A-Za-z0-9]{40,}\b`,
			},
			// Slack tokens: xox[bpas]- followed by at least 20 chars.
			{
				name:    "slack_token",
				pattern: `\bxox[bpas]-[A-Za-z0-9\-]{20,}\b`,
			},
			// Private key PEM headers — RSA, EC, OpenSSH, PKCS8, etc.
			{
				name:    "private_key_header",
				pattern: `-----BEGIN (?:RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`,
			},
			// Database connection strings with embedded credentials.
			// Matches scheme://user:password@host (requires password portion).
			{
				name:    "connection_string",
				pattern: `(?i)\b(?:postgres(?:ql)?|mysql|mongodb(?:\+srv)?|redis|amqp)://[^:@\s]+:[^@\s]+@[^\s'"]+`,
			},
			// HTTP Authorization header with Bearer token (>= 20 chars).
			{
				name:    "bearer_token",
				pattern: `(?i)\bauthorization:\s*bearer\s+([A-Za-z0-9\-_\.]{20,})`,
			},
			// Generic api_key / apikey / api-key assignments with a long value.
			{
				name:    "generic_api_key",
				pattern: `(?i)\bapi[_\-]?key["']?\s*[:=]\s*["']?([A-Za-z0-9\-_\.\/\+]{20,})`,
			},
		}

		patternRegistry = make([]compiledPattern, 0, len(defs))
		for _, d := range defs {
			patternRegistry = append(patternRegistry, compiledPattern{
				name: d.name,
				re:   regexp.MustCompile(d.pattern),
			})
		}
	})
}

// DetectPatterns scans text for known secret patterns and returns all matches found.
// Patterns are compiled once at package initialisation so repeated calls are fast.
// The function is safe for concurrent use.
func DetectPatterns(text string) []PatternMatch {
	initPatterns()

	if text == "" {
		return nil
	}

	var matches []PatternMatch
	for _, p := range patternRegistry {
		locs := p.re.FindAllStringIndex(text, -1)
		for _, loc := range locs {
			matches = append(matches, PatternMatch{
				PatternName: p.name,
				Value:       text[loc[0]:loc[1]],
				Start:       loc[0],
				End:         loc[1],
			})
		}
	}
	return matches
}

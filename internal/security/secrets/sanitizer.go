package secrets

import (
	"regexp"
	"strings"
	"sync"
)

// SanitizeTarget describes who will receive the sanitized text, which controls
// how aggressively secrets are redacted.
type SanitizeTarget int

const (
	// TargetLLM is the most aggressive target — text is destined for an LLM
	// provider and must have all possible secrets removed.
	TargetLLM SanitizeTarget = iota
	// TargetChannel is text destined for an external messaging channel (Telegram,
	// Discord, Slack, WhatsApp).  Redaction level is high but may preserve some
	// contextual information.
	TargetChannel
	// TargetMemory is text being persisted to the vector memory store.  Secrets
	// must be removed to prevent future LLM exposure.
	TargetMemory
	// TargetUser is text shown directly to the local user.  Secrets are still
	// redacted but the output may be slightly less aggressive.
	TargetUser
)

// SanitizeResult contains the output of a Sanitize call.
type SanitizeResult struct {
	// Output is the sanitized text with secret values replaced by placeholders.
	Output string
	// Redacted is true if any substitutions were made.
	Redacted bool
	// Findings is a list of human-readable descriptions of what was redacted.
	Findings []string
}

// minRegisteredSecretLen is the minimum length a manually registered secret
// must have to be considered for redaction (prevents over-redaction of short tokens).
const minRegisteredSecretLen = 8

// Sanitizer is the top-level orchestrator that combines pattern detection,
// registered-secret replacement, and (for connection strings) additional
// context-aware scrubbing.
//
// Sanitizer is safe for concurrent use after construction.
type Sanitizer struct {
	mu      sync.RWMutex
	secrets []string // manually registered secret values
}

// NewSanitizer creates a new Sanitizer with default settings.
func NewSanitizer() *Sanitizer {
	return &Sanitizer{}
}

// RegisterSecret adds a known secret value to the sanitizer's registry.
// Future calls to Sanitize will redact occurrences of this value.
// Values shorter than minRegisteredSecretLen are silently ignored to
// prevent over-redaction of common short tokens.
func (s *Sanitizer) RegisterSecret(value string) {
	if len(value) < minRegisteredSecretLen {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.secrets = append(s.secrets, value)
}

// Sanitize scans text for secrets and returns a SanitizeResult.
// The input string is never modified; a new string is returned in Output.
func (s *Sanitizer) Sanitize(text string, target SanitizeTarget) SanitizeResult {
	if text == "" {
		return SanitizeResult{Output: text}
	}

	output := text
	var findings []string
	redacted := false

	// Phase 1: Redact registered (known) secrets.
	s.mu.RLock()
	registered := make([]string, len(s.secrets))
	copy(registered, s.secrets)
	s.mu.RUnlock()

	for _, secret := range registered {
		if strings.Contains(output, secret) {
			output = strings.ReplaceAll(output, secret, "[REDACTED:registered_secret]")
			findings = append(findings, "registered_secret")
			redacted = true
		}
	}

	// Phase 2: Redact pattern matches.
	// We re-scan on every call because the output may have been partially
	// modified in Phase 1.
	matches := DetectPatterns(output)
	if len(matches) > 0 {
		output = redactByPatternMatches(output, matches)
		for _, m := range matches {
			findings = append(findings, m.PatternName)
		}
		redacted = true
	}

	// Phase 3: For TargetLLM and TargetMemory apply additional connection-string
	// scrubbing in case the generic regex missed an edge case.
	if target == TargetLLM || target == TargetMemory {
		scrubbed, changed := scrubConnectionPasswords(output)
		if changed {
			output = scrubbed
			findings = append(findings, "connection_string_password")
			redacted = true
		}
	}

	return SanitizeResult{
		Output:   output,
		Redacted: redacted,
		Findings: deduplicateStrings(findings),
	}
}

// redactByPatternMatches replaces the matched substrings in text (from right to
// left to preserve offsets) with [REDACTED:<patternName>] placeholders.
func redactByPatternMatches(text string, matches []PatternMatch) string {
	// Sort descending by start offset so we can splice without offset drift.
	sorted := make([]PatternMatch, len(matches))
	copy(sorted, matches)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].Start > sorted[i].Start {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	var sb strings.Builder
	sb.Grow(len(text))

	pos := len(text)
	// We'll build the output right-to-left using the original text.
	segments := make([]string, 0, len(sorted)*2+1)
	for _, m := range sorted {
		if m.End > pos {
			// Overlapping match — skip.
			continue
		}
		// Append the tail segment after this match.
		segments = append(segments, text[m.End:pos])
		// Append the redaction placeholder.
		segments = append(segments, "[REDACTED:"+m.PatternName+"]")
		pos = m.Start
	}
	// Append the head (everything before the first match).
	segments = append(segments, text[:pos])

	// Reverse and join.
	for i := len(segments) - 1; i >= 0; i-- {
		sb.WriteString(segments[i])
	}
	return sb.String()
}

// connectionPasswordRe matches passwords embedded in DSN URLs.
// It captures the password portion (group 1) between ':' and '@'.
var connectionPasswordRe = regexp.MustCompile(
	`(?i)((?:postgres(?:ql)?|mysql|mongodb(?:\+srv)?|redis|amqp)://[^:@\s]+:)([^@\s]+)(@)`,
)

// scrubConnectionPasswords replaces password portions of DSN URLs with
// [REDACTED:dsn_password].  Returns the new string and whether a replacement
// was made.
func scrubConnectionPasswords(text string) (string, bool) {
	if !connectionPasswordRe.MatchString(text) {
		return text, false
	}
	result := connectionPasswordRe.ReplaceAllString(text, "${1}[REDACTED:dsn_password]${3}")
	return result, result != text
}

// deduplicateStrings returns a new slice with duplicate strings removed,
// preserving insertion order.
func deduplicateStrings(s []string) []string {
	if len(s) == 0 {
		return s
	}
	seen := make(map[string]bool, len(s))
	out := make([]string, 0, len(s))
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

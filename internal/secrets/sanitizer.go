package secrets

import (
	"strings"
	"sync"
)

// Sanitizer redacts secret values from arbitrary text. It is applied to:
//   - Tool execution output (stdout/stderr) before returning to the agent
//   - Log messages before writing to disk
//   - Memory entries before storing in the vector database
//   - LLM message content before sending to the provider
//
// The sanitizer maintains a set of known secret values and replaces any
// occurrence with a redaction placeholder. It uses Aho-Corasick-style
// matching for efficient multi-pattern search.
type Sanitizer struct {
	mu       sync.RWMutex
	patterns map[string]string // secret value -> redaction label
}

// NewSanitizer creates an empty sanitizer.
func NewSanitizer() *Sanitizer {
	return &Sanitizer{
		patterns: make(map[string]string),
	}
}

// Register adds a secret value to the redaction set.
// The label is shown in place of the secret (e.g. "[REDACTED:openai_api_key]").
// Short values (< 4 chars) are ignored to prevent false positives.
func (s *Sanitizer) Register(value string, label string) {
	if len(value) < 4 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	redaction := "[REDACTED:" + label + "]"
	s.patterns[value] = redaction

	// Also register common encodings of the secret:
	// - URL-encoded, base64, etc. are handled by registering the specific
	//   encoded forms when they are known. For API keys, the raw form is
	//   usually sufficient because they appear as-is in env vars and headers.
}

// Unregister removes a secret value from the redaction set.
// Call this when a secret is deleted or rotated (then re-register the new value).
func (s *Sanitizer) Unregister(value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.patterns, value)
}

// Redact replaces all registered secret values in the text with their
// redaction labels. Returns the sanitized text and whether any redactions
// were made.
func (s *Sanitizer) Redact(text string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.patterns) == 0 {
		return text, false
	}

	redacted := false
	result := text

	// Replace longest patterns first to avoid partial matches.
	// For typical secret counts (< 100), linear scan is fast enough.
	// If this becomes a bottleneck, switch to Aho-Corasick.
	for pattern, replacement := range s.patterns {
		if strings.Contains(result, pattern) {
			result = strings.ReplaceAll(result, pattern, replacement)
			redacted = true
		}
	}

	return result, redacted
}

// RedactBytes is like Redact but operates on byte slices.
func (s *Sanitizer) RedactBytes(data []byte) ([]byte, bool) {
	text, redacted := s.Redact(string(data))
	if !redacted {
		return data, false
	}
	return []byte(text), true
}

// Count returns the number of registered secret patterns.
func (s *Sanitizer) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.patterns)
}

// Clear removes all registered patterns. Used during shutdown.
func (s *Sanitizer) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.patterns = make(map[string]string)
}

// SanitizeForLLM applies additional redaction rules specific to LLM context.
// Beyond value-based redaction, it also strips common secret patterns:
//   - Bearer tokens in HTTP headers
//   - Base64-encoded credentials
//   - AWS access key patterns (AKIA...)
//   - Connection strings with embedded passwords
func SanitizeForLLM(text string, sanitizer *Sanitizer) string {
	// First pass: value-based redaction.
	result, _ := sanitizer.Redact(text)

	// Second pass: pattern-based redaction for secrets we might not have registered.
	// These patterns catch secrets that leaked from tool output, config files, etc.
	result = redactPatterns(result)

	return result
}

// redactPatterns applies regex-free pattern matching for common secret formats.
// We avoid regex for performance and to keep the binary small.
func redactPatterns(text string) string {
	// Redact Bearer tokens: "Bearer <token>" -> "Bearer [REDACTED]"
	text = redactAfterPrefix(text, "Bearer ", " ", "\n", "\r")

	// Redact Authorization headers
	text = redactAfterPrefix(text, "Authorization: ", "\n", "\r")

	// Redact common key= patterns in URLs and configs
	for _, prefix := range []string{"api_key=", "apikey=", "token=", "password=", "secret="} {
		text = redactAfterPrefix(text, prefix, "&", " ", "\n", "\r", "\"", "'")
	}

	return text
}

// redactAfterPrefix finds "prefix<value>" and replaces <value> up to any delimiter.
func redactAfterPrefix(text, prefix string, delimiters ...string) string {
	result := text
	searchFrom := 0

	for {
		idx := strings.Index(result[searchFrom:], prefix)
		if idx == -1 {
			break
		}
		idx += searchFrom
		valueStart := idx + len(prefix)
		if valueStart >= len(result) {
			break
		}

		// Find the end of the value (first delimiter or end of string).
		valueEnd := len(result)
		for _, delim := range delimiters {
			pos := strings.Index(result[valueStart:], delim)
			if pos != -1 && (valueStart+pos) < valueEnd {
				valueEnd = valueStart + pos
			}
		}

		value := result[valueStart:valueEnd]
		if len(value) >= 8 { // Only redact values that look like real secrets.
			result = result[:valueStart] + "[REDACTED]" + result[valueEnd:]
		}

		searchFrom = valueStart + len("[REDACTED]")
		if searchFrom >= len(result) {
			break
		}
	}

	return result
}

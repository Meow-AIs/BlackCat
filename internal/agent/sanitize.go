package agent

import "regexp"

// sanitizeForLLM is a stopgap sanitizer that scrubs the most critical secret
// patterns from tool output before it is added to the LLM message history.
// A full sanitizer will be wired in once internal/security/secrets is complete.
//
// Patterns covered:
//  1. Postgres/MySQL/Redis connection strings with embedded credentials
//     e.g. postgres://user:password@host/db  →  postgres://[REDACTED]@host/db
//  2. Bearer tokens in Authorization headers
//     e.g. Bearer eyJ…  →  Bearer [REDACTED]
//  3. api_key= query-string or form parameters
//     e.g. api_key=sk-abc123  →  api_key=[REDACTED]
var (
	// reConnString matches DSNs of the form scheme://user:pass@host…
	// Capture group 1: scheme+authority prefix before credentials.
	// Capture group 2: user:pass@ section to redact.
	// Capture group 3: remainder (host/path).
	reConnString = regexp.MustCompile(`(?i)([a-z][a-z0-9+\-.]*://)([^:@/\s]+:[^@\s]+@)([^\s]*)`)

	// reBearerToken matches "Bearer <token>" where the token is non-whitespace.
	reBearerToken = regexp.MustCompile(`(?i)(Bearer\s+)\S+`)

	// reAPIKeyParam matches api_key=<value> in query strings / form data.
	reAPIKeyParam = regexp.MustCompile(`(?i)(api[_-]key\s*=\s*)\S+`)
)

// sanitizeForLLM removes well-known secret patterns from text so they are not
// forwarded to an LLM provider. It returns a new string; the original is
// never modified (immutability requirement).
func sanitizeForLLM(text string) string {
	if text == "" {
		return text
	}

	// 1. Redact credentials in connection strings.
	text = reConnString.ReplaceAllString(text, "${1}[REDACTED]@${3}")

	// 2. Redact Bearer token values.
	text = reBearerToken.ReplaceAllString(text, "${1}[REDACTED]")

	// 3. Redact api_key parameter values.
	text = reAPIKeyParam.ReplaceAllString(text, "${1}[REDACTED]")

	return text
}

package remote

import (
	"regexp"
	"strings"
)

var (
	// ipPattern matches IPv4 addresses and replaces with first-octet.X.X.X
	ipPattern = regexp.MustCompile(`\b(\d{1,3})\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)

	// sshKeyPattern matches PEM-encoded private keys (RSA, EC, DSA, OpenSSH, etc.)
	sshKeyPattern = regexp.MustCompile(
		`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----`,
	)

	// bearerPattern matches Bearer tokens in headers
	bearerPattern = regexp.MustCompile(`(?i)(Bearer\s+)\S+`)

	// secretFieldPattern matches key=value or key: "value" patterns for
	// sensitive field names (password, token, api_key, secret).
	secretFieldPattern = regexp.MustCompile(
		`(?i)((?:password|token|api_key|secret)\s*[:=]\s*)"[^"]*"`,
	)

	// secretFieldBare matches unquoted key=value for sensitive fields.
	secretFieldBare = regexp.MustCompile(
		`(?i)((?:password|token|api_key|secret)\s*[:=]\s*)\S+`,
	)
)

// SanitizeRemoteOutput redacts sensitive information from remote command
// output. It removes IP addresses (keeping first octet), SSH private keys,
// bearer tokens, and common secret patterns.
func SanitizeRemoteOutput(output string) string {
	if output == "" {
		return ""
	}

	result := output

	// Redact SSH keys first (multi-line, most distinctive)
	result = sshKeyPattern.ReplaceAllString(result, "[REDACTED SSH KEY]")

	// Redact bearer tokens
	result = bearerPattern.ReplaceAllString(result, "${1}[REDACTED]")

	// Redact quoted secret fields
	result = secretFieldPattern.ReplaceAllString(result, "${1}[REDACTED]")

	// Redact bare secret fields (password=xxx, token=xxx)
	result = secretFieldBare.ReplaceAllString(result, "${1}[REDACTED]")

	// Redact IP addresses last (keep first octet)
	result = redactIPs(result)

	return result
}

// redactIPs replaces full IPv4 addresses with first-octet.X.X.X, but only
// for patterns that look like real IPs (each octet 0-255). We skip version
// numbers like "2.0" which don't have 4 octets.
func redactIPs(s string) string {
	return ipPattern.ReplaceAllStringFunc(s, func(match string) string {
		parts := strings.SplitN(match, ".", 4)
		if len(parts) != 4 {
			return match
		}
		return parts[0] + ".X.X.X"
	})
}

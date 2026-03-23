package secrets

import (
	"strings"
)

// blockedCommandRule defines a rule that, when matched, blocks execution.
type blockedCommandRule struct {
	// reason is returned to the caller when this rule fires.
	reason string
	// matcher reports whether the (lower-cased, trimmed) command matches.
	matcher func(cmd string) bool
}

// blockedRules is evaluated in order; the first match wins.
var blockedRules = []blockedCommandRule{
	// -----------------------------------------------------------------------
	// Environment enumeration commands.
	// -----------------------------------------------------------------------
	{
		reason: "dumps all environment variables including secrets",
		matcher: func(cmd string) bool {
			return cmd == "env" || strings.HasPrefix(cmd, "env ") || strings.HasPrefix(cmd, "env\t")
		},
	},
	{
		reason: "dumps all environment variables including secrets",
		matcher: func(cmd string) bool {
			return cmd == "printenv" || strings.HasPrefix(cmd, "printenv ")
		},
	},
	{
		reason: "dumps shell state including all environment variables",
		matcher: func(cmd string) bool {
			return cmd == "set"
		},
	},
	// -----------------------------------------------------------------------
	// Reading sensitive files directly.
	// -----------------------------------------------------------------------
	{
		reason: "reads SSH private key",
		matcher: func(cmd string) bool {
			return containsSensitiveFileRead(cmd, ".ssh/id_")
		},
	},
	{
		reason: "reads AWS credentials",
		matcher: func(cmd string) bool {
			return containsSensitiveFileRead(cmd, ".aws/credentials") ||
				containsSensitiveFileRead(cmd, ".aws/config")
		},
	},
	{
		reason: "reads dotenv file containing secrets",
		matcher: func(cmd string) bool {
			return matchesDotenvRead(cmd)
		},
	},
	{
		reason: "reads netrc file containing credentials",
		matcher: func(cmd string) bool {
			return containsSensitiveFileRead(cmd, ".netrc")
		},
	},
	// -----------------------------------------------------------------------
	// Echoing / printing environment variable expansions.
	// -----------------------------------------------------------------------
	{
		reason: "prints expanded environment variable that may contain a secret",
		matcher: func(cmd string) bool {
			return matchesSecretEcho(cmd)
		},
	},
	// -----------------------------------------------------------------------
	// Container / orchestration secret exposure.
	// -----------------------------------------------------------------------
	{
		reason: "docker inspect can expose container environment variables and secrets",
		matcher: func(cmd string) bool {
			return strings.HasPrefix(cmd, "docker inspect")
		},
	},
	{
		reason: "kubectl get/describe secret exposes Kubernetes secret contents",
		matcher: func(cmd string) bool {
			return matchesKubectlSecret(cmd)
		},
	},
}

// secretEnvVarPrefixes lists environment variable name patterns that commonly hold secrets.
// Used when detecting `echo $VAR` commands.
var secretEnvVarPrefixes = []string{
	"api_key", "api_secret",
	"secret", "password", "passwd", "token",
	"auth", "credential",
	"private_key", "private_token",
	"aws_secret", "aws_access",
	"openai", "anthropic",
	"github_token", "gitlab_token",
	"stripe", "sendgrid", "twilio",
	"db_pass", "database_pass",
	"jwt_",
}

// IsSecretExposingCommand reports whether a command is likely to expose secrets.
// It returns (blocked=true, reason) when blocked, or (false, "") when safe.
// Matching is case-insensitive.
func IsSecretExposingCommand(command string) (blocked bool, reason string) {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return false, ""
	}
	lower := strings.ToLower(trimmed)

	for _, rule := range blockedRules {
		if rule.matcher(lower) {
			return true, rule.reason
		}
	}
	return false, ""
}

// containsSensitiveFileRead reports whether cmd contains a read of a path segment
// (e.g. "cat ~/.ssh/id_rsa").
func containsSensitiveFileRead(cmd, pathSegment string) bool {
	if !strings.Contains(cmd, pathSegment) {
		return false
	}
	// Confirm it is a read command (cat, head, tail, less, more, tee, type).
	readCmds := []string{"cat ", "head ", "tail ", "less ", "more ", "tee ", "type "}
	for _, rc := range readCmds {
		if strings.HasPrefix(cmd, rc) || strings.Contains(cmd, " "+rc) || strings.Contains(cmd, "\t"+rc) {
			return true
		}
	}
	// Also match bare cat/head/etc. without trailing space when combined with pipe.
	return false
}

// matchesDotenvRead reports whether cmd reads a .env file.
func matchesDotenvRead(cmd string) bool {
	// Matches: cat .env, cat .env.production, cat /app/.env.staging, etc.
	for _, rc := range []string{"cat ", "head ", "tail ", "less ", "more ", "type "} {
		if !strings.HasPrefix(cmd, rc) {
			continue
		}
		rest := strings.TrimSpace(cmd[len(rc):])
		// Iterate over all space-separated tokens (handles flags like -n).
		for _, token := range strings.Fields(rest) {
			// Extract the basename of the token.
			base := token
			if idx := strings.LastIndexAny(token, "/\\"); idx >= 0 {
				base = token[idx+1:]
			}
			// Strip leading ~ from the whole token when no separator was found.
			base = strings.TrimLeft(base, "~")
			if base == ".env" || strings.HasPrefix(base, ".env.") {
				return true
			}
		}
	}
	return false
}

// matchesSecretEcho reports whether cmd is an echo/print of a secret env var.
func matchesSecretEcho(cmd string) bool {
	if !strings.HasPrefix(cmd, "echo ") && !strings.HasPrefix(cmd, "printf ") {
		return false
	}
	rest := cmd[strings.Index(cmd, " ")+1:]
	// Look for $VAR or ${VAR} patterns.
	lower := strings.ToLower(rest)
	for _, prefix := range secretEnvVarPrefixes {
		if strings.Contains(lower, "$"+prefix) ||
			strings.Contains(lower, "${"+prefix) {
			return true
		}
	}
	return false
}

// matchesKubectlSecret reports whether cmd is a kubectl command targeting secrets.
func matchesKubectlSecret(cmd string) bool {
	if !strings.HasPrefix(cmd, "kubectl ") {
		return false
	}
	lower := cmd
	return (strings.Contains(lower, " get ") || strings.Contains(lower, " describe ")) &&
		strings.Contains(lower, "secret")
}

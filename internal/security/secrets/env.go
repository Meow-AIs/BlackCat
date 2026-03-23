package secrets

import "strings"

// safeVars is the explicit allow-list of environment variable names that are
// safe to pass through without inspection.
var safeVars = map[string]bool{
	"PATH":        true,
	"HOME":        true,
	"USER":        true,
	"LOGNAME":     true,
	"LANG":        true,
	"LC_ALL":      true,
	"LC_CTYPE":    true,
	"GOPATH":      true,
	"GOROOT":      true,
	"GOMODCACHE":  true,
	"GOCACHE":     true,
	"GOENV":       true,
	"TERM":        true,
	"TERM_PROGRAM": true,
	"COLORTERM":   true,
	"SHELL":       true,
	"PWD":         true,
	"OLDPWD":      true,
	"TMPDIR":      true,
	"TEMP":        true,
	"TMP":         true,
	"XDG_RUNTIME_DIR": true,
	"XDG_DATA_DIRS":   true,
	"XDG_CONFIG_DIRS": true,
	"DISPLAY":     true,
	"WAYLAND_DISPLAY": true,
	"DBUS_SESSION_BUS_ADDRESS": true,
	// Blackcat non-secret config.
	"BLACKCAT_MODEL":  true,
	"BLACKCAT_LOG":    true,
	"BLACKCAT_CONFIG": true,
}

// secretVarSubstrings are substrings that, when found in an upper-cased variable
// name, indicate the variable likely holds a secret.  Checked after the safe-list
// so that e.g. PATH is never blocked.
var secretVarSubstrings = []string{
	"SECRET",
	"PASSWORD",
	"PASSWD",
	"API_KEY",
	"APIKEY",
	"API_SECRET",
	"TOKEN",
	"AUTH",
	"CREDENTIAL",
	"PRIVATE_KEY",
	"PRIVATE_TOKEN",
	"ACCESS_KEY",
	"AWS_",
	"OPENAI_",
	"ANTHROPIC_",
	"GITHUB_TOKEN",
	"GITLAB_TOKEN",
	"NPM_TOKEN",
	"STRIPE_",
	"SENDGRID_",
	"TWILIO_",
	"DB_PASS",
	"DATABASE_PASS",
	"REDIS_PASS",
	"POSTGRES_PASS",
	"MYSQL_",
	"JWT_",
}

// FilterEnvironment returns a new slice containing only the environment entries
// from env that are considered safe to expose to sub-processes or LLM providers.
//
// env must be a slice of "KEY=value" strings as returned by os.Environ().
// The input slice is never mutated.
func FilterEnvironment(env []string) []string {
	if len(env) == 0 {
		return []string{}
	}

	result := make([]string, 0, len(env))
	for _, entry := range env {
		// Entries without '=' are malformed; include them unchanged to avoid
		// information loss and let the caller decide.
		eqIdx := strings.IndexByte(entry, '=')
		if eqIdx < 0 {
			result = append(result, entry)
			continue
		}

		name := entry[:eqIdx]
		upper := strings.ToUpper(name)

		// BLACKCAT_SECRET_* is always stripped.
		if strings.HasPrefix(upper, "BLACKCAT_SECRET_") {
			continue
		}

		// If the name is on the explicit safe list, keep it.
		if safeVars[upper] {
			result = append(result, entry)
			continue
		}

		// If the name contains any known-secret substring, strip it.
		if containsSecretSubstring(upper) {
			continue
		}

		// Default: keep unknown vars (they may be needed by the process).
		result = append(result, entry)
	}
	return result
}

// containsSecretSubstring reports whether varName (already upper-cased) contains
// any of the substrings that indicate a secret variable.
func containsSecretSubstring(upper string) bool {
	for _, sub := range secretVarSubstrings {
		if strings.Contains(upper, sub) {
			return true
		}
	}
	return false
}

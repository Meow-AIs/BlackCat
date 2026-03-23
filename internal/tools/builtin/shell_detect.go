package builtin

import (
	"regexp"
	"strings"
)

// interactiveCommands lists commands that are typically interactive when run
// without specific flags or arguments.
var interactiveCommands = map[string]bool{
	"ssh":        true,
	"python":     true,
	"python3":    true,
	"node":       true,
	"irb":        true,
	"mysql":      true,
	"psql":       true,
	"mongo":      true,
	"mongosh":    true,
	"redis-cli":  true,
	"sqlite3":    true,
	"vim":        true,
	"vi":         true,
	"nano":       true,
	"less":       true,
	"more":       true,
	"top":        true,
	"htop":       true,
	"ftp":        true,
	"sftp":       true,
	"telnet":     true,
	"nslookup":   true,
	"bash":       true,
	"sh":         true,
	"zsh":        true,
	"fish":       true,
	"powershell": true,
	"pwsh":       true,
}

// interpreterCommands are commands that become non-interactive when given a
// script file or -c/-e flag.
var interpreterCommands = map[string]bool{
	"python":  true,
	"python3": true,
	"node":    true,
	"bash":    true,
	"sh":      true,
	"zsh":     true,
	"fish":    true,
}

// nonInteractiveFlags maps commands to flags that make them non-interactive.
var nonInteractiveFlags = map[string][]string{
	"python":  {"-c", "-m"},
	"python3": {"-c", "-m"},
	"node":    {"-e", "--eval", "-p", "--print"},
	"bash":    {"-c"},
	"sh":      {"-c"},
	"zsh":     {"-c"},
	"mysql":   {"-e", "--execute"},
	"psql":    {"-c", "--command"},
}

// IsInteractiveCommand detects if a command will likely require interactive input.
func IsInteractiveCommand(command string) bool {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false
	}

	baseCmd := parts[0]

	// Check for docker exec -it / kubectl exec -it patterns
	if (baseCmd == "docker" || baseCmd == "kubectl") && len(parts) >= 2 && parts[1] == "exec" {
		for _, p := range parts[2:] {
			if p == "-it" || p == "-ti" || p == "--interactive" {
				return true
			}
		}
		return false
	}

	// Not a known interactive command
	if !interactiveCommands[baseCmd] {
		return false
	}

	// For interpreter commands, check if they have a script arg or non-interactive flag
	if interpreterCommands[baseCmd] && len(parts) > 1 {
		// Check for non-interactive flags
		if flags, ok := nonInteractiveFlags[baseCmd]; ok {
			for _, arg := range parts[1:] {
				for _, flag := range flags {
					if arg == flag {
						return false
					}
				}
			}
		}

		// If the interpreter has an argument that doesn't start with '-',
		// it's likely a script file
		for _, arg := range parts[1:] {
			if !strings.HasPrefix(arg, "-") {
				return false
			}
		}
	}

	// For mysql/psql with non-interactive flags
	if (baseCmd == "mysql" || baseCmd == "psql") && len(parts) > 1 {
		if flags, ok := nonInteractiveFlags[baseCmd]; ok {
			for _, arg := range parts[1:] {
				for _, flag := range flags {
					if arg == flag {
						return false
					}
				}
			}
		}
	}

	return true
}

// suggestions maps base commands to non-interactive alternatives.
var suggestions = map[string]string{
	"ssh":        "ssh user@host 'command'",
	"python":     "python -c 'code' or python script.py",
	"python3":    "python3 -c 'code' or python3 script.py",
	"node":       "node -e 'code' or node script.js",
	"mysql":      "mysql -e 'query'",
	"psql":       "psql -c 'query'",
	"mongo":      "mongosh --eval 'query'",
	"mongosh":    "mongosh --eval 'query'",
	"redis-cli":  "redis-cli command [args]",
	"sqlite3":    "sqlite3 db 'query'",
	"bash":       "bash -c 'command'",
	"sh":         "sh -c 'command'",
	"zsh":        "zsh -c 'command'",
	"fish":       "fish -c 'command'",
	"powershell": "powershell -Command 'command'",
	"pwsh":       "pwsh -Command 'command'",
	"ftp":        "curl or wget for file transfers",
	"sftp":       "scp for file transfers",
	"telnet":     "curl or nc for network testing",
	"nslookup":   "nslookup -type=A domain or dig domain",
	"irb":        "ruby -e 'code'",
}

// editorCommands should use the file edit tool instead.
var editorCommands = map[string]bool{
	"vim":  true,
	"vi":   true,
	"nano": true,
}

// pagerCommands should use the file read tool instead.
var pagerCommands = map[string]bool{
	"less": true,
	"more": true,
}

// monitorCommands have no simple non-interactive alternative.
var monitorCommands = map[string]bool{
	"top":  true,
	"htop": true,
}

// SuggestNonInteractive returns a suggestion for a non-interactive alternative.
// Returns empty string if the command is not known to be interactive.
func SuggestNonInteractive(command string) string {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return ""
	}

	baseCmd := parts[0]

	if !interactiveCommands[baseCmd] {
		return ""
	}

	if editorCommands[baseCmd] {
		return "Use the file edit tool instead of " + baseCmd
	}

	if pagerCommands[baseCmd] {
		return "Use the file read tool instead of " + baseCmd
	}

	if monitorCommands[baseCmd] {
		return baseCmd + " is a non-interactive monitor; consider using ps, vmstat, or similar one-shot commands"
	}

	if s, ok := suggestions[baseCmd]; ok {
		return "Try: " + s
	}

	return "Consider running " + baseCmd + " with appropriate flags for non-interactive use"
}

// promptPatterns are regex patterns that detect interactive prompts in output.
var promptPatterns = []*regexp.Regexp{
	regexp.MustCompile(`>>>\s*$`),                           // Python REPL
	regexp.MustCompile(`mysql>\s*$`),                         // MySQL
	regexp.MustCompile(`postgres=>\s*$`),                     // PostgreSQL
	regexp.MustCompile(`(?:^|\n)[#$]\s+$`),                   // Shell prompt
	regexp.MustCompile(`(?:^|\n)>\s+$`),                      // Node.js / generic
	regexp.MustCompile(`irb\([^)]*\):\d+:\d+>\s*$`),         // Ruby IRB
	regexp.MustCompile(`(?i)(?:enter\s+)?password\s*:`),      // Password prompt
	regexp.MustCompile(`\[(?:[Yy]/[Nn]|[Yy]\/[Nn])\]`),     // Y/n confirmation
	regexp.MustCompile(`(?i)press\s+(?:any\s+key|enter)`),   // Press any key
}

// DetectPromptPattern checks if the output contains patterns indicating
// the program is waiting for user input.
func DetectPromptPattern(output string) bool {
	for _, pat := range promptPatterns {
		if pat.MatchString(output) {
			return true
		}
	}
	return false
}

package security

import (
	"regexp"
	"strings"
)

// RiskLevel classifies how risky a command is.
type RiskLevel string

const (
	RiskSafe      RiskLevel = "safe"
	RiskModerate  RiskLevel = "moderate"
	RiskDangerous RiskLevel = "dangerous"
)

// RiskAssessment is the result of analyzing a command's risk.
type RiskAssessment struct {
	Command  string
	Level    RiskLevel
	Category string
	Reason   string
}

type riskRule struct {
	pattern  *regexp.Regexp
	level    RiskLevel
	category string
	reason   string
}

// safeCommands that are always safe regardless of arguments.
var safeCommands = map[string]bool{
	"ls": true, "cat": true, "head": true, "tail": true,
	"grep": true, "rg": true, "find": true, "echo": true,
	"pwd": true, "which": true, "whoami": true, "wc": true,
	"tree": true, "env": true, "printenv": true, "date": true,
	"uname": true, "hostname": true, "id": true, "df": true,
	"du": true, "free": true, "uptime": true, "file": true,
	"stat": true, "readlink": true, "basename": true, "dirname": true,
	"sort": true, "uniq": true, "diff": true, "less": true,
	"more": true, "man": true, "help": true, "type": true,
}

var dangerousRules []riskRule
var moderateRules []riskRule

func init() {
	dangerousRules = []riskRule{
		{regexp.MustCompile(`(?i)\brm\s+.*-[a-zA-Z]*r[a-zA-Z]*f|rm\s+.*-[a-zA-Z]*f[a-zA-Z]*r`), RiskDangerous, "filesystem", "recursive forced deletion can destroy data"},
		{regexp.MustCompile(`(?i)\bmkfs\b`), RiskDangerous, "filesystem", "filesystem format destroys all data on device"},
		{regexp.MustCompile(`(?i)\bdd\s+if=`), RiskDangerous, "filesystem", "dd can overwrite entire devices"},
		{regexp.MustCompile(`(?i)\bchmod\s+777\b`), RiskDangerous, "filesystem", "world-writable permissions are a security risk"},
		{regexp.MustCompile(`(?i)\bchmod\s+-[a-zA-Z]*R`), RiskDangerous, "filesystem", "recursive permission changes affect entire trees"},
		{regexp.MustCompile(`(?i)\bdrop\s+table\b`), RiskDangerous, "database", "drops an entire database table"},
		{regexp.MustCompile(`(?i)\bdrop\s+database\b`), RiskDangerous, "database", "drops an entire database"},
		{regexp.MustCompile(`(?i)\btruncate\s+table\b`), RiskDangerous, "database", "truncates all rows from a table"},
		{regexp.MustCompile(`(?i)\bkubectl\s+delete\s+(namespace|ns)\b`), RiskDangerous, "kubernetes", "deletes an entire Kubernetes namespace"},
		{regexp.MustCompile(`(?i)\bkubectl\s+delete\s+-f\b`), RiskDangerous, "kubernetes", "deletes resources from a manifest file"},
		{regexp.MustCompile(`(?i)\bgit\s+push\s+.*--force`), RiskDangerous, "git", "force push can overwrite remote history"},
		{regexp.MustCompile(`(?i)\bgit\s+reset\s+--hard`), RiskDangerous, "git", "hard reset discards all uncommitted changes"},
		{regexp.MustCompile(`(?i)\bterraform\s+destroy\b`), RiskDangerous, "infrastructure", "destroys all managed infrastructure"},
		{regexp.MustCompile(`(?i)\bkill\s+-9\b`), RiskDangerous, "process", "SIGKILL cannot be caught; process killed immediately"},
		{regexp.MustCompile(`(?i)\bpkill\b`), RiskDangerous, "process", "pkill can terminate multiple processes by pattern"},
		{regexp.MustCompile(`(?i)\bkillall\b`), RiskDangerous, "process", "killall terminates all processes by name"},
	}

	moderateRules = []riskRule{
		{regexp.MustCompile(`(?i)\bgit\s+push\b`), RiskModerate, "git", "pushes commits to remote repository"},
		{regexp.MustCompile(`(?i)\bdocker\s+rm\b`), RiskModerate, "container", "removes a Docker container"},
		{regexp.MustCompile(`(?i)\bdocker\s+rmi\b`), RiskModerate, "container", "removes a Docker image"},
		{regexp.MustCompile(`(?i)\bnpm\s+publish\b`), RiskModerate, "package", "publishes package to npm registry"},
		{regexp.MustCompile(`(?i)\bcargo\s+publish\b`), RiskModerate, "package", "publishes crate to crates.io"},
		{regexp.MustCompile(`(?i)\bcurl\s+.*-X\s+(POST|PUT|DELETE|PATCH)\b`), RiskModerate, "network", "sends mutating HTTP request"},
		{regexp.MustCompile(`(?i)\bsed\s+-i\b`), RiskModerate, "filesystem", "in-place file modification"},
		{regexp.MustCompile(`(?i)\bawk\s+-i\s+inplace\b`), RiskModerate, "filesystem", "in-place file modification with awk"},
	}
}

// safeGitSubcommands that don't modify state.
var safeGitSubcommands = map[string]bool{
	"status": true, "log": true, "diff": true, "show": true,
	"branch": true, "remote": true, "tag": true, "stash": true,
	"blame": true, "shortlog": true, "describe": true, "rev-parse": true,
	"ls-files": true, "ls-tree": true, "config": true,
}

// safeBuildCommands that are read-only or local-only.
var safeBuildCommands = map[string]bool{
	"go":    true,
	"cargo": true,
	"make":  true,
	"npm":   true,
	"pnpm":  true,
	"yarn":  true,
	"node":  true,
	"tsc":   true,
	"rustc": true,
	"gcc":   true,
	"clang": true,
	"javac": true,
	"mvn":   true,
}

// AssessCommandRisk evaluates a shell command and returns its risk level.
func AssessCommandRisk(cmd string) RiskAssessment {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return RiskAssessment{Command: cmd, Level: RiskSafe}
	}

	// If the command contains chains (&&, ||, ;, |), assess each part and return worst.
	if strings.ContainsAny(cmd, "&|;") {
		return assessChainedCommand(cmd)
	}

	return assessSingleCommand(cmd)
}

func assessChainedCommand(cmd string) RiskAssessment {
	// Split on &&, ||, ;, |
	parts := splitChained(cmd)
	worst := RiskAssessment{Command: cmd, Level: RiskSafe}
	for _, part := range parts {
		a := assessSingleCommand(strings.TrimSpace(part))
		if riskOrd(a.Level) > riskOrd(worst.Level) {
			worst.Level = a.Level
			worst.Category = a.Category
			worst.Reason = a.Reason
		}
	}
	worst.Command = cmd
	return worst
}

func splitChained(cmd string) []string {
	// Simple split on &&, ||, ;, |
	var parts []string
	current := ""
	for i := 0; i < len(cmd); i++ {
		ch := cmd[i]
		if ch == ';' {
			parts = append(parts, current)
			current = ""
		} else if ch == '|' {
			if i+1 < len(cmd) && cmd[i+1] == '|' {
				parts = append(parts, current)
				current = ""
				i++ // skip second |
			} else {
				parts = append(parts, current)
				current = ""
			}
		} else if ch == '&' {
			if i+1 < len(cmd) && cmd[i+1] == '&' {
				parts = append(parts, current)
				current = ""
				i++ // skip second &
			} else {
				current += string(ch)
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func riskOrd(l RiskLevel) int {
	switch l {
	case RiskSafe:
		return 0
	case RiskModerate:
		return 1
	case RiskDangerous:
		return 2
	}
	return 1
}

func assessSingleCommand(cmd string) RiskAssessment {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return RiskAssessment{Command: cmd, Level: RiskSafe}
	}

	// Check dangerous rules first
	for _, r := range dangerousRules {
		if r.pattern.MatchString(cmd) {
			return RiskAssessment{
				Command:  cmd,
				Level:    RiskDangerous,
				Category: r.category,
				Reason:   r.reason,
			}
		}
	}

	// Check moderate rules
	for _, r := range moderateRules {
		if r.pattern.MatchString(cmd) {
			return RiskAssessment{
				Command:  cmd,
				Level:    RiskModerate,
				Category: r.category,
				Reason:   r.reason,
			}
		}
	}

	// Extract the base command
	baseCmd := extractBaseCommand(cmd)

	// Check safe commands
	if safeCommands[baseCmd] {
		return RiskAssessment{Command: cmd, Level: RiskSafe}
	}

	// Check git subcommands
	if baseCmd == "git" {
		return assessGitCommand(cmd)
	}

	// Check curl without mutating method
	if baseCmd == "curl" {
		return RiskAssessment{Command: cmd, Level: RiskSafe, Category: "network"}
	}

	// Check safe build commands
	if safeBuildCommands[baseCmd] {
		return RiskAssessment{Command: cmd, Level: RiskSafe}
	}

	// Unknown command defaults to moderate
	return RiskAssessment{
		Command:  cmd,
		Level:    RiskModerate,
		Category: "unknown",
		Reason:   "unrecognized command; requires review",
	}
}

func extractBaseCommand(cmd string) string {
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return ""
	}
	base := fields[0]
	// Strip path prefix
	if idx := strings.LastIndex(base, "/"); idx >= 0 {
		base = base[idx+1:]
	}
	return strings.ToLower(base)
}

func assessGitCommand(cmd string) RiskAssessment {
	fields := strings.Fields(cmd)
	if len(fields) < 2 {
		return RiskAssessment{Command: cmd, Level: RiskSafe, Category: "git"}
	}
	sub := strings.ToLower(fields[1])
	if safeGitSubcommands[sub] {
		return RiskAssessment{Command: cmd, Level: RiskSafe, Category: "git"}
	}
	// git add, git commit are safe-ish
	if sub == "add" || sub == "commit" || sub == "fetch" || sub == "pull" || sub == "clone" || sub == "init" || sub == "checkout" || sub == "switch" || sub == "merge" || sub == "rebase" {
		return RiskAssessment{Command: cmd, Level: RiskSafe, Category: "git"}
	}
	return RiskAssessment{
		Command:  cmd,
		Level:    RiskModerate,
		Category: "git",
		Reason:   "git subcommand may modify state",
	}
}

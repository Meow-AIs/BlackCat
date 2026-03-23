package security

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// defaultDenyPatterns are regex patterns for commands that should never run.
var defaultDenyPatterns = []string{
	`(?i)\brm\s+(-[a-z]*f[a-z]*\s+)?-[a-z]*r[a-z]*\s+/`,  // rm -rf /
	`(?i)\brm\s+(-[a-z]*r[a-z]*\s+)?-[a-z]*f[a-z]*\s+/`,  // rm -fr /
	`(?i)\bmkfs\b`,                                          // mkfs (any variant)
	`(?i)\bdd\s+if=`,                                        // dd if=
	`(?i)\bformat\b`,                                        // format
	`(?i)\bshutdown\b`,                                      // shutdown
	`(?i)\breboot\b`,                                        // reboot
}

// pathTraversalPattern detects suspicious path traversal sequences.
var pathTraversalPattern = regexp.MustCompile(`\.\./\.\./`)

// CommandValidator validates shell commands before execution.
type CommandValidator struct {
	DenyPatterns  []string
	AllowPaths    []string
	MaxArgLength  int
	compiledDeny  []*regexp.Regexp
}

// NewCommandValidator creates a validator with default deny patterns.
func NewCommandValidator() *CommandValidator {
	v := &CommandValidator{
		DenyPatterns: make([]string, len(defaultDenyPatterns)),
		MaxArgLength: 10000,
	}
	copy(v.DenyPatterns, defaultDenyPatterns)
	v.compileDenyPatterns()
	return v
}

// compileDenyPatterns compiles all deny patterns into regex objects.
func (v *CommandValidator) compileDenyPatterns() {
	v.compiledDeny = make([]*regexp.Regexp, 0, len(v.DenyPatterns))
	for _, pattern := range v.DenyPatterns {
		compiled, err := regexp.Compile(pattern)
		if err == nil {
			v.compiledDeny = append(v.compiledDeny, compiled)
		}
	}
}

// Validate checks if a command is safe to execute in the given working directory.
// Returns nil if the command is allowed, or an error explaining why it was denied.
func (v *CommandValidator) Validate(command string, workDir string) error {
	if command == "" {
		return fmt.Errorf("command must not be empty")
	}

	// Check max argument length
	if v.MaxArgLength > 0 && len(command) > v.MaxArgLength {
		return fmt.Errorf("command exceeds maximum length of %d characters", v.MaxArgLength)
	}

	// Check deny patterns
	for _, pattern := range v.compiledDeny {
		if pattern.MatchString(command) {
			return fmt.Errorf("command matches deny pattern: %s", pattern.String())
		}
	}

	// Check for path traversal in command
	if pathTraversalPattern.MatchString(command) {
		return fmt.Errorf("command contains suspicious path traversal sequence")
	}

	// Check working directory against allowed paths
	if len(v.AllowPaths) > 0 && workDir != "" {
		absWorkDir, err := filepath.Abs(workDir)
		if err != nil {
			return fmt.Errorf("cannot resolve working directory: %w", err)
		}
		absWorkDir = filepath.Clean(absWorkDir)

		allowed := false
		for _, allowPath := range v.AllowPaths {
			absAllow, err := filepath.Abs(allowPath)
			if err != nil {
				continue
			}
			absAllow = filepath.Clean(absAllow)

			if absWorkDir == absAllow || strings.HasPrefix(absWorkDir, absAllow+string(filepath.Separator)) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("working directory %q is not in allowed paths", workDir)
		}
	}

	return nil
}

// AddDenyPattern adds a regex pattern to the deny list.
func (v *CommandValidator) AddDenyPattern(pattern string) {
	v.DenyPatterns = append(v.DenyPatterns, pattern)
	compiled, err := regexp.Compile(pattern)
	if err == nil {
		v.compiledDeny = append(v.compiledDeny, compiled)
	}
}

// AddAllowPath adds a directory to the list of allowed working directories.
func (v *CommandValidator) AddAllowPath(path string) {
	v.AllowPaths = append(v.AllowPaths, path)
}

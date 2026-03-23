package security

import (
	"path/filepath"
	"strings"
)

// PermissionChecker evaluates actions against configured permission rules.
type PermissionChecker struct {
	rules []PermissionRule
}

// NewPermissionChecker creates a checker with default rules.
func NewPermissionChecker() *PermissionChecker {
	pc := &PermissionChecker{}
	pc.addDefaults()
	return pc
}

func (pc *PermissionChecker) addDefaults() {
	// Level 1: Always allow — safe read-only operations
	for _, action := range []ActionType{ActionReadFile, ActionListDir, ActionSearchCode} {
		pc.rules = append(pc.rules, PermissionRule{
			Action: action,
			Level:  LevelAllow,
		})
	}

	// Allow git read commands
	pc.rules = append(pc.rules, PermissionRule{
		Action:   ActionShell,
		Patterns: []string{"git status*", "git log*", "git diff*", "git branch*", "git show*"},
		Level:    LevelAllow,
	})

	// Level 4: Deny — dangerous commands (checked before other shell rules)
	pc.rules = append(pc.rules, PermissionRule{
		Action:   ActionShell,
		Patterns: []string{"rm -rf /*", ":(){ :|:& };:*", "mkfs*", "rm -rf .*"},
		Level:    LevelDeny,
	})

	// Deny writing sensitive files
	pc.rules = append(pc.rules, PermissionRule{
		Action:   ActionWriteFile,
		Patterns: []string{"*.env", "*.key", "*.pem", "*.secret*", "credentials*"},
		Level:    LevelDeny,
	})

	// Level 3: Ask — default for shell and write_file
	pc.rules = append(pc.rules, PermissionRule{
		Action: ActionShell,
		Level:  LevelAsk,
	})
	pc.rules = append(pc.rules, PermissionRule{
		Action: ActionWriteFile,
		Level:  LevelAsk,
	})
	pc.rules = append(pc.rules, PermissionRule{
		Action: ActionWeb,
		Level:  LevelAsk,
	})
}

func (pc *PermissionChecker) Check(action Action) Decision {
	target := actionTarget(action)

	// Pass 1: Check deny rules first (highest priority)
	for _, rule := range pc.rules {
		if rule.Action != action.Type || rule.Level != LevelDeny {
			continue
		}
		if matchesRule(rule, target) {
			return Decision{Level: LevelDeny, Allowed: false, Reason: "blocked by deny rule"}
		}
	}

	// Pass 2: Check allow rules
	for _, rule := range pc.rules {
		if rule.Action != action.Type || rule.Level != LevelAllow {
			continue
		}
		if matchesRule(rule, target) {
			return Decision{Level: LevelAllow, Allowed: true, Reason: "allowed by default"}
		}
	}

	// Pass 3: Check auto_approve rules
	for _, rule := range pc.rules {
		if rule.Action != action.Type || rule.Level != LevelAutoApprove {
			continue
		}
		if matchesRule(rule, target) && !matchesExcludes(rule, target) {
			return Decision{Level: LevelAutoApprove, Allowed: true, Reason: "auto-approved by pattern"}
		}
	}

	// Pass 4: Check ask rules (fallback)
	for _, rule := range pc.rules {
		if rule.Action != action.Type || rule.Level != LevelAsk {
			continue
		}
		if matchesRule(rule, target) {
			return Decision{Level: LevelAsk, Allowed: false, Reason: "requires user confirmation"}
		}
	}

	// Default: ask
	return Decision{Level: LevelAsk, Allowed: false, Reason: "no matching rule, defaulting to ask"}
}

func (pc *PermissionChecker) AddRule(rule PermissionRule) {
	// Insert custom rules before the default fallback rules
	// Deny rules go to the front, others before fallback ask rules
	switch rule.Level {
	case LevelDeny:
		pc.rules = append([]PermissionRule{rule}, pc.rules...)
	default:
		// Insert before the last ask/default rules
		insertIdx := len(pc.rules)
		for i, r := range pc.rules {
			if r.Level == LevelAsk && len(r.Patterns) == 0 {
				insertIdx = i
				break
			}
		}
		pc.rules = append(pc.rules[:insertIdx], append([]PermissionRule{rule}, pc.rules[insertIdx:]...)...)
	}
}

func (pc *PermissionChecker) Rules() []PermissionRule {
	result := make([]PermissionRule, len(pc.rules))
	copy(result, pc.rules)
	return result
}

// actionTarget returns the string to match against patterns.
func actionTarget(action Action) string {
	switch action.Type {
	case ActionShell:
		return action.Command
	case ActionWriteFile, ActionReadFile:
		return action.Path
	default:
		return action.Command + action.Path
	}
}

// matchesRule checks if a target string matches any of the rule's patterns.
// If no patterns, it matches all targets of that action type.
func matchesRule(rule PermissionRule, target string) bool {
	if len(rule.Patterns) == 0 {
		return true
	}
	for _, pattern := range rule.Patterns {
		if matched, _ := filepath.Match(pattern, target); matched {
			return true
		}
		// Also try prefix match for commands like "go test ./..."
		if strings.HasSuffix(pattern, "*") {
			prefix := strings.TrimSuffix(pattern, "*")
			if strings.HasPrefix(target, prefix) {
				return true
			}
		}
	}
	return false
}

// matchesExcludes checks if a target matches any exclude pattern.
func matchesExcludes(rule PermissionRule, target string) bool {
	for _, pattern := range rule.Excludes {
		if matched, _ := filepath.Match(pattern, target); matched {
			return true
		}
		// Check just the filename for patterns like "*.env"
		base := filepath.Base(target)
		if matched, _ := filepath.Match(pattern, base); matched {
			return true
		}
	}
	return false
}

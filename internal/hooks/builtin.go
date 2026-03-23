package hooks

import (
	"fmt"
	"regexp"
	"strings"
)

// RegisterBuiltinHooks registers all pre-configured builtin hooks.
func RegisterBuiltinHooks(engine *Engine) {
	registerSafetyGuard(engine)
	registerOutputSanitizer(engine)
	registerCostWarning(engine)
	registerSessionLogger(engine)
	registerPermissionAudit(engine)
	registerMemoryQuality(engine)
}

// registerSafetyGuard blocks extremely dangerous commands.
func registerSafetyGuard(engine *Engine) {
	dangerous := []string{
		"rm -rf /",
		"rm -rf ~",
		"mkfs",
		"dd if=/dev/zero of=/dev/sda",
	}

	engine.Register(EventBeforeTool, "safety-guard", PriorityFirst, func(ctx HookContext) HookResult {
		cmd, ok := ctx.Data["command"].(string)
		if !ok {
			return HookResult{Allow: true}
		}

		for _, pattern := range dangerous {
			if matchesDangerous(cmd, pattern) {
				return HookResult{
					Allow:   false,
					Message: fmt.Sprintf("Blocked by safety-guard: %q matches dangerous pattern %q", cmd, pattern),
				}
			}
		}

		return HookResult{Allow: true}
	})
}

// matchesDangerous checks if a command matches a dangerous pattern.
// For "rm -rf /" and "rm -rf ~", we need to ensure these target root/home
// and not subdirectories like "rm -rf /tmp/test".
func matchesDangerous(cmd, pattern string) bool {
	switch pattern {
	case "rm -rf /":
		// Match "rm -rf /" but not "rm -rf /tmp/test".
		return matchesRmRfRoot(cmd)
	case "rm -rf ~":
		return strings.Contains(cmd, "rm -rf ~") &&
			!strings.Contains(cmd, "rm -rf ~/")
	default:
		return strings.Contains(cmd, pattern)
	}
}

// matchesRmRfRoot checks for "rm -rf /" targeting filesystem root.
func matchesRmRfRoot(cmd string) bool {
	// Match patterns like "rm -rf /", "rm -rf / ", "sudo rm -rf /"
	// but NOT "rm -rf /tmp" or "rm -rf /var/log".
	re := regexp.MustCompile(`rm\s+-rf\s+/(\s|$)`)
	return re.MatchString(cmd)
}

// registerOutputSanitizer redacts secrets from tool output.
func registerOutputSanitizer(engine *Engine) {
	secretPatterns := []*regexp.Regexp{
		// API keys and tokens (sk-..., ghp_..., etc.).
		regexp.MustCompile(`(?i)(api[_-]?key|token|secret|password|passwd|pwd)\s*[=:]\s*\S+`),
		// Bearer tokens.
		regexp.MustCompile(`(?i)Bearer\s+\S+`),
		// Generic long hex/base64 secrets.
		regexp.MustCompile(`(?i)(sk|ghp|gho|ghu|ghs|ghr|glpat|xox[bps])-[A-Za-z0-9_\-]{20,}`),
	}

	engine.Register(EventAfterTool, "output-sanitizer", HookPriority(10), func(ctx HookContext) HookResult {
		output, ok := ctx.Data["output"].(string)
		if !ok {
			return HookResult{Allow: true}
		}

		redacted := output
		modified := false
		for _, re := range secretPatterns {
			if re.MatchString(redacted) {
				redacted = re.ReplaceAllString(redacted, "[REDACTED]")
				modified = true
			}
		}

		if !modified {
			return HookResult{Allow: true}
		}

		return HookResult{
			Allow:    true,
			Modified: map[string]any{"output": redacted},
		}
	})
}

// registerCostWarning warns when session cost exceeds threshold.
func registerCostWarning(engine *Engine) {
	const costThreshold = 1.0

	engine.Register(EventAfterResponse, "cost-warning", PriorityNormal, func(ctx HookContext) HookResult {
		cost, ok := toFloat64(ctx.Data["cost"])
		if !ok {
			return HookResult{Allow: true}
		}

		if cost > costThreshold {
			return HookResult{
				Allow:   true,
				Message: fmt.Sprintf("Cost warning: session cost $%.2f exceeds threshold $%.2f", cost, costThreshold),
			}
		}

		return HookResult{Allow: true}
	})
}

// registerSessionLogger logs session summary at session end.
func registerSessionLogger(engine *Engine) {
	engine.Register(EventSessionEnd, "session-logger", PriorityLast, func(ctx HookContext) HookResult {
		toolCount := ctx.Data["tool_count"]
		duration := ctx.Data["duration"]
		errs := ctx.Data["errors"]

		return HookResult{
			Allow:   true,
			Message: fmt.Sprintf("Session ended: tools=%v, duration=%v, errors=%v", toolCount, duration, errs),
		}
	})
}

// registerPermissionAudit logs all permission decisions.
func registerPermissionAudit(engine *Engine) {
	engine.Register(EventPermissionAsk, "permission-audit", PriorityLast, func(ctx HookContext) HookResult {
		tool := ctx.Data["tool"]
		action := ctx.Data["action"]

		return HookResult{
			Allow:   true,
			Message: fmt.Sprintf("Permission requested: tool=%v, action=%v", tool, action),
		}
	})
}

// registerMemoryQuality filters low-quality memory entries.
func registerMemoryQuality(engine *Engine) {
	lowQualityEntries := map[string]bool{
		"ok":      true,
		"thanks":  true,
		"yes":     true,
		"no":      true,
		"sure":    true,
		"k":       true,
		"ty":      true,
		"thx":     true,
	}

	engine.Register(EventMemoryStore, "memory-quality", PriorityNormal, func(ctx HookContext) HookResult {
		content, ok := ctx.Data["content"].(string)
		if !ok {
			return HookResult{Allow: true}
		}

		trimmed := strings.TrimSpace(strings.ToLower(content))

		if len(trimmed) < 10 {
			return HookResult{
				Allow:   false,
				Message: fmt.Sprintf("Memory filtered: content too short (%d chars)", len(trimmed)),
			}
		}

		if lowQualityEntries[trimmed] {
			return HookResult{
				Allow:   false,
				Message: fmt.Sprintf("Memory filtered: low-quality entry %q", trimmed),
			}
		}

		return HookResult{Allow: true}
	})
}

// toFloat64 converts various numeric types to float64.
func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	default:
		return 0, false
	}
}

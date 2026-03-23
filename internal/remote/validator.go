package remote

import (
	"errors"
	"fmt"
	"strings"
)

// ValidateRemoteCommand checks whether a command is permitted given the
// profile's RemotePerms. It enforces deny-list, allow-list, and exec
// permission checks. Deny rules take precedence over allow rules.
func ValidateRemoteCommand(cmd string, perms RemotePerms) error {
	trimmed := strings.TrimSpace(cmd)
	if trimmed == "" {
		return errors.New("command cannot be empty")
	}

	if !perms.AllowExec {
		return errors.New("command execution is not allowed for this profile")
	}

	// Deny list check (substring match, case-insensitive). Deny takes precedence.
	cmdLower := strings.ToLower(trimmed)
	for _, denied := range perms.DeniedCmds {
		if strings.Contains(cmdLower, strings.ToLower(denied)) {
			return fmt.Errorf("command contains denied pattern: %q", denied)
		}
	}

	// Allow list check (whitelist mode: if AllowedCmds is non-empty, cmd must
	// start with one of them).
	if len(perms.AllowedCmds) > 0 {
		allowed := false
		for _, prefix := range perms.AllowedCmds {
			if strings.HasPrefix(trimmed, prefix) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("command %q is not in the allowed commands list", trimmed)
		}
	}

	return nil
}

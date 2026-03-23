package remote

import (
	"testing"
)

func TestValidateRemoteCommand_AllowedByDefault(t *testing.T) {
	perms := RemotePerms{AllowExec: true}
	err := ValidateRemoteCommand("ls -la", perms)
	if err != nil {
		t.Errorf("expected allowed, got error: %v", err)
	}
}

func TestValidateRemoteCommand_ExecDisabled(t *testing.T) {
	perms := RemotePerms{AllowExec: false}
	err := ValidateRemoteCommand("ls", perms)
	if err == nil {
		t.Error("expected error when exec is disabled")
	}
}

func TestValidateRemoteCommand_DeniedCmd_SubstringMatch(t *testing.T) {
	perms := RemotePerms{
		AllowExec:  true,
		DeniedCmds: []string{"rm -rf", "DROP", "shutdown"},
	}

	tests := []struct {
		cmd     string
		blocked bool
	}{
		{"rm -rf /", true},
		{"sudo rm -rf /tmp", true},
		{"DROP TABLE users", true},
		{"echo DROP something", true},
		{"shutdown -h now", true},
		{"ls -la", false},
		{"cat /var/log/syslog", false},
		{"rm file.txt", false}, // "rm" alone is not "rm -rf"
	}

	for _, tt := range tests {
		err := ValidateRemoteCommand(tt.cmd, perms)
		if tt.blocked && err == nil {
			t.Errorf("expected %q to be blocked", tt.cmd)
		}
		if !tt.blocked && err != nil {
			t.Errorf("expected %q to be allowed, got error: %v", tt.cmd, err)
		}
	}
}

func TestValidateRemoteCommand_AllowedCmds_WhitelistMode(t *testing.T) {
	perms := RemotePerms{
		AllowExec:   true,
		AllowedCmds: []string{"ls", "cat", "kubectl get"},
	}

	tests := []struct {
		cmd     string
		allowed bool
	}{
		{"ls -la /tmp", true},
		{"cat /etc/hosts", true},
		{"kubectl get pods", true},
		{"rm file", false},
		{"wget http://evil.com", false},
	}

	for _, tt := range tests {
		err := ValidateRemoteCommand(tt.cmd, perms)
		if tt.allowed && err != nil {
			t.Errorf("expected %q to be allowed, got error: %v", tt.cmd, err)
		}
		if !tt.allowed && err == nil {
			t.Errorf("expected %q to be blocked in whitelist mode", tt.cmd)
		}
	}
}

func TestValidateRemoteCommand_EmptyCommand(t *testing.T) {
	perms := RemotePerms{AllowExec: true}
	err := ValidateRemoteCommand("", perms)
	if err == nil {
		t.Error("expected error for empty command")
	}
}

func TestValidateRemoteCommand_WhitespaceOnlyCommand(t *testing.T) {
	perms := RemotePerms{AllowExec: true}
	err := ValidateRemoteCommand("   ", perms)
	if err == nil {
		t.Error("expected error for whitespace-only command")
	}
}

func TestValidateRemoteCommand_DenyTakesPrecedenceOverAllow(t *testing.T) {
	perms := RemotePerms{
		AllowExec:   true,
		AllowedCmds: []string{"rm"},
		DeniedCmds:  []string{"rm -rf"},
	}

	// "rm file" is allowed (in whitelist, not denied)
	if err := ValidateRemoteCommand("rm file.txt", perms); err != nil {
		t.Errorf("expected 'rm file.txt' to be allowed: %v", err)
	}

	// "rm -rf /" is denied (deny takes precedence)
	if err := ValidateRemoteCommand("rm -rf /", perms); err == nil {
		t.Error("expected 'rm -rf /' to be denied")
	}
}

func TestValidateRemoteCommand_CaseInsensitiveDeny(t *testing.T) {
	perms := RemotePerms{
		AllowExec:  true,
		DeniedCmds: []string{"DROP"},
	}

	// "drop" should match "DROP" (case-insensitive)
	if err := ValidateRemoteCommand("drop table users", perms); err == nil {
		t.Error("expected case-insensitive deny match")
	}
}

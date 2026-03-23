package secrets

import "testing"

// TestIsSecretExposingCommand_Blocked verifies commands that expose secrets are blocked.
func TestIsSecretExposingCommand_Blocked(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{"bare env", "env"},
		{"env with flags", "env -0"},
		{"printenv", "printenv"},
		{"printenv specific var", "printenv AWS_SECRET_ACCESS_KEY"},
		{"cat ssh private key", "cat ~/.ssh/id_rsa"},
		{"cat dotenv", "cat .env"},
		{"cat dotenv production", "cat .env.production"},
		{"echo secret env var", "echo $API_KEY"},
		{"echo secret env var double-quoted", `echo "$OPENAI_API_KEY"`},
		{"echo AWS secret", "echo $AWS_SECRET_ACCESS_KEY"},
		{"docker inspect container", "docker inspect my_container"},
		{"docker inspect with format", "docker inspect --format '{{.Config.Env}}' app"},
		{"kubectl get secret", "kubectl get secret my-secret"},
		{"kubectl get secret with output", "kubectl get secret -o yaml"},
		{"kubectl describe secret", "kubectl describe secret my-secret"},
		{"set command (dumps env on bash)", "set"},
		{"cat aws credentials", "cat ~/.aws/credentials"},
		{"cat netrc", "cat ~/.netrc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocked, reason := IsSecretExposingCommand(tt.command)
			if !blocked {
				t.Errorf("IsSecretExposingCommand(%q) blocked=false, want true", tt.command)
			}
			if reason == "" {
				t.Errorf("IsSecretExposingCommand(%q) reason is empty, want descriptive reason", tt.command)
			}
		})
	}
}

// TestIsSecretExposingCommand_Allowed verifies safe commands are not blocked.
func TestIsSecretExposingCommand_Allowed(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{"ls", "ls -la"},
		{"git status", "git status"},
		{"git log", "git log --oneline"},
		{"go build", "go build ./..."},
		{"go test", "go test ./..."},
		{"make build", "make build"},
		{"cat regular file", "cat main.go"},
		{"cat readme", "cat README.md"},
		{"echo hello", "echo hello"},
		{"echo static string", "echo 'hello world'"},
		{"grep in source", "grep -r TODO ./internal"},
		{"docker ps", "docker ps"},
		{"kubectl get pods", "kubectl get pods"},
		{"curl public url", "curl https://example.com/api"},
		{"npm install", "npm install"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocked, _ := IsSecretExposingCommand(tt.command)
			if blocked {
				t.Errorf("IsSecretExposingCommand(%q) blocked=true, want false", tt.command)
			}
		})
	}
}

// TestIsSecretExposingCommand_ReasonNonEmpty ensures blocked commands always carry a reason.
func TestIsSecretExposingCommand_ReasonNonEmpty(t *testing.T) {
	blocked, reason := IsSecretExposingCommand("env")
	if !blocked {
		t.Fatal("expected env to be blocked")
	}
	if reason == "" {
		t.Error("expected non-empty reason for blocked command")
	}
}

// TestIsSecretExposingCommand_EmptyInput handles empty/whitespace without panic.
func TestIsSecretExposingCommand_EmptyInput(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("IsSecretExposingCommand(\"\") panicked: %v", r)
		}
	}()
	blocked, _ := IsSecretExposingCommand("")
	if blocked {
		t.Error("empty command should not be blocked")
	}
}

// TestIsSecretExposingCommand_WhitespaceOnly handles whitespace-only input.
func TestIsSecretExposingCommand_WhitespaceOnly(t *testing.T) {
	blocked, _ := IsSecretExposingCommand("   ")
	if blocked {
		t.Error("whitespace-only command should not be blocked")
	}
}

// TestIsSecretExposingCommand_CaseInsensitive verifies case-insensitive matching for known commands.
func TestIsSecretExposingCommand_CaseInsensitive(t *testing.T) {
	tests := []struct {
		command string
	}{
		{"ENV"},
		{"Env"},
		{"PRINTENV"},
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			blocked, _ := IsSecretExposingCommand(tt.command)
			if !blocked {
				t.Errorf("IsSecretExposingCommand(%q) should be blocked (case-insensitive)", tt.command)
			}
		})
	}
}

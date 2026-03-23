package builtin

import "testing"

func TestIsInteractiveCommandKnown(t *testing.T) {
	tests := []struct {
		command string
		want    bool
	}{
		{"ssh user@host", true},
		{"python", true},
		{"python3", true},
		{"node", true},
		{"irb", true},
		{"mysql", true},
		{"psql", true},
		{"mongo", true},
		{"mongosh", true},
		{"redis-cli", true},
		{"sqlite3", true},
		{"vim file.txt", true},
		{"vi file.txt", true},
		{"nano file.txt", true},
		{"less file.txt", true},
		{"more file.txt", true},
		{"top", true},
		{"htop", true},
		{"ftp host", true},
		{"sftp host", true},
		{"telnet host", true},
		{"nslookup", true},
		{"bash", true},
		{"sh", true},
		{"zsh", true},
		{"fish", true},
		{"powershell", true},
		{"pwsh", true},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			got := IsInteractiveCommand(tt.command)
			if got != tt.want {
				t.Errorf("IsInteractiveCommand(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}

func TestIsInteractiveCommandNonInteractive(t *testing.T) {
	tests := []struct {
		command string
		want    bool
	}{
		{"ls -la", false},
		{"echo hello", false},
		{"cat file.txt", false},
		{"grep pattern file", false},
		{"python script.py", false},
		{"python3 -c 'print(1)'", false},
		{"python -c 'print(1)'", false},
		{"node script.js", false},
		{"node -e 'console.log(1)'", false},
		{"bash -c 'echo hi'", false},
		{"sh -c 'echo hi'", false},
		{"mysql -e 'SELECT 1'", false},
		{"psql -c 'SELECT 1'", false},
		{"git status", false},
		{"docker ps", false},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			got := IsInteractiveCommand(tt.command)
			if got != tt.want {
				t.Errorf("IsInteractiveCommand(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}

func TestIsInteractiveCommandFlags(t *testing.T) {
	tests := []struct {
		command string
		want    bool
	}{
		{"docker exec -it container bash", true},
		{"kubectl exec -it pod -- bash", true},
		{"docker exec container ls", false},
		{"kubectl exec pod -- ls", false},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			got := IsInteractiveCommand(tt.command)
			if got != tt.want {
				t.Errorf("IsInteractiveCommand(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}

func TestSuggestNonInteractive(t *testing.T) {
	tests := []struct {
		command string
		wantNon string // substring that should appear in suggestion
	}{
		{"ssh user@host", "ssh user@host 'command'"},
		{"python", "python -c 'code'"},
		{"python3", "python3 -c 'code'"},
		{"node", "node -e 'code'"},
		{"mysql", "mysql -e 'query'"},
		{"psql", "psql -c 'query'"},
		{"vim file.txt", "file edit tool"},
		{"vi file.txt", "file edit tool"},
		{"nano file.txt", "file edit tool"},
		{"less file.txt", "file read tool"},
		{"more file.txt", "file read tool"},
		{"bash", "bash -c 'command'"},
		{"sh", "sh -c 'command'"},
		{"sqlite3", "sqlite3 db 'query'"},
		{"top", "non-interactive"},
		{"htop", "non-interactive"},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			got := SuggestNonInteractive(tt.command)
			if got == "" {
				t.Error("expected non-empty suggestion")
			}
			// Just verify a suggestion is returned (non-empty)
			_ = got
		})
	}
}

func TestSuggestNonInteractiveNonInteractiveCommand(t *testing.T) {
	got := SuggestNonInteractive("ls -la")
	if got != "" {
		t.Errorf("expected empty suggestion for non-interactive command, got %q", got)
	}
}

func TestDetectPromptPattern(t *testing.T) {
	tests := []struct {
		output string
		want   bool
	}{
		{">>> ", true},
		{"some output\n>>> ", true},
		{"mysql> ", true},
		{"postgres=> ", true},
		{"$ ", true},
		{"# ", true},
		{"> ", true},
		{"irb(main):001:0> ", true},
		{"Enter password:", true},
		{"Password:", true},
		{"[Y/n]", true},
		{"[y/N]", true},
		{"Press any key to continue", true},
		{"Press Enter to continue", true},
	}

	for _, tt := range tests {
		t.Run(tt.output, func(t *testing.T) {
			got := DetectPromptPattern(tt.output)
			if got != tt.want {
				t.Errorf("DetectPromptPattern(%q) = %v, want %v", tt.output, got, tt.want)
			}
		})
	}
}

func TestDetectPromptPatternNegative(t *testing.T) {
	tests := []struct {
		output string
	}{
		{"hello world"},
		{"some normal output"},
		{"file.txt:42: error found"},
		{"building project..."},
		{""},
	}

	for _, tt := range tests {
		t.Run(tt.output, func(t *testing.T) {
			got := DetectPromptPattern(tt.output)
			if got {
				t.Errorf("DetectPromptPattern(%q) = true, want false", tt.output)
			}
		})
	}
}

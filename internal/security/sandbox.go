package security

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// sensitiveEnvSubstrings is the list of substrings that, when found in an
// environment variable name (case-insensitive), cause it to be stripped before
// passing the environment to a sandboxed child process.
var sensitiveEnvSubstrings = []string{
	"SECRET",
	"KEY",
	"TOKEN",
	"PASSWORD",
	"PASSWD",
	"PASS",
	"CREDENTIAL",
	"AUTH",
	"API_",
	"PRIVATE",
	"SIGNING",
}

// safeEnvPrefixes lists prefixes whose variables are always safe to forward
// regardless of whether the name contains a sensitive substring.
var safeEnvPrefixes = []string{
	"PATH",
	"HOME",
	"USER",
	"LANG",
	"LC_",
	"TERM",
	"SHELL",
	"EDITOR",
	"TMPDIR",
	"TMP",
	"TEMP",
	"XDG_",
	"GOPATH",
	"GOROOT",
}

// filterEnvironment returns a copy of os.Environ() with sensitive variables
// removed.  A variable is stripped when its name (uppercased) contains any of
// the sensitiveEnvSubstrings, unless the name also matches a safeEnvPrefixes
// entry — but note that none of the safe prefixes overlap with the sensitive
// substrings in practice, so the safe prefix list is kept as an explicit
// allow-list override.
func filterEnvironment() []string {
	raw := os.Environ()
	filtered := make([]string, 0, len(raw))

	for _, entry := range raw {
		// Split on the first '=' to extract the variable name.
		eqIdx := strings.IndexByte(entry, '=')
		if eqIdx < 0 {
			// Malformed entry — skip it.
			continue
		}
		name := entry[:eqIdx]
		upper := strings.ToUpper(name)

		// Check explicit safe prefixes first so they are never stripped.
		safe := false
		for _, prefix := range safeEnvPrefixes {
			if strings.HasPrefix(upper, prefix) {
				safe = true
				break
			}
		}
		if safe {
			filtered = append(filtered, entry)
			continue
		}

		// Strip if the name contains any sensitive keyword.
		sensitive := false
		for _, substr := range sensitiveEnvSubstrings {
			if strings.Contains(upper, substr) {
				sensitive = true
				break
			}
		}
		if !sensitive {
			filtered = append(filtered, entry)
		}
	}

	return filtered
}

// SandboxConfig controls sandbox behavior.
type SandboxConfig struct {
	Timeout        time.Duration
	MaxOutputBytes int
	WorkDir        string // if empty, uses current directory
}

// DefaultSandboxConfig returns safe defaults.
func DefaultSandboxConfig() SandboxConfig {
	return SandboxConfig{
		Timeout:        120 * time.Second,
		MaxOutputBytes: 1024 * 1024, // 1MB
	}
}

// SandboxResult is the output of a sandboxed execution.
type SandboxResult struct {
	Output   string
	ExitCode int
	TimedOut bool
}

// Sandbox executes commands with isolation and resource limits.
type Sandbox struct {
	config SandboxConfig
}

// NewSandbox creates a sandbox with the given config.
func NewSandbox(config SandboxConfig) *Sandbox {
	return &Sandbox{config: config}
}

// Execute runs a shell command within the sandbox constraints.
func (s *Sandbox) Execute(ctx context.Context, command string) (SandboxResult, error) {
	ctx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/c", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	if s.config.WorkDir != "" {
		cmd.Dir = s.config.WorkDir
	}

	// Use a filtered environment so that secrets from the parent process are
	// not leaked to the sandboxed child process.
	cmd.Env = filterEnvironment()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	timedOut := ctx.Err() == context.DeadlineExceeded

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
			} else {
				exitCode = exitErr.ExitCode()
			}
		} else if timedOut {
			exitCode = -1
		} else {
			return SandboxResult{}, fmt.Errorf("execute command: %w", err)
		}
	}

	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" && !strings.HasSuffix(output, "\n") {
			output += "\n"
		}
		output += stderr.String()
	}

	// Truncate output if too large
	if s.config.MaxOutputBytes > 0 && len(output) > s.config.MaxOutputBytes {
		output = output[:s.config.MaxOutputBytes] + "\n... (output truncated)"
	}

	return SandboxResult{
		Output:   output,
		ExitCode: exitCode,
		TimedOut: timedOut,
	}, nil
}

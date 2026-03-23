package builtin

import (
	"context"
	"fmt"
	"time"

	"github.com/meowai/blackcat/internal/security"
	"github.com/meowai/blackcat/internal/tools"
)

// ShellTool executes shell commands through a sandbox.
type ShellTool struct {
	sandbox *security.Sandbox
	checker *security.PermissionChecker
}

// NewShellTool creates a shell tool with the given sandbox and permission checker.
func NewShellTool(checker *security.PermissionChecker) *ShellTool {
	return &ShellTool{
		sandbox: security.NewSandbox(security.SandboxConfig{
			Timeout:        120 * time.Second,
			MaxOutputBytes: 1024 * 1024,
		}),
		checker: checker,
	}
}

func (t *ShellTool) Info() tools.Definition {
	return tools.Definition{
		Name:        "execute",
		Description: "Execute a shell command (sandboxed)",
		Category:    "shell",
		Parameters: []tools.Parameter{
			{Name: "command", Type: "string", Description: "Shell command to execute", Required: true},
		},
	}
}

func (t *ShellTool) Execute(ctx context.Context, args map[string]any) (tools.Result, error) {
	command, err := requireStringArg(args, "command")
	if err != nil {
		return tools.Result{}, err
	}

	if t.checker != nil {
		decision := t.checker.Check(security.Action{
			Type:    security.ActionShell,
			Command: command,
		})
		if decision.Level == security.LevelDeny {
			return tools.Result{
				Error:    fmt.Sprintf("command denied: %s", decision.Reason),
				ExitCode: -1,
			}, nil
		}
	}

	result, err := t.sandbox.Execute(ctx, command)
	if err != nil {
		return tools.Result{Error: err.Error(), ExitCode: -1}, nil
	}

	return tools.Result{
		Output:   result.Output,
		ExitCode: result.ExitCode,
	}, nil
}

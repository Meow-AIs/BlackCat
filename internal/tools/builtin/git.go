package builtin

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/meowai/blackcat/internal/tools"
)

// runGit executes a git command in the given directory and returns the combined output.
func runGit(ctx context.Context, dir string, args ...string) (string, int, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return "", -1, err
		}
	}

	output := stdout.String()
	if stderr.Len() > 0 && exitCode != 0 {
		if output != "" && !strings.HasSuffix(output, "\n") {
			output += "\n"
		}
		output += stderr.String()
	}

	return output, exitCode, nil
}

// --- GitStatusTool ---

// GitStatusTool returns the porcelain status of a git repo.
type GitStatusTool struct{}

// NewGitStatusTool creates a new git status tool.
func NewGitStatusTool() *GitStatusTool { return &GitStatusTool{} }

func (t *GitStatusTool) Info() tools.Definition {
	return tools.Definition{
		Name:        "git_status",
		Description: "Show the working tree status (porcelain format)",
		Category:    "git",
		Parameters: []tools.Parameter{
			{Name: "path", Type: "string", Description: "Path to the git repository", Required: true},
		},
	}
}

func (t *GitStatusTool) Execute(ctx context.Context, args map[string]any) (tools.Result, error) {
	path, err := requireStringArg(args, "path")
	if err != nil {
		return tools.Result{}, err
	}

	output, exitCode, err := runGit(ctx, path, "status", "--porcelain")
	if err != nil {
		return tools.Result{Error: err.Error(), ExitCode: -1}, nil
	}
	return tools.Result{Output: output, ExitCode: exitCode}, nil
}

// --- GitDiffTool ---

// GitDiffTool returns the diff output of a git repo.
type GitDiffTool struct{}

// NewGitDiffTool creates a new git diff tool.
func NewGitDiffTool() *GitDiffTool { return &GitDiffTool{} }

func (t *GitDiffTool) Info() tools.Definition {
	return tools.Definition{
		Name:        "git_diff",
		Description: "Show changes in the working tree or staging area",
		Category:    "git",
		Parameters: []tools.Parameter{
			{Name: "path", Type: "string", Description: "Path to the git repository", Required: true},
			{Name: "staged", Type: "boolean", Description: "Show staged changes (default: false)"},
		},
	}
}

func (t *GitDiffTool) Execute(ctx context.Context, args map[string]any) (tools.Result, error) {
	path, err := requireStringArg(args, "path")
	if err != nil {
		return tools.Result{}, err
	}

	gitArgs := []string{"diff"}
	if staged, ok := args["staged"].(bool); ok && staged {
		gitArgs = append(gitArgs, "--cached")
	}

	output, exitCode, err := runGit(ctx, path, gitArgs...)
	if err != nil {
		return tools.Result{Error: err.Error(), ExitCode: -1}, nil
	}
	return tools.Result{Output: output, ExitCode: exitCode}, nil
}

// --- GitLogTool ---

// GitLogTool returns the oneline log of a git repo.
type GitLogTool struct{}

// NewGitLogTool creates a new git log tool.
func NewGitLogTool() *GitLogTool { return &GitLogTool{} }

func (t *GitLogTool) Info() tools.Definition {
	return tools.Definition{
		Name:        "git_log",
		Description: "Show commit log in oneline format",
		Category:    "git",
		Parameters: []tools.Parameter{
			{Name: "path", Type: "string", Description: "Path to the git repository", Required: true},
			{Name: "count", Type: "integer", Description: "Number of commits to show (default: 10)"},
		},
	}
}

func (t *GitLogTool) Execute(ctx context.Context, args map[string]any) (tools.Result, error) {
	path, err := requireStringArg(args, "path")
	if err != nil {
		return tools.Result{}, err
	}

	count := 10
	if c, ok := args["count"]; ok {
		switch v := c.(type) {
		case int:
			count = v
		case float64:
			count = int(v)
		case string:
			if parsed, parseErr := strconv.Atoi(v); parseErr == nil {
				count = parsed
			}
		}
	}

	output, exitCode, err := runGit(ctx, path, "log", "--oneline", fmt.Sprintf("-%d", count))
	if err != nil {
		return tools.Result{Error: err.Error(), ExitCode: -1}, nil
	}
	return tools.Result{Output: output, ExitCode: exitCode}, nil
}

// --- GitCommitTool ---

// GitCommitTool stages all changes and commits with a message.
type GitCommitTool struct{}

// NewGitCommitTool creates a new git commit tool.
func NewGitCommitTool() *GitCommitTool { return &GitCommitTool{} }

func (t *GitCommitTool) Info() tools.Definition {
	return tools.Definition{
		Name:        "git_commit",
		Description: "Stage all changes and commit with a message",
		Category:    "git",
		Parameters: []tools.Parameter{
			{Name: "path", Type: "string", Description: "Path to the git repository", Required: true},
			{Name: "message", Type: "string", Description: "Commit message", Required: true},
		},
	}
}

func (t *GitCommitTool) Execute(ctx context.Context, args map[string]any) (tools.Result, error) {
	path, err := requireStringArg(args, "path")
	if err != nil {
		return tools.Result{}, err
	}
	message, err := requireStringArg(args, "message")
	if err != nil {
		return tools.Result{}, err
	}

	// Stage all changes
	_, exitCode, err := runGit(ctx, path, "add", "-A")
	if err != nil {
		return tools.Result{Error: err.Error(), ExitCode: -1}, nil
	}
	if exitCode != 0 {
		return tools.Result{Error: "git add failed", ExitCode: exitCode}, nil
	}

	// Commit — on Windows use cmd to avoid shell quoting issues
	var commitOutput string
	if runtime.GOOS == "windows" {
		commitOutput, exitCode, err = runGit(ctx, path, "commit", "-m", message)
	} else {
		commitOutput, exitCode, err = runGit(ctx, path, "commit", "-m", message)
	}
	if err != nil {
		return tools.Result{Error: err.Error(), ExitCode: -1}, nil
	}
	return tools.Result{Output: commitOutput, ExitCode: exitCode}, nil
}

// --- GitBranchTool ---

// GitBranchTool lists or creates branches.
type GitBranchTool struct{}

// NewGitBranchTool creates a new git branch tool.
func NewGitBranchTool() *GitBranchTool { return &GitBranchTool{} }

func (t *GitBranchTool) Info() tools.Definition {
	return tools.Definition{
		Name:        "git_branch",
		Description: "List or create branches",
		Category:    "git",
		Parameters: []tools.Parameter{
			{Name: "path", Type: "string", Description: "Path to the git repository", Required: true},
			{Name: "name", Type: "string", Description: "Branch name (for create)"},
			{Name: "create", Type: "boolean", Description: "Create the branch (default: false)"},
		},
	}
}

func (t *GitBranchTool) Execute(ctx context.Context, args map[string]any) (tools.Result, error) {
	path, err := requireStringArg(args, "path")
	if err != nil {
		return tools.Result{}, err
	}

	create, _ := args["create"].(bool)

	if create {
		name, _ := args["name"].(string)
		if name == "" {
			return tools.Result{
				Error:    "branch name is required when create is true",
				ExitCode: 1,
			}, nil
		}
		output, exitCode, err := runGit(ctx, path, "branch", name)
		if err != nil {
			return tools.Result{Error: err.Error(), ExitCode: -1}, nil
		}
		return tools.Result{Output: output, ExitCode: exitCode}, nil
	}

	// List branches
	output, exitCode, err := runGit(ctx, path, "branch", "--list")
	if err != nil {
		return tools.Result{Error: err.Error(), ExitCode: -1}, nil
	}
	return tools.Result{Output: output, ExitCode: exitCode}, nil
}

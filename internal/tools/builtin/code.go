package builtin

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/meowai/blackcat/internal/tools"
)

// runCommand executes a command in the given directory and returns the output.
func runCommand(ctx context.Context, dir string, name string, args ...string) (string, int) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return err.Error(), -1
		}
	}

	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" && !strings.HasSuffix(output, "\n") {
			output += "\n"
		}
		output += stderr.String()
	}

	return output, exitCode
}

// resolveLanguage returns the language string, defaulting to "go".
func resolveLanguage(args map[string]any) string {
	lang, ok := args["language"].(string)
	if !ok || lang == "" {
		return "go"
	}
	return strings.ToLower(lang)
}

// --- CodeAnalyzeTool ---

// CodeAnalyzeTool runs static analysis on code.
type CodeAnalyzeTool struct{}

// NewCodeAnalyzeTool creates a new code analysis tool.
func NewCodeAnalyzeTool() *CodeAnalyzeTool { return &CodeAnalyzeTool{} }

func (t *CodeAnalyzeTool) Info() tools.Definition {
	return tools.Definition{
		Name:        "code_analyze",
		Description: "Run static analysis (linter) on code",
		Category:    "code",
		Parameters: []tools.Parameter{
			{Name: "path", Type: "string", Description: "Path to analyze", Required: true},
			{Name: "language", Type: "string", Description: "Language (default: go). Supported: go, javascript, typescript"},
		},
	}
}

func (t *CodeAnalyzeTool) Execute(ctx context.Context, args map[string]any) (tools.Result, error) {
	path, err := requireStringArg(args, "path")
	if err != nil {
		return tools.Result{}, err
	}

	lang := resolveLanguage(args)

	var output string
	var exitCode int

	switch lang {
	case "go":
		output, exitCode = runCommand(ctx, path, "go", "vet", "./...")
	case "javascript", "typescript", "js", "ts":
		output, exitCode = runCommand(ctx, path, "npx", "eslint", ".")
	default:
		return tools.Result{
			Output:   fmt.Sprintf("unsupported language: %s", lang),
			ExitCode: 1,
		}, nil
	}

	if output == "" && exitCode == 0 {
		output = "no issues found"
	}

	return tools.Result{Output: output, ExitCode: exitCode}, nil
}

// --- CodeFormatTool ---

// CodeFormatTool runs code formatters.
type CodeFormatTool struct{}

// NewCodeFormatTool creates a new code format tool.
func NewCodeFormatTool() *CodeFormatTool { return &CodeFormatTool{} }

func (t *CodeFormatTool) Info() tools.Definition {
	return tools.Definition{
		Name:        "code_format",
		Description: "Format code using language-specific formatter",
		Category:    "code",
		Parameters: []tools.Parameter{
			{Name: "path", Type: "string", Description: "Path to format", Required: true},
			{Name: "language", Type: "string", Description: "Language (default: go). Supported: go, javascript, typescript"},
		},
	}
}

func (t *CodeFormatTool) Execute(ctx context.Context, args map[string]any) (tools.Result, error) {
	path, err := requireStringArg(args, "path")
	if err != nil {
		return tools.Result{}, err
	}

	lang := resolveLanguage(args)

	var output string
	var exitCode int

	switch lang {
	case "go":
		output, exitCode = runCommand(ctx, path, "gofmt", "-w", ".")
	case "javascript", "typescript", "js", "ts":
		output, exitCode = runCommand(ctx, path, "npx", "prettier", "--write", ".")
	default:
		return tools.Result{
			Output:   fmt.Sprintf("unsupported language: %s", lang),
			ExitCode: 1,
		}, nil
	}

	if output == "" && exitCode == 0 {
		output = "formatting complete"
	}

	return tools.Result{Output: output, ExitCode: exitCode}, nil
}

// --- CodeTestTool ---

// CodeTestTool runs tests in a project.
type CodeTestTool struct{}

// NewCodeTestTool creates a new code test tool.
func NewCodeTestTool() *CodeTestTool { return &CodeTestTool{} }

func (t *CodeTestTool) Info() tools.Definition {
	return tools.Definition{
		Name:        "code_test",
		Description: "Run tests in a project",
		Category:    "code",
		Parameters: []tools.Parameter{
			{Name: "path", Type: "string", Description: "Path to the project", Required: true},
			{Name: "pattern", Type: "string", Description: "Test name pattern (e.g., TestAdd)"},
			{Name: "language", Type: "string", Description: "Language (default: go)"},
		},
	}
}

func (t *CodeTestTool) Execute(ctx context.Context, args map[string]any) (tools.Result, error) {
	path, err := requireStringArg(args, "path")
	if err != nil {
		return tools.Result{}, err
	}

	lang := resolveLanguage(args)
	pattern, _ := args["pattern"].(string)

	var output string
	var exitCode int

	switch lang {
	case "go":
		goArgs := []string{"test", "./..."}
		if pattern != "" {
			goArgs = append(goArgs, "-run", pattern)
		}
		goArgs = append(goArgs, "-v")
		output, exitCode = runCommand(ctx, path, "go", goArgs...)
	case "javascript", "typescript", "js", "ts":
		testArgs := []string{"vitest", "run"}
		if pattern != "" {
			testArgs = append(testArgs, "-t", pattern)
		}
		output, exitCode = runCommand(ctx, path, "npx", testArgs...)
	default:
		return tools.Result{
			Output:   fmt.Sprintf("unsupported language: %s", lang),
			ExitCode: 1,
		}, nil
	}

	return tools.Result{Output: output, ExitCode: exitCode}, nil
}

// --- CodeBuildTool ---

// CodeBuildTool runs the build command for a project.
type CodeBuildTool struct{}

// NewCodeBuildTool creates a new code build tool.
func NewCodeBuildTool() *CodeBuildTool { return &CodeBuildTool{} }

func (t *CodeBuildTool) Info() tools.Definition {
	return tools.Definition{
		Name:        "code_build",
		Description: "Build a project",
		Category:    "code",
		Parameters: []tools.Parameter{
			{Name: "path", Type: "string", Description: "Path to the project", Required: true},
			{Name: "language", Type: "string", Description: "Language (default: go)"},
		},
	}
}

func (t *CodeBuildTool) Execute(ctx context.Context, args map[string]any) (tools.Result, error) {
	path, err := requireStringArg(args, "path")
	if err != nil {
		return tools.Result{}, err
	}

	lang := resolveLanguage(args)

	var output string
	var exitCode int

	switch lang {
	case "go":
		output, exitCode = runCommand(ctx, path, "go", "build", "./...")
	case "javascript", "typescript", "js", "ts":
		output, exitCode = runCommand(ctx, path, "npx", "tsc", "--noEmit")
	default:
		return tools.Result{
			Output:   fmt.Sprintf("unsupported language: %s", lang),
			ExitCode: 1,
		}, nil
	}

	if output == "" && exitCode == 0 {
		output = "build successful"
	}

	return tools.Result{Output: output, ExitCode: exitCode}, nil
}

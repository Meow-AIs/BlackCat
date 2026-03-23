package tools

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// CustomToolDef describes a tool defined in a YAML configuration file.
type CustomToolDef struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Command     string            `yaml:"command"`
	Args        []Parameter       `yaml:"args"`
	Env         map[string]string `yaml:"env,omitempty"`
}

// customTool wraps a CustomToolDef to implement the Tool interface.
type customTool struct {
	def CustomToolDef
}

func (t *customTool) Info() Definition {
	return Definition{
		Name:        t.def.Name,
		Description: t.def.Description,
		Category:    "custom",
		Parameters:  t.def.Args,
	}
}

func (t *customTool) Execute(ctx context.Context, args map[string]any) (Result, error) {
	// Render the command template with the provided args
	tmpl, err := template.New("cmd").Parse(t.def.Command)
	if err != nil {
		return Result{Error: fmt.Sprintf("invalid command template: %v", err), ExitCode: 1}, nil
	}

	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, args); err != nil {
		return Result{Error: fmt.Sprintf("template render failed: %v", err), ExitCode: 1}, nil
	}

	command := rendered.String()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/c", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	// Apply custom environment variables
	if len(t.def.Env) > 0 {
		env := os.Environ()
		for k, v := range t.def.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = env
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return Result{Error: err.Error(), ExitCode: -1}, nil
		}
	}

	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" && !strings.HasSuffix(output, "\n") {
			output += "\n"
		}
		output += stderr.String()
	}

	return Result{Output: output, ExitCode: exitCode}, nil
}

// LoadCustomTools reads a YAML file containing custom tool definitions and
// returns them as Tool implementations. The YAML must contain a list of
// CustomToolDef objects.
func LoadCustomTools(path string) ([]Tool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read custom tools file: %w", err)
	}

	var defs []CustomToolDef
	if err := yaml.Unmarshal(data, &defs); err != nil {
		return nil, fmt.Errorf("parse custom tools YAML: %w", err)
	}

	tools := make([]Tool, 0, len(defs))
	for _, def := range defs {
		if def.Name == "" {
			return nil, fmt.Errorf("custom tool has empty name")
		}
		if def.Command == "" {
			return nil, fmt.Errorf("custom tool %q has empty command", def.Name)
		}
		tools = append(tools, &customTool{def: def})
	}

	return tools, nil
}

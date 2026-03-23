package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadCustomToolsValidYAML(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `- name: greet
  description: "Say hello"
  command: "echo hello {{.name}}"
  args:
    - name: name
      type: string
      description: "Name to greet"
      required: true
- name: count_files
  description: "Count files in directory"
  command: "ls {{.path}} | wc -l"
  args:
    - name: path
      type: string
      description: "Directory path"
      required: true
`
	fpath := filepath.Join(dir, "tools.yaml")
	os.WriteFile(fpath, []byte(yamlContent), 0644)

	tools, err := LoadCustomTools(fpath)
	if err != nil {
		t.Fatalf("LoadCustomTools failed: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
	if tools[0].Info().Name != "greet" {
		t.Errorf("expected first tool name 'greet', got %q", tools[0].Info().Name)
	}
	if tools[1].Info().Name != "count_files" {
		t.Errorf("expected second tool name 'count_files', got %q", tools[1].Info().Name)
	}
}

func TestLoadCustomToolsInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "bad.yaml")
	os.WriteFile(fpath, []byte("{{invalid yaml"), 0644)

	_, err := LoadCustomTools(fpath)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadCustomToolsNonexistentFile(t *testing.T) {
	_, err := LoadCustomTools("/nonexistent/tools.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadCustomToolsEmptyName(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `- name: ""
  description: "empty"
  command: "echo"
  args: []
`
	fpath := filepath.Join(dir, "tools.yaml")
	os.WriteFile(fpath, []byte(yamlContent), 0644)

	_, err := LoadCustomTools(fpath)
	if err == nil {
		t.Error("expected error for empty tool name")
	}
}

func TestLoadCustomToolsEmptyCommand(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `- name: nocommand
  description: "no command"
  command: ""
  args: []
`
	fpath := filepath.Join(dir, "tools.yaml")
	os.WriteFile(fpath, []byte(yamlContent), 0644)

	_, err := LoadCustomTools(fpath)
	if err == nil {
		t.Error("expected error for empty command")
	}
}

func TestCustomToolInfo(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `- name: hello
  description: "Say hello"
  command: "echo hello"
  args:
    - name: name
      type: string
      description: "Name"
      required: true
`
	fpath := filepath.Join(dir, "tools.yaml")
	os.WriteFile(fpath, []byte(yamlContent), 0644)

	tools, err := LoadCustomTools(fpath)
	if err != nil {
		t.Fatalf("LoadCustomTools failed: %v", err)
	}
	info := tools[0].Info()
	if info.Name != "hello" {
		t.Errorf("expected name 'hello', got %q", info.Name)
	}
	if info.Category != "custom" {
		t.Errorf("expected category 'custom', got %q", info.Category)
	}
	if len(info.Parameters) != 1 {
		t.Errorf("expected 1 parameter, got %d", len(info.Parameters))
	}
}

func TestCustomToolExecute(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `- name: echo_tool
  description: "Echo something"
  command: "echo {{.message}}"
  args:
    - name: message
      type: string
      description: "Message"
      required: true
`
	fpath := filepath.Join(dir, "tools.yaml")
	os.WriteFile(fpath, []byte(yamlContent), 0644)

	tools, err := LoadCustomTools(fpath)
	if err != nil {
		t.Fatalf("LoadCustomTools failed: %v", err)
	}

	result, err := tools[0].Execute(context.Background(), map[string]any{
		"message": "world",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(result.Output, "world") {
		t.Errorf("expected output to contain 'world', got %q", result.Output)
	}
}

func TestCustomToolWithEnv(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `- name: env_tool
  description: "Echo env var"
  command: "echo ${CUSTOM_VAR}"
  args: []
  env:
    CUSTOM_VAR: "test_value"
`
	fpath := filepath.Join(dir, "tools.yaml")
	os.WriteFile(fpath, []byte(yamlContent), 0644)

	tools, err := LoadCustomTools(fpath)
	if err != nil {
		t.Fatalf("LoadCustomTools failed: %v", err)
	}

	info := tools[0].Info()
	if info.Name != "env_tool" {
		t.Errorf("expected name 'env_tool', got %q", info.Name)
	}
}

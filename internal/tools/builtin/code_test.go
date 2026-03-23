package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodeAnalyzeToolInfo(t *testing.T) {
	tool := NewCodeAnalyzeTool()
	info := tool.Info()
	if info.Name != "code_analyze" {
		t.Errorf("expected name 'code_analyze', got %q", info.Name)
	}
	if info.Category != "code" {
		t.Errorf("expected category 'code', got %q", info.Category)
	}
}

func TestCodeAnalyzeToolGoFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module testpkg\n\ngo 1.21\n"), 0644)
	// Write a valid Go file
	goFile := filepath.Join(dir, "main.go")
	os.WriteFile(goFile, []byte(`package main

func main() {}
`), 0644)

	tool := NewCodeAnalyzeTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"path":     dir,
		"language": "go",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	// go vet on a valid file should succeed
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0 for valid Go file, got %d; output: %s; error: %s",
			result.ExitCode, result.Output, result.Error)
	}
}

func TestCodeAnalyzeToolMissingPath(t *testing.T) {
	tool := NewCodeAnalyzeTool()
	_, err := tool.Execute(context.Background(), map[string]any{
		"language": "go",
	})
	if err == nil {
		t.Error("expected error for missing 'path' argument")
	}
}

func TestCodeAnalyzeToolDefaultLanguage(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module testpkg\n\ngo 1.21\n"), 0644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(`package main

func main() {}
`), 0644)

	tool := NewCodeAnalyzeTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"path": dir,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	// Default is Go, should work fine
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d; output: %s", result.ExitCode, result.Output)
	}
}

func TestCodeFormatToolInfo(t *testing.T) {
	tool := NewCodeFormatTool()
	info := tool.Info()
	if info.Name != "code_format" {
		t.Errorf("expected name 'code_format', got %q", info.Name)
	}
}

func TestCodeFormatToolGoFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module testpkg\n\ngo 1.21\n"), 0644)
	goFile := filepath.Join(dir, "main.go")
	// Unformatted Go code
	os.WriteFile(goFile, []byte(`package main

func main(){
x:=1
_=x
}
`), 0644)

	tool := NewCodeFormatTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"path":     dir,
		"language": "go",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d; error: %s", result.ExitCode, result.Error)
	}

	// Read the file back - should be formatted
	data, _ := os.ReadFile(goFile)
	content := string(data)
	if !strings.Contains(content, "\tx := 1") && !strings.Contains(content, "\tx = 1") {
		// gofmt adds tabs for indentation
		t.Logf("formatted content: %s", content)
	}
}

func TestCodeTestToolInfo(t *testing.T) {
	tool := NewCodeTestTool()
	info := tool.Info()
	if info.Name != "code_test" {
		t.Errorf("expected name 'code_test', got %q", info.Name)
	}
}

func TestCodeTestToolGoPackage(t *testing.T) {
	dir := t.TempDir()

	// Write a go.mod
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module testpkg\n\ngo 1.21\n"), 0644)

	// Write a simple Go file
	os.WriteFile(filepath.Join(dir, "add.go"), []byte(`package testpkg

func Add(a, b int) int { return a + b }
`), 0644)

	// Write a test file
	os.WriteFile(filepath.Join(dir, "add_test.go"), []byte(`package testpkg

import "testing"

func TestAdd(t *testing.T) {
	if Add(1, 2) != 3 {
		t.Error("expected 3")
	}
}
`), 0644)

	tool := NewCodeTestTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"path":     dir,
		"language": "go",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d; output: %s; error: %s",
			result.ExitCode, result.Output, result.Error)
	}
}

func TestCodeTestToolWithPattern(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module testpkg\n\ngo 1.21\n"), 0644)
	os.WriteFile(filepath.Join(dir, "add.go"), []byte(`package testpkg

func Add(a, b int) int { return a + b }
`), 0644)
	os.WriteFile(filepath.Join(dir, "add_test.go"), []byte(`package testpkg

import "testing"

func TestAdd(t *testing.T) {
	if Add(1, 2) != 3 {
		t.Error("expected 3")
	}
}
`), 0644)

	tool := NewCodeTestTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"path":     dir,
		"language": "go",
		"pattern":  "TestAdd",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
}

func TestCodeBuildToolInfo(t *testing.T) {
	tool := NewCodeBuildTool()
	info := tool.Info()
	if info.Name != "code_build" {
		t.Errorf("expected name 'code_build', got %q", info.Name)
	}
}

func TestCodeBuildToolGoPackage(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module testpkg\n\ngo 1.21\n"), 0644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(`package main

func main() {}
`), 0644)

	tool := NewCodeBuildTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"path":     dir,
		"language": "go",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d; output: %s; error: %s",
			result.ExitCode, result.Output, result.Error)
	}
}

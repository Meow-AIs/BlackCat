package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFileTool(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "test.txt")
	os.WriteFile(fpath, []byte("hello blackcat"), 0644)

	tool := NewReadFileTool()
	if tool.Info().Name != "read_file" {
		t.Errorf("expected name 'read_file', got %q", tool.Info().Name)
	}

	result, err := tool.Execute(context.Background(), map[string]any{"path": fpath})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Output != "hello blackcat" {
		t.Errorf("expected 'hello blackcat', got %q", result.Output)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
}

func TestReadFileToolNotFound(t *testing.T) {
	tool := NewReadFileTool()
	result, err := tool.Execute(context.Background(), map[string]any{"path": "/nonexistent/file.txt"})
	if err != nil {
		t.Fatalf("expected no error (error in result), got: %v", err)
	}
	if result.Error == "" {
		t.Error("expected error message for missing file")
	}
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code")
	}
}

func TestReadFileToolMissingPathArg(t *testing.T) {
	tool := NewReadFileTool()
	_, err := tool.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Error("expected error for missing 'path' argument")
	}
}

func TestWriteFileTool(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "output.txt")

	tool := NewWriteFileTool()
	if tool.Info().Name != "write_file" {
		t.Errorf("expected name 'write_file', got %q", tool.Info().Name)
	}

	result, err := tool.Execute(context.Background(), map[string]any{
		"path":    fpath,
		"content": "written by blackcat",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	data, _ := os.ReadFile(fpath)
	if string(data) != "written by blackcat" {
		t.Errorf("expected 'written by blackcat', got %q", string(data))
	}
}

func TestWriteFileToolCreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "sub", "deep", "file.txt")

	tool := NewWriteFileTool()
	_, err := tool.Execute(context.Background(), map[string]any{
		"path":    fpath,
		"content": "nested",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	data, _ := os.ReadFile(fpath)
	if string(data) != "nested" {
		t.Errorf("expected 'nested', got %q", string(data))
	}
}

func TestListDirTool(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("text"), 0644)
	os.Mkdir(filepath.Join(dir, "sub"), 0755)

	tool := NewListDirTool()
	if tool.Info().Name != "list_dir" {
		t.Errorf("expected name 'list_dir', got %q", tool.Info().Name)
	}

	result, err := tool.Execute(context.Background(), map[string]any{"path": dir})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Output, "a.go") {
		t.Error("expected output to contain 'a.go'")
	}
	if !strings.Contains(result.Output, "b.txt") {
		t.Error("expected output to contain 'b.txt'")
	}
	if !strings.Contains(result.Output, "sub") {
		t.Error("expected output to contain 'sub'")
	}
}

func TestSearchFilesTool(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(dir, "util.go"), []byte("package util"), 0644)
	os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# readme"), 0644)

	tool := NewSearchFilesTool()
	if tool.Info().Name != "search_files" {
		t.Errorf("expected name 'search_files', got %q", tool.Info().Name)
	}

	result, err := tool.Execute(context.Background(), map[string]any{
		"path":    dir,
		"pattern": "*.go",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Output, "main.go") {
		t.Error("expected output to contain 'main.go'")
	}
	if !strings.Contains(result.Output, "util.go") {
		t.Error("expected output to contain 'util.go'")
	}
	if strings.Contains(result.Output, "readme.md") {
		t.Error("expected output NOT to contain 'readme.md'")
	}
}

func TestSearchContentTool(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("func main() {\n\tfmt.Println(\"hello\")\n}"), 0644)
	os.WriteFile(filepath.Join(dir, "util.go"), []byte("func helper() {\n\treturn nil\n}"), 0644)

	tool := NewSearchContentTool()
	if tool.Info().Name != "search_content" {
		t.Errorf("expected name 'search_content', got %q", tool.Info().Name)
	}

	result, err := tool.Execute(context.Background(), map[string]any{
		"path":    dir,
		"pattern": "hello",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Output, "main.go") {
		t.Error("expected output to reference 'main.go'")
	}
	if !strings.Contains(result.Output, "hello") {
		t.Error("expected output to contain matched text 'hello'")
	}
}

func TestSearchContentToolNoResults(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644)

	tool := NewSearchContentTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"path":    dir,
		"pattern": "nonexistent_pattern_xyz",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Output != "" && !strings.Contains(result.Output, "no matches") {
		t.Errorf("expected empty or 'no matches' output, got %q", result.Output)
	}
}

// --- P0 Fix 1: Sensitive Path Denylist Tests ---

func TestIsSensitivePathBlocksSSHDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home dir")
	}
	sshKey := filepath.Join(home, ".ssh", "id_rsa")
	if !isSensitivePath(sshKey) {
		t.Errorf("expected %q to be sensitive, but was not", sshKey)
	}
}

func TestIsSensitivePathBlocksAWSCredentials(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home dir")
	}
	awsCreds := filepath.Join(home, ".aws", "credentials")
	if !isSensitivePath(awsCreds) {
		t.Errorf("expected %q to be sensitive, but was not", awsCreds)
	}
}

func TestIsSensitivePathBlocksBlackcatConfig(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home dir")
	}
	bcConfig := filepath.Join(home, ".blackcat", "config.yaml")
	if !isSensitivePath(bcConfig) {
		t.Errorf("expected %q to be sensitive, but was not", bcConfig)
	}
}

func TestIsSensitivePathBlocksPEMFile(t *testing.T) {
	if !isSensitivePath("/some/path/server.pem") {
		t.Error("expected .pem file to be sensitive")
	}
}

func TestIsSensitivePathBlocksKeyFile(t *testing.T) {
	if !isSensitivePath("/some/path/private.key") {
		t.Error("expected .key file to be sensitive")
	}
}

func TestIsSensitivePathBlocksIdRsa(t *testing.T) {
	if !isSensitivePath("/home/user/.ssh/id_rsa") {
		t.Error("expected id_rsa to be sensitive")
	}
}

func TestIsSensitivePathBlocksIdEd25519(t *testing.T) {
	if !isSensitivePath("/home/user/.ssh/id_ed25519") {
		t.Error("expected id_ed25519 to be sensitive")
	}
}

func TestIsSensitivePathAllowsRegularFile(t *testing.T) {
	dir := t.TempDir()
	regularFile := filepath.Join(dir, "main.go")
	if isSensitivePath(regularFile) {
		t.Errorf("expected %q to be allowed, but was flagged sensitive", regularFile)
	}
}

func TestIsSensitivePathAllowsReadme(t *testing.T) {
	if isSensitivePath("/home/user/projects/myapp/README.md") {
		t.Error("expected README.md to be allowed")
	}
}

func TestReadFileToolBlocksSensitivePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home dir")
	}
	sshKey := filepath.Join(home, ".ssh", "id_rsa")

	tool := NewReadFileTool()
	result, err := tool.Execute(context.Background(), map[string]any{"path": sshKey})
	if err != nil {
		t.Fatalf("expected no Go error (blocked in result), got: %v", err)
	}
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code for blocked sensitive path")
	}
	if !strings.Contains(result.Error, "sensitive") && !strings.Contains(result.Error, "denied") && !strings.Contains(result.Error, "blocked") {
		t.Errorf("expected error message mentioning block reason, got %q", result.Error)
	}
}

func TestReadFileToolBlocksTraversalToSSH(t *testing.T) {
	// Path traversal attempt: ../../../../.ssh/id_rsa resolved from a temp dir
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home dir")
	}
	// Build a traversal path that resolves to ~/.ssh/id_rsa
	traversal := filepath.Join(home, "projects", "..", "..", ".ssh", "id_rsa")

	tool := NewReadFileTool()
	result, err := tool.Execute(context.Background(), map[string]any{"path": traversal})
	if err != nil {
		t.Fatalf("expected no Go error (blocked in result), got: %v", err)
	}
	// The cleaned path should still hit the SSH denylist
	if result.ExitCode == 0 {
		t.Error("expected traversal path to be blocked")
	}
}

func TestReadFileToolAllowsNormalFile(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "safe.txt")
	os.WriteFile(fpath, []byte("safe content"), 0644)

	tool := NewReadFileTool()
	result, err := tool.Execute(context.Background(), map[string]any{"path": fpath})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected normal file to be readable, got error: %q", result.Error)
	}
	if result.Output != "safe content" {
		t.Errorf("expected 'safe content', got %q", result.Output)
	}
}

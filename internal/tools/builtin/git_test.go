package builtin

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// initTestRepo creates a fresh git repo in a temp dir with one initial commit.
func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	run("init")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "test")

	// Create initial commit
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test"), 0644)
	run("add", "-A")
	run("commit", "-m", "initial commit")

	return dir
}

func TestGitStatusToolInfo(t *testing.T) {
	tool := NewGitStatusTool()
	info := tool.Info()
	if info.Name != "git_status" {
		t.Errorf("expected name 'git_status', got %q", info.Name)
	}
	if info.Category != "git" {
		t.Errorf("expected category 'git', got %q", info.Category)
	}
}

func TestGitStatusToolCleanRepo(t *testing.T) {
	dir := initTestRepo(t)
	tool := NewGitStatusTool()

	result, err := tool.Execute(context.Background(), map[string]any{"path": dir})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d; error: %s", result.ExitCode, result.Error)
	}
	// Clean repo should have empty porcelain output
	if strings.TrimSpace(result.Output) != "" {
		t.Errorf("expected empty output for clean repo, got %q", result.Output)
	}
}

func TestGitStatusToolWithChanges(t *testing.T) {
	dir := initTestRepo(t)
	os.WriteFile(filepath.Join(dir, "new.txt"), []byte("hello"), 0644)

	tool := NewGitStatusTool()
	result, err := tool.Execute(context.Background(), map[string]any{"path": dir})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(result.Output, "new.txt") {
		t.Errorf("expected output to contain 'new.txt', got %q", result.Output)
	}
}

func TestGitStatusToolMissingArg(t *testing.T) {
	tool := NewGitStatusTool()
	_, err := tool.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Error("expected error for missing 'path' argument")
	}
}

func TestGitDiffToolInfo(t *testing.T) {
	tool := NewGitDiffTool()
	if tool.Info().Name != "git_diff" {
		t.Errorf("expected name 'git_diff', got %q", tool.Info().Name)
	}
}

func TestGitDiffToolUnstaged(t *testing.T) {
	dir := initTestRepo(t)
	fpath := filepath.Join(dir, "README.md")
	os.WriteFile(fpath, []byte("# Test\nmodified"), 0644)

	tool := NewGitDiffTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"path":   dir,
		"staged": false,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(result.Output, "modified") {
		t.Errorf("expected diff to contain 'modified', got %q", result.Output)
	}
}

func TestGitDiffToolStaged(t *testing.T) {
	dir := initTestRepo(t)
	fpath := filepath.Join(dir, "README.md")
	os.WriteFile(fpath, []byte("# Test\nstaged change"), 0644)

	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = dir
	cmd.Run()

	tool := NewGitDiffTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"path":   dir,
		"staged": true,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(result.Output, "staged change") {
		t.Errorf("expected diff to contain 'staged change', got %q", result.Output)
	}
}

func TestGitLogToolInfo(t *testing.T) {
	tool := NewGitLogTool()
	if tool.Info().Name != "git_log" {
		t.Errorf("expected name 'git_log', got %q", tool.Info().Name)
	}
}

func TestGitLogToolDefaultCount(t *testing.T) {
	dir := initTestRepo(t)
	tool := NewGitLogTool()

	result, err := tool.Execute(context.Background(), map[string]any{"path": dir})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(result.Output, "initial commit") {
		t.Errorf("expected log to contain 'initial commit', got %q", result.Output)
	}
}

func TestGitLogToolCustomCount(t *testing.T) {
	dir := initTestRepo(t)
	tool := NewGitLogTool()

	result, err := tool.Execute(context.Background(), map[string]any{
		"path":  dir,
		"count": 1,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(result.Output), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 log line, got %d: %q", len(lines), result.Output)
	}
}

func TestGitCommitToolInfo(t *testing.T) {
	tool := NewGitCommitTool()
	if tool.Info().Name != "git_commit" {
		t.Errorf("expected name 'git_commit', got %q", tool.Info().Name)
	}
}

func TestGitCommitToolCommit(t *testing.T) {
	dir := initTestRepo(t)
	os.WriteFile(filepath.Join(dir, "new.txt"), []byte("hello"), 0644)

	tool := NewGitCommitTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"path":    dir,
		"message": "add new file",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d; error: %s; output: %s", result.ExitCode, result.Error, result.Output)
	}

	// Verify the commit exists in log
	logTool := NewGitLogTool()
	logResult, _ := logTool.Execute(context.Background(), map[string]any{
		"path":  dir,
		"count": 1,
	})
	if !strings.Contains(logResult.Output, "add new file") {
		t.Errorf("expected log to contain commit message, got %q", logResult.Output)
	}
}

func TestGitCommitToolMissingMessage(t *testing.T) {
	tool := NewGitCommitTool()
	_, err := tool.Execute(context.Background(), map[string]any{"path": "/tmp"})
	if err == nil {
		t.Error("expected error for missing 'message' argument")
	}
}

func TestGitBranchToolInfo(t *testing.T) {
	tool := NewGitBranchTool()
	if tool.Info().Name != "git_branch" {
		t.Errorf("expected name 'git_branch', got %q", tool.Info().Name)
	}
}

func TestGitBranchToolList(t *testing.T) {
	dir := initTestRepo(t)
	tool := NewGitBranchTool()

	result, err := tool.Execute(context.Background(), map[string]any{"path": dir})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	// Should list at least the default branch
	if strings.TrimSpace(result.Output) == "" {
		t.Error("expected at least one branch in output")
	}
}

func TestGitBranchToolCreate(t *testing.T) {
	dir := initTestRepo(t)
	tool := NewGitBranchTool()

	result, err := tool.Execute(context.Background(), map[string]any{
		"path":   dir,
		"name":   "feature-test",
		"create": true,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d; error: %s", result.ExitCode, result.Error)
	}

	// Verify branch exists
	listResult, _ := tool.Execute(context.Background(), map[string]any{"path": dir})
	if !strings.Contains(listResult.Output, "feature-test") {
		t.Errorf("expected branch list to contain 'feature-test', got %q", listResult.Output)
	}
}

func TestGitBranchToolCreateMissingName(t *testing.T) {
	dir := initTestRepo(t)
	tool := NewGitBranchTool()

	result, err := tool.Execute(context.Background(), map[string]any{
		"path":   dir,
		"create": true,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code when creating branch without name")
	}
}

func TestGitToolsInvalidPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("path behavior differs on Windows")
	}
	tool := NewGitStatusTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"path": "/nonexistent/repo/path",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code for invalid repo path")
	}
}

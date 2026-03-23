package pipeline

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Generate – GitHub Actions
// ---------------------------------------------------------------------------

func TestGenerate_GitHubActions_Go(t *testing.T) {
	req := PipelineRequest{
		Platform:    PlatformGitHubActions,
		Language:    LangGo,
		ProjectName: "myapp",
		GoVersion:   "1.22",
	}
	res, err := Generate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Platform != PlatformGitHubActions {
		t.Errorf("expected platform %q, got %q", PlatformGitHubActions, res.Platform)
	}
	if res.Filename != ".github/workflows/ci.yml" {
		t.Errorf("expected filename .github/workflows/ci.yml, got %q", res.Filename)
	}
	// Must contain key workflow elements
	for _, want := range []string{
		"name:",
		"on:",
		"actions/checkout@",
		"actions/setup-go@",
		"go test",
		"go build",
		"1.22",
	} {
		if !strings.Contains(res.Content, want) {
			t.Errorf("expected content to contain %q", want)
		}
	}
}

func TestGenerate_GitHubActions_Node(t *testing.T) {
	req := PipelineRequest{
		Platform:    PlatformGitHubActions,
		Language:    LangNode,
		ProjectName: "webapp",
		NodeVersion: "20",
	}
	res, err := Generate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{
		"actions/setup-node@",
		"npm ci",
		"npm test",
		"npm run build",
		"20",
	} {
		if !strings.Contains(res.Content, want) {
			t.Errorf("expected content to contain %q", want)
		}
	}
}

func TestGenerate_GitHubActions_Python(t *testing.T) {
	req := PipelineRequest{
		Platform:      PlatformGitHubActions,
		Language:      LangPython,
		ProjectName:   "pyapp",
		PythonVersion: "3.12",
	}
	res, err := Generate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{
		"actions/setup-python@",
		"pip install",
		"pytest",
		"3.12",
	} {
		if !strings.Contains(res.Content, want) {
			t.Errorf("expected content to contain %q", want)
		}
	}
}

func TestGenerate_GitHubActions_Rust(t *testing.T) {
	req := PipelineRequest{
		Platform:    PlatformGitHubActions,
		Language:    LangRust,
		ProjectName: "rustapp",
	}
	res, err := Generate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{
		"cargo test",
		"cargo build",
		"cargo clippy",
	} {
		if !strings.Contains(res.Content, want) {
			t.Errorf("expected content to contain %q", want)
		}
	}
}

func TestGenerate_GitHubActions_WithSecurityGates(t *testing.T) {
	req := PipelineRequest{
		Platform:             PlatformGitHubActions,
		Language:             LangGo,
		ProjectName:          "secure-app",
		GoVersion:            "1.22",
		IncludeSecurityGates: true,
	}
	res, err := Generate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{
		"gitleaks",
		"semgrep",
	} {
		if !strings.Contains(res.Content, want) {
			t.Errorf("expected security content to contain %q", want)
		}
	}
}

func TestGenerate_GitHubActions_WithDocker(t *testing.T) {
	req := PipelineRequest{
		Platform:       PlatformGitHubActions,
		Language:       LangGo,
		ProjectName:    "dockerapp",
		GoVersion:      "1.22",
		IncludeDocker:  true,
		DockerRegistry: "ghcr.io/myorg",
	}
	res, err := Generate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{
		"docker",
		"ghcr.io/myorg",
		"docker/build-push-action@",
	} {
		if !strings.Contains(res.Content, want) {
			t.Errorf("expected docker content to contain %q", want)
		}
	}
}

// ---------------------------------------------------------------------------
// Generate – GitLab CI
// ---------------------------------------------------------------------------

func TestGenerate_GitLabCI_Go(t *testing.T) {
	req := PipelineRequest{
		Platform:    PlatformGitLabCI,
		Language:    LangGo,
		ProjectName: "myapp",
		GoVersion:   "1.22",
	}
	res, err := Generate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Platform != PlatformGitLabCI {
		t.Errorf("expected platform %q, got %q", PlatformGitLabCI, res.Platform)
	}
	if res.Filename != ".gitlab-ci.yml" {
		t.Errorf("expected filename .gitlab-ci.yml, got %q", res.Filename)
	}
	for _, want := range []string{
		"stages:",
		"go test",
		"go build",
		"golang:",
	} {
		if !strings.Contains(res.Content, want) {
			t.Errorf("expected content to contain %q", want)
		}
	}
}

func TestGenerate_GitLabCI_Node(t *testing.T) {
	req := PipelineRequest{
		Platform:    PlatformGitLabCI,
		Language:    LangNode,
		ProjectName: "webapp",
		NodeVersion: "20",
	}
	res, err := Generate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{
		"stages:",
		"node:",
		"npm ci",
		"npm test",
	} {
		if !strings.Contains(res.Content, want) {
			t.Errorf("expected content to contain %q", want)
		}
	}
}

func TestGenerate_GitLabCI_WithSecurityGates(t *testing.T) {
	req := PipelineRequest{
		Platform:             PlatformGitLabCI,
		Language:             LangGo,
		ProjectName:          "secure-app",
		GoVersion:            "1.22",
		IncludeSecurityGates: true,
	}
	res, err := Generate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{
		"gitleaks",
		"semgrep",
	} {
		if !strings.Contains(res.Content, want) {
			t.Errorf("expected security content to contain %q", want)
		}
	}
}

func TestGenerate_GitLabCI_WithDocker(t *testing.T) {
	req := PipelineRequest{
		Platform:       PlatformGitLabCI,
		Language:       LangGo,
		ProjectName:    "dockerapp",
		GoVersion:      "1.22",
		IncludeDocker:  true,
		DockerRegistry: "registry.gitlab.com/myorg",
	}
	res, err := Generate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{
		"docker build",
		"docker push",
		"registry.gitlab.com/myorg",
	} {
		if !strings.Contains(res.Content, want) {
			t.Errorf("expected docker content to contain %q", want)
		}
	}
}

// ---------------------------------------------------------------------------
// Error cases
// ---------------------------------------------------------------------------

func TestGenerate_UnsupportedPlatform(t *testing.T) {
	req := PipelineRequest{
		Platform:    Platform("circle_ci"),
		Language:    LangGo,
		ProjectName: "myapp",
	}
	_, err := Generate(req)
	if err == nil {
		t.Fatal("expected error for unsupported platform")
	}
	if !strings.Contains(err.Error(), "unsupported platform") {
		t.Errorf("expected 'unsupported platform' in error, got %q", err.Error())
	}
}

func TestGenerate_UnsupportedLanguage(t *testing.T) {
	req := PipelineRequest{
		Platform:    PlatformGitHubActions,
		Language:    Language("haskell"),
		ProjectName: "myapp",
	}
	_, err := Generate(req)
	if err == nil {
		t.Fatal("expected error for unsupported language")
	}
	if !strings.Contains(err.Error(), "unsupported language") {
		t.Errorf("expected 'unsupported language' in error, got %q", err.Error())
	}
}

func TestGenerate_EmptyProjectName(t *testing.T) {
	req := PipelineRequest{
		Platform: PlatformGitHubActions,
		Language: LangGo,
	}
	_, err := Generate(req)
	if err == nil {
		t.Fatal("expected error for empty project name")
	}
}

// ---------------------------------------------------------------------------
// Content is valid YAML (basic structure check)
// ---------------------------------------------------------------------------

func TestGenerate_GitHubActions_HasPermissions(t *testing.T) {
	req := PipelineRequest{
		Platform:    PlatformGitHubActions,
		Language:    LangGo,
		ProjectName: "myapp",
		GoVersion:   "1.22",
	}
	res, err := Generate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(res.Content, "permissions:") {
		t.Error("expected GitHub Actions workflow to include permissions block")
	}
}

func TestGenerate_GitHubActions_ActionsPinned(t *testing.T) {
	req := PipelineRequest{
		Platform:    PlatformGitHubActions,
		Language:    LangGo,
		ProjectName: "myapp",
		GoVersion:   "1.22",
	}
	res, err := Generate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Actions should use @v<N> format at minimum
	if !strings.Contains(res.Content, "actions/checkout@v") {
		t.Error("expected pinned actions/checkout version")
	}
}

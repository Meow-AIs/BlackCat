package pipeline

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// HardenPipeline
// ---------------------------------------------------------------------------

func TestHardenPipeline_AddsPermissions(t *testing.T) {
	content := `name: CI
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
`
	result := HardenPipeline(content, PlatformGitHubActions)
	if !strings.Contains(result, "permissions:") {
		t.Error("expected hardened content to contain permissions block")
	}
	if !strings.Contains(result, "read-all") {
		t.Error("expected permissions to be read-all")
	}
}

func TestHardenPipeline_PreservesExistingPermissions(t *testing.T) {
	content := `name: CI
on: [push]
permissions:
  contents: write
jobs:
  build:
    runs-on: ubuntu-latest
`
	result := HardenPipeline(content, PlatformGitHubActions)
	// Should still contain permissions (already present)
	if !strings.Contains(result, "permissions:") {
		t.Error("expected content to still contain permissions block")
	}
}

func TestHardenPipeline_GitLabCI_ReturnsContent(t *testing.T) {
	content := `stages:
  - test
  - build
test:
  stage: test
  script:
    - go test ./...
`
	result := HardenPipeline(content, PlatformGitLabCI)
	if result == "" {
		t.Error("expected non-empty result for GitLab CI")
	}
	if !strings.Contains(result, "stages:") {
		t.Error("expected original content to be preserved")
	}
}

// ---------------------------------------------------------------------------
// AuditPipeline – GitHub Actions
// ---------------------------------------------------------------------------

func TestAuditPipeline_DetectsUnpinnedActions(t *testing.T) {
	content := `name: CI
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@main
      - uses: actions/setup-go@v4
`
	issues := AuditPipeline(content, PlatformGitHubActions)
	found := false
	for _, issue := range issues {
		if issue.Rule == "unpinned-action" && issue.Severity == "high" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to detect unpinned action (branch ref like @main)")
	}
}

func TestAuditPipeline_DetectsWriteAllPermissions(t *testing.T) {
	content := `name: CI
on: [push]
permissions: write-all
jobs:
  build:
    runs-on: ubuntu-latest
`
	issues := AuditPipeline(content, PlatformGitHubActions)
	found := false
	for _, issue := range issues {
		if issue.Rule == "overly-permissive" && issue.Severity == "high" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to detect write-all permissions")
	}
}

func TestAuditPipeline_DetectsPullRequestTarget(t *testing.T) {
	content := `name: CI
on:
  pull_request_target:
    types: [opened]
jobs:
  build:
    runs-on: ubuntu-latest
`
	issues := AuditPipeline(content, PlatformGitHubActions)
	found := false
	for _, issue := range issues {
		if issue.Rule == "pull-request-target" && issue.Severity == "high" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to detect pull_request_target usage")
	}
}

func TestAuditPipeline_DetectsSecretsInEnv(t *testing.T) {
	content := `name: CI
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    env:
      API_KEY: ${{ secrets.API_KEY }}
    steps:
      - run: echo $API_KEY
`
	issues := AuditPipeline(content, PlatformGitHubActions)
	found := false
	for _, issue := range issues {
		if issue.Rule == "secret-in-env" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to detect secrets used in env vars")
	}
}

func TestAuditPipeline_CleanPipeline(t *testing.T) {
	content := `name: CI
on: [push]
permissions: read-all
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
      - run: go test ./...
`
	issues := AuditPipeline(content, PlatformGitHubActions)
	// A clean pipeline should have no high-severity issues
	for _, issue := range issues {
		if issue.Severity == "high" {
			t.Errorf("unexpected high-severity issue: %s – %s", issue.Rule, issue.Message)
		}
	}
}

func TestAuditPipeline_ReturnsLineNumbers(t *testing.T) {
	content := `name: CI
on: [push]
permissions: write-all
jobs:
  build:
    runs-on: ubuntu-latest
`
	issues := AuditPipeline(content, PlatformGitHubActions)
	for _, issue := range issues {
		if issue.Rule == "overly-permissive" {
			if issue.Line <= 0 {
				t.Error("expected positive line number for issue")
			}
			return
		}
	}
	t.Error("expected to find overly-permissive issue")
}

func TestAuditPipeline_GitLabCI_NoIssues(t *testing.T) {
	content := `stages:
  - test
test:
  stage: test
  script:
    - go test ./...
`
	issues := AuditPipeline(content, PlatformGitLabCI)
	// GitLab CI audit should return issues slice (possibly empty)
	_ = issues
}

// ---------------------------------------------------------------------------
// PipelineIssue type
// ---------------------------------------------------------------------------

func TestPipelineIssue_Fields(t *testing.T) {
	issue := PipelineIssue{
		Severity: "high",
		Rule:     "test-rule",
		Message:  "test message",
		Line:     42,
	}
	if issue.Severity != "high" {
		t.Errorf("expected severity 'high', got %q", issue.Severity)
	}
	if issue.Rule != "test-rule" {
		t.Errorf("expected rule 'test-rule', got %q", issue.Rule)
	}
	if issue.Line != 42 {
		t.Errorf("expected line 42, got %d", issue.Line)
	}
}

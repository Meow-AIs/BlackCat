package devsecops

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func writeDockerfile(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "Dockerfile")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func scanDockerfile(t *testing.T, dir string) []Finding {
	t.Helper()
	s := NewDockerfileScanner()
	result, err := s.Scan(context.Background(), ScanRequest{Path: dir, Recursive: true})
	if err != nil {
		t.Fatal(err)
	}
	return result.Findings
}

func hasRule(findings []Finding, ruleID string) bool {
	for _, f := range findings {
		if f.RuleID == ruleID {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Interface
// ---------------------------------------------------------------------------

func TestDockerfileScanner_Name(t *testing.T) {
	s := NewDockerfileScanner()
	if s.Name() != "scan_dockerfile" {
		t.Errorf("expected scan_dockerfile, got %q", s.Name())
	}
}

// ---------------------------------------------------------------------------
// DL3006 — image tagging
// ---------------------------------------------------------------------------

func TestDockerfile_UntaggedImage(t *testing.T) {
	dir := t.TempDir()
	writeDockerfile(t, dir, "FROM ubuntu\nRUN echo hello\n")
	findings := scanDockerfile(t, dir)
	if !hasRule(findings, "DL3006") {
		t.Error("expected DL3006 for untagged image")
	}
}

func TestDockerfile_LatestTag(t *testing.T) {
	dir := t.TempDir()
	writeDockerfile(t, dir, "FROM ubuntu:latest\nRUN echo hello\n")
	findings := scanDockerfile(t, dir)
	if !hasRule(findings, "DL3006") {
		t.Error("expected DL3006 for :latest tag")
	}
}

func TestDockerfile_PinnedImage_OK(t *testing.T) {
	dir := t.TempDir()
	writeDockerfile(t, dir, "FROM ubuntu:22.04\nRUN echo hello\n")
	findings := scanDockerfile(t, dir)
	if hasRule(findings, "DL3006") {
		t.Error("should not flag pinned image")
	}
}

func TestDockerfile_ScratchImage_OK(t *testing.T) {
	dir := t.TempDir()
	writeDockerfile(t, dir, "FROM scratch\nCOPY app /app\n")
	findings := scanDockerfile(t, dir)
	if hasRule(findings, "DL3006") {
		t.Error("should not flag scratch image")
	}
}

// ---------------------------------------------------------------------------
// DL3002 — last USER not root
// ---------------------------------------------------------------------------

func TestDockerfile_LastUserRoot(t *testing.T) {
	dir := t.TempDir()
	writeDockerfile(t, dir, "FROM ubuntu:22.04\nUSER root\n")
	findings := scanDockerfile(t, dir)
	if !hasRule(findings, "DL3002") {
		t.Error("expected DL3002 for last USER=root")
	}
}

func TestDockerfile_LastUserNonRoot_OK(t *testing.T) {
	dir := t.TempDir()
	writeDockerfile(t, dir, "FROM ubuntu:22.04\nUSER root\nRUN apt-get update\nUSER appuser\n")
	findings := scanDockerfile(t, dir)
	if hasRule(findings, "DL3002") {
		t.Error("should not flag when last USER is non-root")
	}
}

// ---------------------------------------------------------------------------
// DL3020 — COPY instead of ADD
// ---------------------------------------------------------------------------

func TestDockerfile_ADDLocalFile(t *testing.T) {
	dir := t.TempDir()
	writeDockerfile(t, dir, "FROM ubuntu:22.04\nADD myapp /app\n")
	findings := scanDockerfile(t, dir)
	if !hasRule(findings, "DL3020") {
		t.Error("expected DL3020 for ADD with local file")
	}
}

func TestDockerfile_ADDRemoteURL_OK(t *testing.T) {
	dir := t.TempDir()
	writeDockerfile(t, dir, "FROM ubuntu:22.04\nADD https://example.com/file.tar.gz /app/\n")
	findings := scanDockerfile(t, dir)
	if hasRule(findings, "DL3020") {
		t.Error("should not flag ADD with remote URL")
	}
}

// ---------------------------------------------------------------------------
// DL3007 — --no-install-recommends
// ---------------------------------------------------------------------------

func TestDockerfile_AptGetWithoutNoInstallRecommends(t *testing.T) {
	dir := t.TempDir()
	writeDockerfile(t, dir, "FROM ubuntu:22.04\nRUN apt-get install -y curl\n")
	findings := scanDockerfile(t, dir)
	if !hasRule(findings, "DL3007") {
		t.Error("expected DL3007 for apt-get without --no-install-recommends")
	}
}

func TestDockerfile_AptGetWithNoInstallRecommends_OK(t *testing.T) {
	dir := t.TempDir()
	writeDockerfile(t, dir, "FROM ubuntu:22.04\nRUN apt-get install --no-install-recommends -y curl\n")
	findings := scanDockerfile(t, dir)
	if hasRule(findings, "DL3007") {
		t.Error("should not flag apt-get with --no-install-recommends")
	}
}

// ---------------------------------------------------------------------------
// DL3000 — absolute WORKDIR
// ---------------------------------------------------------------------------

func TestDockerfile_RelativeWorkdir(t *testing.T) {
	dir := t.TempDir()
	writeDockerfile(t, dir, "FROM ubuntu:22.04\nWORKDIR app\n")
	findings := scanDockerfile(t, dir)
	if !hasRule(findings, "DL3000") {
		t.Error("expected DL3000 for relative WORKDIR")
	}
}

func TestDockerfile_AbsoluteWorkdir_OK(t *testing.T) {
	dir := t.TempDir()
	writeDockerfile(t, dir, "FROM ubuntu:22.04\nWORKDIR /app\n")
	findings := scanDockerfile(t, dir)
	if hasRule(findings, "DL3000") {
		t.Error("should not flag absolute WORKDIR")
	}
}

// ---------------------------------------------------------------------------
// Clean Dockerfile — minimal findings
// ---------------------------------------------------------------------------

func TestDockerfile_CleanDockerfile(t *testing.T) {
	dir := t.TempDir()
	writeDockerfile(t, dir, `FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /blackcat .

FROM scratch
COPY --from=builder /blackcat /blackcat
USER 1000
ENTRYPOINT ["/blackcat"]
`)
	findings := scanDockerfile(t, dir)
	// Should have 0 findings for a clean multi-stage Dockerfile
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for clean Dockerfile, got %d: %v", len(findings), findingRules(findings))
	}
}

// ---------------------------------------------------------------------------
// No Dockerfile — no findings
// ---------------------------------------------------------------------------

func TestDockerfileScanner_NoDockerfile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.go", "package main")
	findings := scanDockerfile(t, dir)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings with no Dockerfile, got %d", len(findings))
	}
}

func findingRules(findings []Finding) []string {
	rules := make([]string, len(findings))
	for i, f := range findings {
		rules[i] = f.RuleID
	}
	return rules
}

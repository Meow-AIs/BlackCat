package domains

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// AllDomains
// ---------------------------------------------------------------------------

func TestAllDomains_ContainsExpected(t *testing.T) {
	all := AllDomains()
	if len(all) < 4 {
		t.Errorf("expected at least 4 domains, got %d", len(all))
	}
	expected := map[Domain]bool{
		DomainGeneral: false, DomainDevSecOps: false,
		DomainArchitect: false, DomainSysAdmin: false,
	}
	for _, d := range all {
		expected[d] = true
	}
	for d, found := range expected {
		if !found {
			t.Errorf("missing domain %q in AllDomains()", d)
		}
	}
}

// ---------------------------------------------------------------------------
// DefaultManager — Register & Get
// ---------------------------------------------------------------------------

func TestDefaultManager_RegisterAndGet(t *testing.T) {
	mgr := NewDefaultManager()
	cfg := DomainConfig{
		Name:         DomainDevSecOps,
		Description:  "DevSecOps specialization",
		SystemPrompt: "You are a DevSecOps expert.",
		Tools:        []string{"scan_secrets", "scan_deps"},
	}

	if err := mgr.Register(cfg); err != nil {
		t.Fatalf("Register: %v", err)
	}

	got, err := mgr.Get(DomainDevSecOps)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Description != cfg.Description {
		t.Errorf("Description = %q, want %q", got.Description, cfg.Description)
	}
	if got.SystemPrompt != cfg.SystemPrompt {
		t.Errorf("SystemPrompt = %q, want %q", got.SystemPrompt, cfg.SystemPrompt)
	}
}

func TestDefaultManager_Get_NotFound(t *testing.T) {
	mgr := NewDefaultManager()
	_, err := mgr.Get(Domain("nonexistent"))
	if err == nil {
		t.Error("expected error for nonexistent domain")
	}
}

func TestDefaultManager_Register_DuplicateOverwrites(t *testing.T) {
	mgr := NewDefaultManager()
	cfg1 := DomainConfig{Name: DomainGeneral, Description: "v1"}
	cfg2 := DomainConfig{Name: DomainGeneral, Description: "v2"}

	_ = mgr.Register(cfg1)
	_ = mgr.Register(cfg2)

	got, _ := mgr.Get(DomainGeneral)
	if got.Description != "v2" {
		t.Errorf("expected overwritten description 'v2', got %q", got.Description)
	}
}

// ---------------------------------------------------------------------------
// SystemPromptFor
// ---------------------------------------------------------------------------

func TestDefaultManager_SystemPromptFor_RegisteredDomain(t *testing.T) {
	mgr := NewDefaultManager()
	_ = mgr.Register(DomainConfig{
		Name:         DomainArchitect,
		SystemPrompt: "You are an architecture expert.",
	})

	prompt := mgr.SystemPromptFor(DomainArchitect)
	if prompt != "You are an architecture expert." {
		t.Errorf("SystemPromptFor = %q, want %q", prompt, "You are an architecture expert.")
	}
}

func TestDefaultManager_SystemPromptFor_UnregisteredDomain_Empty(t *testing.T) {
	mgr := NewDefaultManager()
	prompt := mgr.SystemPromptFor(Domain("unknown"))
	if prompt != "" {
		t.Errorf("expected empty prompt for unknown domain, got %q", prompt)
	}
}

// ---------------------------------------------------------------------------
// ToolsFor
// ---------------------------------------------------------------------------

func TestDefaultManager_ToolsFor(t *testing.T) {
	mgr := NewDefaultManager()
	_ = mgr.Register(DomainConfig{
		Name:  DomainDevSecOps,
		Tools: []string{"scan_secrets", "scan_deps", "check_cve"},
	})

	tools := mgr.ToolsFor(DomainDevSecOps)
	if len(tools) != 3 {
		t.Errorf("expected 3 tools, got %d", len(tools))
	}
}

func TestDefaultManager_ToolsFor_UnregisteredDomain_Empty(t *testing.T) {
	mgr := NewDefaultManager()
	tools := mgr.ToolsFor(Domain("unknown"))
	if len(tools) != 0 {
		t.Errorf("expected 0 tools for unknown domain, got %d", len(tools))
	}
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestDefaultManager_List(t *testing.T) {
	mgr := NewDefaultManager()
	_ = mgr.Register(DomainConfig{Name: DomainGeneral})
	_ = mgr.Register(DomainConfig{Name: DomainDevSecOps})

	list := mgr.List()
	if len(list) != 2 {
		t.Errorf("expected 2 domains, got %d", len(list))
	}
}

func TestDefaultManager_List_Empty(t *testing.T) {
	mgr := NewDefaultManager()
	list := mgr.List()
	if len(list) != 0 {
		t.Errorf("expected 0 domains, got %d", len(list))
	}
}

// ---------------------------------------------------------------------------
// Detect — heuristic-based
// ---------------------------------------------------------------------------

func TestDefaultManager_Detect_Dockerfile(t *testing.T) {
	mgr := NewDefaultManager()
	_ = mgr.Register(DomainConfig{
		Name:           DomainDevSecOps,
		DetectionFiles: []string{"Dockerfile", ".github/workflows/*", "Jenkinsfile"},
	})
	_ = mgr.Register(DomainConfig{
		Name:           DomainGeneral,
		DetectionFiles: []string{},
	})

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM alpine"), 0644)

	result, err := mgr.Detect(context.Background(), dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if result.Domain != DomainDevSecOps {
		t.Errorf("expected devsecops domain, got %q", result.Domain)
	}
	if result.Confidence <= 0 {
		t.Error("expected positive confidence")
	}
}

func TestDefaultManager_Detect_GoMod(t *testing.T) {
	mgr := NewDefaultManager()
	_ = mgr.Register(DomainConfig{
		Name:              DomainArchitect,
		DetectionKeywords: []string{"microservice", "grpc", "protobuf"},
	})

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/example/microservice"), 0644)

	result, err := mgr.Detect(context.Background(), dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if result.Domain != DomainArchitect {
		t.Errorf("expected architect domain, got %q", result.Domain)
	}
}

func TestDefaultManager_Detect_NoMatch_ReturnsGeneral(t *testing.T) {
	mgr := NewDefaultManager()
	_ = mgr.Register(DomainConfig{
		Name:           DomainDevSecOps,
		DetectionFiles: []string{"Dockerfile"},
	})

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644)

	result, err := mgr.Detect(context.Background(), dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if result.Domain != DomainGeneral {
		t.Errorf("expected general domain for no match, got %q", result.Domain)
	}
}

func TestDefaultManager_Detect_EmptyDir_ReturnsGeneral(t *testing.T) {
	mgr := NewDefaultManager()
	dir := t.TempDir()

	result, err := mgr.Detect(context.Background(), dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if result.Domain != DomainGeneral {
		t.Errorf("expected general domain for empty dir, got %q", result.Domain)
	}
}

func TestDefaultManager_Detect_MultipleMatches_HighestConfidence(t *testing.T) {
	mgr := NewDefaultManager()
	_ = mgr.Register(DomainConfig{
		Name:           DomainDevSecOps,
		DetectionFiles: []string{"Dockerfile"},
	})
	_ = mgr.Register(DomainConfig{
		Name:           DomainSysAdmin,
		DetectionFiles: []string{"Dockerfile", "ansible.cfg", "playbook.yml"},
	})

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM alpine"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "ansible.cfg"), []byte("[defaults]"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "playbook.yml"), []byte("---"), 0644)

	result, err := mgr.Detect(context.Background(), dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	// sysadmin has more matching files, should win
	if result.Domain != DomainSysAdmin {
		t.Errorf("expected sysadmin domain (more matches), got %q", result.Domain)
	}
}

// ---------------------------------------------------------------------------
// Detect — keyword-based
// ---------------------------------------------------------------------------

func TestDefaultManager_Detect_Keywords_InREADME(t *testing.T) {
	mgr := NewDefaultManager()
	_ = mgr.Register(DomainConfig{
		Name:              DomainDevSecOps,
		DetectionKeywords: []string{"security", "vulnerability", "owasp"},
	})

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "README.md"), []byte("This project focuses on security scanning and OWASP compliance"), 0644)

	result, err := mgr.Detect(context.Background(), dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if result.Domain != DomainDevSecOps {
		t.Errorf("expected devsecops from keyword match, got %q", result.Domain)
	}
}

// ---------------------------------------------------------------------------
// BuildSystemPrompt
// ---------------------------------------------------------------------------

func TestBuildSystemPrompt_WithDomain(t *testing.T) {
	mgr := NewDefaultManager()
	_ = mgr.Register(DomainConfig{
		Name:         DomainDevSecOps,
		SystemPrompt: "You are a security expert.",
	})

	base := "You are BlackCat, an AI agent."
	result := BuildSystemPrompt(base, mgr, DomainDevSecOps)

	if !strings.Contains(result, base) {
		t.Error("result should contain the base prompt")
	}
	if !strings.Contains(result, "security expert") {
		t.Error("result should contain the domain system prompt")
	}
}

func TestBuildSystemPrompt_NoDomain_ReturnsBase(t *testing.T) {
	mgr := NewDefaultManager()
	base := "You are BlackCat."
	result := BuildSystemPrompt(base, mgr, DomainGeneral)
	if result != base {
		t.Errorf("expected base prompt only for general domain, got %q", result)
	}
}

func TestBuildSystemPrompt_UnregisteredDomain_ReturnsBase(t *testing.T) {
	mgr := NewDefaultManager()
	base := "You are BlackCat."
	result := BuildSystemPrompt(base, mgr, Domain("nonexistent"))
	if result != base {
		t.Errorf("expected base prompt for unregistered domain, got %q", result)
	}
}

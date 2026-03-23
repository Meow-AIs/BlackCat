package devsecops

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// LoadDevSecOpsWorkflows
// ---------------------------------------------------------------------------

func TestLoadDevSecOpsWorkflows_Returns10Workflows(t *testing.T) {
	workflows := LoadDevSecOpsWorkflows()
	if len(workflows) != 10 {
		t.Errorf("expected 10 workflows, got %d", len(workflows))
	}
}

func TestLoadDevSecOpsWorkflows_AllHaveRequiredFields(t *testing.T) {
	for _, w := range LoadDevSecOpsWorkflows() {
		if w.Name == "" {
			t.Error("workflow has empty Name")
		}
		if w.Description == "" {
			t.Errorf("workflow %q has empty Description", w.Name)
		}
		if w.Trigger == "" {
			t.Errorf("workflow %q has empty Trigger", w.Name)
		}
		if len(w.Steps) == 0 {
			t.Errorf("workflow %q has no Steps", w.Name)
		}
		if len(w.Tags) == 0 {
			t.Errorf("workflow %q has no Tags", w.Name)
		}
	}
}

func TestLoadDevSecOpsWorkflows_StepsHaveRequiredFields(t *testing.T) {
	for _, w := range LoadDevSecOpsWorkflows() {
		for i, s := range w.Steps {
			if s.Name == "" {
				t.Errorf("workflow %q step %d has empty Name", w.Name, i)
			}
			if s.Tool == "" {
				t.Errorf("workflow %q step %q has empty Tool", w.Name, s.Name)
			}
			if s.Description == "" {
				t.Errorf("workflow %q step %q has empty Description", w.Name, s.Name)
			}
		}
	}
}

func TestLoadDevSecOpsWorkflows_UniqueNames(t *testing.T) {
	seen := map[string]bool{}
	for _, w := range LoadDevSecOpsWorkflows() {
		if seen[w.Name] {
			t.Errorf("duplicate workflow name: %q", w.Name)
		}
		seen[w.Name] = true
	}
}

// ---------------------------------------------------------------------------
// GetDevSecOpsWorkflow
// ---------------------------------------------------------------------------

func TestGetDevSecOpsWorkflow_Found(t *testing.T) {
	names := []string{
		"pre-commit-security",
		"dependency-audit",
		"container-hardening",
		"k8s-security-review",
		"iac-security-review",
		"incident-triage",
		"compliance-gap-analysis",
		"sbom-pipeline",
		"threat-model",
		"pentest-assist",
	}
	for _, name := range names {
		w, ok := GetDevSecOpsWorkflow(name)
		if !ok {
			t.Errorf("expected to find workflow %q", name)
			continue
		}
		if w.Name != name {
			t.Errorf("expected workflow name %q, got %q", name, w.Name)
		}
	}
}

func TestGetDevSecOpsWorkflow_NotFound(t *testing.T) {
	_, ok := GetDevSecOpsWorkflow("nonexistent-workflow")
	if ok {
		t.Error("expected workflow not found")
	}
}

// ---------------------------------------------------------------------------
// ListDevSecOpsWorkflows
// ---------------------------------------------------------------------------

func TestListDevSecOpsWorkflows_SameAsLoad(t *testing.T) {
	loaded := LoadDevSecOpsWorkflows()
	listed := ListDevSecOpsWorkflows()
	if len(loaded) != len(listed) {
		t.Errorf("ListDevSecOpsWorkflows length %d != LoadDevSecOpsWorkflows %d",
			len(listed), len(loaded))
	}
}

// ---------------------------------------------------------------------------
// FormatMarkdown
// ---------------------------------------------------------------------------

func TestDevSecOpsWorkflow_FormatMarkdown(t *testing.T) {
	w, ok := GetDevSecOpsWorkflow("pre-commit-security")
	if !ok {
		t.Fatal("expected to find pre-commit-security")
	}
	md := w.FormatMarkdown()
	if md == "" {
		t.Fatal("expected non-empty markdown")
	}
	if !strings.Contains(md, "pre-commit-security") {
		t.Error("expected markdown to contain workflow name")
	}
	if !strings.Contains(md, "Steps") {
		t.Error("expected markdown to contain 'Steps'")
	}
}

func TestDevSecOpsWorkflow_FormatMarkdown_AllWorkflows(t *testing.T) {
	for _, w := range LoadDevSecOpsWorkflows() {
		md := w.FormatMarkdown()
		if md == "" {
			t.Errorf("workflow %q produced empty markdown", w.Name)
		}
	}
}

// ---------------------------------------------------------------------------
// Specific workflow content checks
// ---------------------------------------------------------------------------

func TestWorkflow_PreCommitSecurity_HasSecretScan(t *testing.T) {
	w, _ := GetDevSecOpsWorkflow("pre-commit-security")
	hasSecretStep := false
	for _, s := range w.Steps {
		if strings.Contains(strings.ToLower(s.Name), "secret") ||
			strings.Contains(strings.ToLower(s.Description), "secret") {
			hasSecretStep = true
			break
		}
	}
	if !hasSecretStep {
		t.Error("pre-commit-security should have a secret scanning step")
	}
}

func TestWorkflow_DependencyAudit_HasOSVStep(t *testing.T) {
	w, _ := GetDevSecOpsWorkflow("dependency-audit")
	hasOSV := false
	for _, s := range w.Steps {
		if strings.Contains(strings.ToLower(s.Description), "osv") ||
			strings.Contains(strings.ToLower(s.Tool), "osv") {
			hasOSV = true
			break
		}
	}
	if !hasOSV {
		t.Error("dependency-audit should reference OSV")
	}
}

func TestWorkflow_ThreatModel_HasSTRIDE(t *testing.T) {
	w, _ := GetDevSecOpsWorkflow("threat-model")
	hasSTRIDE := false
	for _, s := range w.Steps {
		if strings.Contains(strings.ToUpper(s.Description), "STRIDE") ||
			strings.Contains(strings.ToUpper(s.Name), "STRIDE") {
			hasSTRIDE = true
			break
		}
	}
	if !hasSTRIDE {
		t.Error("threat-model should reference STRIDE methodology")
	}
}

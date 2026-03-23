package architect

import (
	"strings"
	"testing"
)

func TestLoadTerraformPatterns(t *testing.T) {
	patterns := LoadTerraformPatterns()
	if len(patterns) < 15 {
		t.Errorf("expected at least 15 patterns, got %d", len(patterns))
	}
	// Every pattern must have required fields.
	for _, p := range patterns {
		if p.Name == "" {
			t.Error("pattern has empty name")
		}
		if p.Provider == "" {
			t.Errorf("pattern %q has empty provider", p.Name)
		}
		if p.Category == "" {
			t.Errorf("pattern %q has empty category", p.Name)
		}
		if p.Template == "" {
			t.Errorf("pattern %q has empty template", p.Name)
		}
		if len(p.BestPractices) == 0 {
			t.Errorf("pattern %q has no best practices", p.Name)
		}
	}
}

func TestGetTerraformPattern(t *testing.T) {
	p, ok := GetTerraformPattern("VPC Network")
	if !ok {
		t.Fatal("expected to find VPC Network pattern")
	}
	if p.Provider != "aws" {
		t.Errorf("expected provider aws, got %q", p.Provider)
	}

	_, ok = GetTerraformPattern("nonexistent-pattern")
	if ok {
		t.Error("expected not found for nonexistent pattern")
	}
}

func TestSearchTerraformPatterns(t *testing.T) {
	results := SearchTerraformPatterns("database")
	if len(results) == 0 {
		t.Error("expected at least one result for 'database'")
	}

	results = SearchTerraformPatterns("encrypt")
	if len(results) == 0 {
		t.Error("expected at least one result for 'encrypt'")
	}

	results = SearchTerraformPatterns("xyznonexistent123")
	if len(results) != 0 {
		t.Errorf("expected no results for nonsense query, got %d", len(results))
	}
}

func TestStateBackendAdvice(t *testing.T) {
	tests := []struct {
		provider string
		backend  string
	}{
		{"aws", "s3"},
		{"gcp", "gcs"},
		{"azure", "azurerm"},
		{"other", "local"},
	}
	for _, tt := range tests {
		advice := StateBackendAdvice(tt.provider)
		if advice.Backend != tt.backend {
			t.Errorf("StateBackendAdvice(%q): expected backend %q, got %q", tt.provider, tt.backend, advice.Backend)
		}
		if advice.BestPractice == "" {
			t.Errorf("StateBackendAdvice(%q): empty best practice", tt.provider)
		}
	}

	// AWS and GCP should recommend encryption.
	awsAdvice := StateBackendAdvice("aws")
	if !awsAdvice.Encryption {
		t.Error("expected encryption=true for aws")
	}
}

func TestGetOpenTofuInfo(t *testing.T) {
	info := GetOpenTofuInfo()
	if !info.Compatible {
		t.Error("expected OpenTofu to be compatible")
	}
	if info.ProviderRegistry != "registry.opentofu.org" {
		t.Errorf("unexpected registry: %q", info.ProviderRegistry)
	}
	if !strings.Contains(info.LicenseNote, "MPL-2.0") {
		t.Errorf("expected MPL-2.0 in license note, got %q", info.LicenseNote)
	}
}

func TestTerraformBestPractices(t *testing.T) {
	practices := TerraformBestPractices()
	if len(practices) < 15 {
		t.Errorf("expected at least 15 best practices, got %d", len(practices))
	}
	for i, p := range practices {
		if p == "" {
			t.Errorf("best practice %d is empty", i)
		}
	}
}

func TestTerraformPatternCategories(t *testing.T) {
	patterns := LoadTerraformPatterns()
	categories := make(map[string]bool)
	for _, p := range patterns {
		categories[p.Category] = true
	}
	required := []string{"compute", "networking", "database", "security", "storage", "monitoring"}
	for _, cat := range required {
		if !categories[cat] {
			t.Errorf("missing category %q in patterns", cat)
		}
	}
}

func TestTerraformPatternProviders(t *testing.T) {
	patterns := LoadTerraformPatterns()
	providers := make(map[string]bool)
	for _, p := range patterns {
		providers[p.Provider] = true
	}
	if !providers["aws"] {
		t.Error("expected aws provider in patterns")
	}
}

func TestTFVariableFields(t *testing.T) {
	patterns := LoadTerraformPatterns()
	foundVars := false
	for _, p := range patterns {
		if len(p.Variables) > 0 {
			foundVars = true
			for _, v := range p.Variables {
				if v.Name == "" || v.Type == "" {
					t.Errorf("pattern %q has variable with empty name or type", p.Name)
				}
			}
		}
	}
	if !foundVars {
		t.Error("expected at least one pattern to have variables")
	}
}

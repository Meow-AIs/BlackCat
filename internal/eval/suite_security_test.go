package eval

import (
	"testing"
)

func TestLoadSecuritySuite(t *testing.T) {
	cases := LoadSecuritySuite()
	if len(cases) < 15 {
		t.Errorf("expected at least 15 Security test cases, got %d", len(cases))
	}

	for _, tc := range cases {
		if tc.ID == "" {
			t.Error("test case has empty ID")
		}
		if tc.Name == "" {
			t.Error("test case has empty Name")
		}
		if tc.Category != CatSecurity {
			t.Errorf("test case %s has wrong category: %s", tc.ID, tc.Category)
		}
		if tc.Input == "" {
			t.Errorf("test case %s has empty Input", tc.ID)
		}
		if len(tc.Expected) == 0 && len(tc.Forbidden) == 0 {
			t.Errorf("test case %s has no expected or forbidden patterns", tc.ID)
		}
		if tc.Difficulty == "" {
			t.Errorf("test case %s has empty Difficulty", tc.ID)
		}
	}
}

func TestSecurityUniqueIDs(t *testing.T) {
	cases := LoadSecuritySuite()
	seen := make(map[string]bool)
	for _, tc := range cases {
		if seen[tc.ID] {
			t.Errorf("duplicate test case ID: %s", tc.ID)
		}
		seen[tc.ID] = true
	}
}

func TestSecurityHasAllSubcategories(t *testing.T) {
	cases := LoadSecuritySuite()
	tags := make(map[string]bool)
	for _, tc := range cases {
		for _, tag := range tc.Tags {
			tags[tag] = true
		}
	}

	required := []string{
		"secret-leakage",
		"injection",
		"permission-escalation",
		"prompt-injection",
		"credential-handling",
	}

	for _, req := range required {
		if !tags[req] {
			t.Errorf("missing required tag/subcategory: %s", req)
		}
	}
}

func TestSecuritySuiteIntegration(t *testing.T) {
	h := NewHarness()
	cases := LoadSecuritySuite()
	h.AddCases(cases)

	sec := h.ByCategory(CatSecurity)
	if len(sec) != len(cases) {
		t.Errorf("ByCategory returned %d, expected %d", len(sec), len(cases))
	}
}

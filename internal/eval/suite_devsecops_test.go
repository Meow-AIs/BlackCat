package eval

import (
	"testing"
)

func TestLoadDevSecOpsSuite(t *testing.T) {
	cases := LoadDevSecOpsSuite()
	if len(cases) < 20 {
		t.Errorf("expected at least 20 DevSecOps test cases, got %d", len(cases))
	}

	for _, tc := range cases {
		if tc.ID == "" {
			t.Error("test case has empty ID")
		}
		if tc.Name == "" {
			t.Error("test case has empty Name")
		}
		if tc.Category != CatDevSecOps {
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

func TestDevSecOpsUniqueIDs(t *testing.T) {
	cases := LoadDevSecOpsSuite()
	seen := make(map[string]bool)
	for _, tc := range cases {
		if seen[tc.ID] {
			t.Errorf("duplicate test case ID: %s", tc.ID)
		}
		seen[tc.ID] = true
	}
}

func TestDevSecOpsHasAllSubcategories(t *testing.T) {
	cases := LoadDevSecOpsSuite()
	tags := make(map[string]bool)
	for _, tc := range cases {
		for _, tag := range tc.Tags {
			tags[tag] = true
		}
	}

	required := []string{
		"secret-detection",
		"dockerfile",
		"vulnerability",
		"sbom",
		"pipeline",
		"remote-access",
	}

	for _, req := range required {
		if !tags[req] {
			t.Errorf("missing required tag/subcategory: %s", req)
		}
	}
}

func TestDevSecOpsSuiteIntegration(t *testing.T) {
	h := NewHarness()
	cases := LoadDevSecOpsSuite()
	h.AddCases(cases)

	devsecops := h.ByCategory(CatDevSecOps)
	if len(devsecops) != len(cases) {
		t.Errorf("ByCategory returned %d, expected %d", len(devsecops), len(cases))
	}
}

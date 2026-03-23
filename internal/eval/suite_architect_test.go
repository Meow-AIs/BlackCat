package eval

import (
	"testing"
)

func TestLoadArchitectSuite(t *testing.T) {
	cases := LoadArchitectSuite()
	if len(cases) < 20 {
		t.Errorf("expected at least 20 Architecture test cases, got %d", len(cases))
	}

	for _, tc := range cases {
		if tc.ID == "" {
			t.Error("test case has empty ID")
		}
		if tc.Name == "" {
			t.Error("test case has empty Name")
		}
		if tc.Category != CatArchitecture {
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

func TestArchitectUniqueIDs(t *testing.T) {
	cases := LoadArchitectSuite()
	seen := make(map[string]bool)
	for _, tc := range cases {
		if seen[tc.ID] {
			t.Errorf("duplicate test case ID: %s", tc.ID)
		}
		seen[tc.ID] = true
	}
}

func TestArchitectHasAllSubcategories(t *testing.T) {
	cases := LoadArchitectSuite()
	tags := make(map[string]bool)
	for _, tc := range cases {
		for _, tag := range tc.Tags {
			tags[tag] = true
		}
	}

	required := []string{
		"pattern-selection",
		"database",
		"capacity",
		"c4-diagram",
		"tech-comparison",
		"waf",
		"adr",
		"cloud-service",
	}

	for _, req := range required {
		if !tags[req] {
			t.Errorf("missing required tag/subcategory: %s", req)
		}
	}
}

func TestArchitectSuiteIntegration(t *testing.T) {
	h := NewHarness()
	cases := LoadArchitectSuite()
	h.AddCases(cases)

	arch := h.ByCategory(CatArchitecture)
	if len(arch) != len(cases) {
		t.Errorf("ByCategory returned %d, expected %d", len(arch), len(cases))
	}
}

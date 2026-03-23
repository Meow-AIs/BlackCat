package eval

import (
	"testing"
)

func TestLoadCodingSuite(t *testing.T) {
	cases := LoadCodingSuite()
	if len(cases) < 15 {
		t.Errorf("expected at least 15 Coding test cases, got %d", len(cases))
	}

	for _, tc := range cases {
		if tc.ID == "" {
			t.Error("test case has empty ID")
		}
		if tc.Name == "" {
			t.Error("test case has empty Name")
		}
		if tc.Category != CatCoding {
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

func TestCodingUniqueIDs(t *testing.T) {
	cases := LoadCodingSuite()
	seen := make(map[string]bool)
	for _, tc := range cases {
		if seen[tc.ID] {
			t.Errorf("duplicate test case ID: %s", tc.ID)
		}
		seen[tc.ID] = true
	}
}

func TestCodingHasAllSubcategories(t *testing.T) {
	cases := LoadCodingSuite()
	tags := make(map[string]bool)
	for _, tc := range cases {
		for _, tag := range tc.Tags {
			tags[tag] = true
		}
	}

	required := []string{
		"code-review",
		"error-handling",
		"concurrency",
		"api-design",
		"testing",
	}

	for _, req := range required {
		if !tags[req] {
			t.Errorf("missing required tag/subcategory: %s", req)
		}
	}
}

func TestCodingSuiteIntegration(t *testing.T) {
	h := NewHarness()
	cases := LoadCodingSuite()
	h.AddCases(cases)

	coding := h.ByCategory(CatCoding)
	if len(coding) != len(cases) {
		t.Errorf("ByCategory returned %d, expected %d", len(coding), len(cases))
	}
}

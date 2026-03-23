package skills

import (
	"encoding/json"
	"testing"
)

func TestParseSkillPackage_Valid(t *testing.T) {
	pkg := SkillPackage{
		APIVersion: "v1",
		Kind:       "skill",
		Metadata: PackageMetadata{
			Name:        "devsecops/secret-scanner",
			Version:     "1.2.0",
			Author:      "alice",
			Description: "Scans repos for leaked secrets",
			License:     "MIT",
			Tags:        []string{"security", "scanner"},
		},
		Spec: SkillSpec{
			Trigger: "scan*",
			Steps: []SkillStep{
				{Name: "detect", Tool: "grep", Description: "Find secrets"},
			},
		},
	}
	data, err := json.Marshal(pkg)
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := ParseSkillPackage(data)
	if err != nil {
		t.Fatalf("ParseSkillPackage failed: %v", err)
	}
	if parsed.APIVersion != "v1" {
		t.Errorf("expected api_version v1, got %s", parsed.APIVersion)
	}
	if parsed.Metadata.Name != "devsecops/secret-scanner" {
		t.Errorf("expected name devsecops/secret-scanner, got %s", parsed.Metadata.Name)
	}
	if len(parsed.Spec.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(parsed.Spec.Steps))
	}
}

func TestParseSkillPackage_Invalid(t *testing.T) {
	_, err := ParseSkillPackage([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestSkillPackage_Validate_Valid(t *testing.T) {
	pkg := SkillPackage{
		APIVersion: "v1",
		Kind:       "skill",
		Metadata: PackageMetadata{
			Name:        "testing/example",
			Version:     "1.0.0",
			Author:      "bob",
			Description: "An example skill for testing purposes",
			License:     "Apache-2.0",
		},
		Spec: SkillSpec{
			Trigger: "test*",
			Steps: []SkillStep{
				{Name: "run", Description: "Run the tests"},
			},
		},
	}
	if err := pkg.Validate(); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestSkillPackage_Validate_MissingFields(t *testing.T) {
	tests := []struct {
		name string
		pkg  SkillPackage
	}{
		{
			name: "missing api_version",
			pkg: SkillPackage{
				Kind:     "skill",
				Metadata: PackageMetadata{Name: "a/b", Version: "1.0.0", Author: "x", Description: "long enough desc", License: "MIT"},
				Spec:     SkillSpec{Trigger: "*", Steps: []SkillStep{{Name: "s", Description: "d"}}},
			},
		},
		{
			name: "missing name",
			pkg: SkillPackage{
				APIVersion: "v1",
				Kind:       "skill",
				Metadata:   PackageMetadata{Version: "1.0.0", Author: "x", Description: "long enough desc", License: "MIT"},
				Spec:       SkillSpec{Trigger: "*", Steps: []SkillStep{{Name: "s", Description: "d"}}},
			},
		},
		{
			name: "invalid name format",
			pkg: SkillPackage{
				APIVersion: "v1",
				Kind:       "skill",
				Metadata:   PackageMetadata{Name: "no-slash", Version: "1.0.0", Author: "x", Description: "long enough desc", License: "MIT"},
				Spec:       SkillSpec{Trigger: "*", Steps: []SkillStep{{Name: "s", Description: "d"}}},
			},
		},
		{
			name: "invalid semver",
			pkg: SkillPackage{
				APIVersion: "v1",
				Kind:       "skill",
				Metadata:   PackageMetadata{Name: "a/b", Version: "not-semver", Author: "x", Description: "long enough desc", License: "MIT"},
				Spec:       SkillSpec{Trigger: "*", Steps: []SkillStep{{Name: "s", Description: "d"}}},
			},
		},
		{
			name: "missing author",
			pkg: SkillPackage{
				APIVersion: "v1",
				Kind:       "skill",
				Metadata:   PackageMetadata{Name: "a/b", Version: "1.0.0", Description: "long enough desc", License: "MIT"},
				Spec:       SkillSpec{Trigger: "*", Steps: []SkillStep{{Name: "s", Description: "d"}}},
			},
		},
		{
			name: "description too short",
			pkg: SkillPackage{
				APIVersion: "v1",
				Kind:       "skill",
				Metadata:   PackageMetadata{Name: "a/b", Version: "1.0.0", Author: "x", Description: "short", License: "MIT"},
				Spec:       SkillSpec{Trigger: "*", Steps: []SkillStep{{Name: "s", Description: "d"}}},
			},
		},
		{
			name: "missing license",
			pkg: SkillPackage{
				APIVersion: "v1",
				Kind:       "skill",
				Metadata:   PackageMetadata{Name: "a/b", Version: "1.0.0", Author: "x", Description: "long enough desc"},
				Spec:       SkillSpec{Trigger: "*", Steps: []SkillStep{{Name: "s", Description: "d"}}},
			},
		},
		{
			name: "no steps",
			pkg: SkillPackage{
				APIVersion: "v1",
				Kind:       "skill",
				Metadata:   PackageMetadata{Name: "a/b", Version: "1.0.0", Author: "x", Description: "long enough desc", License: "MIT"},
				Spec:       SkillSpec{Trigger: "*", Steps: []SkillStep{}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.pkg.Validate(); err == nil {
				t.Error("expected validation error")
			}
		})
	}
}

func TestSkillPackage_ToSkill(t *testing.T) {
	pkg := SkillPackage{
		APIVersion: "v1",
		Kind:       "skill",
		Metadata: PackageMetadata{
			Name:        "tools/formatter",
			Version:     "2.1.0",
			Author:      "carol",
			Description: "Auto-format code files",
			License:     "MIT",
			Tags:        []string{"tools", "format"},
			Repository:  "https://github.com/carol/formatter",
		},
		Spec: SkillSpec{
			Trigger: "format*",
			Steps: []SkillStep{
				{Name: "detect-lang", Description: "Detect language"},
				{Name: "format", Tool: "prettier", Description: "Format files"},
			},
		},
	}

	skill := pkg.ToSkill()

	if skill.Name != "tools/formatter" {
		t.Errorf("expected name tools/formatter, got %s", skill.Name)
	}
	if skill.Version != "2.1.0" {
		t.Errorf("expected version 2.1.0, got %s", skill.Version)
	}
	if skill.Author != "carol" {
		t.Errorf("expected author carol, got %s", skill.Author)
	}
	if skill.Trigger != "format*" {
		t.Errorf("expected trigger format*, got %s", skill.Trigger)
	}
	if len(skill.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(skill.Steps))
	}
	if skill.Source != "marketplace" {
		t.Errorf("expected source marketplace, got %s", skill.Source)
	}
	if skill.License != "MIT" {
		t.Errorf("expected license MIT, got %s", skill.License)
	}
	if skill.Repository != "https://github.com/carol/formatter" {
		t.Errorf("expected repository URL, got %s", skill.Repository)
	}
	if len(skill.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(skill.Tags))
	}
}

func TestSkillPackage_Marshal(t *testing.T) {
	pkg := SkillPackage{
		APIVersion: "v1",
		Kind:       "skill",
		Metadata: PackageMetadata{
			Name:    "a/b",
			Version: "1.0.0",
			Author:  "x",
		},
	}
	data, err := pkg.Marshal()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var roundTrip SkillPackage
	if err := json.Unmarshal(data, &roundTrip); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if roundTrip.Metadata.Name != "a/b" {
		t.Errorf("expected name a/b, got %s", roundTrip.Metadata.Name)
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.0.1", "1.0.0", 1},
		{"1.1.0", "1.0.9", 1},
		{"2.0.0", "1.9.9", 1},
		{"0.1.0", "0.2.0", -1},
	}
	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			got := CompareVersions(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestIsCompatibleVersion(t *testing.T) {
	tests := []struct {
		required string
		current  string
		want     bool
	}{
		{"1.0.0", "1.0.0", true},
		{"1.0.0", "1.1.0", true},
		{"1.0.0", "2.0.0", true},
		{"2.0.0", "1.9.9", false},
		{"1.5.0", "1.4.9", false},
	}
	for _, tt := range tests {
		t.Run(tt.required+"_on_"+tt.current, func(t *testing.T) {
			got := IsCompatibleVersion(tt.required, tt.current)
			if got != tt.want {
				t.Errorf("IsCompatibleVersion(%q, %q) = %v, want %v", tt.required, tt.current, got, tt.want)
			}
		})
	}
}

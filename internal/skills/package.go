package skills

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// SkillPackage is the portable package format for marketplace skills.
type SkillPackage struct {
	APIVersion string          `json:"api_version"`
	Kind       string          `json:"kind"`
	Metadata   PackageMetadata `json:"metadata"`
	Spec       SkillSpec       `json:"spec"`
}

// PackageMetadata describes the skill package identity and authorship.
type PackageMetadata struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Author      string   `json:"author"`
	Description string   `json:"description"`
	License     string   `json:"license"`
	Tags        []string `json:"tags,omitempty"`
	Repository  string   `json:"repository,omitempty"`
	Homepage    string   `json:"homepage,omitempty"`
	MinVersion  string   `json:"min_version,omitempty"`
}

// SkillSpec defines the executable behavior of a skill.
type SkillSpec struct {
	Trigger string            `json:"trigger"`
	Steps   []SkillStep       `json:"steps"`
	Tools   []string          `json:"tools,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Prompts map[string]string `json:"prompts,omitempty"`
}

// SkillStep is a single step within a skill's execution plan.
type SkillStep struct {
	Name        string `json:"name"`
	Tool        string `json:"tool,omitempty"`
	Command     string `json:"command,omitempty"`
	Prompt      string `json:"prompt,omitempty"`
	Description string `json:"description"`
}

// ParseSkillPackage deserializes a JSON-encoded skill package.
func ParseSkillPackage(data []byte) (*SkillPackage, error) {
	var pkg SkillPackage
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("parse skill package: %w", err)
	}
	return &pkg, nil
}

// Validate checks that all required fields are present and correctly formatted.
func (p *SkillPackage) Validate() error {
	var errs []string

	if p.APIVersion == "" {
		errs = append(errs, "api_version is required")
	}
	if p.Metadata.Name == "" {
		errs = append(errs, "name is required")
	} else if !strings.Contains(p.Metadata.Name, "/") {
		errs = append(errs, "name must be in category/name format (e.g. devsecops/secret-scanner)")
	}
	if p.Metadata.Version == "" {
		errs = append(errs, "version is required")
	} else if !isValidSemver(p.Metadata.Version) {
		errs = append(errs, "version must be valid semver (e.g. 1.2.3)")
	}
	if p.Metadata.Author == "" {
		errs = append(errs, "author is required")
	}
	if len(p.Metadata.Description) < 10 {
		errs = append(errs, "description is required (min 10 characters)")
	}
	if p.Metadata.License == "" {
		errs = append(errs, "license is required")
	}
	if len(p.Spec.Steps) == 0 {
		errs = append(errs, "at least 1 step is required")
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation failed: %s", strings.Join(errs, "; "))
	}
	return nil
}

// ToSkill converts a SkillPackage to the internal Skill type.
func (p *SkillPackage) ToSkill() Skill {
	stepStrings := make([]string, len(p.Spec.Steps))
	for i, s := range p.Spec.Steps {
		stepStrings[i] = s.Name + ": " + s.Description
	}

	tags := make([]string, len(p.Metadata.Tags))
	copy(tags, p.Metadata.Tags)

	return Skill{
		ID:          p.Metadata.Name + "@" + p.Metadata.Version,
		Name:        p.Metadata.Name,
		Description: p.Metadata.Description,
		Trigger:     p.Spec.Trigger,
		Steps:       stepStrings,
		Source:      "marketplace",
		CreatedAt:   time.Now().Unix(),
		Version:     p.Metadata.Version,
		Author:      p.Metadata.Author,
		Tags:        tags,
		License:     p.Metadata.License,
		Repository:  p.Metadata.Repository,
	}
}

// Marshal serializes the package to JSON.
func (p *SkillPackage) Marshal() ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}

// CompareVersions compares two semver strings. Returns -1 if a < b, 0 if equal, 1 if a > b.
func CompareVersions(a, b string) int {
	aParts := parseSemver(a)
	bParts := parseSemver(b)

	for i := 0; i < 3; i++ {
		if aParts[i] < bParts[i] {
			return -1
		}
		if aParts[i] > bParts[i] {
			return 1
		}
	}
	return 0
}

// IsCompatibleVersion returns true if current >= required.
func IsCompatibleVersion(required, current string) bool {
	return CompareVersions(current, required) >= 0
}

// isValidSemver checks if a string is a valid semver (major.minor.patch).
func isValidSemver(v string) bool {
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if _, err := strconv.Atoi(part); err != nil {
			return false
		}
	}
	return true
}

// parseSemver splits a semver string into [major, minor, patch].
func parseSemver(v string) [3]int {
	var result [3]int
	parts := strings.Split(v, ".")
	for i := 0; i < 3 && i < len(parts); i++ {
		result[i], _ = strconv.Atoi(parts[i])
	}
	return result
}

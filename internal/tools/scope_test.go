package tools

import (
	"testing"
)

func TestScopeForDomainDevsecops(t *testing.T) {
	scope := ScopeForDomain("devsecops")
	if scope.Name != "devsecops" {
		t.Errorf("scope name = %q, want %q", scope.Name, "devsecops")
	}
	if len(scope.AllowedCategories) == 0 {
		t.Error("devsecops scope should have allowed categories")
	}
	if len(scope.RequiredTools) == 0 {
		t.Error("devsecops scope should have required tools")
	}
}

func TestScopeForDomainArchitect(t *testing.T) {
	scope := ScopeForDomain("architect")
	if scope.Name != "architect" {
		t.Errorf("scope name = %q, want %q", scope.Name, "architect")
	}
}

func TestScopeForDomainGeneral(t *testing.T) {
	scope := ScopeForDomain("general")
	if scope.Name != "general" {
		t.Errorf("scope name = %q, want %q", scope.Name, "general")
	}
	if len(scope.AllowedCategories) != 0 {
		t.Error("general scope should have empty AllowedCategories (allow all)")
	}
}

func TestScopeForDomainUnknown(t *testing.T) {
	scope := ScopeForDomain("nonexistent")
	if scope.Name != "general" {
		t.Errorf("unknown domain should return general scope, got %q", scope.Name)
	}
}

func TestScopeApplyDevsecops(t *testing.T) {
	scope := ScopeForDomain("devsecops")
	tools := sampleTools()
	filtered := scope.Apply(tools)

	// Should include security, shell, filesystem, git, code categories
	for _, tool := range filtered {
		allowed := false
		for _, cat := range scope.AllowedCategories {
			if tool.Category == cat {
				allowed = true
				break
			}
		}
		// Required tools bypass category check
		isRequired := false
		for _, req := range scope.RequiredTools {
			if tool.Name == req {
				isRequired = true
				break
			}
		}
		if !allowed && !isRequired {
			t.Errorf("tool %q (category %q) should not be in devsecops scope", tool.Name, tool.Category)
		}
	}

	// web_search and docker_build should be excluded
	for _, tool := range filtered {
		if tool.Name == "web_search" || tool.Name == "docker_build" {
			t.Errorf("tool %q should not be in devsecops scope", tool.Name)
		}
	}
}

func TestScopeGeneralAllowsAll(t *testing.T) {
	scope := ScopeForDomain("general")
	tools := sampleTools()
	filtered := scope.Apply(tools)
	if len(filtered) != len(tools) {
		t.Errorf("general scope returned %d tools, want %d (all)", len(filtered), len(tools))
	}
}

func TestIsAllowed(t *testing.T) {
	scope := ScopeForDomain("devsecops")
	secTool := Definition{Name: "scan_secrets", Category: "security"}
	webTool := Definition{Name: "web_search", Category: "web"}

	if !scope.IsAllowed(secTool) {
		t.Error("security tool should be allowed in devsecops scope")
	}
	if scope.IsAllowed(webTool) {
		t.Error("web tool should not be allowed in devsecops scope")
	}
}

func TestIsAllowedRequiredTool(t *testing.T) {
	scope := ToolScope{
		Name:              "test",
		AllowedCategories: []string{"shell"},
		RequiredTools:     []string{"special_tool"},
	}
	special := Definition{Name: "special_tool", Category: "exotic"}
	if !scope.IsAllowed(special) {
		t.Error("required tool should always be allowed regardless of category")
	}
}

func TestIsAllowedExcludedTool(t *testing.T) {
	scope := ToolScope{
		Name:              "test",
		AllowedCategories: []string{"security"},
		ExcludedTools:     []string{"scan_secrets"},
	}
	excluded := Definition{Name: "scan_secrets", Category: "security"}
	if scope.IsAllowed(excluded) {
		t.Error("excluded tool should not be allowed even if category matches")
	}
}

func TestRequiredToolsAlwaysIncluded(t *testing.T) {
	scope := ToolScope{
		Name:              "custom",
		AllowedCategories: []string{"shell"},
		RequiredTools:     []string{"scan_secrets"},
	}
	tools := sampleTools()
	filtered := scope.Apply(tools)

	found := false
	for _, tool := range filtered {
		if tool.Name == "scan_secrets" {
			found = true
			break
		}
	}
	if !found {
		t.Error("required tool scan_secrets should be included even if category doesn't match")
	}
}

func TestExcludedToolsRemoved(t *testing.T) {
	scope := ToolScope{
		Name:              "custom",
		AllowedCategories: []string{}, // allow all
		ExcludedTools:     []string{"docker_build"},
	}
	tools := sampleTools()
	filtered := scope.Apply(tools)

	for _, tool := range filtered {
		if tool.Name == "docker_build" {
			t.Error("excluded tool docker_build should not be in filtered results")
		}
	}
}

func TestClassifyTaskSecurity(t *testing.T) {
	tc := ClassifyTask("scan for vulnerabilities and secrets")
	if tc.Type != "security" {
		t.Errorf("classification = %q, want %q", tc.Type, "security")
	}
	if tc.Confidence <= 0 {
		t.Error("confidence should be > 0")
	}
}

func TestClassifyTaskArchitecture(t *testing.T) {
	tc := ClassifyTask("design the system architecture and generate a diagram")
	if tc.Type != "architecture" {
		t.Errorf("classification = %q, want %q", tc.Type, "architecture")
	}
}

func TestClassifyTaskDevops(t *testing.T) {
	tc := ClassifyTask("deploy docker container to kubernetes")
	if tc.Type != "devops" {
		t.Errorf("classification = %q, want %q", tc.Type, "devops")
	}
}

func TestClassifyTaskCoding(t *testing.T) {
	tc := ClassifyTask("fix the bug in the test suite and refactor")
	if tc.Type != "coding" {
		t.Errorf("classification = %q, want %q", tc.Type, "coding")
	}
}

func TestClassifyTaskGeneral(t *testing.T) {
	tc := ClassifyTask("hello how are you today")
	if tc.Type != "general" {
		t.Errorf("classification = %q, want %q", tc.Type, "general")
	}
}

func TestScopeForTask(t *testing.T) {
	scope := ScopeForTask("scan for security vulnerabilities")
	if scope.Name != "devsecops" {
		t.Errorf("scope = %q, want %q", scope.Name, "devsecops")
	}
}

func TestCombineScopes(t *testing.T) {
	a := ToolScope{
		Name:              "a",
		AllowedCategories: []string{"security", "shell"},
		RequiredTools:     []string{"tool_a"},
		ExcludedTools:     []string{"bad_a"},
	}
	b := ToolScope{
		Name:              "b",
		AllowedCategories: []string{"shell", "web"},
		RequiredTools:     []string{"tool_b"},
		ExcludedTools:     []string{"bad_b"},
	}
	combined := CombineScopes(a, b)

	// Should have union of categories: security, shell, web
	if len(combined.AllowedCategories) != 3 {
		t.Errorf("combined categories = %v, want 3 unique", combined.AllowedCategories)
	}
	// Should have union of required tools
	if len(combined.RequiredTools) != 2 {
		t.Errorf("combined required = %v, want 2", combined.RequiredTools)
	}
	// Should have union of excluded tools
	if len(combined.ExcludedTools) != 2 {
		t.Errorf("combined excluded = %v, want 2", combined.ExcludedTools)
	}
}

func TestCombineScopesEmptyCategories(t *testing.T) {
	// If either scope has empty categories (allow all), combined should too
	a := ToolScope{AllowedCategories: []string{}}
	b := ToolScope{AllowedCategories: []string{"security"}}
	combined := CombineScopes(a, b)
	if len(combined.AllowedCategories) != 0 {
		t.Error("combining with allow-all scope should result in allow-all")
	}
}

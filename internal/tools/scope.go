package tools

import "strings"

// ToolScope defines which tools are available within a domain or task context.
type ToolScope struct {
	Name              string
	AllowedCategories []string // empty = allow all categories
	RequiredTools     []string // always include these tool names
	ExcludedTools     []string // never include these tool names
}

// DomainScopes maps domain names to their predefined tool scopes.
var DomainScopes = map[string]ToolScope{
	"devsecops": {
		Name:              "devsecops",
		AllowedCategories: []string{"security", "shell", "filesystem", "git", "code"},
		RequiredTools:     []string{"scan_secrets", "scan_dependencies", "scan_dockerfile"},
	},
	"architect": {
		Name:              "architect",
		AllowedCategories: []string{"filesystem", "shell", "code", "web"},
		RequiredTools:     []string{"generate_diagram", "compare_tech"},
	},
	"general": {
		Name:              "general",
		AllowedCategories: []string{}, // empty = allow all
	},
}

// ScopeForDomain returns the scope for the given domain, falling back to "general".
func ScopeForDomain(domain string) ToolScope {
	scope, ok := DomainScopes[domain]
	if !ok {
		return DomainScopes["general"]
	}
	return scope
}

// Apply filters tools according to this scope's rules. Required tools are
// always included. Excluded tools are always removed. When AllowedCategories
// is non-empty, only tools in those categories pass through.
func (s ToolScope) Apply(allTools []Definition) []Definition {
	excludedSet := toSet(s.ExcludedTools)
	requiredSet := toSet(s.RequiredTools)
	catSet := toSet(s.AllowedCategories)
	allowAll := len(s.AllowedCategories) == 0

	result := make([]Definition, 0, len(allTools))
	for _, tool := range allTools {
		if excludedSet[tool.Name] {
			continue
		}
		if requiredSet[tool.Name] {
			result = append(result, tool)
			continue
		}
		if allowAll || catSet[tool.Category] {
			result = append(result, tool)
		}
	}
	return result
}

// IsAllowed checks whether a single tool is permitted under this scope.
func (s ToolScope) IsAllowed(tool Definition) bool {
	for _, name := range s.ExcludedTools {
		if tool.Name == name {
			return false
		}
	}
	for _, name := range s.RequiredTools {
		if tool.Name == name {
			return true
		}
	}
	if len(s.AllowedCategories) == 0 {
		return true
	}
	for _, cat := range s.AllowedCategories {
		if tool.Category == cat {
			return true
		}
	}
	return false
}

// TaskClassification describes the inferred type of a task based on its description.
type TaskClassification struct {
	Type       string   // "coding", "security", "architecture", "devops", "general"
	Confidence float64  // 0-1 confidence in the classification
	Keywords   []string // which keywords drove the classification
}

// taskKeywords maps task types to their trigger keywords.
var taskKeywords = map[string][]string{
	"security":     {"security", "scan", "vulnerability", "cve", "secret", "exploit", "malware", "threat"},
	"architecture": {"architect", "design", "diagram", "pattern", "compare", "blueprint", "topology"},
	"devops":       {"deploy", "pipeline", "docker", "k8s", "kubernetes", "terraform", "ci", "cd", "helm"},
	"coding":       {"test", "fix", "bug", "implement", "refactor", "code", "function", "compile", "build"},
}

// taskToScope maps task types to domain scope names.
var taskToScope = map[string]string{
	"security":     "devsecops",
	"architecture": "architect",
	"devops":       "general",
	"coding":       "general",
	"general":      "general",
}

// ClassifyTask determines the type of a task from its description using keyword matching.
func ClassifyTask(description string) TaskClassification {
	tokens := tokenize(description)
	if len(tokens) == 0 {
		return TaskClassification{Type: "general", Confidence: 0.0}
	}

	tokenSet := make(map[string]bool, len(tokens))
	for _, t := range tokens {
		tokenSet[t] = true
	}

	bestType := "general"
	bestScore := 0
	var bestMatched []string

	for taskType, keywords := range taskKeywords {
		var matched []string
		for _, kw := range keywords {
			if tokenSet[kw] {
				matched = append(matched, kw)
			}
		}
		if len(matched) > bestScore {
			bestScore = len(matched)
			bestType = taskType
			bestMatched = matched
		}
	}

	confidence := 0.0
	if bestScore > 0 {
		confidence = float64(bestScore) / float64(len(tokens))
		if confidence > 1.0 {
			confidence = 1.0
		}
	}

	return TaskClassification{
		Type:       bestType,
		Confidence: confidence,
		Keywords:   bestMatched,
	}
}

// ScopeForTask classifies a task description and returns the appropriate tool scope.
func ScopeForTask(description string) ToolScope {
	classification := ClassifyTask(description)
	scopeName := taskToScope[classification.Type]
	return ScopeForDomain(scopeName)
}

// CombineScopes merges two scopes. If either has empty AllowedCategories
// (allow-all), the result allows all. Otherwise categories, required tools,
// and excluded tools are unioned.
func CombineScopes(a, b ToolScope) ToolScope {
	combined := ToolScope{
		Name: a.Name + "+" + b.Name,
	}

	// If either allows all, combined allows all
	if len(a.AllowedCategories) == 0 || len(b.AllowedCategories) == 0 {
		combined.AllowedCategories = []string{}
	} else {
		combined.AllowedCategories = unionStrings(a.AllowedCategories, b.AllowedCategories)
	}

	combined.RequiredTools = unionStrings(a.RequiredTools, b.RequiredTools)
	combined.ExcludedTools = unionStrings(a.ExcludedTools, b.ExcludedTools)

	return combined
}

// unionStrings returns the deduplicated union of two string slices, preserving order.
func unionStrings(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	result := make([]string, 0, len(a)+len(b))
	for _, s := range a {
		lower := strings.ToLower(s)
		if !seen[lower] {
			seen[lower] = true
			result = append(result, s)
		}
	}
	for _, s := range b {
		lower := strings.ToLower(s)
		if !seen[lower] {
			seen[lower] = true
			result = append(result, s)
		}
	}
	return result
}

// toSet converts a string slice to a lookup map.
func toSet(items []string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, item := range items {
		m[item] = true
	}
	return m
}

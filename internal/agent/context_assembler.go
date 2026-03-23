package agent

import (
	"fmt"
	"sort"
	"strings"
)

// LayerLevel controls whether a layer can be dropped under token pressure.
type LayerLevel int

const (
	LayerRequired  LayerLevel = 0 // always included, even over budget
	LayerImportant LayerLevel = 1 // included before optional, dropped last
	LayerOptional  LayerLevel = 2 // dropped first when over budget
)

// ContextLayer is a named section of the system prompt.
type ContextLayer struct {
	Name     string     // identifier (e.g., "persona", "domain", "memory", "skills")
	Content  string     // the actual text
	Priority int        // 0-100, higher = appears earlier in prompt
	Level    LayerLevel // required > important > optional
}

// NewContextLayer creates a context layer.
func NewContextLayer(name, content string, priority int, level LayerLevel) ContextLayer {
	return ContextLayer{
		Name:     name,
		Content:  content,
		Priority: priority,
		Level:    level,
	}
}

// ContextAssembler builds the system prompt from multiple layers within a
// token budget. Required layers are always included. Important layers are
// added next, then optional — each dropped if the budget would be exceeded.
type ContextAssembler struct {
	tokenBudget int
	layers      []ContextLayer
	tokensUsed  int
}

// NewContextAssembler creates an assembler with the given token budget
// for the system prompt section.
func NewContextAssembler(tokenBudget int) *ContextAssembler {
	return &ContextAssembler{
		tokenBudget: tokenBudget,
	}
}

// AddLayer appends a context layer. Call Assemble() to build the final prompt.
func (a *ContextAssembler) AddLayer(layer ContextLayer) {
	a.layers = append(a.layers, layer)
}

// Assemble builds the system prompt by:
//  1. Sorting layers by priority (descending)
//  2. Including all Required layers unconditionally
//  3. Adding Important layers within remaining budget
//  4. Adding Optional layers within remaining budget
//
// Returns the assembled system prompt string.
func (a *ContextAssembler) Assemble() string {
	if len(a.layers) == 0 {
		a.tokensUsed = 0
		return ""
	}

	// Sort by priority descending
	sorted := make([]ContextLayer, len(a.layers))
	copy(sorted, a.layers)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority > sorted[j].Priority
	})

	// Separate by level
	var required, important, optional []ContextLayer
	for _, l := range sorted {
		switch l.Level {
		case LayerRequired:
			required = append(required, l)
		case LayerImportant:
			important = append(important, l)
		default:
			optional = append(optional, l)
		}
	}

	var sections []string
	usedTokens := 0

	// Phase 1: Required layers — always included
	for _, l := range required {
		sections = append(sections, formatLayer(l))
		usedTokens += EstimateTokens(l.Content) + 5 // header overhead
	}

	// Phase 2: Important layers — within budget
	for _, l := range important {
		cost := EstimateTokens(l.Content) + 5
		if usedTokens+cost <= a.tokenBudget {
			sections = append(sections, formatLayer(l))
			usedTokens += cost
		}
	}

	// Phase 3: Optional layers — within remaining budget
	for _, l := range optional {
		cost := EstimateTokens(l.Content) + 5
		if usedTokens+cost <= a.tokenBudget {
			sections = append(sections, formatLayer(l))
			usedTokens += cost
		}
	}

	a.tokensUsed = usedTokens
	return strings.Join(sections, "\n\n")
}

// TokensUsed returns the estimated tokens used after the last Assemble call.
func (a *ContextAssembler) TokensUsed() int {
	return a.tokensUsed
}

// TokensRemaining returns the estimated remaining token budget.
func (a *ContextAssembler) TokensRemaining() int {
	remaining := a.tokenBudget - a.tokensUsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

// formatLayer renders a layer with an optional section header.
// The "persona" layer is rendered without a header (it's the base prompt text).
func formatLayer(l ContextLayer) string {
	if l.Name == "persona" {
		return l.Content
	}
	header := sectionHeader(l.Name)
	return fmt.Sprintf("## %s\n%s", header, l.Content)
}

// sectionHeader converts a layer name to a display header.
func sectionHeader(name string) string {
	headers := map[string]string{
		"domain":    "Domain Expertise",
		"memory":    "Memory",
		"skills":    "Skills",
		"reasoning": "Reasoning Instructions",
		"nudges":    "Reminders",
		"tools":     "Available Tools",
		"workflows": "Workflows",
		"context":   "Project Context",
		"cloud":     "Cloud Knowledge",
		"user":      "User Preferences",
	}
	if h, ok := headers[name]; ok {
		return h
	}
	// Title-case the name as fallback
	if len(name) == 0 {
		return "Context"
	}
	return strings.ToUpper(name[:1]) + name[1:]
}

// EstimateTokens estimates token count using ~4 characters per token.
func EstimateTokens(text string) int {
	if len(text) == 0 {
		return 0
	}
	return (len(text) + 3) / 4
}

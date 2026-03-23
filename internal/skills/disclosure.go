package skills

import (
	"context"
	"fmt"
	"strings"
)

// SkillIndex is a compact representation of a skill for system prompt injection.
type SkillIndex struct {
	Name        string // skill name
	Description string // max 60 chars
}

// SkillDisclosure implements three-tier progressive skill disclosure.
type SkillDisclosure struct {
	mgr Manager
}

// NewSkillDisclosure creates a SkillDisclosure backed by the given Manager.
func NewSkillDisclosure(mgr Manager) *SkillDisclosure {
	return &SkillDisclosure{mgr: mgr}
}

// BuildCompactIndex returns all skills as compact index entries with
// descriptions truncated to 60 characters.
func (d *SkillDisclosure) BuildCompactIndex(ctx context.Context) ([]SkillIndex, error) {
	all, err := d.mgr.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing skills: %w", err)
	}

	index := make([]SkillIndex, 0, len(all))
	for _, s := range all {
		index = append(index, SkillIndex{
			Name:        s.Name,
			Description: truncate(s.Description, 60),
		})
	}
	return index, nil
}

// FormatForSystemPrompt formats a compact index as a numbered list with an
// instruction header for injection into the system prompt.
func (d *SkillDisclosure) FormatForSystemPrompt(index []SkillIndex) string {
	var b strings.Builder

	b.WriteString("Before replying, scan the skills below. If one matches your task, load it with skill_view(name).\n\n")

	for i, entry := range index {
		fmt.Fprintf(&b, "%d. **%s** — %s\n", i+1, entry.Name, entry.Description)
	}

	return b.String()
}

// LoadFullSkill loads the complete skill details (Tier 2 disclosure).
func (d *SkillDisclosure) LoadFullSkill(ctx context.Context, name string) (Skill, error) {
	return d.mgr.Get(ctx, name)
}

// EstimateTokens estimates the token count of a compact index.
// Uses the rough heuristic of ~4 characters per token.
func (d *SkillDisclosure) EstimateTokens(index []SkillIndex) int {
	if len(index) == 0 {
		return 0
	}

	totalChars := 0
	for _, entry := range index {
		// Account for numbering, formatting, name, and description.
		totalChars += len(entry.Name) + len(entry.Description) + 10 // overhead per line
	}

	return (totalChars + 3) / 4 // ceil division by 4
}

// truncate returns s trimmed to maxLen characters. If truncated, the last
// three characters are replaced with "...".
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// memSkillManager is an in-memory implementation of Manager for testing.
type memSkillManager struct {
	skills map[string]Skill
}

func newMemSkillManager() *memSkillManager {
	return &memSkillManager{skills: make(map[string]Skill)}
}

func (m *memSkillManager) Store(_ context.Context, skill Skill) error {
	m.skills[skill.ID] = skill
	return nil
}

func (m *memSkillManager) Get(_ context.Context, id string) (Skill, error) {
	s, ok := m.skills[id]
	if !ok {
		return Skill{}, fmt.Errorf("skill %q not found", id)
	}
	return s, nil
}

func (m *memSkillManager) Match(_ context.Context, _ string, limit int) ([]Skill, error) {
	var result []Skill
	for _, s := range m.skills {
		result = append(result, s)
		if len(result) >= limit {
			break
		}
	}
	return result, nil
}

func (m *memSkillManager) RecordOutcome(_ context.Context, id string, success bool) error {
	s, ok := m.skills[id]
	if !ok {
		return fmt.Errorf("skill %q not found", id)
	}
	s.UsageCount++
	if success {
		s.SuccessRate = (s.SuccessRate*float64(s.UsageCount-1) + 1.0) / float64(s.UsageCount)
	} else {
		s.SuccessRate = s.SuccessRate * float64(s.UsageCount-1) / float64(s.UsageCount)
	}
	m.skills[id] = s
	return nil
}

func (m *memSkillManager) List(_ context.Context) ([]Skill, error) {
	var result []Skill
	for _, s := range m.skills {
		result = append(result, s)
	}
	return result, nil
}

func (m *memSkillManager) Delete(_ context.Context, id string) error {
	delete(m.skills, id)
	return nil
}

func (m *memSkillManager) Export(_ context.Context) ([]byte, error) {
	var all []Skill
	for _, s := range m.skills {
		all = append(all, s)
	}
	return json.Marshal(all)
}

func (m *memSkillManager) Import(_ context.Context, data []byte) (int, error) {
	var imported []Skill
	if err := json.Unmarshal(data, &imported); err != nil {
		return 0, err
	}
	for _, s := range imported {
		m.skills[s.ID] = s
	}
	return len(imported), nil
}

func seedSkillManager() *memSkillManager {
	mgr := newMemSkillManager()
	mgr.skills["deploy-k8s"] = Skill{
		ID:          "deploy-k8s",
		Name:        "deploy-k8s",
		Description: "Deploy applications to Kubernetes using kubectl and helm charts",
		Trigger:     "deploy*",
		Steps:       []string{"build image", "push to registry", "apply manifests"},
		SuccessRate: 0.95,
		UsageCount:  20,
		Source:      "manual",
	}
	mgr.skills["git-rebase"] = Skill{
		ID:          "git-rebase",
		Name:        "git-rebase",
		Description: "Interactive git rebase workflow for cleaning up commit history",
		Trigger:     "rebase*",
		Steps:       []string{"fetch origin", "rebase -i", "force push"},
		SuccessRate: 0.85,
		UsageCount:  10,
		Source:      "auto-learned",
	}
	mgr.skills["debug-memory"] = Skill{
		ID:          "debug-memory",
		Name:        "debug-memory",
		Description: "Debug memory leaks in Go applications using pprof and runtime metrics analysis tools",
		Trigger:     "memory*",
		Steps:       []string{"enable pprof", "capture heap", "analyze goroutines", "check finalizers"},
		SuccessRate: 0.70,
		UsageCount:  5,
		Source:      "auto-learned",
	}
	return mgr
}

func TestBuildCompactIndex(t *testing.T) {
	mgr := seedSkillManager()
	disc := NewSkillDisclosure(mgr)
	ctx := context.Background()

	index, err := disc.BuildCompactIndex(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(index) != 3 {
		t.Fatalf("expected 3 index entries, got %d", len(index))
	}

	for _, entry := range index {
		if entry.Name == "" {
			t.Error("expected non-empty name")
		}
		if len(entry.Description) > 60 {
			t.Errorf("description for %q exceeds 60 chars: %d", entry.Name, len(entry.Description))
		}
	}
}

func TestBuildCompactIndex_Truncation(t *testing.T) {
	mgr := seedSkillManager()
	disc := NewSkillDisclosure(mgr)
	ctx := context.Background()

	index, err := disc.BuildCompactIndex(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// "debug-memory" has a description longer than 60 chars, should be truncated.
	for _, entry := range index {
		if entry.Name == "debug-memory" {
			if len(entry.Description) > 60 {
				t.Errorf("expected truncated description, got %d chars", len(entry.Description))
			}
			return
		}
	}
	t.Error("debug-memory skill not found in index")
}

func TestBuildCompactIndex_Empty(t *testing.T) {
	mgr := newMemSkillManager()
	disc := NewSkillDisclosure(mgr)
	ctx := context.Background()

	index, err := disc.BuildCompactIndex(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(index) != 0 {
		t.Errorf("expected 0 entries, got %d", len(index))
	}
}

func TestFormatForSystemPrompt(t *testing.T) {
	disc := NewSkillDisclosure(nil)
	index := []SkillIndex{
		{Name: "deploy-k8s", Description: "Deploy to Kubernetes"},
		{Name: "git-rebase", Description: "Interactive git rebase"},
	}

	result := disc.FormatForSystemPrompt(index)

	if !strings.Contains(result, "1.") {
		t.Error("expected numbered list starting with 1.")
	}
	if !strings.Contains(result, "2.") {
		t.Error("expected numbered list with 2.")
	}
	if !strings.Contains(result, "deploy-k8s") {
		t.Error("expected skill name deploy-k8s in output")
	}
	if !strings.Contains(result, "git-rebase") {
		t.Error("expected skill name git-rebase in output")
	}
	if !strings.Contains(result, "skill_view") {
		t.Error("expected instruction mentioning skill_view")
	}
	if !strings.Contains(result, "Before replying") {
		t.Error("expected instruction starting with 'Before replying'")
	}
}

func TestFormatForSystemPrompt_Empty(t *testing.T) {
	disc := NewSkillDisclosure(nil)
	result := disc.FormatForSystemPrompt(nil)
	// Should still contain the instruction header.
	if !strings.Contains(result, "skill_view") {
		t.Error("expected instruction even with empty index")
	}
}

func TestLoadFullSkill(t *testing.T) {
	mgr := seedSkillManager()
	disc := NewSkillDisclosure(mgr)
	ctx := context.Background()

	skill, err := disc.LoadFullSkill(ctx, "deploy-k8s")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill.Name != "deploy-k8s" {
		t.Errorf("expected name deploy-k8s, got %s", skill.Name)
	}
	if len(skill.Steps) != 3 {
		t.Errorf("expected 3 steps, got %d", len(skill.Steps))
	}
}

func TestLoadFullSkill_NotFound(t *testing.T) {
	mgr := seedSkillManager()
	disc := NewSkillDisclosure(mgr)
	ctx := context.Background()

	_, err := disc.LoadFullSkill(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent skill")
	}
}

func TestEstimateTokens(t *testing.T) {
	disc := NewSkillDisclosure(nil)

	index := []SkillIndex{
		{Name: "deploy-k8s", Description: "Deploy to Kubernetes"},
		{Name: "git-rebase", Description: "Interactive git rebase"},
	}

	tokens := disc.EstimateTokens(index)
	if tokens <= 0 {
		t.Errorf("expected positive token count, got %d", tokens)
	}

	// More entries should produce more tokens.
	biggerIndex := append(index, SkillIndex{Name: "debug", Description: "Debug stuff"})
	biggerTokens := disc.EstimateTokens(biggerIndex)
	if biggerTokens <= tokens {
		t.Errorf("expected more tokens for bigger index: %d <= %d", biggerTokens, tokens)
	}
}

func TestEstimateTokens_Empty(t *testing.T) {
	disc := NewSkillDisclosure(nil)
	tokens := disc.EstimateTokens(nil)
	if tokens != 0 {
		t.Errorf("expected 0 tokens for empty index, got %d", tokens)
	}
}

func TestSkillDisclosure_RoundTrip(t *testing.T) {
	mgr := seedSkillManager()
	disc := NewSkillDisclosure(mgr)
	ctx := context.Background()

	// Build index, format it, then load a specific skill.
	index, err := disc.BuildCompactIndex(ctx)
	if err != nil {
		t.Fatalf("BuildCompactIndex: %v", err)
	}

	prompt := disc.FormatForSystemPrompt(index)
	if len(prompt) == 0 {
		t.Fatal("expected non-empty prompt")
	}

	// Simulate the agent loading a skill after seeing the index.
	skill, err := disc.LoadFullSkill(ctx, "git-rebase")
	if err != nil {
		t.Fatalf("LoadFullSkill: %v", err)
	}
	if skill.Name != "git-rebase" {
		t.Errorf("expected git-rebase, got %s", skill.Name)
	}

	tokens := disc.EstimateTokens(index)
	if tokens <= 0 {
		t.Error("expected positive token estimate")
	}
}

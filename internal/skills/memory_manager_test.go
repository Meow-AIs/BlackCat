package skills

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func makeSkill(id, name, trigger string) Skill {
	return Skill{
		ID:          id,
		Name:        name,
		Description: "desc for " + name,
		Trigger:     trigger,
		Steps:       []string{"step1", "step2"},
		Source:      "manual",
		Version:     "1.0.0",
		Author:      "tester",
	}
}

// --- Store / Get ---

func TestInMemoryManager_StoreAndGet(t *testing.T) {
	m := NewInMemoryManager()
	ctx := context.Background()
	s := makeSkill("skill-1", "Secret Scanner", "scan*")

	if err := m.Store(ctx, s); err != nil {
		t.Fatalf("Store() error: %v", err)
	}

	got, err := m.Get(ctx, "skill-1")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got.ID != "skill-1" {
		t.Errorf("expected ID skill-1, got %q", got.ID)
	}
	if got.Name != "Secret Scanner" {
		t.Errorf("expected name 'Secret Scanner', got %q", got.Name)
	}
}

func TestInMemoryManager_GetNotFound(t *testing.T) {
	m := NewInMemoryManager()
	ctx := context.Background()

	_, err := m.Get(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent skill, got nil")
	}
}

func TestInMemoryManager_StoreOverwrites(t *testing.T) {
	m := NewInMemoryManager()
	ctx := context.Background()

	s := makeSkill("skill-1", "Original", "orig*")
	if err := m.Store(ctx, s); err != nil {
		t.Fatalf("Store() error: %v", err)
	}

	s.Name = "Updated"
	if err := m.Store(ctx, s); err != nil {
		t.Fatalf("Store() (update) error: %v", err)
	}

	got, err := m.Get(ctx, "skill-1")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got.Name != "Updated" {
		t.Errorf("expected name 'Updated' after overwrite, got %q", got.Name)
	}
}

func TestInMemoryManager_StoreCreatedAt(t *testing.T) {
	m := NewInMemoryManager()
	ctx := context.Background()

	s := makeSkill("ts-skill", "TimestampSkill", "*")
	s.CreatedAt = 0 // should be set automatically

	if err := m.Store(ctx, s); err != nil {
		t.Fatalf("Store() error: %v", err)
	}

	got, err := m.Get(ctx, "ts-skill")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got.CreatedAt == 0 {
		t.Error("expected CreatedAt to be set automatically when zero")
	}
}

// --- List ---

func TestInMemoryManager_ListEmpty(t *testing.T) {
	m := NewInMemoryManager()
	ctx := context.Background()

	skills, err := m.List(ctx)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("expected empty list, got %d", len(skills))
	}
}

func TestInMemoryManager_ListMultiple(t *testing.T) {
	m := NewInMemoryManager()
	ctx := context.Background()

	if err := m.Store(ctx, makeSkill("a", "Alpha", "a*")); err != nil {
		t.Fatalf("Store a: %v", err)
	}
	if err := m.Store(ctx, makeSkill("b", "Beta", "b*")); err != nil {
		t.Fatalf("Store b: %v", err)
	}
	if err := m.Store(ctx, makeSkill("c", "Gamma", "c*")); err != nil {
		t.Fatalf("Store c: %v", err)
	}

	skills, err := m.List(ctx)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(skills) != 3 {
		t.Errorf("expected 3 skills, got %d", len(skills))
	}
}

// --- Delete ---

func TestInMemoryManager_Delete(t *testing.T) {
	m := NewInMemoryManager()
	ctx := context.Background()

	if err := m.Store(ctx, makeSkill("del-1", "ToDelete", "d*")); err != nil {
		t.Fatalf("Store() error: %v", err)
	}

	if err := m.Delete(ctx, "del-1"); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	_, err := m.Get(ctx, "del-1")
	if err == nil {
		t.Error("expected error after delete, skill still exists")
	}
}

func TestInMemoryManager_DeleteNonExistent(t *testing.T) {
	m := NewInMemoryManager()
	ctx := context.Background()

	// Deleting a nonexistent skill should not return an error (idempotent).
	if err := m.Delete(ctx, "ghost"); err != nil {
		t.Errorf("Delete() nonexistent: unexpected error: %v", err)
	}
}

// --- Match ---

func TestInMemoryManager_MatchByDescription(t *testing.T) {
	m := NewInMemoryManager()
	ctx := context.Background()

	s := makeSkill("match-1", "Scanner", "scan*")
	s.Description = "scans for leaked secrets"
	if err := m.Store(ctx, s); err != nil {
		t.Fatalf("Store() error: %v", err)
	}
	if err := m.Store(ctx, makeSkill("match-2", "Docker Deploy", "docker*")); err != nil {
		t.Fatalf("Store() error: %v", err)
	}

	results, err := m.Match(ctx, "secrets", 5)
	if err != nil {
		t.Fatalf("Match() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 match for 'secrets', got %d", len(results))
	}
	if results[0].ID != "match-1" {
		t.Errorf("expected match-1, got %q", results[0].ID)
	}
}

func TestInMemoryManager_MatchByTrigger(t *testing.T) {
	m := NewInMemoryManager()
	ctx := context.Background()

	s := makeSkill("trig-1", "Docker Deploy", "docker*")
	if err := m.Store(ctx, s); err != nil {
		t.Fatalf("Store() error: %v", err)
	}

	results, err := m.Match(ctx, "docker", 5)
	if err != nil {
		t.Fatalf("Match() error: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected at least 1 match for 'docker'")
	}
}

func TestInMemoryManager_MatchLimit(t *testing.T) {
	m := NewInMemoryManager()
	ctx := context.Background()

	// Store 5 skills all matching "test"
	for i := 0; i < 5; i++ {
		s := makeSkill("lim-"+string(rune('0'+i)), "TestSkill", "test*")
		s.Description = "tests something"
		if err := m.Store(ctx, s); err != nil {
			t.Fatalf("Store() error: %v", err)
		}
	}

	results, err := m.Match(ctx, "test", 3)
	if err != nil {
		t.Fatalf("Match() error: %v", err)
	}
	if len(results) > 3 {
		t.Errorf("expected at most 3 results with limit=3, got %d", len(results))
	}
}

func TestInMemoryManager_MatchDefaultLimit(t *testing.T) {
	m := NewInMemoryManager()
	ctx := context.Background()

	// Limit <= 0 should default to 5
	for i := 0; i < 10; i++ {
		s := makeSkill("def-"+string(rune('0'+i)), "Common", "all*")
		s.Description = "common description"
		if err := m.Store(ctx, s); err != nil {
			t.Fatalf("Store() error: %v", err)
		}
	}

	results, err := m.Match(ctx, "common", 0)
	if err != nil {
		t.Fatalf("Match() error: %v", err)
	}
	if len(results) > 5 {
		t.Errorf("expected at most 5 results (default limit), got %d", len(results))
	}
}

func TestInMemoryManager_MatchNoResults(t *testing.T) {
	m := NewInMemoryManager()
	ctx := context.Background()

	if err := m.Store(ctx, makeSkill("x", "Xray", "xray*")); err != nil {
		t.Fatalf("Store() error: %v", err)
	}

	results, err := m.Match(ctx, "zzznomatch", 5)
	if err != nil {
		t.Fatalf("Match() error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 matches, got %d", len(results))
	}
}

// --- RecordOutcome ---

func TestInMemoryManager_RecordOutcomeSuccess(t *testing.T) {
	m := NewInMemoryManager()
	ctx := context.Background()

	s := makeSkill("out-1", "Outcome", "o*")
	s.SuccessRate = 0.5
	s.UsageCount = 4
	if err := m.Store(ctx, s); err != nil {
		t.Fatalf("Store() error: %v", err)
	}

	if err := m.RecordOutcome(ctx, "out-1", true); err != nil {
		t.Fatalf("RecordOutcome() error: %v", err)
	}

	got, _ := m.Get(ctx, "out-1")
	if got.UsageCount != 5 {
		t.Errorf("expected UsageCount=5, got %d", got.UsageCount)
	}
	// 3 successes out of 5 = 0.6
	if got.SuccessRate < 0.59 || got.SuccessRate > 0.61 {
		t.Errorf("expected SuccessRate~0.6, got %f", got.SuccessRate)
	}
}

func TestInMemoryManager_RecordOutcomeFailure(t *testing.T) {
	m := NewInMemoryManager()
	ctx := context.Background()

	s := makeSkill("out-2", "Outcome2", "o*")
	s.SuccessRate = 1.0
	s.UsageCount = 4
	if err := m.Store(ctx, s); err != nil {
		t.Fatalf("Store() error: %v", err)
	}

	if err := m.RecordOutcome(ctx, "out-2", false); err != nil {
		t.Fatalf("RecordOutcome() error: %v", err)
	}

	got, _ := m.Get(ctx, "out-2")
	if got.UsageCount != 5 {
		t.Errorf("expected UsageCount=5, got %d", got.UsageCount)
	}
	// 4 successes out of 5 = 0.8
	if got.SuccessRate < 0.79 || got.SuccessRate > 0.81 {
		t.Errorf("expected SuccessRate~0.8, got %f", got.SuccessRate)
	}
}

func TestInMemoryManager_RecordOutcomeNotFound(t *testing.T) {
	m := NewInMemoryManager()
	ctx := context.Background()

	err := m.RecordOutcome(ctx, "ghost", true)
	if err == nil {
		t.Error("expected error for nonexistent skill in RecordOutcome")
	}
}

// --- Export / Import ---

func TestInMemoryManager_ExportEmpty(t *testing.T) {
	m := NewInMemoryManager()
	ctx := context.Background()

	data, err := m.Export(ctx)
	if err != nil {
		t.Fatalf("Export() error: %v", err)
	}
	if string(data) == "" {
		t.Error("Export() returned empty bytes")
	}
	// Should be valid JSON array
	var skills []Skill
	if err := json.Unmarshal(data, &skills); err != nil {
		t.Errorf("Export() produced invalid JSON: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("expected empty array, got %d skills", len(skills))
	}
}

func TestInMemoryManager_ExportImportRoundTrip(t *testing.T) {
	m1 := NewInMemoryManager()
	ctx := context.Background()

	if err := m1.Store(ctx, makeSkill("exp-1", "ExportMe", "e*")); err != nil {
		t.Fatalf("Store() error: %v", err)
	}
	if err := m1.Store(ctx, makeSkill("exp-2", "ExportMeToo", "e2*")); err != nil {
		t.Fatalf("Store() error: %v", err)
	}

	data, err := m1.Export(ctx)
	if err != nil {
		t.Fatalf("Export() error: %v", err)
	}

	m2 := NewInMemoryManager()
	count, err := m2.Import(ctx, data)
	if err != nil {
		t.Fatalf("Import() error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected Import() count=2, got %d", count)
	}

	skills, _ := m2.List(ctx)
	if len(skills) != 2 {
		t.Errorf("expected 2 skills after import, got %d", len(skills))
	}
}

func TestInMemoryManager_ImportInvalidJSON(t *testing.T) {
	m := NewInMemoryManager()
	ctx := context.Background()

	_, err := m.Import(ctx, []byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON in Import")
	}
}

func TestInMemoryManager_ImportSetsSource(t *testing.T) {
	m := NewInMemoryManager()
	ctx := context.Background()

	// Skill with no Source should get "imported" source
	data, _ := json.Marshal([]Skill{
		{ID: "imp-1", Name: "Imported", Steps: []string{"s1"}},
	})

	count, err := m.Import(ctx, data)
	if err != nil {
		t.Fatalf("Import() error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 imported, got %d", count)
	}

	got, _ := m.Get(ctx, "imp-1")
	if got.Source != "imported" {
		t.Errorf("expected source='imported', got %q", got.Source)
	}
}

// --- Thread safety ---

func TestInMemoryManager_ConcurrentAccess(t *testing.T) {
	m := NewInMemoryManager()
	ctx := context.Background()

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func(n int) {
			s := makeSkill(string(rune('a'+n)), "Concurrent", "c*")
			_ = m.Store(ctx, s)
			_, _ = m.List(ctx)
			done <- struct{}{}
		}(i)
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}

// --- Interface compliance ---

func TestInMemoryManager_ImplementsManager(t *testing.T) {
	var _ Manager = NewInMemoryManager()
}

// --- FormatSkillContext integration ---

func TestFormatSkillContext_WithInMemory(t *testing.T) {
	m := NewInMemoryManager()
	ctx := context.Background()

	s := makeSkill("fmt-1", "Formatter", "fmt*")
	s.SuccessRate = 0.9
	s.UsageCount = 10
	_ = m.Store(ctx, s)

	skills, _ := m.List(ctx)
	result := FormatSkillContext(skills)

	if !strings.Contains(result, "Formatter") {
		t.Errorf("expected skill name in context, got %q", result)
	}
	if !strings.Contains(result, "90%") {
		t.Errorf("expected success rate in context, got %q", result)
	}
}

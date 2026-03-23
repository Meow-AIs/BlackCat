package architect

import "testing"

func loadedKB() *PatternKnowledgeBase {
	kb := NewPatternKnowledgeBase()
	kb.LoadBuiltinPatterns()
	return kb
}

func TestNewPatternKnowledgeBase_Empty(t *testing.T) {
	kb := NewPatternKnowledgeBase()
	if kb.Count() != 0 {
		t.Errorf("expected 0, got %d", kb.Count())
	}
}

func TestLoadBuiltinPatterns_HasPatterns(t *testing.T) {
	kb := loadedKB()
	if kb.Count() < 8 {
		t.Errorf("expected at least 8 builtin patterns, got %d", kb.Count())
	}
}

func TestGet_CircuitBreaker(t *testing.T) {
	kb := loadedKB()
	p, ok := kb.Get("Circuit Breaker")
	if !ok {
		t.Fatal("Circuit Breaker not found")
	}
	if p.Category != CatResilience {
		t.Errorf("expected resilience, got %q", p.Category)
	}
	if len(p.Tradeoffs) == 0 {
		t.Error("expected tradeoffs")
	}
	if len(p.RelatedTo) == 0 {
		t.Error("expected related patterns")
	}
}

func TestGet_NotFound(t *testing.T) {
	kb := loadedKB()
	_, ok := kb.Get("Nonexistent")
	if ok {
		t.Error("should not find nonexistent pattern")
	}
}

func TestSearch_ByName(t *testing.T) {
	kb := loadedKB()
	results := kb.Search("circuit")
	if len(results) == 0 {
		t.Error("expected results for 'circuit'")
	}
}

func TestSearch_ByTag(t *testing.T) {
	kb := loadedKB()
	results := kb.Search("microservices")
	if len(results) < 3 {
		t.Errorf("expected >=3 results for 'microservices' tag, got %d", len(results))
	}
}

func TestSearch_NoMatch(t *testing.T) {
	kb := loadedKB()
	results := kb.Search("xyzzynonexistent")
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestByCategory_Resilience(t *testing.T) {
	kb := loadedKB()
	results := kb.ByCategory(CatResilience)
	if len(results) < 2 {
		t.Errorf("expected >=2 resilience patterns, got %d", len(results))
	}
	for _, p := range results {
		if p.Category != CatResilience {
			t.Errorf("expected resilience category, got %q", p.Category)
		}
	}
}

func TestByCategory_Empty(t *testing.T) {
	kb := loadedKB()
	results := kb.ByCategory(PatternCategory("nonexistent"))
	if len(results) != 0 {
		t.Errorf("expected 0, got %d", len(results))
	}
}

func TestRelated_CircuitBreaker(t *testing.T) {
	kb := loadedKB()
	results := kb.Related("Circuit Breaker")
	if len(results) == 0 {
		t.Error("expected related patterns for Circuit Breaker")
	}
	names := make(map[string]bool)
	for _, r := range results {
		names[r.Name] = true
	}
	if !names["Bulkhead"] {
		t.Error("expected Bulkhead as related pattern")
	}
}

func TestRelated_NotFound(t *testing.T) {
	kb := loadedKB()
	results := kb.Related("Nonexistent")
	if len(results) != 0 {
		t.Errorf("expected 0 related for nonexistent, got %d", len(results))
	}
}

func TestAll_MatchesCount(t *testing.T) {
	kb := loadedKB()
	all := kb.All()
	if len(all) != kb.Count() {
		t.Errorf("All() length %d != Count() %d", len(all), kb.Count())
	}
}

func TestAdd_CustomPattern(t *testing.T) {
	kb := NewPatternKnowledgeBase()
	kb.Add(Pattern{
		Name:     "Custom Pattern",
		Category: CatArchitect,
		Tags:     []string{"custom"},
	})
	if kb.Count() != 1 {
		t.Errorf("expected 1 after Add, got %d", kb.Count())
	}
	p, ok := kb.Get("Custom Pattern")
	if !ok {
		t.Fatal("custom pattern not found")
	}
	if p.Category != CatArchitect {
		t.Errorf("expected architect, got %q", p.Category)
	}
}

func TestAllBuiltinPatterns_HaveRequiredFields(t *testing.T) {
	kb := loadedKB()
	for _, p := range kb.All() {
		if p.Name == "" {
			t.Error("pattern has empty name")
		}
		if p.Category == "" {
			t.Errorf("pattern %q has empty category", p.Name)
		}
		if p.Description == "" {
			t.Errorf("pattern %q has empty description", p.Name)
		}
		if p.Problem == "" {
			t.Errorf("pattern %q has empty problem", p.Name)
		}
		if p.Solution == "" {
			t.Errorf("pattern %q has empty solution", p.Name)
		}
		if len(p.Tags) == 0 {
			t.Errorf("pattern %q has no tags", p.Name)
		}
	}
}

package architect

import (
	"strings"
	"testing"
)

func TestNewADR_Defaults(t *testing.T) {
	adr := NewADR(1, "Use PostgreSQL for primary storage")
	if adr.Number != 1 {
		t.Errorf("expected number 1, got %d", adr.Number)
	}
	if adr.Title != "Use PostgreSQL for primary storage" {
		t.Errorf("unexpected title: %q", adr.Title)
	}
	if adr.Status != ADRProposed {
		t.Errorf("expected proposed status, got %q", adr.Status)
	}
	if adr.Date == "" {
		t.Error("expected date to be set")
	}
	if adr.Supersedes != 0 {
		t.Errorf("expected supersedes 0, got %d", adr.Supersedes)
	}
}

func TestADR_AddOption(t *testing.T) {
	adr := NewADR(2, "Choose messaging system")
	adr.AddOption("RabbitMQ", []string{"mature", "flexible routing"}, []string{"complex setup"})
	adr.AddOption("Kafka", []string{"high throughput"}, []string{"overkill for small scale"})

	if len(adr.Options) != 2 {
		t.Fatalf("expected 2 options, got %d", len(adr.Options))
	}
	if adr.Options[0].Name != "RabbitMQ" {
		t.Errorf("expected RabbitMQ, got %q", adr.Options[0].Name)
	}
	if len(adr.Options[0].Pros) != 2 {
		t.Errorf("expected 2 pros, got %d", len(adr.Options[0].Pros))
	}
	if adr.Options[1].Chosen {
		t.Error("option should not be chosen by default")
	}
}

func TestADR_Choose_Valid(t *testing.T) {
	adr := NewADR(3, "Select cache")
	adr.AddOption("Redis", []string{"fast"}, []string{"memory cost"})
	adr.AddOption("Memcached", []string{"simple"}, []string{"no persistence"})

	err := adr.Choose("Redis")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if adr.Status != ADRAccepted {
		t.Errorf("expected accepted, got %q", adr.Status)
	}

	// Verify only Redis is chosen
	for _, opt := range adr.Options {
		if opt.Name == "Redis" && !opt.Chosen {
			t.Error("Redis should be chosen")
		}
		if opt.Name == "Memcached" && opt.Chosen {
			t.Error("Memcached should not be chosen")
		}
	}
}

func TestADR_Choose_Invalid(t *testing.T) {
	adr := NewADR(4, "Test")
	adr.AddOption("A", nil, nil)

	err := adr.Choose("NonExistent")
	if err == nil {
		t.Error("expected error for nonexistent option")
	}
}

func TestADR_Choose_ClearsOldSelection(t *testing.T) {
	adr := NewADR(5, "Test rechoose")
	adr.AddOption("A", nil, nil)
	adr.AddOption("B", nil, nil)

	_ = adr.Choose("A")
	_ = adr.Choose("B")

	for _, opt := range adr.Options {
		if opt.Name == "A" && opt.Chosen {
			t.Error("A should no longer be chosen")
		}
		if opt.Name == "B" && !opt.Chosen {
			t.Error("B should be chosen")
		}
	}
}

func TestADR_FormatMarkdown_ContainsRequiredSections(t *testing.T) {
	adr := NewADR(1, "Use Go for CLI")
	adr.Context = "We need a compiled language for CLI distribution."
	adr.Decision = "We will use Go."
	adr.Consequences = []string{"+ Easy cross-compilation", "- Learning curve for team"}
	adr.AddOption("Go", []string{"fast compilation"}, []string{"verbose"})
	adr.AddOption("Rust", []string{"memory safety"}, []string{"steep learning curve"})
	_ = adr.Choose("Go")

	md := adr.FormatMarkdown()

	required := []string{
		"# ADR-0001",
		"Use Go for CLI",
		"## Status",
		"accepted",
		"## Context",
		"compiled language",
		"## Decision",
		"We will use Go",
		"## Consequences",
		"Easy cross-compilation",
		"## Options",
		"Go",
		"Rust",
		"Pros",
		"Cons",
	}
	for _, s := range required {
		if !strings.Contains(md, s) {
			t.Errorf("markdown missing %q", s)
		}
	}
}

func TestADR_FormatMarkdown_Supersedes(t *testing.T) {
	adr := NewADR(5, "Switch to Rust")
	adr.Supersedes = 1
	md := adr.FormatMarkdown()
	if !strings.Contains(md, "Supersedes: ADR-0001") {
		t.Error("markdown should mention superseded ADR")
	}
}

func TestParseADRFromMarkdown_Roundtrip(t *testing.T) {
	original := NewADR(7, "Use SQLite for local storage")
	original.Context = "Need embedded database."
	original.Decision = "SQLite is the best fit."
	original.Consequences = []string{"+ Zero setup", "- Limited concurrency"}
	original.AddOption("SQLite", []string{"embedded"}, []string{"no network"})
	original.AddOption("LevelDB", []string{"fast writes"}, []string{"no SQL"})
	_ = original.Choose("SQLite")

	md := original.FormatMarkdown()
	parsed, err := ParseADRFromMarkdown(md)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if parsed.Number != 7 {
		t.Errorf("expected number 7, got %d", parsed.Number)
	}
	if parsed.Title != "Use SQLite for local storage" {
		t.Errorf("unexpected title: %q", parsed.Title)
	}
	if parsed.Status != ADRAccepted {
		t.Errorf("expected accepted, got %q", parsed.Status)
	}
	if !strings.Contains(parsed.Context, "embedded database") {
		t.Error("context not parsed correctly")
	}
	if !strings.Contains(parsed.Decision, "SQLite is the best fit") {
		t.Error("decision not parsed correctly")
	}
	if len(parsed.Consequences) < 2 {
		t.Errorf("expected at least 2 consequences, got %d", len(parsed.Consequences))
	}
}

func TestParseADRFromMarkdown_InvalidInput(t *testing.T) {
	_, err := ParseADRFromMarkdown("not a valid ADR")
	if err == nil {
		t.Error("expected error for invalid markdown")
	}
}

package agent

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// ContextLayer
// ---------------------------------------------------------------------------

func TestNewContextLayer(t *testing.T) {
	l := NewContextLayer("test", "content", 100, LayerRequired)
	if l.Name != "test" || l.Content != "content" || l.Priority != 100 || l.Level != LayerRequired {
		t.Error("fields not set correctly")
	}
}

// ---------------------------------------------------------------------------
// ContextAssembler — basics
// ---------------------------------------------------------------------------

func TestNewContextAssembler(t *testing.T) {
	a := NewContextAssembler(4000)
	if a.tokenBudget != 4000 {
		t.Errorf("expected budget 4000, got %d", a.tokenBudget)
	}
}

func TestAssemble_EmptyLayers(t *testing.T) {
	a := NewContextAssembler(4000)
	result := a.Assemble()
	if result != "" {
		t.Errorf("expected empty, got %q", result)
	}
}

func TestAssemble_SingleLayer(t *testing.T) {
	a := NewContextAssembler(4000)
	a.AddLayer(NewContextLayer("persona", "You are BlackCat.", 100, LayerRequired))
	result := a.Assemble()
	if !strings.Contains(result, "You are BlackCat.") {
		t.Error("expected persona in output")
	}
}

// ---------------------------------------------------------------------------
// Priority ordering
// ---------------------------------------------------------------------------

func TestAssemble_PriorityOrder(t *testing.T) {
	a := NewContextAssembler(4000)
	a.AddLayer(NewContextLayer("low", "LOW PRIORITY", 10, LayerOptional))
	a.AddLayer(NewContextLayer("high", "HIGH PRIORITY", 90, LayerRequired))
	a.AddLayer(NewContextLayer("mid", "MID PRIORITY", 50, LayerImportant))

	result := a.Assemble()

	highIdx := strings.Index(result, "HIGH PRIORITY")
	midIdx := strings.Index(result, "MID PRIORITY")
	lowIdx := strings.Index(result, "LOW PRIORITY")

	if highIdx > midIdx {
		t.Error("high priority should come before mid")
	}
	if midIdx > lowIdx {
		t.Error("mid priority should come before low")
	}
}

// ---------------------------------------------------------------------------
// Token budget enforcement
// ---------------------------------------------------------------------------

func TestAssemble_TokenBudget_DropsOptional(t *testing.T) {
	a := NewContextAssembler(50) // very tight budget (~200 chars)

	a.AddLayer(NewContextLayer("persona", "You are BlackCat, an AI agent.", 100, LayerRequired))
	a.AddLayer(NewContextLayer("optional", strings.Repeat("X", 500), 10, LayerOptional))

	result := a.Assemble()

	if !strings.Contains(result, "BlackCat") {
		t.Error("required layer should be present")
	}
	if strings.Contains(result, "XXXXX") {
		t.Error("optional layer should be dropped when over budget")
	}
}

func TestAssemble_RequiredAlwaysIncluded(t *testing.T) {
	a := NewContextAssembler(10) // extremely tight

	a.AddLayer(NewContextLayer("persona", "You are BlackCat.", 100, LayerRequired))
	result := a.Assemble()

	if !strings.Contains(result, "BlackCat") {
		t.Error("required layer must always be included even over budget")
	}
}

func TestAssemble_ImportantBeforeOptional(t *testing.T) {
	a := NewContextAssembler(100) // tight

	a.AddLayer(NewContextLayer("optional", strings.Repeat("A", 200), 10, LayerOptional))
	a.AddLayer(NewContextLayer("important", "IMPORTANT STUFF", 50, LayerImportant))
	a.AddLayer(NewContextLayer("persona", "Base.", 100, LayerRequired))

	result := a.Assemble()

	if !strings.Contains(result, "IMPORTANT STUFF") {
		t.Error("important should be included before optional")
	}
}

// ---------------------------------------------------------------------------
// Section headers
// ---------------------------------------------------------------------------

func TestAssemble_SectionHeaders(t *testing.T) {
	a := NewContextAssembler(4000)
	a.AddLayer(NewContextLayer("memory", "User prefers Go.", 80, LayerImportant))

	result := a.Assemble()
	if !strings.Contains(result, "## Memory") {
		t.Error("expected section header for memory layer")
	}
}

func TestAssemble_PersonaNoHeader(t *testing.T) {
	a := NewContextAssembler(4000)
	a.AddLayer(NewContextLayer("persona", "You are BlackCat.", 100, LayerRequired))

	result := a.Assemble()
	// Persona should NOT have a section header, it's the base text
	if strings.Contains(result, "## Persona") {
		t.Error("persona should not have a section header")
	}
}

// ---------------------------------------------------------------------------
// EstimateTokens
// ---------------------------------------------------------------------------

func TestEstimateTokens(t *testing.T) {
	tokens := EstimateTokens("hello world") // 11 chars
	if tokens < 2 || tokens > 5 {
		t.Errorf("expected ~3 tokens for 'hello world', got %d", tokens)
	}
}

func TestEstimateTokens_Empty(t *testing.T) {
	if EstimateTokens("") != 0 {
		t.Error("empty string should be 0 tokens")
	}
}

// ---------------------------------------------------------------------------
// Layer levels
// ---------------------------------------------------------------------------

func TestLayerLevels(t *testing.T) {
	if LayerRequired >= LayerImportant {
		t.Error("required should have lower numeric value (higher priority)")
	}
	if LayerImportant >= LayerOptional {
		t.Error("important should be between required and optional")
	}
}

// ---------------------------------------------------------------------------
// Multiple layers of same level
// ---------------------------------------------------------------------------

func TestAssemble_MultipleSameLevel(t *testing.T) {
	a := NewContextAssembler(4000)
	a.AddLayer(NewContextLayer("a", "AAA", 90, LayerRequired))
	a.AddLayer(NewContextLayer("b", "BBB", 80, LayerRequired))
	a.AddLayer(NewContextLayer("c", "CCC", 70, LayerRequired))

	result := a.Assemble()
	if !strings.Contains(result, "AAA") || !strings.Contains(result, "BBB") || !strings.Contains(result, "CCC") {
		t.Error("all required layers should be present")
	}
}

// ---------------------------------------------------------------------------
// TokensUsed
// ---------------------------------------------------------------------------

func TestTokensUsed(t *testing.T) {
	a := NewContextAssembler(4000)
	a.AddLayer(NewContextLayer("persona", "You are BlackCat.", 100, LayerRequired))
	_ = a.Assemble()

	used := a.TokensUsed()
	if used == 0 {
		t.Error("tokens used should be > 0 after assemble")
	}
}

func TestTokensRemaining(t *testing.T) {
	a := NewContextAssembler(4000)
	a.AddLayer(NewContextLayer("persona", "You are BlackCat.", 100, LayerRequired))
	_ = a.Assemble()

	remaining := a.TokensRemaining()
	if remaining >= 4000 {
		t.Error("remaining should be less than budget after adding content")
	}
	if remaining < 0 {
		t.Error("remaining should not be negative")
	}
}

// ---------------------------------------------------------------------------
// Real-world scenario
// ---------------------------------------------------------------------------

func TestAssemble_FullScenario(t *testing.T) {
	a := NewContextAssembler(2000)

	// Persona (required, always included)
	a.AddLayer(NewContextLayer("persona",
		"You are BlackCat, an AI coding agent by MeowAI. You help users with software engineering tasks.",
		100, LayerRequired))

	// Domain knowledge (required when detected)
	a.AddLayer(NewContextLayer("domain",
		"You are also a DevSecOps specialist. Scan for vulnerabilities, check dependencies, harden pipelines.",
		95, LayerRequired))

	// ReAct instructions (important)
	a.AddLayer(NewContextLayer("reasoning",
		"Use structured reasoning: <thinking>, <action>, <critique>. Self-critique with confidence >= 0.7.",
		85, LayerImportant))

	// Memory snapshot (important)
	a.AddLayer(NewContextLayer("memory",
		"User prefers Go. Last session worked on auth middleware.",
		80, LayerImportant))

	// Skills index (important)
	a.AddLayer(NewContextLayer("skills",
		"Available skills:\n1. secret-scanner\n2. dependency-audit\n3. dockerfile-hardener",
		70, LayerImportant))

	// Behavioral nudges (optional)
	a.AddLayer(NewContextLayer("nudges",
		"Consider writing a memory entry about what you learned.",
		20, LayerOptional))

	result := a.Assemble()

	// All required/important should be present
	if !strings.Contains(result, "BlackCat") {
		t.Error("missing persona")
	}
	if !strings.Contains(result, "DevSecOps") {
		t.Error("missing domain")
	}
	if !strings.Contains(result, "structured reasoning") {
		t.Error("missing reasoning")
	}
	if !strings.Contains(result, "prefers Go") {
		t.Error("missing memory")
	}
	if !strings.Contains(result, "secret-scanner") {
		t.Error("missing skills")
	}
}

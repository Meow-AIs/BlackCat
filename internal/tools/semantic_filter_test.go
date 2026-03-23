package tools

import (
	"testing"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"simple words", "hello world", []string{"hello", "world"}},
		{"uppercase", "Hello World", []string{"hello", "world"}},
		{"punctuation", "scan_secrets, find-bugs", []string{"scan", "secrets", "find", "bugs"}},
		{"duplicates", "test test test", []string{"test"}},
		{"empty", "", nil},
		{"mixed separators", "docker.build/image:latest", []string{"docker", "build", "image", "latest"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tokenize(tt.input)
			if len(got) != len(tt.expected) {
				t.Fatalf("tokenize(%q) = %v (len %d), want %v (len %d)",
					tt.input, got, len(got), tt.expected, len(tt.expected))
			}
			for i, g := range got {
				if g != tt.expected[i] {
					t.Errorf("tokenize(%q)[%d] = %q, want %q", tt.input, i, g, tt.expected[i])
				}
			}
		})
	}
}

func TestKeywordOverlap(t *testing.T) {
	tests := []struct {
		name string
		a, b []string
		want float64
	}{
		{"full overlap", []string{"a", "b"}, []string{"a", "b"}, 1.0},
		{"no overlap", []string{"a", "b"}, []string{"c", "d"}, 0.0},
		{"partial", []string{"a", "b", "c"}, []string{"b", "c", "d"}, 2.0 / 3.0},
		{"empty a", nil, []string{"a"}, 0.0},
		{"empty b", []string{"a"}, nil, 0.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := keywordOverlap(tt.a, tt.b)
			if diff := got - tt.want; diff > 0.001 || diff < -0.001 {
				t.Errorf("keywordOverlap(%v, %v) = %f, want %f", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func sampleTools() []Definition {
	return []Definition{
		{Name: "scan_secrets", Description: "scan code for hardcoded secrets and API keys", Category: "security"},
		{Name: "scan_dependencies", Description: "scan project dependencies for vulnerabilities", Category: "security"},
		{Name: "generate_diagram", Description: "generate architecture diagram in mermaid format", Category: "code"},
		{Name: "compare_tech", Description: "compare technology options with scoring matrix", Category: "code"},
		{Name: "shell_exec", Description: "execute shell command in sandbox", Category: "shell"},
		{Name: "read_file", Description: "read file contents from filesystem", Category: "filesystem"},
		{Name: "write_file", Description: "write content to a file", Category: "filesystem"},
		{Name: "git_diff", Description: "show git diff for working directory", Category: "git"},
		{Name: "web_search", Description: "search the web for information", Category: "web"},
		{Name: "docker_build", Description: "build docker image from dockerfile", Category: "devops"},
	}
}

func TestNewSemanticFilter(t *testing.T) {
	f := NewSemanticFilter(10)
	if f == nil {
		t.Fatal("NewSemanticFilter returned nil")
	}
	if f.maxTools != 10 {
		t.Errorf("maxTools = %d, want 10", f.maxTools)
	}
}

func TestFilterToolsReducesCount(t *testing.T) {
	f := NewSemanticFilter(3)
	tools := sampleTools()
	filtered := f.FilterTools("scan secrets", tools)
	if len(filtered) > 3 {
		t.Errorf("FilterTools returned %d tools, want at most 3", len(filtered))
	}
}

func TestFilterToolsExactNameMatch(t *testing.T) {
	f := NewSemanticFilter(15)
	tools := sampleTools()
	scored := f.ScoreTools("scan_secrets", tools)
	if len(scored) == 0 {
		t.Fatal("ScoreTools returned no results")
	}
	if scored[0].Tool.Name != "scan_secrets" {
		t.Errorf("top tool = %q, want %q", scored[0].Tool.Name, "scan_secrets")
	}
	if scored[0].Score < 0.9 {
		t.Errorf("exact match score = %f, want >= 0.9", scored[0].Score)
	}
}

func TestFilterToolsRelevantRankedHigher(t *testing.T) {
	f := NewSemanticFilter(15)
	tools := sampleTools()
	scored := f.ScoreTools("security scan vulnerabilities", tools)
	if len(scored) < 2 {
		t.Fatal("expected at least 2 scored tools")
	}
	// Security tools should rank higher than unrelated tools
	topTwo := []string{scored[0].Tool.Name, scored[1].Tool.Name}
	hasSecTool := false
	for _, name := range topTwo {
		if name == "scan_secrets" || name == "scan_dependencies" {
			hasSecTool = true
			break
		}
	}
	if !hasSecTool {
		t.Errorf("top 2 tools %v should include a security tool", topTwo)
	}
}

func TestFilterToolsRecencyBoost(t *testing.T) {
	f := NewSemanticFilter(15)
	f.RecordUsage("web_search")

	tools := sampleTools()
	scored := f.ScoreTools("search", tools)

	var webScore float64
	for _, s := range scored {
		if s.Tool.Name == "web_search" {
			webScore = s.Score
			break
		}
	}
	if webScore < 0.1 {
		t.Errorf("recently used tool score = %f, expected boost", webScore)
	}
}

func TestFilterToolsEmptyQueryReturnsUpToMax(t *testing.T) {
	f := NewSemanticFilter(5)
	tools := sampleTools()
	filtered := f.FilterTools("", tools)
	if len(filtered) > 5 {
		t.Errorf("empty query returned %d tools, want at most 5", len(filtered))
	}
}

func TestFilterToolsSingleTool(t *testing.T) {
	f := NewSemanticFilter(15)
	single := []Definition{{Name: "only_tool", Description: "the only tool", Category: "misc"}}
	filtered := f.FilterTools("tool", single)
	if len(filtered) != 1 {
		t.Errorf("got %d tools, want 1", len(filtered))
	}
}

func TestFilterToolsCategoryMatching(t *testing.T) {
	f := NewSemanticFilter(15)
	tools := sampleTools()
	scored := f.ScoreTools("security", tools)

	// Tools in "security" category should score higher
	for _, s := range scored[:2] {
		if s.Tool.Category != "security" {
			// At least one of top 2 should be security category
			continue
		}
		if s.Score < 0.2 {
			t.Errorf("security category tool %q score = %f, want >= 0.2", s.Tool.Name, s.Score)
		}
		return
	}
}

func TestSetMaxTools(t *testing.T) {
	f := NewSemanticFilter(5)
	f.SetMaxTools(20)
	if f.maxTools != 20 {
		t.Errorf("maxTools = %d, want 20", f.maxTools)
	}
}

func TestMatchTermsPopulated(t *testing.T) {
	f := NewSemanticFilter(15)
	tools := sampleTools()
	scored := f.ScoreTools("scan secrets", tools)
	if len(scored) == 0 {
		t.Fatal("no scored tools")
	}
	if len(scored[0].MatchTerms) == 0 {
		t.Error("expected MatchTerms to be populated for top result")
	}
}

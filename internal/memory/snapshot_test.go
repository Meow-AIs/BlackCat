package memory

import (
	"strings"
	"testing"
)

func TestBuildSnapshotFromEntries_Basic(t *testing.T) {
	entries := []Entry{
		{ID: "1", Tier: TierEpisodic, Content: "User discussed Go testing patterns.", Score: 0.9},
		{ID: "2", Tier: TierSemantic, Content: "Project uses Go 1.25 with CGo.", Score: 0.8},
		{ID: "3", Tier: TierProcedural, Content: "Run tests with: go test ./...", Score: 0.7},
	}

	snap := BuildSnapshotFromEntries(entries, 1000)

	if snap.EntryCount != 3 {
		t.Errorf("EntryCount = %d, want 3", snap.EntryCount)
	}
	if snap.Content == "" {
		t.Error("Content should not be empty")
	}
	if !strings.Contains(snap.Content, "Go testing patterns") {
		t.Error("Content should include first entry text")
	}
	if snap.TokenCount <= 0 {
		t.Error("TokenCount should be positive")
	}
}

func TestBuildSnapshotFromEntries_RespectMaxTokens(t *testing.T) {
	// Each entry is about 10 words ~= ~13 tokens (at ~4 chars per token)
	entries := make([]Entry, 100)
	for i := range entries {
		entries[i] = Entry{
			ID:      "e-" + string(rune('a'+i%26)),
			Tier:    TierEpisodic,
			Content: "This is a relatively long entry that contains several words for testing purposes in the memory engine.",
			Score:   float64(100-i) / 100.0,
		}
	}

	snap := BuildSnapshotFromEntries(entries, 50)

	if snap.TokenCount > 50 {
		t.Errorf("TokenCount = %d, exceeds max 50", snap.TokenCount)
	}
	if snap.EntryCount >= 100 {
		t.Error("should not include all entries when token budget is small")
	}
}

func TestBuildSnapshotFromEntries_Empty(t *testing.T) {
	snap := BuildSnapshotFromEntries(nil, 1000)

	if snap.EntryCount != 0 {
		t.Errorf("EntryCount = %d, want 0", snap.EntryCount)
	}
	if snap.Content != "" {
		t.Errorf("Content = %q, want empty", snap.Content)
	}
	if snap.TokenCount != 0 {
		t.Errorf("TokenCount = %d, want 0", snap.TokenCount)
	}
}

func TestBuildSnapshotFromEntries_OrderPreserved(t *testing.T) {
	entries := []Entry{
		{ID: "1", Content: "FIRST_ENTRY", Score: 0.9},
		{ID: "2", Content: "SECOND_ENTRY", Score: 0.8},
	}

	snap := BuildSnapshotFromEntries(entries, 1000)

	firstIdx := strings.Index(snap.Content, "FIRST_ENTRY")
	secondIdx := strings.Index(snap.Content, "SECOND_ENTRY")

	if firstIdx == -1 || secondIdx == -1 {
		t.Fatal("both entries should appear in snapshot")
	}
	if firstIdx > secondIdx {
		t.Error("entries should maintain order")
	}
}

func TestBuildSnapshotFromEntries_TierLabels(t *testing.T) {
	entries := []Entry{
		{ID: "1", Tier: TierEpisodic, Content: "episodic content"},
		{ID: "2", Tier: TierSemantic, Content: "semantic content"},
		{ID: "3", Tier: TierProcedural, Content: "procedural content"},
	}

	snap := BuildSnapshotFromEntries(entries, 1000)

	if !strings.Contains(snap.Content, "episodic") {
		t.Error("should contain episodic label or content")
	}
	if !strings.Contains(snap.Content, "semantic") {
		t.Error("should contain semantic label or content")
	}
	if !strings.Contains(snap.Content, "procedural") {
		t.Error("should contain procedural label or content")
	}
}

func TestBuildSnapshotFromEntries_ZeroMaxTokens(t *testing.T) {
	entries := []Entry{
		{ID: "1", Content: "Something"},
	}

	snap := BuildSnapshotFromEntries(entries, 0)
	if snap.EntryCount != 0 {
		t.Errorf("EntryCount = %d, want 0 with zero budget", snap.EntryCount)
	}
}

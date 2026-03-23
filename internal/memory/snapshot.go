package memory

import (
	"fmt"
	"strings"
)

// BuildSnapshotFromEntries selects top entries by order, formats them as text,
// and estimates tokens. Entries are included in order until the token budget
// is exhausted. This function does not require CGo.
func BuildSnapshotFromEntries(entries []Entry, maxTokens int) Snapshot {
	if len(entries) == 0 || maxTokens <= 0 {
		return Snapshot{}
	}

	var parts []string
	totalTokens := 0
	count := 0

	for _, entry := range entries {
		line := formatEntry(entry)
		tokens := estimateTokens(line)

		if totalTokens+tokens > maxTokens {
			break
		}

		parts = append(parts, line)
		totalTokens += tokens
		count++
	}

	return Snapshot{
		Content:    strings.Join(parts, "\n"),
		TokenCount: totalTokens,
		EntryCount: count,
	}
}

// formatEntry renders a single memory entry as a labeled line.
func formatEntry(entry Entry) string {
	tier := string(entry.Tier)
	if tier == "" {
		tier = "general"
	}
	return fmt.Sprintf("[%s] %s", tier, entry.Content)
}

// estimateTokens provides a rough token count using the ~4 characters per
// token heuristic common for English text with GPT-style tokenizers.
func estimateTokens(text string) int {
	if text == "" {
		return 0
	}
	tokens := len(text) / 4
	if tokens == 0 {
		tokens = 1
	}
	return tokens
}

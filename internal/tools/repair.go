package tools

import "strings"

// RepairResult describes the outcome of a tool name repair attempt.
type RepairResult struct {
	Original   string
	Repaired   string
	Strategy   string  // "exact", "lowercase", "normalize", "fuzzy", "none"
	Confidence float64 // 0.0-1.0
}

// RepairToolName attempts to match a potentially misspelled tool name
// against the available list. Returns a RepairResult describing the outcome.
// Strategies are tried in order: exact, lowercase, normalize, fuzzy.
func RepairToolName(name string, available []string) RepairResult {
	noMatch := RepairResult{
		Original:   name,
		Repaired:   "",
		Strategy:   "none",
		Confidence: 0.0,
	}

	if name == "" || len(available) == 0 {
		return noMatch
	}

	// Strategy 1: exact match
	for _, candidate := range available {
		if candidate == name {
			return RepairResult{
				Original:   name,
				Repaired:   candidate,
				Strategy:   "exact",
				Confidence: 1.0,
			}
		}
	}

	// Strategy 2: lowercase match
	lower := strings.ToLower(name)
	for _, candidate := range available {
		if strings.ToLower(candidate) == lower {
			return RepairResult{
				Original:   name,
				Repaired:   candidate,
				Strategy:   "lowercase",
				Confidence: 1.0,
			}
		}
	}

	// Strategy 3: normalize (hyphens and spaces -> underscores), then lowercase compare
	normalized := strings.ToLower(strings.NewReplacer("-", "_", " ", "_").Replace(name))
	for _, candidate := range available {
		normalizedCandidate := strings.ToLower(strings.NewReplacer("-", "_", " ", "_").Replace(candidate))
		if normalizedCandidate == normalized {
			return RepairResult{
				Original:   name,
				Repaired:   candidate,
				Strategy:   "normalize",
				Confidence: 1.0,
			}
		}
	}

	// Strategy 4: fuzzy match using Levenshtein distance
	const fuzzyThreshold = 0.7
	bestScore := 0.0
	bestCandidate := ""

	for _, candidate := range available {
		score := FuzzyScore(strings.ToLower(name), strings.ToLower(candidate))
		if score > bestScore {
			bestScore = score
			bestCandidate = candidate
		}
	}

	if bestScore > fuzzyThreshold {
		return RepairResult{
			Original:   name,
			Repaired:   bestCandidate,
			Strategy:   "fuzzy",
			Confidence: bestScore,
		}
	}

	return noMatch
}

// LevenshteinDistance computes the edit distance between two strings.
func LevenshteinDistance(a, b string) int {
	la, lb := len(a), len(b)

	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	// Use two rows to reduce memory from O(m*n) to O(n).
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)

	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			if a[i-1] == b[j-1] {
				curr[j] = prev[j-1]
			} else {
				curr[j] = 1 + min3(prev[j], curr[j-1], prev[j-1])
			}
		}
		prev, curr = curr, prev
	}

	return prev[lb]
}

// FuzzyScore returns a similarity score (0.0-1.0) between two strings
// based on Levenshtein distance normalized by max length.
func FuzzyScore(a, b string) float64 {
	if a == "" && b == "" {
		return 1.0
	}

	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	if maxLen == 0 {
		return 1.0
	}

	dist := LevenshteinDistance(a, b)
	return 1.0 - float64(dist)/float64(maxLen)
}

// min3 returns the minimum of three integers.
func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

package tools

import (
	"sort"
	"strings"
	"unicode"
)

// ToolRelevance pairs a tool definition with its relevance score.
type ToolRelevance struct {
	Tool       Definition
	Score      float64  // 0-1 relevance score
	MatchTerms []string // which query terms matched
}

// SemanticFilter selects the most relevant tools for a given query,
// reducing the number of tools sent to the LLM and improving selection accuracy.
type SemanticFilter struct {
	maxTools     int
	recencyBoost float64
	recentTools  []string
}

const (
	defaultMaxTools     = 15
	defaultRecencyBoost = 0.2
	maxRecentTools      = 20

	boostExactName   = 1.0
	boostCategory    = 0.3
	boostDescription = 0.5
)

// NewSemanticFilter creates a filter that returns at most maxTools tools.
func NewSemanticFilter(maxTools int) *SemanticFilter {
	if maxTools <= 0 {
		maxTools = defaultMaxTools
	}
	return &SemanticFilter{
		maxTools:     maxTools,
		recencyBoost: defaultRecencyBoost,
		recentTools:  nil,
	}
}

// SetMaxTools changes the maximum number of tools returned.
func (f *SemanticFilter) SetMaxTools(max int) {
	if max > 0 {
		f.maxTools = max
	}
}

// RecordUsage tracks a tool as recently used for recency boosting.
func (f *SemanticFilter) RecordUsage(toolName string) {
	// Remove if already present to move to front
	filtered := make([]string, 0, len(f.recentTools)+1)
	filtered = append(filtered, toolName)
	for _, name := range f.recentTools {
		if name != toolName {
			filtered = append(filtered, name)
		}
	}
	if len(filtered) > maxRecentTools {
		filtered = filtered[:maxRecentTools]
	}
	f.recentTools = filtered
}

// FilterTools returns the top-scoring tools for the query, up to maxTools.
func (f *SemanticFilter) FilterTools(query string, allTools []Definition) []Definition {
	scored := f.ScoreTools(query, allTools)
	limit := f.maxTools
	if limit > len(scored) {
		limit = len(scored)
	}
	result := make([]Definition, limit)
	for i := 0; i < limit; i++ {
		result[i] = scored[i].Tool
	}
	return result
}

// ScoreTools computes relevance scores for all tools and returns them sorted descending.
func (f *SemanticFilter) ScoreTools(query string, allTools []Definition) []ToolRelevance {
	queryTokens := tokenize(query)
	results := make([]ToolRelevance, 0, len(allTools))

	for _, tool := range allTools {
		score, matched := f.scoreTool(queryTokens, tool)
		results = append(results, ToolRelevance{
			Tool:       tool,
			Score:      score,
			MatchTerms: matched,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

// scoreTool computes the relevance score for a single tool.
func (f *SemanticFilter) scoreTool(queryTokens []string, tool Definition) (float64, []string) {
	if len(queryTokens) == 0 {
		return 0.0, nil
	}

	nameTokens := tokenize(tool.Name)
	descTokens := tokenize(tool.Description)
	catTokens := tokenize(tool.Category)

	var score float64
	matchedSet := make(map[string]bool)

	// 1. Exact name match (query contains the full tool name)
	queryLower := strings.ToLower(strings.Join(queryTokens, " "))
	nameLower := strings.ToLower(tool.Name)
	if queryLower == nameLower || strings.Contains(queryLower, nameLower) {
		score += boostExactName
		for _, t := range nameTokens {
			matchedSet[t] = true
		}
	}

	// 2. Category keyword match
	catOverlap := keywordOverlap(queryTokens, catTokens)
	if catOverlap > 0 {
		score += boostCategory * catOverlap
		for _, qt := range queryTokens {
			for _, ct := range catTokens {
				if qt == ct {
					matchedSet[qt] = true
				}
			}
		}
	}

	// 3. Description keyword overlap
	descOverlap := keywordOverlap(queryTokens, descTokens)
	if descOverlap > 0 {
		score += boostDescription * descOverlap
		for _, qt := range queryTokens {
			for _, dt := range descTokens {
				if qt == dt {
					matchedSet[qt] = true
				}
			}
		}
	}

	// 4. Name token overlap (partial name match, not exact)
	nameOverlap := keywordOverlap(queryTokens, nameTokens)
	if nameOverlap > 0 && score < boostExactName {
		score += 0.4 * nameOverlap
		for _, qt := range queryTokens {
			for _, nt := range nameTokens {
				if qt == nt {
					matchedSet[qt] = true
				}
			}
		}
	}

	// 5. Recency boost
	if f.isRecent(tool.Name) {
		score += f.recencyBoost
	}

	// Cap at 1.0 + recency
	maxScore := 1.0 + f.recencyBoost
	if score > maxScore {
		score = maxScore
	}

	matched := make([]string, 0, len(matchedSet))
	for term := range matchedSet {
		matched = append(matched, term)
	}
	sort.Strings(matched)

	return score, matched
}

// isRecent checks whether a tool was recently used.
func (f *SemanticFilter) isRecent(name string) bool {
	for _, n := range f.recentTools {
		if n == name {
			return true
		}
	}
	return false
}

// tokenize splits text into lowercase alphabetic tokens, deduplicated.
func tokenize(text string) []string {
	if text == "" {
		return nil
	}
	words := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	seen := make(map[string]bool, len(words))
	result := make([]string, 0, len(words))
	for _, w := range words {
		if w != "" && !seen[w] {
			seen[w] = true
			result = append(result, w)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// keywordOverlap returns the fraction of tokens in a that appear in b.
// Returns 0 if a is empty.
func keywordOverlap(a, b []string) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}
	bSet := make(map[string]bool, len(b))
	for _, w := range b {
		bSet[w] = true
	}
	count := 0
	for _, w := range a {
		if bSet[w] {
			count++
		}
	}
	return float64(count) / float64(len(a))
}

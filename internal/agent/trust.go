package agent

import (
	"fmt"
	"strings"
)

// TrustSignal represents a single signal contributing to a trust score.
type TrustSignal struct {
	Name   string  // signal identifier
	Score  float64 // 0.0-1.0
	Weight float64 // importance weight
	Reason string  // why this score
}

// TrustScore is the aggregated trust evaluation of an LLM response.
type TrustScore struct {
	Overall     float64       // weighted average 0-1
	Signals     []TrustSignal // individual signals
	Verdict     TrustVerdict  // categorical judgment
	ShouldRetry bool          // true if score below retry threshold
}

// TrustVerdict categorizes the trust level.
type TrustVerdict string

const (
	VerdictTrusted   TrustVerdict = "trusted"   // >= warnThreshold (0.7)
	VerdictUncertain TrustVerdict = "uncertain" // retryThreshold to warnThreshold
	VerdictUntrusted TrustVerdict = "untrusted" // < retryThreshold (0.5)
)

// TrustScorer evaluates LLM response trustworthiness from multiple signals.
type TrustScorer struct {
	retryThreshold float64 // below this triggers retry (default 0.5)
	warnThreshold  float64 // below this is uncertain (default 0.7)
}

// NewTrustScorer creates a TrustScorer with default thresholds.
func NewTrustScorer() *TrustScorer {
	return &TrustScorer{
		retryThreshold: 0.5,
		warnThreshold:  0.7,
	}
}

// SetThresholds updates the retry and warn thresholds.
func (ts *TrustScorer) SetThresholds(retry, warn float64) {
	ts.retryThreshold = retry
	ts.warnThreshold = warn
}

// Score computes an aggregate TrustScore from the given signals.
// Returns untrusted with ShouldRetry if no signals are provided.
func (ts *TrustScorer) Score(signals []TrustSignal) TrustScore {
	if len(signals) == 0 {
		return TrustScore{
			Overall:     0.0,
			Signals:     nil,
			Verdict:     VerdictUntrusted,
			ShouldRetry: true,
		}
	}

	var weightedSum, totalWeight float64
	for _, s := range signals {
		weightedSum += s.Score * s.Weight
		totalWeight += s.Weight
	}

	var overall float64
	if totalWeight > 0 {
		overall = weightedSum / totalWeight
	}

	// Copy signals for immutability.
	copied := make([]TrustSignal, len(signals))
	copy(copied, signals)

	verdict := ts.classify(overall)

	return TrustScore{
		Overall:     overall,
		Signals:     copied,
		Verdict:     verdict,
		ShouldRetry: overall < ts.retryThreshold,
	}
}

// classify maps a numeric score to a TrustVerdict.
func (ts *TrustScorer) classify(score float64) TrustVerdict {
	switch {
	case score >= ts.warnThreshold:
		return VerdictTrusted
	case score >= ts.retryThreshold:
		return VerdictUncertain
	default:
		return VerdictUntrusted
	}
}

// clamp restricts a value to [0.0, 1.0].
func clamp01(v float64) float64 {
	if v < 0.0 {
		return 0.0
	}
	if v > 1.0 {
		return 1.0
	}
	return v
}

// SignalFromConfidence creates a trust signal from self-critique confidence.
func SignalFromConfidence(confidence float64) TrustSignal {
	score := clamp01(confidence)
	return TrustSignal{
		Name:   "confidence",
		Score:  score,
		Weight: 2.0,
		Reason: fmt.Sprintf("self-critique confidence: %.2f", score),
	}
}

// SignalFromToolValidity creates a trust signal from the ratio of valid tool calls.
// If totalCalls is 0, returns a perfect score (no tools means no tool errors).
func SignalFromToolValidity(validCalls, totalCalls int) TrustSignal {
	if totalCalls == 0 {
		return TrustSignal{
			Name:   "tool_validity",
			Score:  1.0,
			Weight: 1.5,
			Reason: "no tool calls to validate",
		}
	}
	score := float64(validCalls) / float64(totalCalls)
	return TrustSignal{
		Name:   "tool_validity",
		Score:  score,
		Weight: 1.5,
		Reason: fmt.Sprintf("%d/%d tool calls valid", validCalls, totalCalls),
	}
}

// SignalFromReasoningCompleteness scores how many fields of a ReasoningStep
// are populated. Each of the 5 fields (Thought, Action, Observation, Critique,
// Confidence) contributes 0.2 to the score.
func SignalFromReasoningCompleteness(step ReasoningStep) TrustSignal {
	filled := 0
	if step.Thought != "" {
		filled++
	}
	if step.Action != "" {
		filled++
	}
	if step.Observation != "" {
		filled++
	}
	if step.Critique != "" {
		filled++
	}
	if step.Confidence > 0.0 {
		filled++
	}

	score := float64(filled) / 5.0
	return TrustSignal{
		Name:   "reasoning_completeness",
		Score:  score,
		Weight: 1.0,
		Reason: fmt.Sprintf("%d/5 reasoning fields populated", filled),
	}
}

// SignalFromResponseLength scores a response based on its length relative
// to expected bounds. Empty responses score 0. Responses within [minLen, maxLen]
// score 1.0. Out-of-range responses are penalized proportionally.
func SignalFromResponseLength(response string, minLen, maxLen int) TrustSignal {
	length := len(response)

	if length == 0 {
		return TrustSignal{
			Name:   "response_length",
			Score:  0.0,
			Weight: 0.5,
			Reason: "empty response",
		}
	}

	var score float64
	switch {
	case length >= minLen && length <= maxLen:
		score = 1.0
	case length < minLen:
		score = float64(length) / float64(minLen)
	default: // length > maxLen
		// Penalize proportionally: at 2x maxLen, score = 0.5
		ratio := float64(maxLen) / float64(length)
		score = clamp01(ratio)
	}

	return TrustSignal{
		Name:   "response_length",
		Score:  score,
		Weight: 0.5,
		Reason: fmt.Sprintf("length %d (expected %d-%d)", length, minLen, maxLen),
	}
}

// SignalFromConsistency compares the current response against a previous
// response using keyword overlap. If previousResponse is empty, returns 1.0
// (nothing to contradict). Otherwise computes Jaccard similarity on words.
func SignalFromConsistency(currentResponse, previousResponse string) TrustSignal {
	if previousResponse == "" {
		return TrustSignal{
			Name:   "consistency",
			Score:  1.0,
			Weight: 1.0,
			Reason: "no previous response to compare",
		}
	}

	currentWords := wordSet(strings.ToLower(currentResponse))
	previousWords := wordSet(strings.ToLower(previousResponse))

	if len(currentWords) == 0 || len(previousWords) == 0 {
		return TrustSignal{
			Name:   "consistency",
			Score:  0.0,
			Weight: 1.0,
			Reason: "empty word set",
		}
	}

	intersection := 0
	for w := range currentWords {
		if previousWords[w] {
			intersection++
		}
	}

	union := len(currentWords)
	for w := range previousWords {
		if !currentWords[w] {
			union++
		}
	}

	score := float64(intersection) / float64(union)

	return TrustSignal{
		Name:   "consistency",
		Score:  score,
		Weight: 1.0,
		Reason: fmt.Sprintf("keyword overlap %.0f%% (%d/%d)", score*100, intersection, union),
	}
}

// wordSet splits text into a set of unique words, filtering short words.
func wordSet(text string) map[string]bool {
	words := strings.Fields(text)
	set := make(map[string]bool, len(words))
	for _, w := range words {
		// Filter very short words and punctuation.
		cleaned := strings.Trim(w, ".,;:!?()[]{}\"'`")
		if len(cleaned) >= 2 {
			set[cleaned] = true
		}
	}
	return set
}

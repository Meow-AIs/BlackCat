package agent

import (
	"testing"
)

func TestNewTrustScorer(t *testing.T) {
	ts := NewTrustScorer()
	if ts == nil {
		t.Fatal("NewTrustScorer returned nil")
	}
	if ts.retryThreshold != 0.5 {
		t.Errorf("expected retryThreshold 0.5, got %f", ts.retryThreshold)
	}
	if ts.warnThreshold != 0.7 {
		t.Errorf("expected warnThreshold 0.7, got %f", ts.warnThreshold)
	}
}

func TestSetThresholds(t *testing.T) {
	ts := NewTrustScorer()
	ts.SetThresholds(0.3, 0.6)
	if ts.retryThreshold != 0.3 {
		t.Errorf("expected retryThreshold 0.3, got %f", ts.retryThreshold)
	}
	if ts.warnThreshold != 0.6 {
		t.Errorf("expected warnThreshold 0.6, got %f", ts.warnThreshold)
	}
}

func TestTrustScoreVerdictTrusted(t *testing.T) {
	ts := NewTrustScorer()
	signals := []TrustSignal{
		{Name: "confidence", Score: 0.9, Weight: 1.0, Reason: "high confidence"},
		{Name: "tool_validity", Score: 0.8, Weight: 1.0, Reason: "valid tools"},
	}
	result := ts.Score(signals)

	// weighted average = (0.9*1.0 + 0.8*1.0) / (1.0 + 1.0) = 0.85
	if result.Overall < 0.84 || result.Overall > 0.86 {
		t.Errorf("expected overall ~0.85, got %f", result.Overall)
	}
	if result.Verdict != VerdictTrusted {
		t.Errorf("expected trusted verdict, got %s", result.Verdict)
	}
	if result.ShouldRetry {
		t.Error("should not retry trusted score")
	}
}

func TestTrustScoreVerdictUncertain(t *testing.T) {
	ts := NewTrustScorer()
	signals := []TrustSignal{
		{Name: "confidence", Score: 0.6, Weight: 1.0, Reason: "medium"},
	}
	result := ts.Score(signals)

	if result.Verdict != VerdictUncertain {
		t.Errorf("expected uncertain verdict, got %s", result.Verdict)
	}
	if result.ShouldRetry {
		t.Error("uncertain should not trigger retry (above 0.5)")
	}
}

func TestTrustScoreVerdictUntrusted(t *testing.T) {
	ts := NewTrustScorer()
	signals := []TrustSignal{
		{Name: "confidence", Score: 0.3, Weight: 1.0, Reason: "low"},
	}
	result := ts.Score(signals)

	if result.Verdict != VerdictUntrusted {
		t.Errorf("expected untrusted verdict, got %s", result.Verdict)
	}
	if !result.ShouldRetry {
		t.Error("untrusted should trigger retry")
	}
}

func TestTrustScoreEmptySignals(t *testing.T) {
	ts := NewTrustScorer()
	result := ts.Score(nil)

	if result.Overall != 0.0 {
		t.Errorf("expected 0.0 for empty signals, got %f", result.Overall)
	}
	if result.Verdict != VerdictUntrusted {
		t.Errorf("expected untrusted for empty signals, got %s", result.Verdict)
	}
	if !result.ShouldRetry {
		t.Error("empty signals should trigger retry")
	}
}

func TestTrustScoreWeightedAverage(t *testing.T) {
	ts := NewTrustScorer()
	signals := []TrustSignal{
		{Name: "a", Score: 1.0, Weight: 3.0, Reason: "high weight"},
		{Name: "b", Score: 0.0, Weight: 1.0, Reason: "low weight"},
	}
	result := ts.Score(signals)

	// weighted average = (1.0*3.0 + 0.0*1.0) / (3.0+1.0) = 0.75
	if result.Overall < 0.74 || result.Overall > 0.76 {
		t.Errorf("expected overall ~0.75, got %f", result.Overall)
	}
}

func TestTrustScoreZeroWeights(t *testing.T) {
	ts := NewTrustScorer()
	signals := []TrustSignal{
		{Name: "a", Score: 0.9, Weight: 0.0, Reason: "zero weight"},
	}
	result := ts.Score(signals)

	if result.Overall != 0.0 {
		t.Errorf("expected 0.0 for zero total weight, got %f", result.Overall)
	}
}

func TestSignalFromConfidence(t *testing.T) {
	tests := []struct {
		name       string
		confidence float64
		wantScore  float64
	}{
		{"high", 0.9, 0.9},
		{"zero", 0.0, 0.0},
		{"one", 1.0, 1.0},
		{"negative_clamped", -0.5, 0.0},
		{"over_one_clamped", 1.5, 1.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sig := SignalFromConfidence(tt.confidence)
			if sig.Score != tt.wantScore {
				t.Errorf("expected score %f, got %f", tt.wantScore, sig.Score)
			}
			if sig.Name != "confidence" {
				t.Errorf("expected name 'confidence', got %s", sig.Name)
			}
		})
	}
}

func TestSignalFromToolValidity(t *testing.T) {
	tests := []struct {
		name      string
		valid     int
		total     int
		wantScore float64
	}{
		{"all_valid", 5, 5, 1.0},
		{"half_valid", 3, 6, 0.5},
		{"none_valid", 0, 5, 0.0},
		{"zero_total", 0, 0, 1.0}, // no tools = no problem
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sig := SignalFromToolValidity(tt.valid, tt.total)
			if sig.Score != tt.wantScore {
				t.Errorf("expected score %f, got %f", tt.wantScore, sig.Score)
			}
		})
	}
}

func TestSignalFromReasoningCompleteness(t *testing.T) {
	complete := ReasoningStep{
		Thought:     "thinking",
		Action:      "do something",
		Observation: "result",
		Critique:    "good",
		Confidence:  0.8,
	}
	sig := SignalFromReasoningCompleteness(complete)
	if sig.Score != 1.0 {
		t.Errorf("expected 1.0 for complete step, got %f", sig.Score)
	}

	partial := ReasoningStep{
		Thought: "thinking",
		Action:  "do something",
	}
	sig2 := SignalFromReasoningCompleteness(partial)
	if sig2.Score >= 1.0 || sig2.Score <= 0.0 {
		t.Errorf("expected partial score, got %f", sig2.Score)
	}

	empty := ReasoningStep{}
	sig3 := SignalFromReasoningCompleteness(empty)
	if sig3.Score != 0.0 {
		t.Errorf("expected 0.0 for empty step, got %f", sig3.Score)
	}
}

func TestSignalFromResponseLength(t *testing.T) {
	sig := SignalFromResponseLength("hello world", 5, 100)
	if sig.Score != 1.0 {
		t.Errorf("expected 1.0 for normal length, got %f", sig.Score)
	}

	sigShort := SignalFromResponseLength("hi", 5, 100)
	if sigShort.Score >= 1.0 {
		t.Errorf("expected penalty for too short, got %f", sigShort.Score)
	}

	sigLong := SignalFromResponseLength(string(make([]byte, 200)), 5, 100)
	if sigLong.Score >= 1.0 {
		t.Errorf("expected penalty for too long, got %f", sigLong.Score)
	}

	sigEmpty := SignalFromResponseLength("", 5, 100)
	if sigEmpty.Score != 0.0 {
		t.Errorf("expected 0.0 for empty response, got %f", sigEmpty.Score)
	}
}

func TestSignalFromConsistency(t *testing.T) {
	sig := SignalFromConsistency(
		"The error is in main.go at line 42",
		"The error is in main.go at line 42",
	)
	if sig.Score < 0.8 {
		t.Errorf("expected high consistency for identical responses, got %f", sig.Score)
	}

	sigDiff := SignalFromConsistency(
		"The file is main.go and it works fine",
		"The server crashes on startup due to nil pointer",
	)
	if sigDiff.Score > 0.5 {
		t.Errorf("expected low consistency for different responses, got %f", sigDiff.Score)
	}

	sigEmpty := SignalFromConsistency("hello", "")
	if sigEmpty.Score != 1.0 {
		t.Errorf("expected 1.0 when previous is empty, got %f", sigEmpty.Score)
	}
}

func TestTrustScoreSignalsPreserved(t *testing.T) {
	ts := NewTrustScorer()
	signals := []TrustSignal{
		{Name: "a", Score: 0.8, Weight: 1.0, Reason: "test"},
		{Name: "b", Score: 0.6, Weight: 1.0, Reason: "test2"},
	}
	result := ts.Score(signals)
	if len(result.Signals) != 2 {
		t.Errorf("expected 2 signals preserved, got %d", len(result.Signals))
	}
}

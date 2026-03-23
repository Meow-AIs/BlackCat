package architect

import (
	"strings"
	"testing"
)

func TestNewWAFReview_HasQuestions(t *testing.T) {
	r := NewWAFReview()
	if len(r.Questions) < 30 {
		t.Errorf("expected at least 30 questions, got %d", len(r.Questions))
	}
}

func TestNewWAFReview_FivePerPillar(t *testing.T) {
	r := NewWAFReview()
	counts := make(map[WAFPillar]int)
	for _, q := range r.Questions {
		counts[q.Pillar]++
	}
	pillars := []WAFPillar{
		PillarSecurity, PillarReliability, PillarPerformance,
		PillarCostOptimization, PillarOperationalExcellence, PillarSustainability,
	}
	for _, p := range pillars {
		if counts[p] < 5 {
			t.Errorf("pillar %q has %d questions, need at least 5", p, counts[p])
		}
	}
}

func TestNewWAFReview_UniqueIDs(t *testing.T) {
	r := NewWAFReview()
	seen := make(map[string]bool)
	for _, q := range r.Questions {
		if seen[q.ID] {
			t.Errorf("duplicate question ID: %s", q.ID)
		}
		seen[q.ID] = true
	}
}

func TestWAFReview_Answer_Valid(t *testing.T) {
	r := NewWAFReview()
	qid := r.Questions[0].ID
	err := r.Answer(qid, 4, "Implemented via WAF")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ans, ok := r.Answers[qid]
	if !ok {
		t.Fatal("answer not stored")
	}
	if ans.Score != 4 {
		t.Errorf("expected score 4, got %d", ans.Score)
	}
	if ans.Notes != "Implemented via WAF" {
		t.Errorf("unexpected notes: %q", ans.Notes)
	}
}

func TestWAFReview_Answer_InvalidQuestion(t *testing.T) {
	r := NewWAFReview()
	err := r.Answer("nonexistent-id", 3, "")
	if err == nil {
		t.Error("expected error for invalid question ID")
	}
}

func TestWAFReview_Answer_InvalidScore(t *testing.T) {
	r := NewWAFReview()
	qid := r.Questions[0].ID

	if err := r.Answer(qid, -1, ""); err == nil {
		t.Error("expected error for negative score")
	}
	if err := r.Answer(qid, 6, ""); err == nil {
		t.Error("expected error for score > 5")
	}
}

func TestWAFReview_PillarScore(t *testing.T) {
	r := NewWAFReview()
	// Answer all security questions with score 4
	for _, q := range r.Questions {
		if q.Pillar == PillarSecurity {
			_ = r.Answer(q.ID, 4, "")
		}
	}
	score := r.PillarScore(PillarSecurity)
	if score != 4.0 {
		t.Errorf("expected pillar score 4.0, got %.2f", score)
	}
}

func TestWAFReview_PillarScore_Unanswered(t *testing.T) {
	r := NewWAFReview()
	score := r.PillarScore(PillarSecurity)
	if score != 0.0 {
		t.Errorf("expected 0.0 for unanswered pillar, got %.2f", score)
	}
}

func TestWAFReview_OverallScore(t *testing.T) {
	r := NewWAFReview()
	// Answer all questions with score 3
	for _, q := range r.Questions {
		_ = r.Answer(q.ID, 3, "")
	}
	overall := r.OverallScore()
	if overall != 3.0 {
		t.Errorf("expected overall score 3.0, got %.2f", overall)
	}
}

func TestWAFReview_OverallScore_Empty(t *testing.T) {
	r := NewWAFReview()
	overall := r.OverallScore()
	if overall != 0.0 {
		t.Errorf("expected 0.0 for no answers, got %.2f", overall)
	}
}

func TestWAFReview_Recommendations(t *testing.T) {
	r := NewWAFReview()
	// Answer some with low scores
	for i, q := range r.Questions {
		if i < 5 {
			_ = r.Answer(q.ID, 1, "Needs work")
		} else {
			_ = r.Answer(q.ID, 5, "Excellent")
		}
	}
	recs := r.Recommendations()
	if len(recs) < 5 {
		t.Errorf("expected at least 5 recommendations, got %d", len(recs))
	}
}

func TestWAFReview_Recommendations_AllHigh(t *testing.T) {
	r := NewWAFReview()
	for _, q := range r.Questions {
		_ = r.Answer(q.ID, 5, "")
	}
	recs := r.Recommendations()
	if len(recs) != 0 {
		t.Errorf("expected 0 recommendations when all scores are 5, got %d", len(recs))
	}
}

func TestWAFReview_FormatMarkdown(t *testing.T) {
	r := NewWAFReview()
	for _, q := range r.Questions {
		_ = r.Answer(q.ID, 3, "Moderate")
	}
	md := r.FormatMarkdown()

	required := []string{
		"Security", "Reliability", "Performance",
		"Cost", "Operational", "Sustainability",
		"Overall",
	}
	for _, s := range required {
		if !strings.Contains(md, s) {
			t.Errorf("markdown missing %q", s)
		}
	}
}

func TestWAFReview_QuestionsHaveBestPractice(t *testing.T) {
	r := NewWAFReview()
	for _, q := range r.Questions {
		if q.BestPractice == "" {
			t.Errorf("question %s missing best practice", q.ID)
		}
		if q.Question == "" {
			t.Errorf("question %s missing question text", q.ID)
		}
	}
}

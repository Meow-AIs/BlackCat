package architect

import (
	"fmt"
	"strings"
)

// WAFPillar represents a Well-Architected Framework pillar.
type WAFPillar string

const (
	PillarSecurity              WAFPillar = "security"
	PillarReliability           WAFPillar = "reliability"
	PillarPerformance           WAFPillar = "performance"
	PillarCostOptimization      WAFPillar = "cost_optimization"
	PillarOperationalExcellence WAFPillar = "operational_excellence"
	PillarSustainability        WAFPillar = "sustainability"
)

// WAFQuestion is a review question within a pillar.
type WAFQuestion struct {
	ID           string
	Pillar       WAFPillar
	Question     string
	BestPractice string
}

// WAFAnswer records a score and notes for a question.
type WAFAnswer struct {
	QuestionID string
	Score      int
	Notes      string
}

// WAFReview holds the full set of questions and collected answers.
type WAFReview struct {
	Questions []WAFQuestion
	Answers   map[string]WAFAnswer
}

// NewWAFReview creates a review loaded with builtin questions (at least 5 per pillar).
func NewWAFReview() *WAFReview {
	return &WAFReview{
		Questions: builtinQuestions(),
		Answers:   make(map[string]WAFAnswer),
	}
}

// Answer records a score (0-5) for the given question.
func (r *WAFReview) Answer(questionID string, score int, notes string) error {
	if score < 0 || score > 5 {
		return fmt.Errorf("score must be 0-5, got %d", score)
	}
	found := false
	for _, q := range r.Questions {
		if q.ID == questionID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("question %q not found", questionID)
	}
	r.Answers[questionID] = WAFAnswer{
		QuestionID: questionID,
		Score:      score,
		Notes:      notes,
	}
	return nil
}

// PillarScore returns the average score for all answered questions in a pillar.
func (r *WAFReview) PillarScore(pillar WAFPillar) float64 {
	var total, count float64
	for _, q := range r.Questions {
		if q.Pillar != pillar {
			continue
		}
		if ans, ok := r.Answers[q.ID]; ok {
			total += float64(ans.Score)
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return total / count
}

// OverallScore returns the average of all pillar scores (only answered pillars).
func (r *WAFReview) OverallScore() float64 {
	pillars := []WAFPillar{
		PillarSecurity, PillarReliability, PillarPerformance,
		PillarCostOptimization, PillarOperationalExcellence, PillarSustainability,
	}
	var total, count float64
	for _, p := range pillars {
		s := r.PillarScore(p)
		if s > 0 || r.pillarHasAnswers(p) {
			total += s
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return total / count
}

func (r *WAFReview) pillarHasAnswers(pillar WAFPillar) bool {
	for _, q := range r.Questions {
		if q.Pillar == pillar {
			if _, ok := r.Answers[q.ID]; ok {
				return true
			}
		}
	}
	return false
}

// Recommendations returns advice for questions scored below 3.
func (r *WAFReview) Recommendations() []string {
	var recs []string
	for _, q := range r.Questions {
		ans, ok := r.Answers[q.ID]
		if !ok {
			continue
		}
		if ans.Score < 3 {
			recs = append(recs, fmt.Sprintf("[%s] %s - Best practice: %s",
				q.Pillar, q.Question, q.BestPractice))
		}
	}
	return recs
}

// FormatMarkdown renders the review as a markdown report.
func (r *WAFReview) FormatMarkdown() string {
	var b strings.Builder

	b.WriteString("# Well-Architected Framework Review\n\n")
	b.WriteString(fmt.Sprintf("**Overall Score: %.1f / 5.0**\n\n", r.OverallScore()))

	pillars := []struct {
		Pillar WAFPillar
		Label  string
	}{
		{PillarSecurity, "Security"},
		{PillarReliability, "Reliability"},
		{PillarPerformance, "Performance"},
		{PillarCostOptimization, "Cost Optimization"},
		{PillarOperationalExcellence, "Operational Excellence"},
		{PillarSustainability, "Sustainability"},
	}

	for _, p := range pillars {
		score := r.PillarScore(p.Pillar)
		b.WriteString(fmt.Sprintf("## %s (%.1f / 5.0)\n\n", p.Label, score))
		b.WriteString("| Question | Score | Notes |\n")
		b.WriteString("|----------|-------|-------|\n")
		for _, q := range r.Questions {
			if q.Pillar != p.Pillar {
				continue
			}
			ans, ok := r.Answers[q.ID]
			scoreStr := "-"
			notesStr := ""
			if ok {
				scoreStr = fmt.Sprintf("%d/5", ans.Score)
				notesStr = ans.Notes
			}
			b.WriteString(fmt.Sprintf("| %s | %s | %s |\n", q.Question, scoreStr, notesStr))
		}
		b.WriteString("\n")
	}

	recs := r.Recommendations()
	if len(recs) > 0 {
		b.WriteString("## Recommendations\n\n")
		for _, rec := range recs {
			b.WriteString(fmt.Sprintf("- %s\n", rec))
		}
	}

	return b.String()
}

func builtinQuestions() []WAFQuestion {
	return []WAFQuestion{
		// Security (5)
		{ID: "sec-01", Pillar: PillarSecurity, Question: "Is identity and access management implemented?", BestPractice: "Use IAM with least-privilege access and MFA for all users"},
		{ID: "sec-02", Pillar: PillarSecurity, Question: "Is data encrypted at rest and in transit?", BestPractice: "Enable TLS for transit and AES-256 for data at rest"},
		{ID: "sec-03", Pillar: PillarSecurity, Question: "Are network controls in place?", BestPractice: "Use VPCs, security groups, and network segmentation"},
		{ID: "sec-04", Pillar: PillarSecurity, Question: "Is there a vulnerability management process?", BestPractice: "Regular scanning, patching cadence, and dependency updates"},
		{ID: "sec-05", Pillar: PillarSecurity, Question: "Are secrets managed securely?", BestPractice: "Use a secrets manager, rotate credentials, never hardcode secrets"},

		// Reliability (5)
		{ID: "rel-01", Pillar: PillarReliability, Question: "Is the system designed for high availability?", BestPractice: "Multi-AZ deployment with automatic failover"},
		{ID: "rel-02", Pillar: PillarReliability, Question: "Are backups automated and tested?", BestPractice: "Automated backups with regular restore testing"},
		{ID: "rel-03", Pillar: PillarReliability, Question: "Is there disaster recovery planning?", BestPractice: "Defined RTO/RPO with documented and tested DR procedures"},
		{ID: "rel-04", Pillar: PillarReliability, Question: "Are health checks and auto-healing configured?", BestPractice: "Implement health endpoints with automatic instance replacement"},
		{ID: "rel-05", Pillar: PillarReliability, Question: "Is there graceful degradation under load?", BestPractice: "Circuit breakers, bulkheads, and load shedding patterns"},

		// Performance (5)
		{ID: "perf-01", Pillar: PillarPerformance, Question: "Are compute resources right-sized?", BestPractice: "Regular benchmarking and right-sizing based on actual usage"},
		{ID: "perf-02", Pillar: PillarPerformance, Question: "Is caching implemented at appropriate layers?", BestPractice: "Multi-layer caching: CDN, application, and database query cache"},
		{ID: "perf-03", Pillar: PillarPerformance, Question: "Are database queries optimized?", BestPractice: "Query analysis, proper indexing, and connection pooling"},
		{ID: "perf-04", Pillar: PillarPerformance, Question: "Is auto-scaling configured?", BestPractice: "Horizontal auto-scaling based on CPU, memory, and custom metrics"},
		{ID: "perf-05", Pillar: PillarPerformance, Question: "Is latency monitored with SLOs?", BestPractice: "P50/P95/P99 latency tracking with defined SLOs and alerts"},

		// Cost Optimization (5)
		{ID: "cost-01", Pillar: PillarCostOptimization, Question: "Is there cost visibility and allocation?", BestPractice: "Tagging strategy with cost allocation per team/service"},
		{ID: "cost-02", Pillar: PillarCostOptimization, Question: "Are reserved/spot instances used where appropriate?", BestPractice: "Reservations for steady state, spot for fault-tolerant workloads"},
		{ID: "cost-03", Pillar: PillarCostOptimization, Question: "Is unused infrastructure cleaned up?", BestPractice: "Regular audits for orphaned resources and idle instances"},
		{ID: "cost-04", Pillar: PillarCostOptimization, Question: "Is data transfer cost optimized?", BestPractice: "CDN for static content, minimize cross-region data transfer"},
		{ID: "cost-05", Pillar: PillarCostOptimization, Question: "Are cost anomaly alerts configured?", BestPractice: "Budget alerts and anomaly detection for unexpected spend"},

		// Operational Excellence (5)
		{ID: "ops-01", Pillar: PillarOperationalExcellence, Question: "Is infrastructure defined as code?", BestPractice: "All infrastructure managed via IaC with version control"},
		{ID: "ops-02", Pillar: PillarOperationalExcellence, Question: "Is there a CI/CD pipeline?", BestPractice: "Automated build, test, and deploy with rollback capability"},
		{ID: "ops-03", Pillar: PillarOperationalExcellence, Question: "Is centralized logging and monitoring in place?", BestPractice: "Structured logging with centralized aggregation and dashboards"},
		{ID: "ops-04", Pillar: PillarOperationalExcellence, Question: "Are runbooks documented for common operations?", BestPractice: "Documented procedures for deployment, scaling, and incident response"},
		{ID: "ops-05", Pillar: PillarOperationalExcellence, Question: "Is there an incident response process?", BestPractice: "Defined on-call rotation, escalation paths, and post-mortem process"},

		// Sustainability (5)
		{ID: "sus-01", Pillar: PillarSustainability, Question: "Is resource utilization maximized?", BestPractice: "Right-sizing and consolidation to maximize utilization rates"},
		{ID: "sus-02", Pillar: PillarSustainability, Question: "Are efficient architectures used?", BestPractice: "Serverless and event-driven patterns to reduce idle resources"},
		{ID: "sus-03", Pillar: PillarSustainability, Question: "Is data lifecycle managed?", BestPractice: "Automated data tiering, archival, and deletion policies"},
		{ID: "sus-04", Pillar: PillarSustainability, Question: "Are sustainable regions/providers considered?", BestPractice: "Prefer cloud regions powered by renewable energy"},
		{ID: "sus-05", Pillar: PillarSustainability, Question: "Is software efficiency measured?", BestPractice: "Track carbon footprint proxy metrics like CPU-hours per transaction"},
	}
}

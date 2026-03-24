package architect

import (
	"fmt"
	"strings"
)

// ReliabilityPattern describes a reliability engineering pattern.
type ReliabilityPattern struct {
	Name           string
	Description    string
	Category       string // "resilience", "observability", "deployment", "data", "chaos"
	Implementation string
	When           string // when to use
	Tradeoffs      []string
}

// LoadReliabilityPatterns returns 15+ built-in reliability patterns.
func LoadReliabilityPatterns() []ReliabilityPattern {
	return []ReliabilityPattern{
		// Resilience
		{Name: "Circuit Breaker", Description: "Stop calling a failing service to let it recover", Category: "resilience",
			Implementation: "Track failures; open after threshold; half-open to probe recovery; close on success",
			When:           "Calling external services or downstream dependencies",
			Tradeoffs:      []string{"Added complexity", "Must tune thresholds", "Need fallback strategy"}},
		{Name: "Bulkhead", Description: "Isolate components so one failure doesn't cascade", Category: "resilience",
			Implementation: "Separate thread pools, connection pools, or process groups per dependency",
			When:           "Multiple services sharing resources",
			Tradeoffs:      []string{"Resource underutilization", "More complex resource management"}},
		{Name: "Retry with Backoff", Description: "Retry failed operations with exponential delay", Category: "resilience",
			Implementation: "Exponential backoff with jitter; cap max retries and total duration",
			When:           "Transient failures in network calls or queue processing",
			Tradeoffs:      []string{"Increased latency on failure paths", "Can amplify load without jitter"}},
		{Name: "Timeout", Description: "Set time limits on all external calls", Category: "resilience",
			Implementation: "Configure connect and read timeouts; use context cancellation",
			When:           "Any call to external service, database, or API",
			Tradeoffs:      []string{"Too short causes false failures", "Too long wastes resources"}},
		{Name: "Fallback", Description: "Provide degraded functionality when primary fails", Category: "resilience",
			Implementation: "Return cached data, default values, or simplified response on failure",
			When:           "Non-critical features that can degrade gracefully",
			Tradeoffs:      []string{"Stale data risk", "User experience degradation", "Complexity in fallback logic"}},
		{Name: "Rate Limiting", Description: "Limit request rate to protect services from overload", Category: "resilience",
			Implementation: "Token bucket or sliding window at API gateway and service level",
			When:           "Public APIs, shared resources, multi-tenant systems",
			Tradeoffs:      []string{"Legitimate traffic may be rejected", "Complexity in setting limits"}},
		{Name: "Queue-based Load Leveling", Description: "Buffer requests through a queue to smooth load spikes", Category: "resilience",
			Implementation: "Place a message queue between producers and consumers; scale consumers independently",
			When:           "Variable workload with predictable processing time",
			Tradeoffs:      []string{"Added latency", "Queue management overhead", "At-least-once delivery semantics"}},

		// Observability
		{Name: "Distributed Tracing", Description: "Trace requests across service boundaries", Category: "observability",
			Implementation: "Instrument with OpenTelemetry; propagate trace context; export to Jaeger/Tempo",
			When:           "Microservices architecture with complex call chains",
			Tradeoffs:      []string{"Performance overhead", "Storage costs", "Instrumentation effort"}},
		{Name: "Health Check API", Description: "Expose health endpoints for liveness and readiness probes", Category: "observability",
			Implementation: "Implement /healthz (liveness) and /readyz (readiness) endpoints checking dependencies",
			When:           "Any service running in Kubernetes or behind a load balancer",
			Tradeoffs:      []string{"Must check meaningful dependencies", "Flapping if checks are too sensitive"}},
		{Name: "Log Aggregation", Description: "Centralize logs from all services for search and analysis", Category: "observability",
			Implementation: "Structured JSON logging; ship via Fluentd/Vector to Elasticsearch/Loki",
			When:           "Multiple services or instances producing logs",
			Tradeoffs:      []string{"Storage costs", "Network bandwidth", "Sensitive data in logs"}},
		{Name: "Metrics Dashboard", Description: "Visualize key metrics with golden signals", Category: "observability",
			Implementation: "Expose Prometheus metrics; build Grafana dashboards for latency, traffic, errors, saturation",
			When:           "Any production service",
			Tradeoffs:      []string{"Dashboard maintenance", "Alert fatigue if poorly configured"}},

		// Deployment
		{Name: "Blue-Green Deployment", Description: "Run two identical environments; switch traffic atomically", Category: "deployment",
			Implementation: "Deploy new version to green; validate; switch load balancer; keep blue for rollback",
			When:           "Need zero-downtime deployments with instant rollback",
			Tradeoffs:      []string{"Double infrastructure cost during deployment", "Database migration complexity"}},
		{Name: "Canary Deployment", Description: "Gradually route traffic to new version", Category: "deployment",
			Implementation: "Route 1-5% traffic to canary; monitor error rates and latency; expand or rollback",
			When:           "High-traffic services where blast radius must be minimized",
			Tradeoffs:      []string{"Slower rollout", "Need good monitoring", "Session affinity concerns"}},
		{Name: "Feature Flags", Description: "Toggle features without deploying new code", Category: "deployment",
			Implementation: "Use feature flag service; wrap new code in flag checks; gradual rollout by percentage or segment",
			When:           "Separating deployment from release; A/B testing; kill switches",
			Tradeoffs:      []string{"Flag debt if not cleaned up", "Testing complexity", "Runtime configuration management"}},

		// Data
		{Name: "Write-Ahead Log", Description: "Write changes to a log before applying to storage", Category: "data",
			Implementation: "Append operations to WAL before committing; replay on crash recovery",
			When:           "Databases, message brokers, any system needing crash recovery",
			Tradeoffs:      []string{"Write amplification", "Recovery time on large logs"}},
		{Name: "Event Sourcing", Description: "Store state as sequence of events", Category: "data",
			Implementation: "Append events to event store; rebuild state by replaying; use snapshots for performance",
			When:           "Audit trails, financial systems, collaborative editing",
			Tradeoffs:      []string{"Complex querying", "Storage growth", "Eventual consistency"}},
		{Name: "Saga Pattern", Description: "Coordinate distributed transactions via compensating actions", Category: "data",
			Implementation: "Orchestrator or choreography; each step has a compensating action for rollback",
			When:           "Multi-service transactions that can't use 2PC",
			Tradeoffs:      []string{"Eventual consistency", "Complex error handling", "No isolation guarantees"}},

		// Chaos
		{Name: "Chaos Monkey", Description: "Randomly terminate instances to test resilience", Category: "chaos",
			Implementation: "Schedule random instance termination in non-peak hours; verify auto-recovery",
			When:           "Validating auto-scaling and self-healing capabilities",
			Tradeoffs:      []string{"Risk of customer impact", "Need strong monitoring first"}},
		{Name: "Fault Injection", Description: "Inject failures to test system behavior under stress", Category: "chaos",
			Implementation: "Use tools like Litmus/Gremlin to inject latency, errors, resource exhaustion",
			When:           "Testing resilience patterns (circuit breakers, retries, fallbacks)",
			Tradeoffs:      []string{"Must scope blast radius", "Requires mature observability"}},
		{Name: "Game Days", Description: "Scheduled exercises simulating production incidents", Category: "chaos",
			Implementation: "Plan scenarios; assign roles; execute in staging or production; debrief and document learnings",
			When:           "Training incident response teams; validating runbooks",
			Tradeoffs:      []string{"Time investment", "Risk if in production", "Need executive buy-in"}},
	}
}

// SearchReliabilityPatterns finds patterns matching a query.
func SearchReliabilityPatterns(query string) []ReliabilityPattern {
	q := toLowerCase(query)
	var results []ReliabilityPattern
	for _, p := range LoadReliabilityPatterns() {
		if containsLower(p.Name, q) || containsLower(p.Description, q) || containsLower(p.Category, q) {
			results = append(results, p)
		}
	}
	return results
}

// GetReliabilityPattern returns a pattern by exact name.
func GetReliabilityPattern(name string) (ReliabilityPattern, bool) {
	for _, p := range LoadReliabilityPatterns() {
		if p.Name == name {
			return p, true
		}
	}
	return ReliabilityPattern{}, false
}

// SLI represents a Service Level Indicator.
type SLI struct {
	Name          string
	Description   string
	Metric        string // "latency_p99", "error_rate", "availability", "throughput"
	Unit          string
	GoodThreshold float64
}

// SLO represents a Service Level Objective.
type SLO struct {
	SLI         SLI
	Target      float64 // e.g., 99.9
	Window      string  // "30d", "7d"
	BurnRate    float64 // current burn rate
	ErrorBudget float64 // remaining %
}

// DefaultSLIs returns standard SLIs for web services.
func DefaultSLIs() []SLI {
	return []SLI{
		{Name: "Request Latency", Description: "99th percentile request latency", Metric: "latency_p99", Unit: "ms", GoodThreshold: 200},
		{Name: "Error Rate", Description: "Percentage of failed requests", Metric: "error_rate", Unit: "%", GoodThreshold: 0.1},
		{Name: "Availability", Description: "Percentage of successful health checks", Metric: "availability", Unit: "%", GoodThreshold: 99.9},
		{Name: "Throughput", Description: "Requests processed per second", Metric: "throughput", Unit: "rps", GoodThreshold: 1000},
	}
}

// CalculateErrorBudget returns remaining error budget in minutes.
// target is the SLO percentage (e.g., 99.9), windowDays is the SLO window,
// downtimeMinutes is the actual downtime consumed so far.
func CalculateErrorBudget(target float64, windowDays int, downtimeMinutes float64) float64 {
	if windowDays <= 0 {
		return 0
	}
	totalMinutes := float64(windowDays) * 24 * 60
	allowedDowntime := totalMinutes * (1 - target/100)
	return allowedDowntime - downtimeMinutes
}

// CalculateBurnRate returns the rate at which error budget is being consumed.
// A burn rate of 1.0 means consuming budget at exactly the expected rate.
func CalculateBurnRate(target float64, windowDays int, currentDowntime float64) float64 {
	if windowDays <= 0 {
		return 0
	}
	totalMinutes := float64(windowDays) * 24 * 60
	allowedDowntime := totalMinutes * (1 - target/100)
	if allowedDowntime == 0 {
		return 0
	}
	return currentDowntime / allowedDowntime
}

// Runbook holds incident response steps for an alert.
type Runbook struct {
	Title      string
	Service    string
	AlertName  string
	Severity   string
	Steps      []RunbookStep
	Escalation []string
	References []string
}

// RunbookStep describes one step in a runbook.
type RunbookStep struct {
	Order    int
	Action   string
	Command  string // optional command to run
	Expected string // expected outcome
	IfFails  string // what to do if this step fails
}

// GenerateRunbook creates a standard runbook for a service and alert.
func GenerateRunbook(service, alertName, severity string) *Runbook {
	rb := &Runbook{
		Title:     fmt.Sprintf("Runbook: %s - %s", service, alertName),
		Service:   service,
		AlertName: alertName,
		Severity:  severity,
		Steps: []RunbookStep{
			{Order: 1, Action: "Verify alert is genuine by checking metrics dashboard",
				Command: fmt.Sprintf("kubectl get pods -l app=%s", service), Expected: "Pods running and ready", IfFails: "Check if cluster is healthy"},
			{Order: 2, Action: "Check service logs for errors",
				Command: fmt.Sprintf("kubectl logs -l app=%s --tail=100", service), Expected: "Identify error pattern", IfFails: "Check if logging pipeline is working"},
			{Order: 3, Action: "Check dependent services health",
				Expected: "All dependencies healthy", IfFails: "Escalate to dependency team"},
			{Order: 4, Action: "Attempt restart if service is unhealthy",
				Command: fmt.Sprintf("kubectl rollout restart deployment/%s", service), Expected: "Pods restart successfully", IfFails: "Check for resource constraints or image issues"},
			{Order: 5, Action: "Verify recovery by checking metrics",
				Expected: "Error rate returns to normal", IfFails: "Escalate to next level"},
		},
		Escalation: []string{
			fmt.Sprintf("L1: On-call engineer for %s", service),
			fmt.Sprintf("L2: %s team lead", service),
			"L3: Platform engineering manager",
			"L4: VP Engineering (SEV1 only)",
		},
		References: []string{
			fmt.Sprintf("Service dashboard: https://grafana.internal/d/%s", service),
			fmt.Sprintf("Architecture docs: https://docs.internal/services/%s", service),
			"Incident management: https://incident.internal",
		},
	}
	return rb
}

// FormatMarkdown renders the runbook as Markdown.
func (rb *Runbook) FormatMarkdown() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# %s\n\n", rb.Title))
	b.WriteString(fmt.Sprintf("**Service:** %s | **Alert:** %s | **Severity:** %s\n\n", rb.Service, rb.AlertName, rb.Severity))

	b.WriteString("## Steps\n\n")
	for _, s := range rb.Steps {
		b.WriteString(fmt.Sprintf("### Step %d: %s\n\n", s.Order, s.Action))
		if s.Command != "" {
			b.WriteString(fmt.Sprintf("```\n%s\n```\n\n", s.Command))
		}
		if s.Expected != "" {
			b.WriteString(fmt.Sprintf("**Expected:** %s\n\n", s.Expected))
		}
		if s.IfFails != "" {
			b.WriteString(fmt.Sprintf("**If fails:** %s\n\n", s.IfFails))
		}
	}

	b.WriteString("## Escalation\n\n")
	for _, e := range rb.Escalation {
		b.WriteString(fmt.Sprintf("- %s\n", e))
	}
	b.WriteString("\n")

	if len(rb.References) > 0 {
		b.WriteString("## References\n\n")
		for _, r := range rb.References {
			b.WriteString(fmt.Sprintf("- %s\n", r))
		}
	}

	return b.String()
}

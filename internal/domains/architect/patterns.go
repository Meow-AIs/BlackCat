package architect

// PatternCategory classifies architecture patterns.
type PatternCategory string

const (
	CatCreational  PatternCategory = "creational"
	CatStructural  PatternCategory = "structural"
	CatBehavioral  PatternCategory = "behavioral"
	CatArchitect   PatternCategory = "architectural"
	CatCloud       PatternCategory = "cloud"
	CatData        PatternCategory = "data"
	CatIntegration PatternCategory = "integration"
	CatResilience  PatternCategory = "resilience"
)

// Pattern represents an architecture or design pattern with structured knowledge.
type Pattern struct {
	Name         string          `json:"name"`
	Category     PatternCategory `json:"category"`
	Description  string          `json:"description"`
	Problem      string          `json:"problem"`
	Solution     string          `json:"solution"`
	Tradeoffs    []string        `json:"tradeoffs"`
	UseCases     []string        `json:"use_cases"`
	AntiPatterns []string        `json:"anti_patterns"`
	RelatedTo    []string        `json:"related_to"`
	Tags         []string        `json:"tags"`
}

// PatternKnowledgeBase stores and queries architecture patterns.
type PatternKnowledgeBase struct {
	patterns map[string]Pattern
}

// NewPatternKnowledgeBase creates an empty knowledge base.
func NewPatternKnowledgeBase() *PatternKnowledgeBase {
	return &PatternKnowledgeBase{patterns: make(map[string]Pattern)}
}

// Add registers a pattern.
func (kb *PatternKnowledgeBase) Add(p Pattern) {
	kb.patterns[p.Name] = p
}

// Get returns a pattern by name.
func (kb *PatternKnowledgeBase) Get(name string) (Pattern, bool) {
	p, ok := kb.patterns[name]
	return p, ok
}

// Search finds patterns matching a query by checking name, tags, and use cases.
func (kb *PatternKnowledgeBase) Search(query string) []Pattern {
	var results []Pattern
	q := toLowerCase(query)

	for _, p := range kb.patterns {
		if containsLower(p.Name, q) || containsLower(p.Description, q) || containsLower(p.Problem, q) {
			results = append(results, p)
			continue
		}
		for _, tag := range p.Tags {
			if containsLower(tag, q) {
				results = append(results, p)
				break
			}
		}
	}
	return results
}

// ByCategory returns all patterns in a given category.
func (kb *PatternKnowledgeBase) ByCategory(cat PatternCategory) []Pattern {
	var results []Pattern
	for _, p := range kb.patterns {
		if p.Category == cat {
			results = append(results, p)
		}
	}
	return results
}

// Related returns patterns related to the given pattern name.
func (kb *PatternKnowledgeBase) Related(name string) []Pattern {
	p, ok := kb.patterns[name]
	if !ok {
		return nil
	}
	var results []Pattern
	for _, rel := range p.RelatedTo {
		if rp, ok := kb.patterns[rel]; ok {
			results = append(results, rp)
		}
	}
	return results
}

// All returns all registered patterns.
func (kb *PatternKnowledgeBase) All() []Pattern {
	result := make([]Pattern, 0, len(kb.patterns))
	for _, p := range kb.patterns {
		result = append(result, p)
	}
	return result
}

// Count returns the number of patterns.
func (kb *PatternKnowledgeBase) Count() int {
	return len(kb.patterns)
}

// LoadBuiltinPatterns populates the knowledge base with common patterns.
func (kb *PatternKnowledgeBase) LoadBuiltinPatterns() {
	builtins := []Pattern{
		{
			Name: "Circuit Breaker", Category: CatResilience,
			Description: "Prevent cascading failures by wrapping calls in a circuit breaker",
			Problem:     "Downstream service failures cascade to upstream, causing system-wide outage",
			Solution:    "Track failures; open circuit after threshold, fail fast; half-open to probe recovery",
			Tradeoffs:   []string{"Added latency for tracking", "Complexity in tuning thresholds"},
			UseCases:    []string{"Microservice communication", "External API calls", "Database connections"},
			AntiPatterns: []string{"No fallback strategy", "Too aggressive thresholds"},
			RelatedTo:   []string{"Bulkhead", "Retry with Backoff", "Timeout"},
			Tags:        []string{"resilience", "microservices", "fault-tolerance"},
		},
		{
			Name: "Bulkhead", Category: CatResilience,
			Description: "Isolate components so failure in one doesn't affect others",
			Problem:     "One slow service consumes all threads/connections, starving other services",
			Solution:    "Partition resources (thread pools, connection pools) per service/feature",
			Tradeoffs:   []string{"Resource underutilization", "More complex resource management"},
			UseCases:    []string{"Thread pool isolation", "Connection pool per service", "Rate limiting per tenant"},
			RelatedTo:   []string{"Circuit Breaker", "Rate Limiting"},
			Tags:        []string{"resilience", "isolation", "microservices"},
		},
		{
			Name: "CQRS", Category: CatArchitect,
			Description: "Separate read and write models for different scalability and optimization",
			Problem:     "Single model for reads and writes leads to complex queries and write contention",
			Solution:    "Separate command (write) and query (read) sides with different data models",
			Tradeoffs:   []string{"Eventual consistency", "Increased complexity", "Data synchronization overhead"},
			UseCases:    []string{"High-read systems", "Complex domain models", "Event sourcing"},
			AntiPatterns: []string{"Using CQRS for simple CRUD", "Synchronous read model updates"},
			RelatedTo:   []string{"Event Sourcing", "Saga"},
			Tags:        []string{"architecture", "scalability", "ddd"},
		},
		{
			Name: "Event Sourcing", Category: CatData,
			Description: "Store state changes as a sequence of events rather than current state",
			Problem:     "Losing history of state changes, difficulty in audit and replay",
			Solution:    "Persist every state change as an immutable event; derive current state by replaying",
			Tradeoffs:   []string{"Storage growth", "Complex querying", "Eventual consistency"},
			UseCases:    []string{"Financial systems", "Audit trails", "Collaborative editing"},
			RelatedTo:   []string{"CQRS", "Saga"},
			Tags:        []string{"data", "events", "audit", "ddd"},
		},
		{
			Name: "Saga", Category: CatData,
			Description: "Manage distributed transactions through a sequence of local transactions",
			Problem:     "Distributed transactions (2PC) don't scale and create tight coupling",
			Solution:    "Break transaction into steps; each step has a compensating action for rollback",
			Tradeoffs:   []string{"Eventual consistency", "Complex error handling", "No isolation"},
			UseCases:    []string{"Order processing", "Booking systems", "Multi-service workflows"},
			AntiPatterns: []string{"Two-Phase Commit at scale", "Missing compensating transactions"},
			RelatedTo:   []string{"Event Sourcing", "CQRS"},
			Tags:        []string{"distributed", "transactions", "microservices"},
		},
		{
			Name: "API Gateway", Category: CatArchitect,
			Description: "Single entry point for all client requests to backend services",
			Problem:     "Clients must know about and communicate with multiple services",
			Solution:    "Route requests through a gateway that handles auth, rate limiting, routing",
			Tradeoffs:   []string{"Single point of failure", "Added latency", "Deployment bottleneck"},
			UseCases:    []string{"Microservice frontends", "Mobile backends", "Multi-tenant APIs"},
			RelatedTo:   []string{"BFF", "Service Mesh"},
			Tags:        []string{"api", "gateway", "microservices", "routing"},
		},
		{
			Name: "Sidecar", Category: CatCloud,
			Description: "Deploy helper component alongside the main service in the same pod/host",
			Problem:     "Cross-cutting concerns (logging, TLS, auth) duplicated in every service",
			Solution:    "Co-locate a sidecar process that handles cross-cutting concerns transparently",
			Tradeoffs:   []string{"Resource overhead per instance", "Complexity in debugging"},
			UseCases:    []string{"Service mesh (Envoy)", "Log collection", "TLS termination"},
			RelatedTo:   []string{"Service Mesh", "Ambassador"},
			Tags:        []string{"cloud", "kubernetes", "infrastructure"},
		},
		{
			Name: "Strangler Fig", Category: CatArchitect,
			Description: "Incrementally migrate a legacy system by routing traffic to new implementation",
			Problem:     "Big-bang rewrites are risky and rarely succeed",
			Solution:    "Gradually replace legacy with new code behind a routing layer; sunset old parts",
			Tradeoffs:   []string{"Long migration period", "Running two systems", "Complex routing"},
			UseCases:    []string{"Legacy modernization", "Monolith to microservices", "Platform migrations"},
			RelatedTo:   []string{"Anti-Corruption Layer"},
			Tags:        []string{"migration", "legacy", "modernization"},
		},
		{
			Name: "Retry with Backoff", Category: CatResilience,
			Description: "Retry failed operations with increasing delay between attempts",
			Problem:     "Transient failures cause immediate errors; retrying without backoff amplifies load",
			Solution:    "Retry with exponential backoff and jitter; cap max retries",
			Tradeoffs:   []string{"Increased latency on failure", "Amplified load without jitter"},
			UseCases:    []string{"HTTP calls", "Queue processing", "Database connections"},
			RelatedTo:   []string{"Circuit Breaker", "Timeout"},
			Tags:        []string{"resilience", "retry", "backoff"},
		},
		{
			Name: "Outbox", Category: CatIntegration,
			Description: "Reliably publish events by writing to a local outbox table in the same transaction",
			Problem:     "Dual writes to database and message broker can be inconsistent",
			Solution:    "Write event to an outbox table in the same DB transaction; a poller publishes",
			Tradeoffs:   []string{"Polling latency", "Outbox table growth", "At-least-once delivery"},
			UseCases:    []string{"Event-driven architectures", "Microservice integration", "CDC"},
			RelatedTo:   []string{"Event Sourcing", "Saga"},
			Tags:        []string{"events", "messaging", "reliability"},
		},
	}
	for _, p := range builtins {
		kb.Add(p)
	}
}

func toLowerCase(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		b[i] = c
	}
	return string(b)
}

func containsLower(haystack, needle string) bool {
	h := toLowerCase(haystack)
	return len(h) >= len(needle) && containsStr(h, needle)
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

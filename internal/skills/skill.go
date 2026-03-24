package skills

import "context"

// Skill is a learned procedure that the agent can reuse.
type Skill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Trigger     string   `json:"trigger"` // glob pattern for matching tasks
	Steps       []string `json:"steps"`
	SuccessRate float64  `json:"success_rate"`
	UsageCount  int      `json:"usage_count"`
	LastUsedAt  int64    `json:"last_used_at"`
	Source      string   `json:"source"` // "auto-learned" | "manual" | "imported" | "marketplace"
	CreatedAt   int64    `json:"created_at"`
	Version     string   `json:"version,omitempty"`
	Author      string   `json:"author,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	License     string   `json:"license,omitempty"`
	Repository  string   `json:"repository,omitempty"`
	Checksum    string   `json:"checksum,omitempty"`
}

// Manager handles skill CRUD and retrieval.
type Manager interface {
	// Store saves a skill.
	Store(ctx context.Context, skill Skill) error

	// Get returns a skill by ID.
	Get(ctx context.Context, id string) (Skill, error)

	// Match finds skills relevant to the given task description.
	Match(ctx context.Context, taskDescription string, limit int) ([]Skill, error)

	// RecordOutcome updates a skill's success rate after use.
	RecordOutcome(ctx context.Context, id string, success bool) error

	// List returns all skills.
	List(ctx context.Context) ([]Skill, error)

	// Delete removes a skill by ID.
	Delete(ctx context.Context, id string) error

	// Export serializes all skills to JSON.
	Export(ctx context.Context) ([]byte, error)

	// Import loads skills from JSON.
	Import(ctx context.Context, data []byte) (int, error)
}

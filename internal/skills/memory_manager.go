package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// InMemoryManager implements Manager using an in-memory map.
// It is safe for concurrent use and is intended for use when no database
// is available (e.g., in tests or lightweight CLI sessions).
type InMemoryManager struct {
	mu     sync.RWMutex
	skills map[string]Skill
}

// NewInMemoryManager creates an empty in-memory skill manager.
func NewInMemoryManager() *InMemoryManager {
	return &InMemoryManager{
		skills: make(map[string]Skill),
	}
}

// Store saves or replaces a skill by its ID.
// If CreatedAt is zero it is set to the current Unix timestamp.
func (m *InMemoryManager) Store(_ context.Context, skill Skill) error {
	if skill.CreatedAt == 0 {
		skill.CreatedAt = time.Now().Unix()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.skills[skill.ID] = skill
	return nil
}

// Get returns the skill with the given ID or an error if not found.
func (m *InMemoryManager) Get(_ context.Context, id string) (Skill, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.skills[id]
	if !ok {
		return Skill{}, fmt.Errorf("skill not found: %s", id)
	}
	return s, nil
}

// Match returns skills whose description or trigger contains taskDescription
// (case-insensitive substring match), ordered by success rate descending,
// capped at limit results. If limit is <= 0 it defaults to 5.
func (m *InMemoryManager) Match(_ context.Context, taskDescription string, limit int) ([]Skill, error) {
	if limit <= 0 {
		limit = 5
	}
	lower := strings.ToLower(taskDescription)

	m.mu.RLock()
	var matched []Skill
	for _, s := range m.skills {
		if strings.Contains(strings.ToLower(s.Description), lower) ||
			strings.Contains(strings.ToLower(s.Trigger), lower) ||
			strings.Contains(strings.ToLower(s.Name), lower) {
			matched = append(matched, s)
		}
	}
	m.mu.RUnlock()

	// Sort by success rate descending (simple insertion sort — small lists).
	for i := 1; i < len(matched); i++ {
		for j := i; j > 0 && matched[j].SuccessRate > matched[j-1].SuccessRate; j-- {
			matched[j], matched[j-1] = matched[j-1], matched[j]
		}
	}

	if len(matched) > limit {
		matched = matched[:limit]
	}
	return matched, nil
}

// RecordOutcome updates a skill's usage count and success rate.
func (m *InMemoryManager) RecordOutcome(ctx context.Context, id string, success bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.skills[id]
	if !ok {
		return fmt.Errorf("skill not found: %s", id)
	}

	total := float64(s.UsageCount + 1)
	successes := s.SuccessRate * float64(s.UsageCount)
	if success {
		successes++
	}
	s.SuccessRate = successes / total
	s.UsageCount++
	s.LastUsedAt = time.Now().Unix()
	m.skills[id] = s
	return nil
}

// List returns all stored skills.
func (m *InMemoryManager) List(_ context.Context) ([]Skill, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]Skill, 0, len(m.skills))
	for _, s := range m.skills {
		result = append(result, s)
	}
	return result, nil
}

// Delete removes the skill with the given ID. It is a no-op if the skill
// does not exist (idempotent).
func (m *InMemoryManager) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.skills, id)
	return nil
}

// Export serializes all skills to JSON.
func (m *InMemoryManager) Export(ctx context.Context) ([]byte, error) {
	skills, err := m.List(ctx)
	if err != nil {
		return nil, err
	}
	if skills == nil {
		skills = []Skill{}
	}
	return json.MarshalIndent(skills, "", "  ")
}

// Import deserializes skills from JSON and stores them.
// Skills without a Source get "imported" as their source.
// Returns the number of successfully imported skills.
func (m *InMemoryManager) Import(ctx context.Context, data []byte) (int, error) {
	var list []Skill
	if err := json.Unmarshal(data, &list); err != nil {
		return 0, fmt.Errorf("parse skills: %w", err)
	}
	count := 0
	for _, s := range list {
		if s.Source == "" {
			s.Source = "imported"
		}
		if err := m.Store(ctx, s); err == nil {
			count++
		}
	}
	return count, nil
}

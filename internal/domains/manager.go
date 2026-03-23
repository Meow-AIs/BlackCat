package domains

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ErrDomainNotFound is returned when a domain is not registered.
var ErrDomainNotFound = fmt.Errorf("domain not found")

// DefaultManager is an in-memory domain manager with heuristic detection.
type DefaultManager struct {
	mu      sync.RWMutex
	configs map[Domain]DomainConfig
}

// NewDefaultManager creates an empty DefaultManager.
func NewDefaultManager() *DefaultManager {
	return &DefaultManager{
		configs: make(map[Domain]DomainConfig),
	}
}

func (m *DefaultManager) Register(config DomainConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configs[config.Name] = config
	return nil
}

func (m *DefaultManager) Get(domain Domain) (DomainConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cfg, ok := m.configs[domain]
	if !ok {
		return DomainConfig{}, ErrDomainNotFound
	}
	return cfg, nil
}

func (m *DefaultManager) SystemPromptFor(domain Domain) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cfg, ok := m.configs[domain]
	if !ok {
		return ""
	}
	return cfg.SystemPrompt
}

func (m *DefaultManager) ToolsFor(domain Domain) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cfg, ok := m.configs[domain]
	if !ok {
		return nil
	}
	return cfg.Tools
}

func (m *DefaultManager) List() []Domain {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]Domain, 0, len(m.configs))
	for d := range m.configs {
		result = append(result, d)
	}
	return result
}

// Detect scans the project directory for files and keywords that match
// registered domain configurations. Returns the domain with the highest
// confidence, or DomainGeneral if nothing matches.
func (m *DefaultManager) Detect(_ context.Context, projectPath string) (DetectionResult, error) {
	m.mu.RLock()
	configs := make(map[Domain]DomainConfig, len(m.configs))
	for k, v := range m.configs {
		configs[k] = v
	}
	m.mu.RUnlock()

	type scored struct {
		domain Domain
		score  float64
		reason string
	}
	var candidates []scored

	// Scan directory entries (top-level only for speed)
	dirEntries, _ := os.ReadDir(projectPath)
	entryNames := make(map[string]bool, len(dirEntries))
	for _, e := range dirEntries {
		entryNames[e.Name()] = true
	}

	// Read text content from key files for keyword matching
	textContent := readProjectText(projectPath, dirEntries)

	for domain, cfg := range configs {
		var score float64
		var reasons []string

		// File-based detection
		for _, pattern := range cfg.DetectionFiles {
			if matchFilePattern(pattern, entryNames, projectPath) {
				score += 1.0
				reasons = append(reasons, fmt.Sprintf("file match: %s", pattern))
			}
		}

		// Keyword-based detection
		if textContent != "" {
			lower := strings.ToLower(textContent)
			for _, kw := range cfg.DetectionKeywords {
				if strings.Contains(lower, strings.ToLower(kw)) {
					score += 0.5
					reasons = append(reasons, fmt.Sprintf("keyword: %s", kw))
				}
			}
		}

		if score > 0 {
			candidates = append(candidates, scored{
				domain: domain,
				score:  score,
				reason: strings.Join(reasons, "; "),
			})
		}
	}

	if len(candidates) == 0 {
		return DetectionResult{
			Domain:     DomainGeneral,
			Confidence: 0.0,
			Reason:     "no matching patterns found",
		}, nil
	}

	// Pick highest score
	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.score > best.score {
			best = c
		}
	}

	// Normalize confidence to 0-1 range (cap at 1.0)
	confidence := best.score / 5.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return DetectionResult{
		Domain:     best.domain,
		Confidence: confidence,
		Reason:     best.reason,
	}, nil
}

// matchFilePattern checks if a file pattern exists in the project.
func matchFilePattern(pattern string, entryNames map[string]bool, projectPath string) bool {
	// Direct name match
	if entryNames[pattern] {
		return true
	}
	// Try glob match for patterns with wildcards or subdirectories
	matches, _ := filepath.Glob(filepath.Join(projectPath, pattern))
	return len(matches) > 0
}

// readProjectText reads content from common project files for keyword matching.
func readProjectText(projectPath string, entries []os.DirEntry) string {
	var buf strings.Builder
	textFiles := []string{"README.md", "readme.md", "README", "go.mod", "package.json", "Cargo.toml", "pyproject.toml"}
	for _, name := range textFiles {
		for _, e := range entries {
			if strings.EqualFold(e.Name(), name) {
				data, err := os.ReadFile(filepath.Join(projectPath, e.Name()))
				if err == nil && len(data) < 100_000 {
					buf.Write(data)
					buf.WriteByte('\n')
				}
				break
			}
		}
	}
	return buf.String()
}

// BuildSystemPrompt combines the base system prompt with the domain-specific extension.
func BuildSystemPrompt(base string, mgr Manager, domain Domain) string {
	ext := mgr.SystemPromptFor(domain)
	if ext == "" {
		return base
	}
	return base + "\n\n" + ext
}

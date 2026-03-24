package domains

import "context"

// Domain represents a specialization area (e.g., devsecops, architect, general).
type Domain string

const (
	DomainGeneral   Domain = "general"
	DomainDevSecOps Domain = "devsecops"
	DomainArchitect Domain = "architect"
	DomainSysAdmin  Domain = "sysadmin"
)

// AllDomains returns the complete list of recognized domains.
func AllDomains() []Domain {
	return []Domain{DomainGeneral, DomainDevSecOps, DomainArchitect, DomainSysAdmin}
}

// DomainConfig holds the configuration for a domain specialization.
type DomainConfig struct {
	Name              Domain   `json:"name" yaml:"name"`
	Description       string   `json:"description" yaml:"description"`
	SystemPrompt      string   `json:"system_prompt" yaml:"system_prompt"`
	Tools             []string `json:"tools" yaml:"tools"` // tool names to register
	RequiredSkills    []string `json:"required_skills" yaml:"required_skills"`
	DetectionFiles    []string `json:"detection_files" yaml:"detection_files"`       // file patterns for auto-detection
	DetectionKeywords []string `json:"detection_keywords" yaml:"detection_keywords"` // keywords in project files
}

// DetectionResult holds the result of domain auto-detection.
type DetectionResult struct {
	Domain     Domain
	Confidence float64 // 0.0-1.0
	Reason     string  // why this domain was detected
}

// Manager handles domain registration, detection, and prompt injection.
type Manager interface {
	// Register adds a domain configuration.
	Register(config DomainConfig) error

	// Get returns the config for a domain.
	Get(domain Domain) (DomainConfig, error)

	// Detect auto-detects the most likely domain for a project.
	Detect(ctx context.Context, projectPath string) (DetectionResult, error)

	// SystemPromptFor returns the system prompt extension for a domain.
	SystemPromptFor(domain Domain) string

	// ToolsFor returns the tool names associated with a domain.
	ToolsFor(domain Domain) []string

	// List returns all registered domains.
	List() []Domain
}

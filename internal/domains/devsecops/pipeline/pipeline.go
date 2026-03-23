// Package pipeline generates CI/CD configuration files for multiple platforms.
package pipeline

// Platform identifies the CI/CD platform.
type Platform string

const (
	PlatformGitHubActions Platform = "github_actions"
	PlatformGitLabCI      Platform = "gitlab_ci"
)

// Language identifies the project programming language.
type Language string

const (
	LangGo     Language = "go"
	LangNode   Language = "node"
	LangPython Language = "python"
	LangRust   Language = "rust"
)

// PipelineRequest describes the desired CI/CD pipeline configuration.
type PipelineRequest struct {
	Platform             Platform
	Language             Language
	ProjectName          string
	GoVersion            string // e.g., "1.22"
	NodeVersion          string // e.g., "20"
	PythonVersion        string // e.g., "3.12"
	IncludeSecurityGates bool   // add SAST/SCA/SBOM steps
	IncludeDocker        bool   // add Docker build+push
	DockerRegistry       string // e.g., "ghcr.io/myorg"
}

// PipelineResult contains the generated pipeline configuration.
type PipelineResult struct {
	Platform Platform
	Filename string // e.g., ".github/workflows/ci.yml"
	Content  string // the YAML content
}

// supportedLanguages is the set of languages we can generate pipelines for.
var supportedLanguages = map[Language]bool{
	LangGo:     true,
	LangNode:   true,
	LangPython: true,
	LangRust:   true,
}

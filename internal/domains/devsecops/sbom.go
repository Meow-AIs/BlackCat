package devsecops

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// CycloneDXBOM represents a minimal CycloneDX 1.5 SBOM.
type CycloneDXBOM struct {
	BOMFormat    string         `json:"bomFormat"`
	SpecVersion  string         `json:"specVersion"`
	SerialNumber string         `json:"serialNumber"`
	Version      int            `json:"version"`
	Metadata     BOMMetadata    `json:"metadata"`
	Components   []BOMComponent `json:"components"`
}

// BOMMetadata holds SBOM metadata.
type BOMMetadata struct {
	Timestamp string        `json:"timestamp"`
	Tools     []BOMTool     `json:"tools,omitempty"`
	Component *BOMComponent `json:"component,omitempty"`
}

// BOMTool identifies the tool that generated the SBOM.
type BOMTool struct {
	Vendor  string `json:"vendor"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

// BOMComponent represents a software component/dependency.
type BOMComponent struct {
	Type      string `json:"type"` // "library", "application", "framework"
	Name      string `json:"name"`
	Version   string `json:"version"`
	PURL      string `json:"purl,omitempty"` // Package URL (https://github.com/package-url/purl-spec)
	Ecosystem string `json:"ecosystem,omitempty"`
	License   string `json:"license,omitempty"`
}

// GenerateSBOM scans a project directory for dependency manifests and produces
// a CycloneDX 1.5 SBOM. Supports: go.mod, package.json, Cargo.toml, pyproject.toml, requirements.txt.
func GenerateSBOM(projectPath string) (CycloneDXBOM, error) {
	bom := CycloneDXBOM{
		BOMFormat:    "CycloneDX",
		SpecVersion:  "1.5",
		SerialNumber: fmt.Sprintf("urn:uuid:blackcat-%d", time.Now().UnixNano()),
		Version:      1,
		Metadata: BOMMetadata{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Tools: []BOMTool{{
				Vendor:  "MeowAI",
				Name:    "blackcat",
				Version: "0.1.0",
			}},
		},
	}

	parsers := []struct {
		filename string
		parser   func(string) ([]BOMComponent, error)
	}{
		{"go.mod", parseGoMod},
		{"go.sum", parseGoSum},
		{"package.json", parsePackageJSON},
		{"requirements.txt", parseRequirementsTxt},
		{"Cargo.toml", parseCargoToml},
	}

	for _, p := range parsers {
		path := filepath.Join(projectPath, p.filename)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		components, err := p.parser(path)
		if err != nil {
			continue
		}
		bom.Components = append(bom.Components, components...)
	}

	return bom, nil
}

// SBOMToJSON serializes the SBOM to JSON.
func SBOMToJSON(bom CycloneDXBOM) ([]byte, error) {
	return json.MarshalIndent(bom, "", "  ")
}

// parseGoMod extracts dependencies from go.mod.
func parseGoMod(path string) ([]BOMComponent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var components []BOMComponent
	re := regexp.MustCompile(`^\s+(\S+)\s+(v\S+)`)
	inRequire := false

	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "require (") || strings.HasPrefix(trimmed, "require(") {
			inRequire = true
			continue
		}
		if trimmed == ")" {
			inRequire = false
			continue
		}
		if inRequire {
			m := re.FindStringSubmatch(line)
			if m != nil {
				name := m[1]
				version := m[2]
				components = append(components, BOMComponent{
					Type:      "library",
					Name:      name,
					Version:   version,
					PURL:      fmt.Sprintf("pkg:golang/%s@%s", name, version),
					Ecosystem: "go",
				})
			}
		}
	}
	return components, nil
}

// parseGoSum extracts dependencies from go.sum (more comprehensive).
func parseGoSum(path string) ([]BOMComponent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var components []BOMComponent

	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		name := parts[0]
		version := strings.TrimSuffix(parts[1], "/go.mod")

		key := name + "@" + version
		if seen[key] {
			continue
		}
		seen[key] = true

		components = append(components, BOMComponent{
			Type:      "library",
			Name:      name,
			Version:   version,
			PURL:      fmt.Sprintf("pkg:golang/%s@%s", name, version),
			Ecosystem: "go",
		})
	}
	return components, nil
}

// parsePackageJSON extracts dependencies from package.json.
func parsePackageJSON(path string) ([]BOMComponent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}

	var components []BOMComponent
	for name, version := range pkg.Dependencies {
		v := strings.TrimLeft(version, "^~>=<")
		components = append(components, BOMComponent{
			Type:      "library",
			Name:      name,
			Version:   v,
			PURL:      fmt.Sprintf("pkg:npm/%s@%s", name, v),
			Ecosystem: "npm",
		})
	}
	for name, version := range pkg.DevDependencies {
		v := strings.TrimLeft(version, "^~>=<")
		components = append(components, BOMComponent{
			Type:      "library",
			Name:      name,
			Version:   v,
			PURL:      fmt.Sprintf("pkg:npm/%s@%s", name, v),
			Ecosystem: "npm",
		})
	}
	return components, nil
}

// parseRequirementsTxt extracts Python dependencies.
func parseRequirementsTxt(path string) ([]BOMComponent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`^([A-Za-z0-9_\-.]+)\s*[=~><]+\s*([0-9][^\s,;#]*)`)
	var components []BOMComponent

	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "-") {
			continue
		}
		m := re.FindStringSubmatch(trimmed)
		if m != nil {
			components = append(components, BOMComponent{
				Type:      "library",
				Name:      m[1],
				Version:   m[2],
				PURL:      fmt.Sprintf("pkg:pypi/%s@%s", strings.ToLower(m[1]), m[2]),
				Ecosystem: "pypi",
			})
		}
	}
	return components, nil
}

// parseCargoToml extracts Rust dependencies from Cargo.toml.
func parseCargoToml(path string) ([]BOMComponent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`^(\w[\w-]*)\s*=\s*"([0-9][^"]*)"`)
	var components []BOMComponent
	inDeps := false

	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "[dependencies]" || trimmed == "[dev-dependencies]" {
			inDeps = true
			continue
		}
		if strings.HasPrefix(trimmed, "[") {
			inDeps = false
			continue
		}
		if inDeps {
			m := re.FindStringSubmatch(trimmed)
			if m != nil {
				components = append(components, BOMComponent{
					Type:      "library",
					Name:      m[1],
					Version:   m[2],
					PURL:      fmt.Sprintf("pkg:cargo/%s@%s", m[1], m[2]),
					Ecosystem: "cargo",
				})
			}
		}
	}
	return components, nil
}

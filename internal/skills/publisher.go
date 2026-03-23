package skills

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// PublishRequest contains the data needed to publish a skill to a registry.
type PublishRequest struct {
	Package       SkillPackage `json:"package"`
	ReadmeContent string       `json:"readme_content,omitempty"`
}

// Publisher handles publishing and unpublishing skills to a remote registry.
type Publisher struct {
	registryURL string
	apiKey      string
	httpClient  *http.Client
}

// NewPublisher creates a publisher targeting the given registry URL.
func NewPublisher(registryURL, apiKey string) *Publisher {
	return &Publisher{
		registryURL: registryURL,
		apiKey:      apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Validate checks a skill package against publishing rules.
// Returns a list of human-readable error strings. An empty slice means valid.
func (p *Publisher) Validate(pkg SkillPackage) []string {
	var errs []string

	if pkg.Metadata.Name == "" || !strings.Contains(pkg.Metadata.Name, "/") {
		errs = append(errs, "name must be in category/name format (e.g. devsecops/secret-scanner)")
	}
	if pkg.Metadata.Version == "" || !isValidSemver(pkg.Metadata.Version) {
		errs = append(errs, "version must be valid semver (e.g. 1.0.0)")
	}
	if len(pkg.Metadata.Description) < 10 {
		errs = append(errs, "description required (min 10 characters)")
	}
	if len(pkg.Spec.Steps) == 0 {
		errs = append(errs, "at least 1 step is required")
	}
	if pkg.Metadata.Author == "" {
		errs = append(errs, "author is required")
	}
	if pkg.Metadata.License == "" {
		errs = append(errs, "license is required")
	}

	return errs
}

// ComputeChecksum returns the SHA-256 hex digest of the marshaled package.
func (p *Publisher) ComputeChecksum(pkg SkillPackage) string {
	data, err := json.Marshal(pkg)
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// Publish sends a skill package to the remote registry.
func (p *Publisher) Publish(ctx context.Context, req PublishRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal publish request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.registryURL+"/publish", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("publish request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("publish failed: status %d", resp.StatusCode)
	}

	return nil
}

// Unpublish removes a skill version from the remote registry.
func (p *Publisher) Unpublish(ctx context.Context, name, version string) error {
	deleteURL := fmt.Sprintf("%s/skills/%s/versions/%s", p.registryURL, name, version)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, deleteURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("unpublish request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unpublish failed: status %d", resp.StatusCode)
	}

	return nil
}

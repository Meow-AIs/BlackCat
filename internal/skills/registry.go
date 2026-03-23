package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// RegistrySource identifies a remote skill registry.
type RegistrySource struct {
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	Type    string `json:"type"` // "http" or "github"
}

// RegistryEntry is a skill package listing in a registry.
type RegistryEntry struct {
	Package     SkillPackage `json:"package"`
	Downloads   int          `json:"downloads"`
	Rating      float64      `json:"rating"`
	PublishedAt string       `json:"published_at"`
	Checksum    string       `json:"checksum"`
}

// RegistrySearchResult is a paginated search response from a registry.
type RegistrySearchResult struct {
	Entries []RegistryEntry `json:"entries"`
	Total   int             `json:"total"`
	Page    int             `json:"page"`
}

// RegistryClient fetches skill packages from remote registries.
type RegistryClient struct {
	sources    []RegistrySource
	httpClient *http.Client
	cacheDir   string
}

// DefaultSources are the built-in registry sources.
var DefaultSources = []RegistrySource{
	{
		Name:    "official",
		BaseURL: "https://raw.githubusercontent.com/Meow-AIs/blackcat-skills/main",
		Type:    "github",
	},
}

// NewRegistryClient creates a client that queries the given sources.
func NewRegistryClient(sources []RegistrySource, cacheDir string) *RegistryClient {
	return &RegistryClient{
		sources: sources,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cacheDir: cacheDir,
	}
}

// Search queries all sources for skills matching the query and tags.
func (c *RegistryClient) Search(ctx context.Context, query string, tags []string, page int) (*RegistrySearchResult, error) {
	var allEntries []RegistryEntry

	for _, src := range c.sources {
		entries, err := c.searchSource(ctx, src, query, tags, page)
		if err != nil {
			continue // skip failed sources
		}
		allEntries = append(allEntries, entries...)
	}

	return &RegistrySearchResult{
		Entries: allEntries,
		Total:   len(allEntries),
		Page:    page,
	}, nil
}

// GetPackage fetches a specific skill package by name and version.
func (c *RegistryClient) GetPackage(ctx context.Context, name string, version string) (*SkillPackage, error) {
	for _, src := range c.sources {
		pkg, err := c.getPackageFromSource(ctx, src, name, version)
		if err == nil {
			return pkg, nil
		}
	}
	return nil, fmt.Errorf("package %s@%s not found in any source", name, version)
}

// ListVersions returns all available versions for a package.
func (c *RegistryClient) ListVersions(ctx context.Context, name string) ([]string, error) {
	for _, src := range c.sources {
		versions, err := c.listVersionsFromSource(ctx, src, name)
		if err == nil {
			return versions, nil
		}
	}
	return nil, fmt.Errorf("no versions found for %s", name)
}

// FetchChecksum retrieves the checksum for a specific package version.
func (c *RegistryClient) FetchChecksum(ctx context.Context, name, version string) (string, error) {
	for _, src := range c.sources {
		checksum, err := c.fetchChecksumFromSource(ctx, src, name, version)
		if err == nil {
			return checksum, nil
		}
	}
	return "", fmt.Errorf("checksum not found for %s@%s", name, version)
}

func (c *RegistryClient) searchSource(ctx context.Context, src RegistrySource, query string, tags []string, page int) ([]RegistryEntry, error) {
	searchURL := src.BaseURL + "/search"
	params := url.Values{}
	if query != "" {
		params.Set("q", query)
	}
	if len(tags) > 0 {
		params.Set("tags", strings.Join(tags, ","))
	}
	params.Set("page", fmt.Sprintf("%d", page))

	fullURL := searchURL + "?" + params.Encode()
	body, err := c.doGet(ctx, fullURL)
	if err != nil {
		return nil, err
	}

	var result RegistrySearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse search results: %w", err)
	}
	return result.Entries, nil
}

func (c *RegistryClient) getPackageFromSource(ctx context.Context, src RegistrySource, name, version string) (*SkillPackage, error) {
	pkgURL := fmt.Sprintf("%s/skills/%s/versions/%s/skill.json", src.BaseURL, name, version)
	body, err := c.doGet(ctx, pkgURL)
	if err != nil {
		return nil, err
	}

	var pkg SkillPackage
	if err := json.Unmarshal(body, &pkg); err != nil {
		return nil, fmt.Errorf("parse package: %w", err)
	}
	return &pkg, nil
}

func (c *RegistryClient) listVersionsFromSource(ctx context.Context, src RegistrySource, name string) ([]string, error) {
	versionsURL := fmt.Sprintf("%s/skills/%s/versions.json", src.BaseURL, name)
	body, err := c.doGet(ctx, versionsURL)
	if err != nil {
		return nil, err
	}

	var versions []string
	if err := json.Unmarshal(body, &versions); err != nil {
		return nil, fmt.Errorf("parse versions: %w", err)
	}
	return versions, nil
}

func (c *RegistryClient) fetchChecksumFromSource(ctx context.Context, src RegistrySource, name, version string) (string, error) {
	checksumURL := fmt.Sprintf("%s/skills/%s/versions/%s/checksum", src.BaseURL, name, version)
	body, err := c.doGet(ctx, checksumURL)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

func (c *RegistryClient) doGet(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: status %d", rawURL, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	return body, nil
}

package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// DefaultPluginRegistryURL is the base URL of the official plugin registry.
var DefaultPluginRegistryURL = "https://plugins.blackcat.dev/api/v1"

// RegistryEntry describes a plugin available in the remote registry.
type RegistryEntry struct {
	Manifest    PluginManifest    `json:"manifest"`
	Downloads   int               `json:"downloads"`
	Rating      float64           `json:"rating"`
	PublishedAt string            `json:"published_at"`
	Checksum    string            `json:"checksum"`
	BinaryURLs  map[string]string `json:"binary_urls"`
}

// PluginRegistry is a client for the remote plugin registry API.
type PluginRegistry struct {
	baseURL    string
	httpClient *http.Client
}

// NewPluginRegistry creates a registry client pointing at baseURL.
func NewPluginRegistry(baseURL string) *PluginRegistry {
	return &PluginRegistry{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

// Search queries the registry for plugins matching the query and optional type.
func (r *PluginRegistry) Search(ctx context.Context, query string, pluginType PluginType) ([]RegistryEntry, error) {
	u, err := url.Parse(r.baseURL + "/search")
	if err != nil {
		return nil, fmt.Errorf("parse search URL: %w", err)
	}

	q := u.Query()
	q.Set("q", query)
	if pluginType != "" {
		q.Set("type", string(pluginType))
	}
	u.RawQuery = q.Encode()

	var entries []RegistryEntry
	if err := r.doJSON(ctx, u.String(), &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// GetManifest fetches the manifest for a specific plugin version.
func (r *PluginRegistry) GetManifest(ctx context.Context, name string, version string) (*PluginManifest, error) {
	endpoint := fmt.Sprintf("%s/plugins/%s/versions/%s", r.baseURL, name, version)

	var manifest PluginManifest
	if err := r.doJSON(ctx, endpoint, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

// Download fetches the binary for a plugin version and platform.
func (r *PluginRegistry) Download(ctx context.Context, name, version, platform string) ([]byte, error) {
	endpoint := fmt.Sprintf("%s/plugins/%s/download/%s/%s", r.baseURL, name, version, platform)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// ListVersions returns all available versions for a plugin.
func (r *PluginRegistry) ListVersions(ctx context.Context, name string) ([]string, error) {
	endpoint := fmt.Sprintf("%s/plugins/%s/versions", r.baseURL, name)

	var versions []string
	if err := r.doJSON(ctx, endpoint, &versions); err != nil {
		return nil, err
	}
	return versions, nil
}

// doJSON performs a GET request and decodes the JSON response into dest.
func (r *PluginRegistry) doJSON(ctx context.Context, endpoint string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: HTTP %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

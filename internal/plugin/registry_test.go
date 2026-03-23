package plugin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupTestRegistry(t *testing.T) (*PluginRegistry, *httptest.Server) {
	t.Helper()

	mux := http.NewServeMux()

	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		pluginType := r.URL.Query().Get("type")

		entries := []RegistryEntry{
			{
				Manifest: PluginManifest{
					Name:         "author/test-plugin",
					Version:      "1.2.0",
					Type:         PluginProvider,
					Description:  "A test provider",
					Author:       "author",
					License:      "MIT",
					Command:      "test-plugin",
					Protocol:     "jsonrpc",
					Capabilities: []string{"chat"},
				},
				Downloads:   1000,
				Rating:      4.5,
				PublishedAt: "2025-01-15T00:00:00Z",
				Checksum:    "sha256:abc123",
				BinaryURLs:  map[string]string{"linux-amd64": "https://example.com/test-plugin-linux-amd64"},
			},
		}

		// Filter by type if provided
		if pluginType != "" {
			var filtered []RegistryEntry
			for _, e := range entries {
				if string(e.Manifest.Type) == pluginType {
					filtered = append(filtered, e)
				}
			}
			entries = filtered
		}

		// Filter by query if provided
		if query == "nonexistent" {
			entries = nil
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entries)
	})

	mux.HandleFunc("/plugins/author/test-plugin/versions/1.2.0", func(w http.ResponseWriter, r *http.Request) {
		manifest := PluginManifest{
			Name:         "author/test-plugin",
			Version:      "1.2.0",
			Type:         PluginProvider,
			Description:  "A test provider",
			Author:       "author",
			License:      "MIT",
			Command:      "test-plugin",
			Protocol:     "jsonrpc",
			Capabilities: []string{"chat"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(manifest)
	})

	mux.HandleFunc("/plugins/author/test-plugin/versions", func(w http.ResponseWriter, r *http.Request) {
		versions := []string{"1.0.0", "1.1.0", "1.2.0"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(versions)
	})

	mux.HandleFunc("/plugins/author/test-plugin/download/1.2.0/linux-amd64", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write([]byte("fake-binary-content"))
	})

	server := httptest.NewServer(mux)
	registry := NewPluginRegistry(server.URL)

	return registry, server
}

func TestRegistrySearch(t *testing.T) {
	reg, server := setupTestRegistry(t)
	defer server.Close()

	results, err := reg.Search(context.Background(), "test", PluginProvider)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
	if results[0].Manifest.Name != "author/test-plugin" {
		t.Errorf("expected author/test-plugin, got %s", results[0].Manifest.Name)
	}
	if results[0].Downloads != 1000 {
		t.Errorf("expected 1000 downloads, got %d", results[0].Downloads)
	}
}

func TestRegistrySearchEmpty(t *testing.T) {
	reg, server := setupTestRegistry(t)
	defer server.Close()

	results, err := reg.Search(context.Background(), "nonexistent", "")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestRegistryGetManifest(t *testing.T) {
	reg, server := setupTestRegistry(t)
	defer server.Close()

	manifest, err := reg.GetManifest(context.Background(), "author/test-plugin", "1.2.0")
	if err != nil {
		t.Fatalf("get manifest failed: %v", err)
	}
	if manifest.Name != "author/test-plugin" {
		t.Errorf("expected author/test-plugin, got %s", manifest.Name)
	}
	if manifest.Version != "1.2.0" {
		t.Errorf("expected version 1.2.0, got %s", manifest.Version)
	}
}

func TestRegistryDownload(t *testing.T) {
	reg, server := setupTestRegistry(t)
	defer server.Close()

	data, err := reg.Download(context.Background(), "author/test-plugin", "1.2.0", "linux-amd64")
	if err != nil {
		t.Fatalf("download failed: %v", err)
	}
	if string(data) != "fake-binary-content" {
		t.Errorf("expected fake-binary-content, got %s", string(data))
	}
}

func TestRegistryListVersions(t *testing.T) {
	reg, server := setupTestRegistry(t)
	defer server.Close()

	versions, err := reg.ListVersions(context.Background(), "author/test-plugin")
	if err != nil {
		t.Fatalf("list versions failed: %v", err)
	}
	if len(versions) != 3 {
		t.Fatalf("expected 3 versions, got %d", len(versions))
	}
	if versions[2] != "1.2.0" {
		t.Errorf("expected 1.2.0, got %s", versions[2])
	}
}

func TestRegistrySearchCancelled(t *testing.T) {
	reg, server := setupTestRegistry(t)
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := reg.Search(ctx, "test", "")
	if err == nil {
		t.Error("expected error with cancelled context")
	}
}

func TestRegistryServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	reg := NewPluginRegistry(server.URL)

	_, err := reg.Search(context.Background(), "test", "")
	if err == nil {
		t.Error("expected error on server error response")
	}
}

package skills

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupMockRegistry(t *testing.T) (*httptest.Server, []RegistryEntry) {
	t.Helper()

	entries := []RegistryEntry{
		{
			Package: SkillPackage{
				APIVersion: "v1",
				Kind:       "skill",
				Metadata: PackageMetadata{
					Name:        "devsecops/secret-scanner",
					Version:     "1.2.0",
					Author:      "alice",
					Description: "Scans repos for leaked secrets and credentials",
					License:     "MIT",
					Tags:        []string{"security", "scanner"},
				},
				Spec: SkillSpec{
					Trigger: "scan*",
					Steps:   []SkillStep{{Name: "detect", Description: "Find secrets"}},
				},
			},
			Downloads:   150,
			Rating:      4.5,
			PublishedAt: "2026-01-15T10:00:00Z",
			Checksum:    "abc123",
		},
		{
			Package: SkillPackage{
				APIVersion: "v1",
				Kind:       "skill",
				Metadata: PackageMetadata{
					Name:        "devops/docker-deploy",
					Version:     "2.0.0",
					Author:      "bob",
					Description: "Deploy containers to production clusters easily",
					License:     "Apache-2.0",
					Tags:        []string{"devops", "docker"},
				},
				Spec: SkillSpec{
					Trigger: "deploy*",
					Steps:   []SkillStep{{Name: "build", Description: "Build image"}, {Name: "push", Description: "Push image"}},
				},
			},
			Downloads:   300,
			Rating:      4.8,
			PublishedAt: "2026-02-01T12:00:00Z",
			Checksum:    "def456",
		},
	}

	mux := http.NewServeMux()

	// Index endpoint
	mux.HandleFunc("/index.json", func(w http.ResponseWriter, r *http.Request) {
		result := RegistrySearchResult{
			Entries: entries,
			Total:   len(entries),
			Page:    1,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	// Search endpoint
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		var matched []RegistryEntry
		for _, e := range entries {
			if query == "" || containsStr(e.Package.Metadata.Name, query) || containsStr(e.Package.Metadata.Description, query) {
				matched = append(matched, e)
			}
		}
		result := RegistrySearchResult{Entries: matched, Total: len(matched), Page: 1}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	// Package endpoint
	mux.HandleFunc("/skills/", func(w http.ResponseWriter, r *http.Request) {
		// Route: /skills/{name}/versions/{version}/skill.json
		for _, e := range entries {
			expectedPath := "/skills/" + e.Package.Metadata.Name + "/versions/" + e.Package.Metadata.Version + "/skill.json"
			if r.URL.Path == expectedPath {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(e.Package)
				return
			}
			// Versions list
			versionsPath := "/skills/" + e.Package.Metadata.Name + "/versions.json"
			if r.URL.Path == versionsPath {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode([]string{e.Package.Metadata.Version})
				return
			}
			// Checksum
			checksumPath := "/skills/" + e.Package.Metadata.Name + "/versions/" + e.Package.Metadata.Version + "/checksum"
			if r.URL.Path == checksumPath {
				w.Write([]byte(e.Checksum))
				return
			}
		}
		http.NotFound(w, r)
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server, entries
}

func containsStr(haystack, needle string) bool {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

func TestRegistryClient_Search(t *testing.T) {
	server, _ := setupMockRegistry(t)
	client := NewRegistryClient(
		[]RegistrySource{{Name: "test", BaseURL: server.URL, Type: "http"}},
		t.TempDir(),
	)

	results, err := client.Search(context.Background(), "secret", nil, 1)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if results.Total == 0 {
		t.Error("expected at least one result")
	}
	if results.Entries[0].Package.Metadata.Name != "devsecops/secret-scanner" {
		t.Errorf("expected secret-scanner, got %s", results.Entries[0].Package.Metadata.Name)
	}
}

func TestRegistryClient_Search_Empty(t *testing.T) {
	server, _ := setupMockRegistry(t)
	client := NewRegistryClient(
		[]RegistrySource{{Name: "test", BaseURL: server.URL, Type: "http"}},
		t.TempDir(),
	)

	results, err := client.Search(context.Background(), "nonexistent-xyz-123", nil, 1)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results.Entries) != 0 {
		t.Errorf("expected 0 results, got %d", len(results.Entries))
	}
}

func TestRegistryClient_GetPackage(t *testing.T) {
	server, _ := setupMockRegistry(t)
	client := NewRegistryClient(
		[]RegistrySource{{Name: "test", BaseURL: server.URL, Type: "http"}},
		t.TempDir(),
	)

	pkg, err := client.GetPackage(context.Background(), "devsecops/secret-scanner", "1.2.0")
	if err != nil {
		t.Fatalf("GetPackage failed: %v", err)
	}
	if pkg.Metadata.Name != "devsecops/secret-scanner" {
		t.Errorf("expected secret-scanner, got %s", pkg.Metadata.Name)
	}
	if pkg.Metadata.Version != "1.2.0" {
		t.Errorf("expected version 1.2.0, got %s", pkg.Metadata.Version)
	}
}

func TestRegistryClient_GetPackage_NotFound(t *testing.T) {
	server, _ := setupMockRegistry(t)
	client := NewRegistryClient(
		[]RegistrySource{{Name: "test", BaseURL: server.URL, Type: "http"}},
		t.TempDir(),
	)

	_, err := client.GetPackage(context.Background(), "nonexistent/pkg", "1.0.0")
	if err == nil {
		t.Error("expected error for nonexistent package")
	}
}

func TestRegistryClient_ListVersions(t *testing.T) {
	server, _ := setupMockRegistry(t)
	client := NewRegistryClient(
		[]RegistrySource{{Name: "test", BaseURL: server.URL, Type: "http"}},
		t.TempDir(),
	)

	versions, err := client.ListVersions(context.Background(), "devsecops/secret-scanner")
	if err != nil {
		t.Fatalf("ListVersions failed: %v", err)
	}
	if len(versions) == 0 {
		t.Error("expected at least one version")
	}
	if versions[0] != "1.2.0" {
		t.Errorf("expected version 1.2.0, got %s", versions[0])
	}
}

func TestRegistryClient_FetchChecksum(t *testing.T) {
	server, _ := setupMockRegistry(t)
	client := NewRegistryClient(
		[]RegistrySource{{Name: "test", BaseURL: server.URL, Type: "http"}},
		t.TempDir(),
	)

	checksum, err := client.FetchChecksum(context.Background(), "devsecops/secret-scanner", "1.2.0")
	if err != nil {
		t.Fatalf("FetchChecksum failed: %v", err)
	}
	if checksum != "abc123" {
		t.Errorf("expected checksum abc123, got %s", checksum)
	}
}

func TestNewRegistryClient_DefaultSources(t *testing.T) {
	if len(DefaultSources) == 0 {
		t.Error("expected at least one default source")
	}
	if DefaultSources[0].Name != "official" {
		t.Errorf("expected official source, got %s", DefaultSources[0].Name)
	}
}

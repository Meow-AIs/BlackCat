package skills

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPublisher_Validate_Valid(t *testing.T) {
	pub := NewPublisher("https://example.com", "test-key")
	pkg := SkillPackage{
		APIVersion: "v1",
		Kind:       "skill",
		Metadata: PackageMetadata{
			Name:        "tools/linter",
			Version:     "1.0.0",
			Author:      "dev",
			Description: "Lint source code files automatically",
			License:     "MIT",
		},
		Spec: SkillSpec{
			Trigger: "lint*",
			Steps:   []SkillStep{{Name: "run", Description: "Run linter"}},
		},
	}

	errs := pub.Validate(pkg)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestPublisher_Validate_Invalid(t *testing.T) {
	pub := NewPublisher("https://example.com", "test-key")

	tests := []struct {
		name string
		pkg  SkillPackage
	}{
		{
			name: "no category slash in name",
			pkg: SkillPackage{
				APIVersion: "v1", Kind: "skill",
				Metadata: PackageMetadata{Name: "noslash", Version: "1.0.0", Author: "x", Description: "long enough desc", License: "MIT"},
				Spec:     SkillSpec{Steps: []SkillStep{{Name: "s", Description: "d"}}},
			},
		},
		{
			name: "bad semver",
			pkg: SkillPackage{
				APIVersion: "v1", Kind: "skill",
				Metadata: PackageMetadata{Name: "a/b", Version: "bad", Author: "x", Description: "long enough desc", License: "MIT"},
				Spec:     SkillSpec{Steps: []SkillStep{{Name: "s", Description: "d"}}},
			},
		},
		{
			name: "short description",
			pkg: SkillPackage{
				APIVersion: "v1", Kind: "skill",
				Metadata: PackageMetadata{Name: "a/b", Version: "1.0.0", Author: "x", Description: "short", License: "MIT"},
				Spec:     SkillSpec{Steps: []SkillStep{{Name: "s", Description: "d"}}},
			},
		},
		{
			name: "no steps",
			pkg: SkillPackage{
				APIVersion: "v1", Kind: "skill",
				Metadata: PackageMetadata{Name: "a/b", Version: "1.0.0", Author: "x", Description: "long enough desc", License: "MIT"},
				Spec:     SkillSpec{Steps: []SkillStep{}},
			},
		},
		{
			name: "missing author",
			pkg: SkillPackage{
				APIVersion: "v1", Kind: "skill",
				Metadata: PackageMetadata{Name: "a/b", Version: "1.0.0", Description: "long enough desc", License: "MIT"},
				Spec:     SkillSpec{Steps: []SkillStep{{Name: "s", Description: "d"}}},
			},
		},
		{
			name: "missing license",
			pkg: SkillPackage{
				APIVersion: "v1", Kind: "skill",
				Metadata: PackageMetadata{Name: "a/b", Version: "1.0.0", Author: "x", Description: "long enough desc"},
				Spec:     SkillSpec{Steps: []SkillStep{{Name: "s", Description: "d"}}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := pub.Validate(tt.pkg)
			if len(errs) == 0 {
				t.Error("expected validation errors")
			}
		})
	}
}

func TestPublisher_ComputeChecksum(t *testing.T) {
	pub := NewPublisher("https://example.com", "test-key")
	pkg := SkillPackage{
		APIVersion: "v1",
		Kind:       "skill",
		Metadata:   PackageMetadata{Name: "a/b", Version: "1.0.0"},
	}

	checksum := pub.ComputeChecksum(pkg)
	if checksum == "" {
		t.Error("expected non-empty checksum")
	}
	if len(checksum) != 64 { // sha256 hex
		t.Errorf("expected 64-char hex checksum, got %d chars", len(checksum))
	}

	// Same input should produce same checksum
	checksum2 := pub.ComputeChecksum(pkg)
	if checksum != checksum2 {
		t.Error("expected deterministic checksum")
	}

	// Different input should produce different checksum
	pkg2 := pkg
	pkg2.Metadata.Version = "2.0.0"
	checksum3 := pub.ComputeChecksum(pkg2)
	if checksum == checksum3 {
		t.Error("expected different checksum for different package")
	}
}

func TestPublisher_Publish(t *testing.T) {
	var receivedBody PublishRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Bearer test-key auth header")
		}
		if r.URL.Path != "/publish" {
			t.Errorf("expected /publish path, got %s", r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	pub := NewPublisher(server.URL, "test-key")
	pkg := SkillPackage{
		APIVersion: "v1",
		Kind:       "skill",
		Metadata: PackageMetadata{
			Name:        "tools/example",
			Version:     "1.0.0",
			Author:      "tester",
			Description: "An example skill for publishing",
			License:     "MIT",
		},
		Spec: SkillSpec{
			Trigger: "example*",
			Steps:   []SkillStep{{Name: "run", Description: "Do the thing"}},
		},
	}

	err := pub.Publish(context.Background(), PublishRequest{Package: pkg, ReadmeContent: "# Example"})
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}
	if receivedBody.Package.Metadata.Name != "tools/example" {
		t.Errorf("expected tools/example, got %s", receivedBody.Package.Metadata.Name)
	}
}

func TestPublisher_Publish_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	pub := NewPublisher(server.URL, "test-key")
	pkg := SkillPackage{
		APIVersion: "v1",
		Kind:       "skill",
		Metadata: PackageMetadata{
			Name: "a/b", Version: "1.0.0", Author: "x",
			Description: "long enough desc", License: "MIT",
		},
		Spec: SkillSpec{Steps: []SkillStep{{Name: "s", Description: "d"}}},
	}

	err := pub.Publish(context.Background(), PublishRequest{Package: pkg})
	if err == nil {
		t.Error("expected error for server error response")
	}
}

func TestPublisher_Unpublish(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/skills/tools/example/versions/1.0.0" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	pub := NewPublisher(server.URL, "test-key")
	err := pub.Unpublish(context.Background(), "tools/example", "1.0.0")
	if err != nil {
		t.Fatalf("Unpublish failed: %v", err)
	}
}

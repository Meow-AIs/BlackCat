package devsecops

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// GenerateSBOM — go.mod
// ---------------------------------------------------------------------------

func TestGenerateSBOM_GoMod(t *testing.T) {
	dir := t.TempDir()
	goMod := `module github.com/example/app

go 1.22

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/lib/pq v1.10.9
)
`
	writeFile(t, dir, "go.mod", goMod)

	bom, err := GenerateSBOM(dir)
	if err != nil {
		t.Fatal(err)
	}
	if bom.BOMFormat != "CycloneDX" {
		t.Errorf("expected CycloneDX format, got %q", bom.BOMFormat)
	}
	if bom.SpecVersion != "1.5" {
		t.Errorf("expected spec 1.5, got %q", bom.SpecVersion)
	}
	if len(bom.Components) != 2 {
		t.Errorf("expected 2 components from go.mod, got %d", len(bom.Components))
	}
	for _, c := range bom.Components {
		if c.Ecosystem != "go" {
			t.Errorf("expected go ecosystem, got %q", c.Ecosystem)
		}
		if c.PURL == "" {
			t.Errorf("expected non-empty PURL for %q", c.Name)
		}
	}
}

// ---------------------------------------------------------------------------
// GenerateSBOM — package.json
// ---------------------------------------------------------------------------

func TestGenerateSBOM_PackageJSON(t *testing.T) {
	dir := t.TempDir()
	pkgJSON := `{
  "dependencies": {
    "express": "^4.18.2",
    "lodash": "~4.17.21"
  },
  "devDependencies": {
    "jest": "^29.7.0"
  }
}`
	writeFile(t, dir, "package.json", pkgJSON)

	bom, err := GenerateSBOM(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(bom.Components) != 3 {
		t.Errorf("expected 3 npm components, got %d", len(bom.Components))
	}
	for _, c := range bom.Components {
		if c.Ecosystem != "npm" {
			t.Errorf("expected npm ecosystem, got %q", c.Ecosystem)
		}
	}
}

// ---------------------------------------------------------------------------
// GenerateSBOM — requirements.txt
// ---------------------------------------------------------------------------

func TestGenerateSBOM_RequirementsTxt(t *testing.T) {
	dir := t.TempDir()
	reqs := `flask==2.3.2
requests>=2.31.0
# comment
numpy==1.24.0
`
	writeFile(t, dir, "requirements.txt", reqs)

	bom, err := GenerateSBOM(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(bom.Components) != 3 {
		t.Errorf("expected 3 pypi components, got %d", len(bom.Components))
	}
}

// ---------------------------------------------------------------------------
// GenerateSBOM — Cargo.toml
// ---------------------------------------------------------------------------

func TestGenerateSBOM_CargoToml(t *testing.T) {
	dir := t.TempDir()
	cargo := `[package]
name = "myapp"
version = "0.1.0"

[dependencies]
serde = "1.0"
tokio = "1.35"

[dev-dependencies]
criterion = "0.5"
`
	writeFile(t, dir, "Cargo.toml", cargo)

	bom, err := GenerateSBOM(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(bom.Components) != 3 {
		t.Errorf("expected 3 cargo components, got %d", len(bom.Components))
	}
}

// ---------------------------------------------------------------------------
// GenerateSBOM — empty project
// ---------------------------------------------------------------------------

func TestGenerateSBOM_EmptyProject(t *testing.T) {
	dir := t.TempDir()
	bom, err := GenerateSBOM(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(bom.Components) != 0 {
		t.Errorf("expected 0 components for empty project, got %d", len(bom.Components))
	}
}

// ---------------------------------------------------------------------------
// GenerateSBOM — multiple manifests combined
// ---------------------------------------------------------------------------

func TestGenerateSBOM_MultipleManifests(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", `module app

go 1.22

require (
	github.com/lib/pq v1.10.9
)
`)
	writeFile(t, dir, "requirements.txt", "flask==2.3.2\n")

	bom, err := GenerateSBOM(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(bom.Components) != 2 {
		t.Errorf("expected 2 total components, got %d", len(bom.Components))
	}
}

// ---------------------------------------------------------------------------
// SBOMToJSON
// ---------------------------------------------------------------------------

func TestSBOMToJSON_ValidJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", `module app

go 1.22

require (
	github.com/lib/pq v1.10.9
)
`)

	bom, _ := GenerateSBOM(dir)
	data, err := SBOMToJSON(bom)
	if err != nil {
		t.Fatal(err)
	}

	var parsed CycloneDXBOM
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if parsed.BOMFormat != "CycloneDX" {
		t.Errorf("round-trip failed: bomFormat = %q", parsed.BOMFormat)
	}
}

// ---------------------------------------------------------------------------
// BOM Metadata
// ---------------------------------------------------------------------------

func TestGenerateSBOM_HasMetadata(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", `module app
go 1.22
require (
	github.com/lib/pq v1.10.9
)
`)
	bom, _ := GenerateSBOM(dir)
	if bom.Metadata.Timestamp == "" {
		t.Error("expected non-empty timestamp")
	}
	if len(bom.Metadata.Tools) == 0 {
		t.Error("expected at least one tool in metadata")
	}
	if bom.Metadata.Tools[0].Name != "blackcat" {
		t.Errorf("expected tool name 'blackcat', got %q", bom.Metadata.Tools[0].Name)
	}
}

// ---------------------------------------------------------------------------
// PURL format
// ---------------------------------------------------------------------------

func TestBOMComponent_PURL_GoFormat(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", `module app
go 1.22
require (
	github.com/lib/pq v1.10.9
)
`)
	bom, _ := GenerateSBOM(dir)
	for _, c := range bom.Components {
		if c.Name == "github.com/lib/pq" {
			expected := "pkg:golang/github.com/lib/pq@v1.10.9"
			if c.PURL != expected {
				t.Errorf("PURL = %q, want %q", c.PURL, expected)
			}
			return
		}
	}
	t.Error("github.com/lib/pq not found in components")
}

// ---------------------------------------------------------------------------
// SerialNumber uniqueness
// ---------------------------------------------------------------------------

func TestGenerateSBOM_UniqueSerialNumbers(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module app\ngo 1.22\n"), 0644)

	bom1, _ := GenerateSBOM(dir)
	bom2, _ := GenerateSBOM(dir)

	if bom1.SerialNumber == bom2.SerialNumber {
		t.Error("serial numbers should be unique across invocations")
	}
}

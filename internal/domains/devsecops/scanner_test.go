package devsecops

import (
	"context"
	"testing"
)

// ---------------------------------------------------------------------------
// ScannerRegistry
// ---------------------------------------------------------------------------

type mockScanner struct {
	name     string
	findings []Finding
}

func (m *mockScanner) Name() string        { return m.name }
func (m *mockScanner) Description() string { return "mock scanner" }
func (m *mockScanner) Scan(_ context.Context, _ ScanRequest) (ScanResult, error) {
	return ScanResult{Scanner: m.name, Findings: m.findings, Scanned: 1}, nil
}

func TestScannerRegistry_RegisterAndGet(t *testing.T) {
	reg := NewScannerRegistry()
	reg.Register(&mockScanner{name: "test_scanner"})

	s, ok := reg.Get("test_scanner")
	if !ok {
		t.Fatal("expected scanner to be found")
	}
	if s.Name() != "test_scanner" {
		t.Errorf("expected name 'test_scanner', got %q", s.Name())
	}
}

func TestScannerRegistry_Get_NotFound(t *testing.T) {
	reg := NewScannerRegistry()
	_, ok := reg.Get("nonexistent")
	if ok {
		t.Error("expected scanner not found")
	}
}

func TestScannerRegistry_Names(t *testing.T) {
	reg := NewScannerRegistry()
	reg.Register(&mockScanner{name: "a"})
	reg.Register(&mockScanner{name: "b"})

	names := reg.Names()
	if len(names) != 2 {
		t.Errorf("expected 2 names, got %d", len(names))
	}
}

func TestScannerRegistry_ScanAll(t *testing.T) {
	reg := NewScannerRegistry()
	reg.Register(&mockScanner{name: "s1", findings: []Finding{{Title: "f1"}}})
	reg.Register(&mockScanner{name: "s2", findings: []Finding{{Title: "f2"}, {Title: "f3"}}})

	results, err := reg.ScanAll(context.Background(), ScanRequest{Path: "."})
	if err != nil {
		t.Fatalf("ScanAll: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	totalFindings := 0
	for _, r := range results {
		totalFindings += len(r.Findings)
	}
	if totalFindings != 3 {
		t.Errorf("expected 3 total findings, got %d", totalFindings)
	}
}

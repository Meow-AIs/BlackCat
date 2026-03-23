package devsecops

import (
	"testing"
	"time"
)

func TestDefaultFeeds(t *testing.T) {
	if len(DefaultFeeds) != 3 {
		t.Fatalf("expected 3 default feeds, got %d", len(DefaultFeeds))
	}

	feedNames := map[string]bool{}
	for _, f := range DefaultFeeds {
		feedNames[f.Name] = true
		if f.URL == "" {
			t.Errorf("feed %q has empty URL", f.Name)
		}
		if f.FeedType == "" {
			t.Errorf("feed %q has empty FeedType", f.Name)
		}
		if f.UpdateFreq <= 0 {
			t.Errorf("feed %q has non-positive UpdateFreq", f.Name)
		}
	}

	for _, name := range []string{"CISA KEV", "EPSS", "OSV"} {
		if !feedNames[name] {
			t.Errorf("missing default feed %q", name)
		}
	}
}

const sampleKEVJSON = `{
  "title": "CISA KEV",
  "catalogVersion": "2025.01.01",
  "vulnerabilities": [
    {
      "cveID": "CVE-2024-1234",
      "vendorProject": "Acme",
      "product": "Widget",
      "dateAdded": "2024-06-01",
      "dueDate": "2024-06-15",
      "knownRansomwareCampaignUse": "Known",
      "shortDescription": "Remote code execution in Widget"
    },
    {
      "cveID": "CVE-2024-5678",
      "vendorProject": "BigCorp",
      "product": "Server",
      "dateAdded": "2024-07-01",
      "dueDate": "2024-07-15",
      "knownRansomwareCampaignUse": "Unknown",
      "shortDescription": "Privilege escalation in Server"
    }
  ]
}`

func TestParseKEVFeed(t *testing.T) {
	entries, err := ParseKEVFeed([]byte(sampleKEVJSON))
	if err != nil {
		t.Fatalf("ParseKEVFeed failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	first := entries[0]
	if first.CVEID != "CVE-2024-1234" {
		t.Errorf("expected CVE-2024-1234, got %s", first.CVEID)
	}
	if first.VendorProject != "Acme" {
		t.Errorf("expected Acme, got %s", first.VendorProject)
	}
	if first.Product != "Widget" {
		t.Errorf("expected Widget, got %s", first.Product)
	}
	if !first.RansomwareUse {
		t.Error("expected RansomwareUse to be true for 'Known'")
	}

	second := entries[1]
	if second.RansomwareUse {
		t.Error("expected RansomwareUse to be false for 'Unknown'")
	}
}

func TestParseKEVFeed_InvalidJSON(t *testing.T) {
	_, err := ParseKEVFeed([]byte(`{invalid`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

const sampleEPSSJSON = `{
  "status": "OK",
  "status-code": 200,
  "version": "1.0",
  "total": 2,
  "data": [
    {"cve": "CVE-2024-1234", "epss": "0.85", "percentile": "0.97"},
    {"cve": "CVE-2024-9999", "epss": "0.02", "percentile": "0.30"}
  ]
}`

func TestParseEPSSResponse(t *testing.T) {
	entries, err := ParseEPSSResponse([]byte(sampleEPSSJSON))
	if err != nil {
		t.Fatalf("ParseEPSSResponse failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	if entries[0].CVEID != "CVE-2024-1234" {
		t.Errorf("expected CVE-2024-1234, got %s", entries[0].CVEID)
	}
	if entries[0].EPSS != 0.85 {
		t.Errorf("expected EPSS 0.85, got %f", entries[0].EPSS)
	}
	if entries[0].Percentile != 0.97 {
		t.Errorf("expected percentile 0.97, got %f", entries[0].Percentile)
	}
}

func TestParseEPSSResponse_InvalidJSON(t *testing.T) {
	_, err := ParseEPSSResponse([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLookupEPSS(t *testing.T) {
	entries := []EPSSEntry{
		{CVEID: "CVE-2024-1234", EPSS: 0.85, Percentile: 0.97},
		{CVEID: "CVE-2024-9999", EPSS: 0.02, Percentile: 0.30},
	}

	entry, found := LookupEPSS(entries, "CVE-2024-1234")
	if !found {
		t.Fatal("expected to find CVE-2024-1234")
	}
	if entry.EPSS != 0.85 {
		t.Errorf("expected 0.85, got %f", entry.EPSS)
	}

	_, found = LookupEPSS(entries, "CVE-2099-0000")
	if found {
		t.Error("expected not found for non-existent CVE")
	}
}

func TestLookupKEV(t *testing.T) {
	entries := []KEVEntry{
		{CVEID: "CVE-2024-1234", VendorProject: "Acme", Product: "Widget", RansomwareUse: true},
		{CVEID: "CVE-2024-5678", VendorProject: "BigCorp", Product: "Server"},
	}

	entry, found := LookupKEV(entries, "CVE-2024-5678")
	if !found {
		t.Fatal("expected to find CVE-2024-5678")
	}
	if entry.VendorProject != "BigCorp" {
		t.Errorf("expected BigCorp, got %s", entry.VendorProject)
	}

	_, found = LookupKEV(entries, "CVE-2099-0000")
	if found {
		t.Error("expected not found for non-existent CVE")
	}
}

func TestIsHighPriority(t *testing.T) {
	tests := []struct {
		name     string
		kev      *KEVEntry
		epss     *EPSSEntry
		cvss     float64
		expected bool
	}{
		{
			name:     "KEV listed is high priority",
			kev:      &KEVEntry{CVEID: "CVE-2024-1234"},
			epss:     nil,
			cvss:     5.0,
			expected: true,
		},
		{
			name:     "EPSS above 0.5 is high priority",
			kev:      nil,
			epss:     &EPSSEntry{CVEID: "CVE-2024-1234", EPSS: 0.6},
			cvss:     5.0,
			expected: true,
		},
		{
			name:     "CVSS >= 9.0 is high priority",
			kev:      nil,
			epss:     nil,
			cvss:     9.0,
			expected: true,
		},
		{
			name:     "CVSS 9.5 is high priority",
			kev:      nil,
			epss:     nil,
			cvss:     9.5,
			expected: true,
		},
		{
			name:     "low everything is not high priority",
			kev:      nil,
			epss:     &EPSSEntry{CVEID: "CVE-2024-1234", EPSS: 0.1},
			cvss:     4.0,
			expected: false,
		},
		{
			name:     "nil epss and nil kev with low cvss",
			kev:      nil,
			epss:     nil,
			cvss:     3.0,
			expected: false,
		},
		{
			name:     "EPSS exactly 0.5 is not high priority",
			kev:      nil,
			epss:     &EPSSEntry{CVEID: "CVE-2024-1234", EPSS: 0.5},
			cvss:     4.0,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsHighPriority(tt.kev, tt.epss, tt.cvss)
			if result != tt.expected {
				t.Errorf("IsHighPriority() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestThreatFeedFields(t *testing.T) {
	feed := ThreatFeed{
		Name:       "Test",
		URL:        "https://example.com",
		FeedType:   "kev",
		UpdateFreq: 24 * time.Hour,
	}
	if feed.Name != "Test" {
		t.Error("ThreatFeed Name field incorrect")
	}
}

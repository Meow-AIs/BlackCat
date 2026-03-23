package devsecops

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// ThreatFeed defines a threat intelligence feed source.
type ThreatFeed struct {
	Name       string
	URL        string
	FeedType   string // "kev", "epss", "osv"
	UpdateFreq time.Duration
}

// KEVEntry represents a CISA Known Exploited Vulnerability catalog entry.
type KEVEntry struct {
	CVEID            string
	VendorProject    string
	Product          string
	DateAdded        string
	DueDate          string
	RansomwareUse    bool
	ShortDescription string
}

// EPSSEntry represents an Exploit Prediction Scoring System entry.
type EPSSEntry struct {
	CVEID      string
	EPSS       float64 // 0-1.0 probability of exploitation in next 30 days
	Percentile float64
}

// DefaultFeeds contains the standard threat intelligence feed sources.
var DefaultFeeds = []ThreatFeed{
	{
		Name:       "CISA KEV",
		URL:        "https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json",
		FeedType:   "kev",
		UpdateFreq: 24 * time.Hour,
	},
	{
		Name:       "EPSS",
		URL:        "https://api.first.org/data/v1/epss",
		FeedType:   "epss",
		UpdateFreq: 24 * time.Hour,
	},
	{
		Name:       "OSV",
		URL:        "https://api.osv.dev/v1/query",
		FeedType:   "osv",
		UpdateFreq: 6 * time.Hour,
	},
}

// kevFeedJSON is the JSON structure of the CISA KEV catalog.
type kevFeedJSON struct {
	Vulnerabilities []kevVulnJSON `json:"vulnerabilities"`
}

type kevVulnJSON struct {
	CVEID                       string `json:"cveID"`
	VendorProject               string `json:"vendorProject"`
	Product                     string `json:"product"`
	DateAdded                   string `json:"dateAdded"`
	DueDate                     string `json:"dueDate"`
	KnownRansomwareCampaignUse  string `json:"knownRansomwareCampaignUse"`
	ShortDescription            string `json:"shortDescription"`
}

// ParseKEVFeed parses raw JSON from the CISA KEV catalog into KEVEntry slices.
func ParseKEVFeed(data []byte) ([]KEVEntry, error) {
	var feed kevFeedJSON
	if err := json.Unmarshal(data, &feed); err != nil {
		return nil, fmt.Errorf("parsing KEV feed: %w", err)
	}

	entries := make([]KEVEntry, 0, len(feed.Vulnerabilities))
	for _, v := range feed.Vulnerabilities {
		entries = append(entries, KEVEntry{
			CVEID:            v.CVEID,
			VendorProject:    v.VendorProject,
			Product:          v.Product,
			DateAdded:        v.DateAdded,
			DueDate:          v.DueDate,
			RansomwareUse:    v.KnownRansomwareCampaignUse == "Known",
			ShortDescription: v.ShortDescription,
		})
	}
	return entries, nil
}

// epssFeedJSON is the JSON structure of the FIRST EPSS API response.
type epssFeedJSON struct {
	Data []epssDataJSON `json:"data"`
}

type epssDataJSON struct {
	CVE        string `json:"cve"`
	EPSS       string `json:"epss"`
	Percentile string `json:"percentile"`
}

// ParseEPSSResponse parses raw JSON from the FIRST EPSS API into EPSSEntry slices.
func ParseEPSSResponse(data []byte) ([]EPSSEntry, error) {
	var feed epssFeedJSON
	if err := json.Unmarshal(data, &feed); err != nil {
		return nil, fmt.Errorf("parsing EPSS response: %w", err)
	}

	entries := make([]EPSSEntry, 0, len(feed.Data))
	for _, d := range feed.Data {
		epss, err := strconv.ParseFloat(d.EPSS, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing EPSS score for %s: %w", d.CVE, err)
		}
		percentile, err := strconv.ParseFloat(d.Percentile, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing percentile for %s: %w", d.CVE, err)
		}
		entries = append(entries, EPSSEntry{
			CVEID:      d.CVE,
			EPSS:       epss,
			Percentile: percentile,
		})
	}
	return entries, nil
}

// LookupEPSS finds an EPSS entry by CVE ID. Returns the entry and whether it was found.
func LookupEPSS(entries []EPSSEntry, cveID string) (EPSSEntry, bool) {
	for _, e := range entries {
		if e.CVEID == cveID {
			return e, true
		}
	}
	return EPSSEntry{}, false
}

// LookupKEV finds a KEV entry by CVE ID. Returns the entry and whether it was found.
func LookupKEV(entries []KEVEntry, cveID string) (KEVEntry, bool) {
	for _, e := range entries {
		if e.CVEID == cveID {
			return e, true
		}
	}
	return KEVEntry{}, false
}

// IsHighPriority returns true if a vulnerability is high priority based on:
// - Being in the CISA KEV catalog (kev != nil)
// - EPSS score > 0.5
// - CVSS score >= 9.0
func IsHighPriority(kev *KEVEntry, epss *EPSSEntry, cvss float64) bool {
	if kev != nil {
		return true
	}
	if epss != nil && epss.EPSS > 0.5 {
		return true
	}
	if cvss >= 9.0 {
		return true
	}
	return false
}

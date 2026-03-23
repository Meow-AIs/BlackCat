package devsecops

import "math"

// VulnEntry represents a vulnerability with scoring data from multiple sources.
type VulnEntry struct {
	CVEID         string  `json:"cve_id"`
	CVSSScore     float64 `json:"cvss_score"`      // 0-10 (CVSS v4.0)
	EPSSScore     float64 `json:"epss_score"`       // 0-1.0 (30-day exploitation probability)
	KEVListed     bool    `json:"kev_listed"`       // CISA Known Exploited Vulnerabilities
	Ransomware    bool    `json:"ransomware"`       // knownRansomwareCampaignUse from KEV
	Reachable     bool    `json:"reachable"`        // is the vulnerable code path callable?
	AssetCritical bool    `json:"asset_critical"`   // is the affected system business-critical?
	FixAvailable  bool    `json:"fix_available"`    // is a patched version published?
	Description   string  `json:"description"`
	AffectedPkg   string  `json:"affected_package"`
}

// PriorityLevel classifies how urgently a vulnerability needs attention.
type PriorityLevel string

const (
	PriorityP0 PriorityLevel = "P0" // Fix immediately (KEV+ransomware, EPSS>0.5+reachable)
	PriorityP1 PriorityLevel = "P1" // Fix within 24h (KEV or EPSS>0.3)
	PriorityP2 PriorityLevel = "P2" // Fix within 1 week (high CVSS, reachable)
	PriorityP3 PriorityLevel = "P3" // Fix within 30 days (medium severity)
	PriorityP4 PriorityLevel = "P4" // Track/defer (low severity, not reachable)
)

// PriorityResult is the output of vulnerability prioritization.
type PriorityResult struct {
	CVEID    string        `json:"cve_id"`
	Priority PriorityLevel `json:"priority"`
	Score    float64       `json:"risk_score"` // 0-100 composite score
	Reason   string        `json:"reason"`
}

// PrioritizeVuln computes a risk score and priority level using the
// EPSS+KEV+CVSS+reachability stack from the IMPROVEMENT-PLAN.
//
// Priority stack (NOT CVSS alone):
// 1. CISA KEV listed (knownRansomwareCampaignUse = instant P0)
// 2. EPSS v4 probability > 0.5 (high exploitation likelihood in 30 days)
// 3. CVSS v4.0 score (severity baseline)
// 4. Reachability analysis (is the vulnerable code path actually callable?)
// 5. Asset criticality (business impact of affected system)
// 6. Fix availability (is a patched version published?)
func PrioritizeVuln(v VulnEntry) PriorityResult {
	score := computeRiskScore(v)
	priority, reason := classifyPriority(v, score)

	return PriorityResult{
		CVEID:    v.CVEID,
		Priority: priority,
		Score:    math.Round(score*100) / 100,
		Reason:   reason,
	}
}

func computeRiskScore(v VulnEntry) float64 {
	// Base: CVSS normalized to 0-40 range
	score := v.CVSSScore * 4.0

	// EPSS multiplier: 0-30 points
	score += v.EPSSScore * 30.0

	// KEV: +15 points
	if v.KEVListed {
		score += 15.0
	}

	// Ransomware: +10 points
	if v.Ransomware {
		score += 10.0
	}

	// Reachability: +10 points if reachable, -10 if not
	if v.Reachable {
		score += 10.0
	} else {
		score -= 10.0
	}

	// Asset criticality: +5 points
	if v.AssetCritical {
		score += 5.0
	}

	// Fix available: -5 points (lowers urgency since fix is ready)
	// This is counter-intuitive but: fix available means it's easier to remediate
	// so the "risk" is slightly lower. The priority should still be high.
	if v.FixAvailable {
		score -= 0 // no penalty — fix being available doesn't reduce risk, just effort
	}

	// Clamp to 0-100
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return score
}

func classifyPriority(v VulnEntry, score float64) (PriorityLevel, string) {
	// P0: KEV + ransomware, or EPSS > 0.5 + reachable
	if v.Ransomware && v.KEVListed {
		return PriorityP0, "KEV-listed with known ransomware campaign use"
	}
	if v.EPSSScore > 0.5 && v.Reachable {
		return PriorityP0, "high exploitation probability (EPSS > 0.5) and reachable code path"
	}

	// P1: KEV listed, or EPSS > 0.3
	if v.KEVListed {
		return PriorityP1, "CISA KEV listed — actively exploited in the wild"
	}
	if v.EPSSScore > 0.3 {
		return PriorityP1, "elevated exploitation probability (EPSS > 0.3)"
	}

	// P2: CVSS >= 7.0 and reachable
	if v.CVSSScore >= 7.0 && v.Reachable {
		return PriorityP2, "high CVSS score with reachable code path"
	}

	// P3: score >= 30
	if score >= 30 {
		return PriorityP3, "medium overall risk score"
	}

	// P4: everything else
	return PriorityP4, "low risk — track for future remediation"
}

// PrioritizeAll sorts vulnerabilities by risk score descending.
func PrioritizeAll(vulns []VulnEntry) []PriorityResult {
	results := make([]PriorityResult, len(vulns))
	for i, v := range vulns {
		results[i] = PrioritizeVuln(v)
	}
	// Sort by score descending
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
	return results
}

package devsecops

import "testing"

// ---------------------------------------------------------------------------
// P0 — Critical: KEV + Ransomware
// ---------------------------------------------------------------------------

func TestPrioritizeVuln_P0_KEVRansomware(t *testing.T) {
	v := VulnEntry{
		CVEID:     "CVE-2024-1234",
		CVSSScore: 9.8,
		EPSSScore: 0.95,
		KEVListed: true,
		Ransomware: true,
		Reachable:  true,
	}
	r := PrioritizeVuln(v)
	if r.Priority != PriorityP0 {
		t.Errorf("expected P0, got %q", r.Priority)
	}
	if r.Score <= 80 {
		t.Errorf("expected high score for KEV+ransomware, got %.2f", r.Score)
	}
}

func TestPrioritizeVuln_P0_HighEPSSReachable(t *testing.T) {
	v := VulnEntry{
		CVEID:     "CVE-2024-5678",
		CVSSScore: 7.5,
		EPSSScore: 0.85,
		Reachable: true,
	}
	r := PrioritizeVuln(v)
	if r.Priority != PriorityP0 {
		t.Errorf("expected P0 for EPSS>0.5+reachable, got %q", r.Priority)
	}
}

// ---------------------------------------------------------------------------
// P1 — KEV listed (no ransomware) or EPSS > 0.3
// ---------------------------------------------------------------------------

func TestPrioritizeVuln_P1_KEVOnly(t *testing.T) {
	v := VulnEntry{
		CVEID:     "CVE-2024-1111",
		CVSSScore: 7.0,
		EPSSScore: 0.2,
		KEVListed: true,
	}
	r := PrioritizeVuln(v)
	if r.Priority != PriorityP1 {
		t.Errorf("expected P1 for KEV-only, got %q", r.Priority)
	}
}

func TestPrioritizeVuln_P1_HighEPSS(t *testing.T) {
	v := VulnEntry{
		CVEID:     "CVE-2024-2222",
		CVSSScore: 6.0,
		EPSSScore: 0.35,
	}
	r := PrioritizeVuln(v)
	if r.Priority != PriorityP1 {
		t.Errorf("expected P1 for EPSS>0.3, got %q", r.Priority)
	}
}

// ---------------------------------------------------------------------------
// P2 — High CVSS + Reachable
// ---------------------------------------------------------------------------

func TestPrioritizeVuln_P2_HighCVSSReachable(t *testing.T) {
	v := VulnEntry{
		CVEID:     "CVE-2024-3333",
		CVSSScore: 8.0,
		EPSSScore: 0.1,
		Reachable: true,
	}
	r := PrioritizeVuln(v)
	if r.Priority != PriorityP2 {
		t.Errorf("expected P2 for high CVSS+reachable, got %q", r.Priority)
	}
}

// ---------------------------------------------------------------------------
// P3 — Medium risk
// ---------------------------------------------------------------------------

func TestPrioritizeVuln_P3_MediumRisk(t *testing.T) {
	v := VulnEntry{
		CVEID:     "CVE-2024-4444",
		CVSSScore: 6.5,
		EPSSScore: 0.15,
		Reachable: true,
	}
	r := PrioritizeVuln(v)
	if r.Priority != PriorityP3 {
		t.Errorf("expected P3 for medium risk, got %q (score=%.2f)", r.Priority, r.Score)
	}
}

// ---------------------------------------------------------------------------
// P4 — Low risk
// ---------------------------------------------------------------------------

func TestPrioritizeVuln_P4_LowRisk(t *testing.T) {
	v := VulnEntry{
		CVEID:     "CVE-2024-5555",
		CVSSScore: 3.0,
		EPSSScore: 0.001,
		Reachable: false,
	}
	r := PrioritizeVuln(v)
	if r.Priority != PriorityP4 {
		t.Errorf("expected P4 for low risk, got %q (score=%.2f)", r.Priority, r.Score)
	}
}

// ---------------------------------------------------------------------------
// Key insight: CVSS 9.8 with EPSS 0.001 < CVSS 7.2 with EPSS 0.85
// ---------------------------------------------------------------------------

func TestPrioritizeVuln_EPSSBeatsHighCVSS(t *testing.T) {
	highCVSS := VulnEntry{
		CVEID:     "CVE-HIGH-CVSS",
		CVSSScore: 9.8,
		EPSSScore: 0.001,
		Reachable: false,
	}
	highEPSS := VulnEntry{
		CVEID:     "CVE-HIGH-EPSS",
		CVSSScore: 7.2,
		EPSSScore: 0.85,
		KEVListed: true,
		Reachable: true,
	}

	rHigh := PrioritizeVuln(highCVSS)
	rEPSS := PrioritizeVuln(highEPSS)

	if rEPSS.Score <= rHigh.Score {
		t.Errorf("EPSS-high vuln (score=%.2f) should outscore CVSS-high vuln (score=%.2f)",
			rEPSS.Score, rHigh.Score)
	}

	// EPSS one should be P0 or P1, CVSS one should be lower priority
	if rEPSS.Priority == PriorityP4 {
		t.Error("EPSS-high vuln should not be P4")
	}
}

// ---------------------------------------------------------------------------
// PrioritizeAll — sorts by score descending
// ---------------------------------------------------------------------------

func TestPrioritizeAll_SortedByScore(t *testing.T) {
	vulns := []VulnEntry{
		{CVEID: "LOW", CVSSScore: 2.0, EPSSScore: 0.001},
		{CVEID: "HIGH", CVSSScore: 9.8, EPSSScore: 0.9, KEVListed: true, Ransomware: true, Reachable: true},
		{CVEID: "MED", CVSSScore: 5.0, EPSSScore: 0.1, Reachable: true},
	}

	results := PrioritizeAll(vulns)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	if results[0].CVEID != "HIGH" {
		t.Errorf("expected HIGH first, got %q", results[0].CVEID)
	}
	if results[2].CVEID != "LOW" {
		t.Errorf("expected LOW last, got %q", results[2].CVEID)
	}

	for i := 0; i < len(results)-1; i++ {
		if results[i].Score < results[i+1].Score {
			t.Errorf("results not sorted: score[%d]=%.2f < score[%d]=%.2f",
				i, results[i].Score, i+1, results[i+1].Score)
		}
	}
}

// ---------------------------------------------------------------------------
// Reason field
// ---------------------------------------------------------------------------

func TestPrioritizeVuln_HasReason(t *testing.T) {
	v := VulnEntry{CVEID: "CVE-TEST", CVSSScore: 5.0}
	r := PrioritizeVuln(v)
	if r.Reason == "" {
		t.Error("expected non-empty reason")
	}
}

// ---------------------------------------------------------------------------
// Score clamping
// ---------------------------------------------------------------------------

func TestPrioritizeVuln_ScoreClamped_0_100(t *testing.T) {
	// Max possible
	v := VulnEntry{
		CVSSScore: 10.0, EPSSScore: 1.0,
		KEVListed: true, Ransomware: true,
		Reachable: true, AssetCritical: true,
	}
	r := PrioritizeVuln(v)
	if r.Score > 100 {
		t.Errorf("score %.2f exceeds 100", r.Score)
	}

	// Min possible
	v2 := VulnEntry{CVSSScore: 0, EPSSScore: 0, Reachable: false}
	r2 := PrioritizeVuln(v2)
	if r2.Score < 0 {
		t.Errorf("score %.2f below 0", r2.Score)
	}
}

package architect

import (
	"fmt"
	"strings"
	"time"
)

// SLATier represents a service level availability target.
type SLATier string

const (
	SLATier999   SLATier = "99.9"   // 8.76h downtime/year
	SLATier9999  SLATier = "99.99"  // 52.6min downtime/year
	SLATier99999 SLATier = "99.999" // 5.26min downtime/year
)

// SLARequirements describes the availability and performance targets.
type SLARequirements struct {
	Availability  SLATier
	RPO           time.Duration // Recovery Point Objective
	RTO           time.Duration // Recovery Time Objective
	LatencyP99    time.Duration // P99 latency target
	ThroughputRPS int
}

// InfraRecommendation holds the infrastructure requirements for an SLA tier.
type InfraRecommendation struct {
	Tier             SLATier
	MinInstances     int
	MinAZs           int  // availability zones
	NeedsMultiRegion bool
	DatabaseHA       string // "single", "read-replica", "multi-az", "multi-region"
	LoadBalancer     string // "alb", "nlb", "global"
	CacheLayer       bool
	CDN              bool
	BackupFrequency  time.Duration
	MonitoringLevel  string // "basic", "enhanced", "full"
	EstMonthlyCost   string // "low", "medium", "high", "very-high"
	Notes            []string
}

const hoursPerYear = 365 * 24

// tierFraction maps SLATier to its allowed-downtime fraction.
func tierFraction(tier SLATier) float64 {
	switch tier {
	case SLATier999:
		return 0.001
	case SLATier9999:
		return 0.0001
	case SLATier99999:
		return 0.00001
	default:
		return 0.001
	}
}

// CalculateDowntime returns the annual allowed downtime for the given tier.
func CalculateDowntime(tier SLATier) time.Duration {
	fraction := tierFraction(tier)
	yearDuration := float64(hoursPerYear) * float64(time.Hour)
	return time.Duration(yearDuration * fraction)
}

// ErrorBudget returns the allowed downtime for a given number of days.
func ErrorBudget(tier SLATier, periodDays int) time.Duration {
	if periodDays <= 0 {
		return 0
	}
	annual := CalculateDowntime(tier)
	return time.Duration(float64(annual) * float64(periodDays) / 365.0)
}

// MapSLAToInfra produces an infrastructure recommendation from SLA requirements.
func MapSLAToInfra(req SLARequirements) InfraRecommendation {
	rec := baseRecommendation(req.Availability)
	applyThroughputOverrides(&rec, req.ThroughputRPS)
	applyLatencyOverrides(&rec, req.LatencyP99)
	rec.Notes = buildSLANotes(req, rec)
	return rec
}

func baseRecommendation(tier SLATier) InfraRecommendation {
	switch tier {
	case SLATier99999:
		return InfraRecommendation{
			Tier:             SLATier99999,
			MinInstances:     5,
			MinAZs:           3,
			NeedsMultiRegion: true,
			DatabaseHA:       "multi-region",
			LoadBalancer:     "global",
			CacheLayer:       true,
			CDN:              true,
			BackupFrequency:  15 * time.Minute,
			MonitoringLevel:  "full",
			EstMonthlyCost:   "very-high",
		}
	case SLATier9999:
		return InfraRecommendation{
			Tier:             SLATier9999,
			MinInstances:     3,
			MinAZs:           3,
			NeedsMultiRegion: false,
			DatabaseHA:       "multi-az",
			LoadBalancer:     "nlb",
			CacheLayer:       true,
			CDN:              false,
			BackupFrequency:  time.Hour,
			MonitoringLevel:  "enhanced",
			EstMonthlyCost:   "high",
		}
	default: // SLATier999
		return InfraRecommendation{
			Tier:             SLATier999,
			MinInstances:     2,
			MinAZs:           2,
			NeedsMultiRegion: false,
			DatabaseHA:       "read-replica",
			LoadBalancer:     "alb",
			CacheLayer:       false,
			CDN:              false,
			BackupFrequency:  24 * time.Hour,
			MonitoringLevel:  "basic",
			EstMonthlyCost:   "medium",
		}
	}
}

func applyThroughputOverrides(rec *InfraRecommendation, rps int) {
	if rps >= 10000 {
		rec.CacheLayer = true
		rec.CDN = true
	} else if rps >= 5000 {
		rec.CacheLayer = true
	}
}

func applyLatencyOverrides(rec *InfraRecommendation, p99 time.Duration) {
	if p99 > 0 && p99 <= 50*time.Millisecond {
		rec.CacheLayer = true
	}
}

func buildSLANotes(req SLARequirements, rec InfraRecommendation) []string {
	var notes []string

	if req.ThroughputRPS > 10000 {
		notes = append(notes, "High throughput: consider horizontal auto-scaling")
	}
	if req.LatencyP99 > 0 && req.LatencyP99 <= 50*time.Millisecond {
		notes = append(notes, "Strict latency target: place compute close to users")
	}
	if req.RPO < 5*time.Minute {
		notes = append(notes, "Tight RPO: use synchronous replication or continuous backup")
	}
	if req.RTO < 15*time.Minute {
		notes = append(notes, "Tight RTO: pre-provision standby instances")
	}
	if rec.NeedsMultiRegion {
		notes = append(notes, "Multi-region deployment requires data residency review")
	}

	return notes
}

// FormatMarkdown renders the recommendation as markdown.
func (r InfraRecommendation) FormatMarkdown() string {
	var b strings.Builder

	b.WriteString("# SLA Infrastructure Recommendation\n\n")
	b.WriteString(fmt.Sprintf("**SLA Tier**: %s%%\n\n", string(r.Tier)))

	b.WriteString("## Compute\n\n")
	b.WriteString(fmt.Sprintf("| Metric | Value |\n"))
	b.WriteString("|--------|-------|\n")
	b.WriteString(fmt.Sprintf("| Min Instances | %d |\n", r.MinInstances))
	b.WriteString(fmt.Sprintf("| Min AZs | %d |\n", r.MinAZs))
	b.WriteString(fmt.Sprintf("| Multi-Region | %v |\n", r.NeedsMultiRegion))

	b.WriteString("\n## Database\n\n")
	b.WriteString(fmt.Sprintf("| Metric | Value |\n"))
	b.WriteString("|--------|-------|\n")
	b.WriteString(fmt.Sprintf("| Database HA | %s |\n", r.DatabaseHA))
	b.WriteString(fmt.Sprintf("| Backup Frequency | %s |\n", r.BackupFrequency))

	b.WriteString("\n## Networking\n\n")
	b.WriteString(fmt.Sprintf("| Metric | Value |\n"))
	b.WriteString("|--------|-------|\n")
	b.WriteString(fmt.Sprintf("| Load Balancer | %s |\n", r.LoadBalancer))
	b.WriteString(fmt.Sprintf("| Cache Layer | %v |\n", r.CacheLayer))
	b.WriteString(fmt.Sprintf("| CDN | %v |\n", r.CDN))

	b.WriteString("\n## Monitoring\n\n")
	b.WriteString(fmt.Sprintf("| Metric | Value |\n"))
	b.WriteString("|--------|-------|\n")
	b.WriteString(fmt.Sprintf("| Monitoring Level | %s |\n", r.MonitoringLevel))
	b.WriteString(fmt.Sprintf("| Est. Monthly Cost | %s |\n", r.EstMonthlyCost))

	if len(r.Notes) > 0 {
		b.WriteString("\n## Notes\n\n")
		for _, n := range r.Notes {
			b.WriteString(fmt.Sprintf("- %s\n", n))
		}
	}

	return b.String()
}

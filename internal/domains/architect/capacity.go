package architect

import (
	"fmt"
	"math"
	"strings"
)

// CapacityInput describes the parameters for capacity estimation.
type CapacityInput struct {
	DailyActiveUsers     int
	RequestsPerUser      int
	AvgPayloadBytes      int
	PeakMultiplier       float64
	GrowthRateMonthly    float64
	PlanningMonths       int
	ReplicationFactor    int
	StorageRetentionDays int
}

// CapacityEstimate holds the computed capacity projections.
type CapacityEstimate struct {
	RPSAverage           float64
	RPSPeak              float64
	BandwidthMbps        float64
	StorageGB            float64
	StorageGBMonthly     []float64
	RecommendedInstances int
	Notes                []string
}

const (
	secondsPerDay  = 86400.0
	bytesPerGB     = 1024.0 * 1024.0 * 1024.0
	bytesPerMbit   = 125000.0
	rpsPerInstance = 1000.0 // assumed capacity per instance
)

// EstimateCapacity computes capacity requirements from the given input.
func EstimateCapacity(input CapacityInput) CapacityEstimate {
	replication := input.ReplicationFactor
	if replication < 1 {
		replication = 1
	}
	peak := input.PeakMultiplier
	if peak < 1.0 {
		peak = 1.0
	}
	months := input.PlanningMonths
	if months < 1 {
		months = 1
	}

	dailyRequests := float64(input.DailyActiveUsers) * float64(input.RequestsPerUser)
	rpsAvg := dailyRequests / secondsPerDay
	rpsPeak := rpsAvg * peak

	// Bandwidth at peak (bits per second -> Mbps)
	bytesPerSecondPeak := rpsPeak * float64(input.AvgPayloadBytes)
	bandwidthMbps := (bytesPerSecondPeak * 8.0) / 1_000_000.0

	// Storage: daily bytes * retention * replication
	dailyBytes := dailyRequests * float64(input.AvgPayloadBytes)
	baseStorageGB := (dailyBytes * float64(input.StorageRetentionDays) * float64(replication)) / bytesPerGB

	// Monthly projections with growth
	monthlyStorage := make([]float64, months)
	for m := 0; m < months; m++ {
		growthFactor := math.Pow(1.0+input.GrowthRateMonthly, float64(m))
		monthlyStorage[m] = baseStorageGB * growthFactor
	}

	// Final storage is at end of planning horizon
	finalStorageGB := monthlyStorage[months-1]

	// Recommended instances based on peak RPS
	instances := int(math.Ceil(rpsPeak / rpsPerInstance))
	if instances < 1 {
		instances = 1
	}

	// Advisory notes
	notes := buildNotes(rpsPeak, finalStorageGB, bandwidthMbps, input)

	return CapacityEstimate{
		RPSAverage:           rpsAvg,
		RPSPeak:              rpsPeak,
		BandwidthMbps:        bandwidthMbps,
		StorageGB:            finalStorageGB,
		StorageGBMonthly:     monthlyStorage,
		RecommendedInstances: instances,
		Notes:                notes,
	}
}

func buildNotes(rpsPeak, storageGB, bandwidthMbps float64, input CapacityInput) []string {
	var notes []string

	if rpsPeak > 10000 {
		notes = append(notes, "High RPS: consider load balancer with auto-scaling")
	}
	if rpsPeak > 50000 {
		notes = append(notes, "Very high RPS: evaluate CDN and edge caching")
	}
	if storageGB > 1000 {
		notes = append(notes, "Large storage: consider tiered storage (hot/warm/cold)")
	}
	if storageGB > 10000 {
		notes = append(notes, "Massive storage: evaluate data lake architecture")
	}
	if bandwidthMbps > 1000 {
		notes = append(notes, "High bandwidth: ensure network capacity and consider compression")
	}
	if input.GrowthRateMonthly > 0.15 {
		notes = append(notes, "Rapid growth: plan for 2x capacity headroom")
	}
	if input.PeakMultiplier > 5.0 {
		notes = append(notes, "High peak multiplier: consider queue-based load leveling")
	}
	if input.ReplicationFactor > 3 {
		notes = append(notes, "High replication: verify consistency requirements justify the cost")
	}
	if input.StorageRetentionDays > 365 {
		notes = append(notes, "Long retention: implement data lifecycle policies")
	}

	return notes
}

// FormatMarkdown renders the capacity estimate as markdown.
func (c CapacityEstimate) FormatMarkdown() string {
	var b strings.Builder

	b.WriteString("# Capacity Estimate\n\n")

	b.WriteString("## Traffic\n\n")
	b.WriteString(fmt.Sprintf("| Metric | Value |\n"))
	b.WriteString(fmt.Sprintf("|--------|-------|\n"))
	b.WriteString(fmt.Sprintf("| RPS (Average) | %.2f |\n", c.RPSAverage))
	b.WriteString(fmt.Sprintf("| RPS (Peak) | %.2f |\n", c.RPSPeak))
	b.WriteString(fmt.Sprintf("| Bandwidth (Peak) | %.2f Mbps |\n", c.BandwidthMbps))

	b.WriteString("\n## Storage\n\n")
	b.WriteString(fmt.Sprintf("| Metric | Value |\n"))
	b.WriteString(fmt.Sprintf("|--------|-------|\n"))
	b.WriteString(fmt.Sprintf("| Storage (Final) | %.2f GB |\n", c.StorageGB))

	if len(c.StorageGBMonthly) > 1 {
		b.WriteString("\n### Monthly Storage Projection\n\n")
		b.WriteString("| Month | Storage (GB) |\n")
		b.WriteString("|-------|--------------|\n")
		for i, gb := range c.StorageGBMonthly {
			b.WriteString(fmt.Sprintf("| %d | %.2f |\n", i+1, gb))
		}
	}

	b.WriteString(fmt.Sprintf("\n## Instances\n\n"))
	b.WriteString(fmt.Sprintf("Recommended Instances: **%d**\n", c.RecommendedInstances))

	if len(c.Notes) > 0 {
		b.WriteString("\n## Notes\n\n")
		for _, n := range c.Notes {
			b.WriteString(fmt.Sprintf("- %s\n", n))
		}
	}

	return b.String()
}

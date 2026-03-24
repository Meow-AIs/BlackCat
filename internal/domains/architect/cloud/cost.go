package cloud

import (
	"fmt"
	"strings"
)

// CostEstimateInput describes the infrastructure to estimate costs for.
type CostEstimateInput struct {
	Provider       CloudProvider
	ComputeUnits   int
	ComputeType    string // "small", "medium", "large", "xlarge"
	StorageGB      float64
	TransferOutGB  float64
	DatabaseType   string // "rds-small", "rds-medium", "rds-large"
	Region         string
	ReservedTerm   int     // 0=on-demand, 1=1yr, 3=3yr
	SpotPercentage float64 // 0-1
}

// CostEstimate holds the estimated costs for a cloud deployment.
type CostEstimate struct {
	Provider          CloudProvider
	MonthlyCompute    float64
	MonthlyStorage    float64
	MonthlyTransfer   float64
	MonthlyDatabase   float64
	MonthlyTotal      float64
	YearlyTotal       float64
	SavingsVsOnDemand float64 // percentage 0-100
	Notes             []string
}

// providerRates holds per-provider base pricing multipliers.
type providerRates struct {
	computeHourly map[string]float64 // per instance per hour
	storagePerGB  float64            // per GB per month
	transferPerGB float64            // egress per GB
	dbMonthly     map[string]float64 // per month
}

var rates = map[CloudProvider]providerRates{
	AWS: {
		computeHourly: map[string]float64{
			"small": 0.023, "medium": 0.046, "large": 0.092, "xlarge": 0.184,
		},
		storagePerGB:  0.023,
		transferPerGB: 0.09,
		dbMonthly: map[string]float64{
			"rds-small": 25.0, "rds-medium": 100.0, "rds-large": 400.0,
		},
	},
	GCP: {
		computeHourly: map[string]float64{
			"small": 0.021, "medium": 0.042, "large": 0.084, "xlarge": 0.168,
		},
		storagePerGB:  0.020,
		transferPerGB: 0.12,
		dbMonthly: map[string]float64{
			"rds-small": 27.0, "rds-medium": 105.0, "rds-large": 420.0,
		},
	},
	Azure: {
		computeHourly: map[string]float64{
			"small": 0.024, "medium": 0.048, "large": 0.096, "xlarge": 0.192,
		},
		storagePerGB:  0.021,
		transferPerGB: 0.087,
		dbMonthly: map[string]float64{
			"rds-small": 28.0, "rds-medium": 110.0, "rds-large": 430.0,
		},
	},
	Alibaba: {
		computeHourly: map[string]float64{
			"small": 0.018, "medium": 0.036, "large": 0.072, "xlarge": 0.144,
		},
		storagePerGB:  0.017,
		transferPerGB: 0.08,
		dbMonthly: map[string]float64{
			"rds-small": 20.0, "rds-medium": 80.0, "rds-large": 320.0,
		},
	},
}

const hoursPerMonth = 730.0

// reservedDiscounts maps term length to discount factor.
var reservedDiscounts = map[int]float64{
	0: 1.0,  // on-demand
	1: 0.60, // 1-year reserved ~40% off
	3: 0.40, // 3-year reserved ~60% off
}

const spotDiscount = 0.35 // spot is ~65% cheaper

// EstimateCloudCost produces a cost estimate for one provider.
func EstimateCloudCost(input CostEstimateInput) CostEstimate {
	r, ok := rates[input.Provider]
	if !ok {
		return CostEstimate{Provider: input.Provider, Notes: []string{"Unknown provider"}}
	}

	// Compute cost
	hourlyRate := r.computeHourly[input.ComputeType]
	onDemandFraction := 1.0 - input.SpotPercentage
	spotFraction := input.SpotPercentage

	perInstanceMonthly := hourlyRate * hoursPerMonth
	onDemandCost := float64(input.ComputeUnits) * perInstanceMonthly * onDemandFraction
	spotCost := float64(input.ComputeUnits) * perInstanceMonthly * spotFraction * spotDiscount
	computeRaw := onDemandCost + spotCost

	// Apply reserved discount to on-demand portion
	discount := reservedDiscounts[input.ReservedTerm]
	if discount == 0 {
		discount = 1.0
	}
	monthlyCompute := computeRaw * discount

	// Storage
	monthlyStorage := input.StorageGB * r.storagePerGB

	// Transfer
	monthlyTransfer := input.TransferOutGB * r.transferPerGB

	// Database
	monthlyDB := r.dbMonthly[input.DatabaseType] * discount

	total := monthlyCompute + monthlyStorage + monthlyTransfer + monthlyDB

	// Calculate on-demand total for savings comparison
	onDemandTotal := (float64(input.ComputeUnits)*perInstanceMonthly +
		monthlyStorage + input.TransferOutGB*r.transferPerGB +
		r.dbMonthly[input.DatabaseType])

	var savings float64
	if onDemandTotal > 0 && total < onDemandTotal {
		savings = ((onDemandTotal - total) / onDemandTotal) * 100
	}

	notes := buildNotes(input)

	return CostEstimate{
		Provider:          input.Provider,
		MonthlyCompute:    monthlyCompute,
		MonthlyStorage:    monthlyStorage,
		MonthlyTransfer:   monthlyTransfer,
		MonthlyDatabase:   monthlyDB,
		MonthlyTotal:      total,
		YearlyTotal:       total * 12,
		SavingsVsOnDemand: savings,
		Notes:             notes,
	}
}

// buildNotes generates descriptive notes for the estimate.
func buildNotes(input CostEstimateInput) []string {
	var notes []string
	switch input.ReservedTerm {
	case 0:
		notes = append(notes, "On-demand pricing")
	case 1:
		notes = append(notes, "1-year reserved instance pricing (~40% savings)")
	case 3:
		notes = append(notes, "3-year reserved instance pricing (~60% savings)")
	}
	if input.SpotPercentage > 0 {
		notes = append(notes, fmt.Sprintf("%.0f%% spot instances (~65%% cheaper)", input.SpotPercentage*100))
	}
	if input.Region != "" {
		notes = append(notes, "Region: "+input.Region)
	}
	return notes
}

// CompareCloudCosts estimates costs across multiple providers/configs.
func CompareCloudCosts(inputs []CostEstimateInput) []CostEstimate {
	results := make([]CostEstimate, 0, len(inputs))
	for _, input := range inputs {
		results = append(results, EstimateCloudCost(input))
	}
	return results
}

// FormatMarkdown renders the cost estimate as a markdown table.
func (c CostEstimate) FormatMarkdown() string {
	md := fmt.Sprintf("## %s Cost Estimate\n\n", strings.ToUpper(string(c.Provider)))
	md += "| Category | Monthly |\n|----------|--------|\n"
	md += fmt.Sprintf("| Compute | $%.2f |\n", c.MonthlyCompute)
	md += fmt.Sprintf("| Storage | $%.2f |\n", c.MonthlyStorage)
	md += fmt.Sprintf("| Transfer | $%.2f |\n", c.MonthlyTransfer)
	md += fmt.Sprintf("| Database | $%.2f |\n", c.MonthlyDatabase)
	md += fmt.Sprintf("| **Total** | **$%.2f** |\n", c.MonthlyTotal)
	md += fmt.Sprintf("\n**Yearly**: $%.2f\n", c.YearlyTotal)
	if c.SavingsVsOnDemand > 0 {
		md += fmt.Sprintf("\n**Savings vs On-Demand**: %.1f%%\n", c.SavingsVsOnDemand)
	}
	if len(c.Notes) > 0 {
		md += "\n**Notes**:\n"
		for _, n := range c.Notes {
			md += "- " + n + "\n"
		}
	}
	return md
}

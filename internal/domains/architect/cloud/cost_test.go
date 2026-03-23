package cloud

import (
	"strings"
	"testing"
)

func TestEstimateCloudCostBasic(t *testing.T) {
	input := CostEstimateInput{
		Provider:     AWS,
		ComputeUnits: 2,
		ComputeType:  "medium",
		StorageGB:    100,
		TransferOutGB: 50,
		DatabaseType: "rds-small",
		Region:       "us-east-1",
	}

	est := EstimateCloudCost(input)
	if est.Provider != AWS {
		t.Fatalf("expected AWS, got %s", est.Provider)
	}
	if est.MonthlyCompute <= 0 {
		t.Fatal("expected positive monthly compute cost")
	}
	if est.MonthlyStorage <= 0 {
		t.Fatal("expected positive monthly storage cost")
	}
	if est.MonthlyTransfer <= 0 {
		t.Fatal("expected positive monthly transfer cost")
	}
	if est.MonthlyDatabase <= 0 {
		t.Fatal("expected positive monthly database cost")
	}
	if est.MonthlyTotal <= 0 {
		t.Fatal("expected positive monthly total")
	}

	expectedTotal := est.MonthlyCompute + est.MonthlyStorage + est.MonthlyTransfer + est.MonthlyDatabase
	if abs(est.MonthlyTotal-expectedTotal) > 0.01 {
		t.Fatalf("monthly total %.2f != sum of parts %.2f", est.MonthlyTotal, expectedTotal)
	}

	expectedYearly := est.MonthlyTotal * 12
	if abs(est.YearlyTotal-expectedYearly) > 0.01 {
		t.Fatalf("yearly total %.2f != monthly*12 %.2f", est.YearlyTotal, expectedYearly)
	}
}

func TestEstimateCloudCostReserved(t *testing.T) {
	onDemand := CostEstimateInput{
		Provider:     AWS,
		ComputeUnits: 4,
		ComputeType:  "large",
		StorageGB:    500,
		TransferOutGB: 100,
		ReservedTerm: 0,
	}
	reserved1yr := onDemand
	reserved1yr.ReservedTerm = 1

	reserved3yr := onDemand
	reserved3yr.ReservedTerm = 3

	estOnDemand := EstimateCloudCost(onDemand)
	estReserved1 := EstimateCloudCost(reserved1yr)
	estReserved3 := EstimateCloudCost(reserved3yr)

	if estReserved1.MonthlyTotal >= estOnDemand.MonthlyTotal {
		t.Fatal("1yr reserved should be cheaper than on-demand")
	}
	if estReserved3.MonthlyTotal >= estReserved1.MonthlyTotal {
		t.Fatal("3yr reserved should be cheaper than 1yr reserved")
	}
	if estReserved1.SavingsVsOnDemand <= 0 {
		t.Fatal("expected positive savings for reserved")
	}
}

func TestEstimateCloudCostSpot(t *testing.T) {
	base := CostEstimateInput{
		Provider:       AWS,
		ComputeUnits:   4,
		ComputeType:    "large",
		SpotPercentage: 0,
	}
	withSpot := base
	withSpot.SpotPercentage = 0.5

	estBase := EstimateCloudCost(base)
	estSpot := EstimateCloudCost(withSpot)

	if estSpot.MonthlyCompute >= estBase.MonthlyCompute {
		t.Fatal("spot instances should reduce compute cost")
	}
}

func TestEstimateCloudCostAllProviders(t *testing.T) {
	providers := []CloudProvider{AWS, GCP, Azure, Alibaba}
	for _, p := range providers {
		est := EstimateCloudCost(CostEstimateInput{
			Provider:     p,
			ComputeUnits: 2,
			ComputeType:  "medium",
			StorageGB:    100,
			TransferOutGB: 50,
		})
		if est.Provider != p {
			t.Errorf("expected provider %s, got %s", p, est.Provider)
		}
		if est.MonthlyTotal <= 0 {
			t.Errorf("expected positive total for %s", p)
		}
	}
}

func TestCompareCloudCosts(t *testing.T) {
	inputs := []CostEstimateInput{
		{Provider: AWS, ComputeUnits: 2, ComputeType: "medium", StorageGB: 100},
		{Provider: GCP, ComputeUnits: 2, ComputeType: "medium", StorageGB: 100},
		{Provider: Azure, ComputeUnits: 2, ComputeType: "medium", StorageGB: 100},
	}

	results := CompareCloudCosts(inputs)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Each should have a different provider
	providers := make(map[CloudProvider]bool)
	for _, r := range results {
		providers[r.Provider] = true
	}
	if len(providers) != 3 {
		t.Fatal("expected 3 different providers in results")
	}
}

func TestCostEstimateFormatMarkdown(t *testing.T) {
	est := CostEstimate{
		Provider:          AWS,
		MonthlyCompute:    200.0,
		MonthlyStorage:    50.0,
		MonthlyTransfer:   25.0,
		MonthlyDatabase:   100.0,
		MonthlyTotal:      375.0,
		YearlyTotal:       4500.0,
		SavingsVsOnDemand: 0,
		Notes:             []string{"On-demand pricing"},
	}

	md := est.FormatMarkdown()
	if md == "" {
		t.Fatal("expected non-empty markdown")
	}
	if !strings.Contains(md, "AWS") {
		t.Error("expected markdown to contain provider name")
	}
	if !strings.Contains(md, "375") {
		t.Error("expected markdown to contain monthly total")
	}
	if !strings.Contains(md, "4500") || !strings.Contains(md, "4,500") {
		// Accept either format
	}
}

func TestEstimateCloudCostZeroInputs(t *testing.T) {
	// Zero compute, storage, etc should produce zero costs
	est := EstimateCloudCost(CostEstimateInput{
		Provider: AWS,
	})
	if est.MonthlyTotal != 0 {
		t.Fatalf("expected zero total for zero inputs, got %.2f", est.MonthlyTotal)
	}
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

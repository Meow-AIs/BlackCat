package architect

import (
	"math"
	"strings"
	"testing"
)

func TestEstimateCapacity_BasicRPS(t *testing.T) {
	input := CapacityInput{
		DailyActiveUsers:     10000,
		RequestsPerUser:      100,
		AvgPayloadBytes:      1024,
		PeakMultiplier:       3.0,
		GrowthRateMonthly:    0.0,
		PlanningMonths:       1,
		ReplicationFactor:    1,
		StorageRetentionDays: 30,
	}
	est := EstimateCapacity(input)

	// 10000 * 100 = 1,000,000 requests/day = ~11.57 RPS
	expectedRPS := 1000000.0 / 86400.0
	if math.Abs(est.RPSAverage-expectedRPS) > 0.5 {
		t.Errorf("expected RPS ~%.2f, got %.2f", expectedRPS, est.RPSAverage)
	}

	expectedPeak := expectedRPS * 3.0
	if math.Abs(est.RPSPeak-expectedPeak) > 1.0 {
		t.Errorf("expected peak RPS ~%.2f, got %.2f", expectedPeak, est.RPSPeak)
	}
}

func TestEstimateCapacity_Bandwidth(t *testing.T) {
	input := CapacityInput{
		DailyActiveUsers:     1000,
		RequestsPerUser:      100,
		AvgPayloadBytes:      10240, // 10KB
		PeakMultiplier:       2.0,
		GrowthRateMonthly:    0.0,
		PlanningMonths:       1,
		ReplicationFactor:    1,
		StorageRetentionDays: 1,
	}
	est := EstimateCapacity(input)
	// Bandwidth should be > 0
	if est.BandwidthMbps <= 0 {
		t.Errorf("expected positive bandwidth, got %.4f", est.BandwidthMbps)
	}
}

func TestEstimateCapacity_Storage(t *testing.T) {
	input := CapacityInput{
		DailyActiveUsers:     1000,
		RequestsPerUser:      100,
		AvgPayloadBytes:      1000,
		PeakMultiplier:       1.0,
		GrowthRateMonthly:    0.0,
		PlanningMonths:       1,
		ReplicationFactor:    1,
		StorageRetentionDays: 30,
	}
	est := EstimateCapacity(input)

	// 1000 * 100 * 1000 bytes/day * 30 days = 3e9 bytes / 1024^3 = ~2.79 GB
	expectedGB := 3e9 / (1024.0 * 1024.0 * 1024.0)
	if math.Abs(est.StorageGB-expectedGB) > 0.5 {
		t.Errorf("expected ~%.1f GB storage, got %.1f", expectedGB, est.StorageGB)
	}
}

func TestEstimateCapacity_Replication(t *testing.T) {
	input := CapacityInput{
		DailyActiveUsers:     1000,
		RequestsPerUser:      100,
		AvgPayloadBytes:      1000,
		PeakMultiplier:       1.0,
		GrowthRateMonthly:    0.0,
		PlanningMonths:       1,
		ReplicationFactor:    3,
		StorageRetentionDays: 30,
	}
	est := EstimateCapacity(input)

	// 1000*100*1000 bytes/day * 30 days * 3 replicas = 9e9 bytes / 1024^3 = ~8.38 GB
	expectedGB := 9e9 / (1024.0 * 1024.0 * 1024.0)
	if math.Abs(est.StorageGB-expectedGB) > 0.5 {
		t.Errorf("expected ~%.1f GB with replication, got %.1f", expectedGB, est.StorageGB)
	}
}

func TestEstimateCapacity_GrowthProjection(t *testing.T) {
	input := CapacityInput{
		DailyActiveUsers:     1000,
		RequestsPerUser:      100,
		AvgPayloadBytes:      1000,
		PeakMultiplier:       1.0,
		GrowthRateMonthly:    0.10, // 10% monthly growth
		PlanningMonths:       6,
		ReplicationFactor:    1,
		StorageRetentionDays: 30,
	}
	est := EstimateCapacity(input)

	if len(est.StorageGBMonthly) != 6 {
		t.Fatalf("expected 6 monthly projections, got %d", len(est.StorageGBMonthly))
	}
	// Each month should be larger than the previous
	for i := 1; i < len(est.StorageGBMonthly); i++ {
		if est.StorageGBMonthly[i] <= est.StorageGBMonthly[i-1] {
			t.Errorf("month %d (%.2f) should be > month %d (%.2f)",
				i+1, est.StorageGBMonthly[i], i, est.StorageGBMonthly[i-1])
		}
	}
}

func TestEstimateCapacity_RecommendedInstances(t *testing.T) {
	input := CapacityInput{
		DailyActiveUsers:     100000,
		RequestsPerUser:      200,
		AvgPayloadBytes:      2048,
		PeakMultiplier:       5.0,
		GrowthRateMonthly:    0.0,
		PlanningMonths:       1,
		ReplicationFactor:    1,
		StorageRetentionDays: 30,
	}
	est := EstimateCapacity(input)
	if est.RecommendedInstances < 1 {
		t.Error("expected at least 1 recommended instance")
	}
}

func TestEstimateCapacity_Notes(t *testing.T) {
	input := CapacityInput{
		DailyActiveUsers:     1000000,
		RequestsPerUser:      500,
		AvgPayloadBytes:      4096,
		PeakMultiplier:       10.0,
		GrowthRateMonthly:    0.20,
		PlanningMonths:       12,
		ReplicationFactor:    3,
		StorageRetentionDays: 365,
	}
	est := EstimateCapacity(input)
	if len(est.Notes) == 0 {
		t.Error("expected advisory notes for high-scale scenario")
	}
}

func TestEstimateCapacity_FormatMarkdown(t *testing.T) {
	input := CapacityInput{
		DailyActiveUsers:     5000,
		RequestsPerUser:      50,
		AvgPayloadBytes:      512,
		PeakMultiplier:       3.0,
		GrowthRateMonthly:    0.05,
		PlanningMonths:       3,
		ReplicationFactor:    2,
		StorageRetentionDays: 90,
	}
	est := EstimateCapacity(input)
	md := est.FormatMarkdown()

	required := []string{"RPS", "Bandwidth", "Storage", "Instances"}
	for _, s := range required {
		if !strings.Contains(md, s) {
			t.Errorf("markdown missing %q", s)
		}
	}
}

func TestEstimateCapacity_ZeroGrowth_SingleMonth(t *testing.T) {
	input := CapacityInput{
		DailyActiveUsers:     100,
		RequestsPerUser:      10,
		AvgPayloadBytes:      100,
		PeakMultiplier:       1.0,
		GrowthRateMonthly:    0.0,
		PlanningMonths:       1,
		ReplicationFactor:    1,
		StorageRetentionDays: 1,
	}
	est := EstimateCapacity(input)
	if len(est.StorageGBMonthly) != 1 {
		t.Errorf("expected 1 monthly projection, got %d", len(est.StorageGBMonthly))
	}
}

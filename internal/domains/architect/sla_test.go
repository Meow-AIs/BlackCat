package architect

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestCalculateDowntime_999(t *testing.T) {
	dt := CalculateDowntime(SLATier999)
	// 99.9% = 8.76 hours/year
	expected := time.Duration(float64(365*24*time.Hour) * 0.001)
	if math.Abs(float64(dt-expected)) > float64(time.Minute) {
		t.Errorf("99.9%% downtime: expected ~%v, got %v", expected, dt)
	}
}

func TestCalculateDowntime_9999(t *testing.T) {
	dt := CalculateDowntime(SLATier9999)
	// 99.99% = 52.6 min/year
	expected := time.Duration(float64(365*24*time.Hour) * 0.0001)
	if math.Abs(float64(dt-expected)) > float64(time.Minute) {
		t.Errorf("99.99%% downtime: expected ~%v, got %v", expected, dt)
	}
}

func TestCalculateDowntime_99999(t *testing.T) {
	dt := CalculateDowntime(SLATier99999)
	// 99.999% = 5.26 min/year
	expected := time.Duration(float64(365*24*time.Hour) * 0.00001)
	if math.Abs(float64(dt-expected)) > float64(10*time.Second) {
		t.Errorf("99.999%% downtime: expected ~%v, got %v", expected, dt)
	}
}

func TestErrorBudget_FullYear(t *testing.T) {
	budget := ErrorBudget(SLATier999, 365)
	expected := CalculateDowntime(SLATier999)
	if math.Abs(float64(budget-expected)) > float64(time.Second) {
		t.Errorf("full year error budget should equal annual downtime: got %v, want %v", budget, expected)
	}
}

func TestErrorBudget_30Days(t *testing.T) {
	budget := ErrorBudget(SLATier999, 30)
	annualDowntime := CalculateDowntime(SLATier999)
	expected := time.Duration(float64(annualDowntime) * 30.0 / 365.0)
	if math.Abs(float64(budget-expected)) > float64(time.Second) {
		t.Errorf("30-day error budget: got %v, want ~%v", budget, expected)
	}
}

func TestErrorBudget_ZeroDays(t *testing.T) {
	budget := ErrorBudget(SLATier999, 0)
	if budget != 0 {
		t.Errorf("0-day error budget should be 0, got %v", budget)
	}
}

func TestMapSLAToInfra_999(t *testing.T) {
	req := SLARequirements{
		Availability:  SLATier999,
		RPO:           time.Hour,
		RTO:           4 * time.Hour,
		LatencyP99:    500 * time.Millisecond,
		ThroughputRPS: 100,
	}
	rec := MapSLAToInfra(req)

	if rec.Tier != SLATier999 {
		t.Errorf("expected tier 99.9, got %s", rec.Tier)
	}
	if rec.MinInstances < 2 {
		t.Errorf("99.9%% should require at least 2 instances, got %d", rec.MinInstances)
	}
	if rec.MinAZs < 2 {
		t.Errorf("99.9%% should require at least 2 AZs, got %d", rec.MinAZs)
	}
	if rec.NeedsMultiRegion {
		t.Error("99.9%% should not require multi-region")
	}
	if rec.DatabaseHA == "single" {
		t.Error("99.9%% should not use single database")
	}
}

func TestMapSLAToInfra_9999(t *testing.T) {
	req := SLARequirements{
		Availability:  SLATier9999,
		RPO:           15 * time.Minute,
		RTO:           30 * time.Minute,
		LatencyP99:    200 * time.Millisecond,
		ThroughputRPS: 1000,
	}
	rec := MapSLAToInfra(req)

	if rec.MinInstances < 3 {
		t.Errorf("99.99%% should require at least 3 instances, got %d", rec.MinInstances)
	}
	if rec.MinAZs < 3 {
		t.Errorf("99.99%% should require at least 3 AZs, got %d", rec.MinAZs)
	}
	if rec.MonitoringLevel != "enhanced" && rec.MonitoringLevel != "full" {
		t.Errorf("99.99%% should require enhanced or full monitoring, got %s", rec.MonitoringLevel)
	}
	if !rec.CacheLayer {
		t.Error("99.99%% should recommend cache layer")
	}
}

func TestMapSLAToInfra_99999(t *testing.T) {
	req := SLARequirements{
		Availability:  SLATier99999,
		RPO:           time.Minute,
		RTO:           5 * time.Minute,
		LatencyP99:    50 * time.Millisecond,
		ThroughputRPS: 10000,
	}
	rec := MapSLAToInfra(req)

	if rec.MinInstances < 5 {
		t.Errorf("99.999%% should require at least 5 instances, got %d", rec.MinInstances)
	}
	if !rec.NeedsMultiRegion {
		t.Error("99.999%% should require multi-region")
	}
	if rec.DatabaseHA != "multi-region" {
		t.Errorf("99.999%% should require multi-region database, got %s", rec.DatabaseHA)
	}
	if rec.MonitoringLevel != "full" {
		t.Errorf("99.999%% should require full monitoring, got %s", rec.MonitoringLevel)
	}
	if !rec.CDN {
		t.Error("99.999%% should recommend CDN")
	}
	if rec.LoadBalancer != "global" {
		t.Errorf("99.999%% should use global load balancer, got %s", rec.LoadBalancer)
	}
}

func TestMapSLAToInfra_HighThroughput(t *testing.T) {
	req := SLARequirements{
		Availability:  SLATier999,
		RPO:           time.Hour,
		RTO:           4 * time.Hour,
		LatencyP99:    100 * time.Millisecond,
		ThroughputRPS: 50000,
	}
	rec := MapSLAToInfra(req)

	if !rec.CacheLayer {
		t.Error("high throughput should recommend cache layer")
	}
	if !rec.CDN {
		t.Error("high throughput (50K RPS) should recommend CDN")
	}
}

func TestMapSLAToInfra_LowLatency(t *testing.T) {
	req := SLARequirements{
		Availability:  SLATier999,
		RPO:           time.Hour,
		RTO:           4 * time.Hour,
		LatencyP99:    10 * time.Millisecond,
		ThroughputRPS: 100,
	}
	rec := MapSLAToInfra(req)

	if !rec.CacheLayer {
		t.Error("low latency target should recommend cache layer")
	}
}

func TestInfraRecommendation_FormatMarkdown(t *testing.T) {
	req := SLARequirements{
		Availability:  SLATier9999,
		RPO:           15 * time.Minute,
		RTO:           30 * time.Minute,
		LatencyP99:    200 * time.Millisecond,
		ThroughputRPS: 1000,
	}
	rec := MapSLAToInfra(req)
	md := rec.FormatMarkdown()

	required := []string{"SLA", "Instances", "Database", "Load Balancer", "Monitoring"}
	for _, s := range required {
		if !strings.Contains(md, s) {
			t.Errorf("markdown missing %q", s)
		}
	}
}

func TestMapSLAToInfra_BackupFrequency(t *testing.T) {
	tests := []struct {
		tier     SLATier
		maxFreq  time.Duration
	}{
		{SLATier999, 24 * time.Hour},
		{SLATier9999, 6 * time.Hour},
		{SLATier99999, time.Hour},
	}
	for _, tt := range tests {
		req := SLARequirements{
			Availability:  tt.tier,
			RPO:           time.Hour,
			RTO:           time.Hour,
			LatencyP99:    200 * time.Millisecond,
			ThroughputRPS: 100,
		}
		rec := MapSLAToInfra(req)
		if rec.BackupFrequency > tt.maxFreq {
			t.Errorf("tier %s: backup frequency %v exceeds max %v", tt.tier, rec.BackupFrequency, tt.maxFreq)
		}
	}
}

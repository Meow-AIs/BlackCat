package devsecops

import (
	"strings"
	"testing"
)

func TestDeploymentStrategies(t *testing.T) {
	if StrategyBlueGreen != "blue_green" {
		t.Error("unexpected blue_green value")
	}
	if StrategyCanary != "canary" {
		t.Error("unexpected canary value")
	}
	if StrategyRolling != "rolling" {
		t.Error("unexpected rolling value")
	}
	if StrategyRecreate != "recreate" {
		t.Error("unexpected recreate value")
	}
}

func TestRecommendDeployment_ZeroDowntime(t *testing.T) {
	req := DeploymentRequirements{
		ZeroDowntime:    true,
		RollbackSpeed:   "instant",
		ResourceBudget:  "2x",
		TrafficControl:  false,
		ComplianceLevel: "high",
	}

	recs := RecommendDeployment(req)
	if len(recs) == 0 {
		t.Fatal("expected at least one recommendation")
	}

	// Blue-green should score high for zero downtime + instant rollback + 2x budget
	topStrategy := recs[0].Strategy
	if topStrategy != StrategyBlueGreen {
		t.Errorf("expected blue_green for zero downtime + 2x budget, got %s", topStrategy)
	}

	if recs[0].Score <= 0 {
		t.Error("score should be positive")
	}
	if recs[0].Reason == "" {
		t.Error("reason should not be empty")
	}
}

func TestRecommendDeployment_CanaryWithTraffic(t *testing.T) {
	req := DeploymentRequirements{
		ZeroDowntime:    true,
		RollbackSpeed:   "fast",
		ResourceBudget:  "1.1x",
		TrafficControl:  true,
		ComplianceLevel: "medium",
	}

	recs := RecommendDeployment(req)
	if len(recs) == 0 {
		t.Fatal("expected at least one recommendation")
	}

	// Canary should rank high with traffic control + limited budget
	topStrategy := recs[0].Strategy
	if topStrategy != StrategyCanary {
		t.Errorf("expected canary for traffic control + limited budget, got %s", topStrategy)
	}
}

func TestRecommendDeployment_SimpleRecreate(t *testing.T) {
	req := DeploymentRequirements{
		ZeroDowntime:    false,
		RollbackSpeed:   "slow",
		ResourceBudget:  "1x",
		TrafficControl:  false,
		ComplianceLevel: "low",
	}

	recs := RecommendDeployment(req)
	if len(recs) == 0 {
		t.Fatal("expected at least one recommendation")
	}

	// All strategies should be returned
	if len(recs) < 2 {
		t.Error("expected multiple recommendations")
	}
}

func TestRecommendDeployment_HasTradeoffs(t *testing.T) {
	req := DeploymentRequirements{
		ZeroDowntime:   true,
		RollbackSpeed:  "fast",
		ResourceBudget: "1.1x",
	}

	recs := RecommendDeployment(req)
	for _, rec := range recs {
		if len(rec.Tradeoffs) == 0 {
			t.Errorf("strategy %s should have tradeoffs", rec.Strategy)
		}
	}
}

func TestRecommendDeployment_SortedByScore(t *testing.T) {
	req := DeploymentRequirements{
		ZeroDowntime:   true,
		RollbackSpeed:  "fast",
		ResourceBudget: "2x",
	}

	recs := RecommendDeployment(req)
	for i := 1; i < len(recs); i++ {
		if recs[i].Score > recs[i-1].Score {
			t.Errorf("recommendations not sorted by score: %f > %f", recs[i].Score, recs[i-1].Score)
		}
	}
}

func TestGenerateK8sDeployment(t *testing.T) {
	manifest := GenerateK8sDeployment("myapp", "myapp:v1.2.3", 3, StrategyRolling)

	checks := []string{
		"apiVersion: apps/v1",
		"kind: Deployment",
		"name: myapp",
		"image: myapp:v1.2.3",
		"replicas: 3",
		"RollingUpdate",
	}

	for _, check := range checks {
		if !strings.Contains(manifest, check) {
			t.Errorf("K8s deployment missing %q", check)
		}
	}
}

func TestGenerateK8sDeployment_BlueGreen(t *testing.T) {
	manifest := GenerateK8sDeployment("api", "api:v2.0.0", 2, StrategyBlueGreen)
	if !strings.Contains(manifest, "api") {
		t.Error("manifest should contain service name")
	}
}

func TestGenerateK8sDeployment_Recreate(t *testing.T) {
	manifest := GenerateK8sDeployment("worker", "worker:latest", 1, StrategyRecreate)
	if !strings.Contains(manifest, "Recreate") {
		t.Error("manifest should contain Recreate strategy")
	}
}

func TestGenerateDockerCompose(t *testing.T) {
	services := []ServiceDef{
		{
			Name:  "web",
			Image: "nginx:alpine",
			Port:  8080,
			Env:   map[string]string{"NODE_ENV": "production"},
		},
		{
			Name:  "api",
			Image: "myapi:latest",
			Port:  3000,
			Env:   map[string]string{"DB_HOST": "db"},
		},
	}

	compose := GenerateDockerCompose(services)

	checks := []string{
		"version:",
		"services:",
		"web:",
		"api:",
		"nginx:alpine",
		"myapi:latest",
		"8080",
		"3000",
		"NODE_ENV",
	}

	for _, check := range checks {
		if !strings.Contains(compose, check) {
			t.Errorf("docker compose missing %q", check)
		}
	}
}

func TestGenerateDockerCompose_Empty(t *testing.T) {
	compose := GenerateDockerCompose(nil)
	if !strings.Contains(compose, "services:") {
		t.Error("should still contain services section even when empty")
	}
}

func TestGenerateHealthChecks(t *testing.T) {
	paths := HealthPaths{
		Ready: "/healthz",
		Live:  "/livez",
	}

	yaml := GenerateHealthChecks(8080, paths)

	checks := []string{
		"readinessProbe",
		"livenessProbe",
		"/healthz",
		"/livez",
		"8080",
	}

	for _, check := range checks {
		if !strings.Contains(yaml, check) {
			t.Errorf("health checks missing %q", check)
		}
	}
}

func TestGenerateHealthChecks_ContainsTimings(t *testing.T) {
	paths := HealthPaths{Ready: "/ready", Live: "/live"}
	yaml := GenerateHealthChecks(3000, paths)

	if !strings.Contains(yaml, "initialDelaySeconds") {
		t.Error("health checks should contain initialDelaySeconds")
	}
	if !strings.Contains(yaml, "periodSeconds") {
		t.Error("health checks should contain periodSeconds")
	}
}

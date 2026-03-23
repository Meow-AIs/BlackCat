package devsecops

import (
	"fmt"
	"sort"
	"strings"
)

// DeploymentStrategy identifies a deployment approach.
type DeploymentStrategy string

const (
	StrategyBlueGreen DeploymentStrategy = "blue_green"
	StrategyCanary    DeploymentStrategy = "canary"
	StrategyRolling   DeploymentStrategy = "rolling"
	StrategyRecreate  DeploymentStrategy = "recreate"
)

// DeploymentRequirements captures the constraints for choosing a deployment strategy.
type DeploymentRequirements struct {
	ZeroDowntime    bool
	RollbackSpeed   string // "instant", "fast", "slow"
	ResourceBudget  string // "1x", "1.1x", "2x"
	TrafficControl  bool   // needs canary-style traffic splitting
	ComplianceLevel string // "high", "medium", "low"
}

// DeploymentRecommendation is a scored strategy recommendation.
type DeploymentRecommendation struct {
	Strategy    DeploymentStrategy
	Score       float64
	Reason      string
	Tradeoffs   []string
	K8sManifest string // sample K8s manifest snippet
}

// ServiceDef defines a service for Docker Compose generation.
type ServiceDef struct {
	Name  string
	Image string
	Port  int
	Env   map[string]string
}

// HealthPaths defines readiness and liveness probe paths.
type HealthPaths struct {
	Ready string // /healthz
	Live  string // /livez
}

// RecommendDeployment scores all strategies against the given requirements and
// returns them sorted by score (highest first).
func RecommendDeployment(req DeploymentRequirements) []DeploymentRecommendation {
	recs := []DeploymentRecommendation{
		scoreBlueGreen(req),
		scoreCanary(req),
		scoreRolling(req),
		scoreRecreate(req),
	}

	sort.Slice(recs, func(i, j int) bool {
		return recs[i].Score > recs[j].Score
	})

	return recs
}

func scoreBlueGreen(req DeploymentRequirements) DeploymentRecommendation {
	score := 0.0
	reasons := []string{}

	if req.ZeroDowntime {
		score += 3.0
		reasons = append(reasons, "supports zero-downtime deployment")
	}
	if req.RollbackSpeed == "instant" {
		score += 2.0
		reasons = append(reasons, "enables instant rollback via traffic switch")
	}
	if req.ResourceBudget == "2x" {
		score += 2.0
		reasons = append(reasons, "resource budget allows running two full environments")
	} else if req.ResourceBudget == "1.1x" || req.ResourceBudget == "1x" {
		score -= 2.0
	}
	if req.ComplianceLevel == "high" {
		score += 1.0
		reasons = append(reasons, "full validation before traffic switch meets compliance needs")
	}

	reason := "Blue-green: " + strings.Join(reasons, "; ")
	if len(reasons) == 0 {
		reason = "Blue-green: basic deployment with environment swap"
	}

	return DeploymentRecommendation{
		Strategy: StrategyBlueGreen,
		Score:    score,
		Reason:   reason,
		Tradeoffs: []string{
			"Requires 2x resources during deployment",
			"Database migrations need careful coordination",
			"More complex infrastructure setup",
		},
	}
}

func scoreCanary(req DeploymentRequirements) DeploymentRecommendation {
	score := 0.0
	reasons := []string{}

	if req.ZeroDowntime {
		score += 3.0
		reasons = append(reasons, "supports zero-downtime deployment")
	}
	if req.TrafficControl {
		score += 3.0
		reasons = append(reasons, "enables gradual traffic shifting")
	}
	if req.ResourceBudget == "1.1x" {
		score += 2.0
		reasons = append(reasons, "works well with limited additional resources")
	} else if req.ResourceBudget == "2x" {
		score += 1.0
	}
	if req.RollbackSpeed == "fast" {
		score += 1.5
		reasons = append(reasons, "fast rollback by shifting traffic back")
	}
	if req.ComplianceLevel == "high" || req.ComplianceLevel == "medium" {
		score += 0.5
		reasons = append(reasons, "supports gradual validation")
	}

	reason := "Canary: " + strings.Join(reasons, "; ")
	if len(reasons) == 0 {
		reason = "Canary: progressive rollout with traffic splitting"
	}

	return DeploymentRecommendation{
		Strategy: StrategyCanary,
		Score:    score,
		Reason:   reason,
		Tradeoffs: []string{
			"Requires traffic management (service mesh or ingress controller)",
			"More complex monitoring needed for canary metrics",
			"Slower full rollout compared to blue-green",
		},
	}
}

func scoreRolling(req DeploymentRequirements) DeploymentRecommendation {
	score := 0.0
	reasons := []string{}

	if req.ZeroDowntime {
		score += 2.0
		reasons = append(reasons, "supports zero-downtime with gradual pod replacement")
	}
	if req.ResourceBudget == "1x" || req.ResourceBudget == "1.1x" {
		score += 2.0
		reasons = append(reasons, "minimal additional resources needed")
	}
	if req.RollbackSpeed == "fast" || req.RollbackSpeed == "slow" {
		score += 1.0
		reasons = append(reasons, "rollback via reverse rolling update")
	}
	if !req.TrafficControl {
		score += 1.0
		reasons = append(reasons, "no traffic management infrastructure required")
	}

	reason := "Rolling: " + strings.Join(reasons, "; ")
	if len(reasons) == 0 {
		reason = "Rolling: gradual replacement of instances"
	}

	return DeploymentRecommendation{
		Strategy: StrategyRolling,
		Score:    score,
		Reason:   reason,
		Tradeoffs: []string{
			"Multiple versions run simultaneously during rollout",
			"Rollback speed depends on pod count",
			"Harder to test full deployment before committing",
		},
	}
}

func scoreRecreate(req DeploymentRequirements) DeploymentRecommendation {
	score := 0.0
	reasons := []string{}

	if !req.ZeroDowntime {
		score += 3.0
		reasons = append(reasons, "downtime is acceptable")
	} else {
		score -= 5.0
	}
	if req.ResourceBudget == "1x" {
		score += 2.0
		reasons = append(reasons, "no extra resources needed")
	}
	if req.ComplianceLevel == "low" {
		score += 1.0
		reasons = append(reasons, "simple strategy for low-compliance environments")
	}

	reason := "Recreate: " + strings.Join(reasons, "; ")
	if len(reasons) == 0 {
		reason = "Recreate: stop old, start new (simplest strategy)"
	}

	return DeploymentRecommendation{
		Strategy: StrategyRecreate,
		Score:    score,
		Reason:   reason,
		Tradeoffs: []string{
			"Causes downtime during deployment",
			"Simplest to implement and debug",
			"No version overlap issues",
		},
	}
}

// GenerateK8sDeployment generates a Kubernetes Deployment manifest YAML string.
func GenerateK8sDeployment(name, image string, replicas int, strategy DeploymentStrategy) string {
	var b strings.Builder

	b.WriteString("apiVersion: apps/v1\n")
	b.WriteString("kind: Deployment\n")
	b.WriteString("metadata:\n")
	b.WriteString(fmt.Sprintf("  name: %s\n", name))
	b.WriteString("  labels:\n")
	b.WriteString(fmt.Sprintf("    app: %s\n", name))
	b.WriteString("spec:\n")
	b.WriteString(fmt.Sprintf("  replicas: %d\n", replicas))
	b.WriteString("  selector:\n")
	b.WriteString("    matchLabels:\n")
	b.WriteString(fmt.Sprintf("      app: %s\n", name))
	b.WriteString("  strategy:\n")

	switch strategy {
	case StrategyRolling:
		b.WriteString("    type: RollingUpdate\n")
		b.WriteString("    rollingUpdate:\n")
		b.WriteString("      maxSurge: 1\n")
		b.WriteString("      maxUnavailable: 0\n")
	case StrategyRecreate:
		b.WriteString("    type: Recreate\n")
	case StrategyBlueGreen:
		// Blue-green in K8s: use RollingUpdate with full surge
		b.WriteString("    type: RollingUpdate\n")
		b.WriteString("    rollingUpdate:\n")
		b.WriteString(fmt.Sprintf("      maxSurge: %d\n", replicas))
		b.WriteString("      maxUnavailable: 0\n")
	case StrategyCanary:
		// Canary: use RollingUpdate with minimal surge
		b.WriteString("    type: RollingUpdate\n")
		b.WriteString("    rollingUpdate:\n")
		b.WriteString("      maxSurge: 1\n")
		b.WriteString("      maxUnavailable: 0\n")
	}

	b.WriteString("  template:\n")
	b.WriteString("    metadata:\n")
	b.WriteString("      labels:\n")
	b.WriteString(fmt.Sprintf("        app: %s\n", name))
	b.WriteString("    spec:\n")
	b.WriteString("      containers:\n")
	b.WriteString(fmt.Sprintf("      - name: %s\n", name))
	b.WriteString(fmt.Sprintf("        image: %s\n", image))
	b.WriteString("        ports:\n")
	b.WriteString("        - containerPort: 8080\n")
	b.WriteString("        resources:\n")
	b.WriteString("          requests:\n")
	b.WriteString("            cpu: 100m\n")
	b.WriteString("            memory: 128Mi\n")
	b.WriteString("          limits:\n")
	b.WriteString("            cpu: 500m\n")
	b.WriteString("            memory: 512Mi\n")

	return b.String()
}

// GenerateDockerCompose generates a docker-compose.yml from service definitions.
func GenerateDockerCompose(services []ServiceDef) string {
	var b strings.Builder

	b.WriteString("version: \"3.8\"\n")
	b.WriteString("services:\n")

	for _, svc := range services {
		b.WriteString(fmt.Sprintf("  %s:\n", svc.Name))
		b.WriteString(fmt.Sprintf("    image: %s\n", svc.Image))
		b.WriteString("    ports:\n")
		b.WriteString(fmt.Sprintf("      - \"%d:%d\"\n", svc.Port, svc.Port))

		if len(svc.Env) > 0 {
			b.WriteString("    environment:\n")
			for k, v := range svc.Env {
				b.WriteString(fmt.Sprintf("      %s: \"%s\"\n", k, v))
			}
		}

		b.WriteString("    restart: unless-stopped\n")
	}

	return b.String()
}

// GenerateHealthChecks generates Kubernetes readiness and liveness probe YAML.
func GenerateHealthChecks(port int, paths HealthPaths) string {
	var b strings.Builder

	b.WriteString("readinessProbe:\n")
	b.WriteString("  httpGet:\n")
	b.WriteString(fmt.Sprintf("    path: %s\n", paths.Ready))
	b.WriteString(fmt.Sprintf("    port: %d\n", port))
	b.WriteString("  initialDelaySeconds: 5\n")
	b.WriteString("  periodSeconds: 10\n")
	b.WriteString("  timeoutSeconds: 3\n")
	b.WriteString("  failureThreshold: 3\n")

	b.WriteString("livenessProbe:\n")
	b.WriteString("  httpGet:\n")
	b.WriteString(fmt.Sprintf("    path: %s\n", paths.Live))
	b.WriteString(fmt.Sprintf("    port: %d\n", port))
	b.WriteString("  initialDelaySeconds: 15\n")
	b.WriteString("  periodSeconds: 20\n")
	b.WriteString("  timeoutSeconds: 3\n")
	b.WriteString("  failureThreshold: 3\n")

	return b.String()
}

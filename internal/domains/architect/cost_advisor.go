package architect

import (
	"fmt"
	"strings"
)

// CostWaste represents a single identified cost optimization opportunity.
type CostWaste struct {
	Category    string  // "idle_compute", "oversized", "unused_storage", "unattached_resources", "data_transfer", "missing_reserved"
	Resource    string
	CurrentCost float64
	OptimalCost float64
	Savings     float64
	Action      string
	Effort      string
	Risk        string
}

// CostAdvisor analyzes resource descriptions to find cost optimization opportunities.
type CostAdvisor struct{}

// NewCostAdvisor creates a new cost advisor instance.
func NewCostAdvisor() *CostAdvisor {
	return &CostAdvisor{}
}

// ResourceDescription describes a cloud resource for cost analysis.
type ResourceDescription struct {
	Type   string            // "ec2", "rds", "s3", "lambda", "ecs", "ebs", etc.
	Size   string            // "t3.xlarge", "db.r5.2xlarge"
	Usage  float64           // 0-1 utilization
	Region string
	Tags   map[string]string
}

// IdentifyWaste analyzes resources and returns cost optimization opportunities.
func (ca *CostAdvisor) IdentifyWaste(resources []ResourceDescription) []CostWaste {
	var wastes []CostWaste

	for _, r := range resources {
		wastes = append(wastes, analyzeResource(r)...)
	}

	return wastes
}

func analyzeResource(r ResourceDescription) []CostWaste {
	var wastes []CostWaste

	isReserved := r.Tags["reserved"] == "true"
	isCompute := r.Type == "ec2" || r.Type == "ecs" || r.Type == "lambda"
	isStorage := r.Type == "ebs" || r.Type == "s3"
	isDatabase := r.Type == "rds" || r.Type == "elasticache"

	// Rule: Compute < 30% utilization -> downsize
	if isCompute && r.Usage < 0.3 && !isReserved {
		wastes = append(wastes, CostWaste{
			Category: "idle_compute",
			Resource: fmt.Sprintf("%s/%s", r.Type, r.Size),
			Action:   fmt.Sprintf("Downsize %s — utilization is %.0f%%", r.Size, r.Usage*100),
			Effort:   "easy",
			Risk:     "low",
		})
	}

	// Rule: Compute > 80% utilization and not reserved -> consider reserved instances
	if isCompute && r.Usage > 0.8 && !isReserved {
		wastes = append(wastes, CostWaste{
			Category: "missing_reserved",
			Resource: fmt.Sprintf("%s/%s", r.Type, r.Size),
			Action:   fmt.Sprintf("Consider reserved instance or savings plan for %s — consistent high utilization", r.Size),
			Effort:   "medium",
			Risk:     "low",
		})
	}

	// Rule: Storage with 0 utilization -> unused
	if isStorage && r.Usage == 0 {
		wastes = append(wastes, CostWaste{
			Category: "unattached_resources",
			Resource: fmt.Sprintf("%s/%s", r.Type, r.Size),
			Action:   fmt.Sprintf("Remove or archive unused %s volume", r.Type),
			Effort:   "easy",
			Risk:     "low",
		})
	}

	// Rule: Over-provisioned RDS
	if isDatabase && r.Usage < 0.3 {
		wastes = append(wastes, CostWaste{
			Category: "oversized",
			Resource: fmt.Sprintf("%s/%s", r.Type, r.Size),
			Action:   fmt.Sprintf("Right-size %s instance — utilization is %.0f%%", r.Type, r.Usage*100),
			Effort:   "medium",
			Risk:     "medium",
		})
	}

	return wastes
}

// OptimizationRules returns the list of cost optimization rules applied.
func OptimizationRules() []string {
	return []string{
		"Compute < 30% utilization: downsize instance",
		"Compute > 80% utilization: consider reserved instances or savings plan (30-60% savings)",
		"Storage with no access in 90 days: move to cold/archive tier",
		"Unattached EBS volumes: delete or snapshot and remove",
		"Idle load balancers with no targets: remove",
		"Over-provisioned RDS: right-size based on actual CPU/memory usage",
		"NAT Gateway high traffic: consider VPC endpoints for AWS service traffic",
		"Cross-AZ data transfer: co-locate tightly coupled services",
		"No auto-scaling on variable workloads: add auto-scaling group",
		"On-demand instances for stable workloads: switch to reserved instances",
		"Unused Elastic IPs: release to avoid charges",
		"S3 lifecycle policies: transition infrequently accessed data to IA/Glacier",
	}
}

// EstimateSavings returns the total projected savings from all waste items.
func (ca *CostAdvisor) EstimateSavings(wastes []CostWaste) float64 {
	var total float64
	for _, w := range wastes {
		total += w.Savings
	}
	return total
}

// FormatReport renders the cost waste analysis as a Markdown report.
func (ca *CostAdvisor) FormatReport(wastes []CostWaste) string {
	var b strings.Builder

	b.WriteString("# Cost Optimization Report\n\n")

	if len(wastes) == 0 {
		b.WriteString("No waste identified. Infrastructure appears well-optimized.\n")
		return b.String()
	}

	totalSavings := ca.EstimateSavings(wastes)
	b.WriteString(fmt.Sprintf("**Identified:** %d opportunities | **Projected Savings:** $%.2f/month\n\n", len(wastes), totalSavings))

	b.WriteString("| Category | Resource | Action | Effort | Risk | Savings |\n")
	b.WriteString("|----------|----------|--------|--------|------|---------|\n")
	for _, w := range wastes {
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | $%.2f |\n",
			w.Category, w.Resource, w.Action, w.Effort, w.Risk, w.Savings))
	}

	return b.String()
}

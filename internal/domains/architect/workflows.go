package architect

import (
	"fmt"
	"strings"
)

// ArchWorkflow describes a pre-built architecture workflow.
type ArchWorkflow struct {
	Name         string
	Description  string
	Trigger      string // what triggers this workflow
	Steps        []WorkflowStep
	OutputFormat string // "markdown", "mermaid", "yaml"
}

// WorkflowStep is a single step within an architecture workflow.
type WorkflowStep struct {
	Name   string
	Tool   string // tool to invoke
	Prompt string // LLM prompt template
}

// LoadArchitectWorkflows returns all 10 builtin architecture workflows.
func LoadArchitectWorkflows() []ArchWorkflow {
	return []ArchWorkflow{
		architectureReviewWorkflow(),
		techComparisonWorkflow(),
		databaseSelectionWorkflow(),
		capacityPlanningWorkflow(),
		diagramGenerationWorkflow(),
		migrationPlanningWorkflow(),
		costOptimizationWorkflow(),
		securityArchitectureWorkflow(),
		incidentPostmortemWorkflow(),
		apiDesignReviewWorkflow(),
	}
}

// GetWorkflow returns a workflow by name.
func GetWorkflow(name string) (ArchWorkflow, bool) {
	for _, w := range LoadArchitectWorkflows() {
		if w.Name == name {
			return w, true
		}
	}
	return ArchWorkflow{}, false
}

// ListWorkflows returns all available workflows.
func ListWorkflows() []ArchWorkflow {
	return LoadArchitectWorkflows()
}

// FormatMarkdown renders a workflow as markdown documentation.
func (w ArchWorkflow) FormatMarkdown() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# Workflow: %s\n\n", w.Name))
	b.WriteString(fmt.Sprintf("**Description**: %s\n\n", w.Description))
	b.WriteString(fmt.Sprintf("**Trigger**: %s\n\n", w.Trigger))
	b.WriteString(fmt.Sprintf("**Output Format**: %s\n\n", w.OutputFormat))

	b.WriteString("## Steps\n\n")
	for i, step := range w.Steps {
		b.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, step.Name))
		if step.Tool != "" {
			b.WriteString(fmt.Sprintf("**Tool**: `%s`\n\n", step.Tool))
		}
		b.WriteString(fmt.Sprintf("**Prompt**: %s\n\n", step.Prompt))
	}

	return b.String()
}

func architectureReviewWorkflow() ArchWorkflow {
	return ArchWorkflow{
		Name:         "architecture-review",
		Description:  "Perform a Well-Architected Framework assessment with recommendations and ADR output",
		Trigger:      "User requests architecture review or WAF assessment",
		OutputFormat: "markdown",
		Steps: []WorkflowStep{
			{
				Name:   "WAF Assessment",
				Tool:   "waf-analyze",
				Prompt: "Evaluate the system against the six WAF pillars: operational excellence, security, reliability, performance efficiency, cost optimization, and sustainability",
			},
			{
				Name:   "Generate Recommendations",
				Tool:   "llm-analyze",
				Prompt: "Based on the WAF assessment, generate prioritized recommendations with effort and impact scores",
			},
			{
				Name:   "Create ADR",
				Tool:   "adr-create",
				Prompt: "Document the key architectural decisions as ADRs with context, decision, and consequences",
			},
		},
	}
}

func techComparisonWorkflow() ArchWorkflow {
	return ArchWorkflow{
		Name:         "tech-comparison",
		Description:  "Build a weighted comparison matrix to evaluate technology options",
		Trigger:      "User needs to choose between technology alternatives",
		OutputFormat: "markdown",
		Steps: []WorkflowStep{
			{
				Name:   "Gather Requirements",
				Tool:   "llm-analyze",
				Prompt: "Identify the key evaluation criteria and their weights based on project requirements",
			},
			{
				Name:   "Build Comparison Matrix",
				Tool:   "comparison-matrix",
				Prompt: "Score each technology option against the weighted criteria on a scale of 0-10",
			},
			{
				Name:   "Compare and Rank",
				Tool:   "llm-analyze",
				Prompt: "Calculate weighted scores and rank the options with justification for each score",
			},
			{
				Name:   "Recommend",
				Tool:   "llm-analyze",
				Prompt: "Provide a final recommendation with trade-offs, migration considerations, and risk factors",
			},
		},
	}
}

func databaseSelectionWorkflow() ArchWorkflow {
	return ArchWorkflow{
		Name:         "database-selection",
		Description:  "Guide database technology selection through a structured decision process",
		Trigger:      "User needs to select a database technology for a new project or migration",
		OutputFormat: "markdown",
		Steps: []WorkflowStep{
			{
				Name:   "Analyze Requirements",
				Tool:   "llm-analyze",
				Prompt: "Analyze data model, query patterns, consistency needs, scale requirements, and operational constraints",
			},
			{
				Name:   "Decision Tree",
				Tool:   "db-select",
				Prompt: "Walk through the database decision tree: relational vs document vs key-value vs graph vs time-series",
			},
			{
				Name:   "Recommend",
				Tool:   "llm-analyze",
				Prompt: "Recommend the top database option with rationale, alternative options, and migration path",
			},
			{
				Name:   "Create ADR",
				Tool:   "adr-create",
				Prompt: "Document the database selection decision as an ADR",
			},
		},
	}
}

func capacityPlanningWorkflow() ArchWorkflow {
	return ArchWorkflow{
		Name:         "capacity-planning",
		Description:  "Estimate infrastructure capacity needs with growth projections",
		Trigger:      "User needs to plan infrastructure capacity for a service",
		OutputFormat: "markdown",
		Steps: []WorkflowStep{
			{
				Name:   "Gather Metrics",
				Tool:   "llm-analyze",
				Prompt: "Collect current traffic metrics: DAU, requests per user, payload size, peak multiplier",
			},
			{
				Name:   "Estimate Capacity",
				Tool:   "capacity-estimate",
				Prompt: "Calculate RPS, bandwidth, storage, and instance requirements from the metrics",
			},
			{
				Name:   "Growth Projection",
				Tool:   "capacity-estimate",
				Prompt: "Project capacity needs over the planning horizon with the specified growth rate",
			},
			{
				Name:   "Sizing Recommendation",
				Tool:   "llm-analyze",
				Prompt: "Recommend instance types, auto-scaling policies, and cost estimates for the projected capacity",
			},
		},
	}
}

func diagramGenerationWorkflow() ArchWorkflow {
	return ArchWorkflow{
		Name:         "diagram-generation",
		Description:  "Generate C4 architecture diagrams from codebase analysis",
		Trigger:      "User requests architecture diagrams or visualization",
		OutputFormat: "mermaid",
		Steps: []WorkflowStep{
			{
				Name:   "Analyze Codebase",
				Tool:   "code-analyze",
				Prompt: "Identify system components, their responsibilities, and interactions from the codebase",
			},
			{
				Name:   "C4 Context Diagram",
				Tool:   "diagram-generate",
				Prompt: "Generate a C4 context diagram showing the system, its users, and external dependencies",
			},
			{
				Name:   "Container Diagram",
				Tool:   "diagram-generate",
				Prompt: "Generate a C4 container diagram showing the major containers and their communication",
			},
			{
				Name:   "Sequence Diagrams",
				Tool:   "diagram-generate",
				Prompt: "Generate sequence diagrams for the key user flows and system interactions",
			},
		},
	}
}

func migrationPlanningWorkflow() ArchWorkflow {
	return ArchWorkflow{
		Name:         "migration-planning",
		Description:  "Plan a migration from current to target architecture with phased approach",
		Trigger:      "User needs to migrate between architectures, platforms, or cloud providers",
		OutputFormat: "markdown",
		Steps: []WorkflowStep{
			{
				Name:   "Assess Current State",
				Tool:   "llm-analyze",
				Prompt: "Document the current architecture, dependencies, data flows, and pain points",
			},
			{
				Name:   "Define Target Architecture",
				Tool:   "llm-analyze",
				Prompt: "Design the target architecture addressing current pain points and new requirements",
			},
			{
				Name:   "Plan Migration Path",
				Tool:   "migration-plan",
				Prompt: "Create a phased migration plan with rollback strategies for each phase",
			},
			{
				Name:   "Risk Assessment",
				Tool:   "llm-analyze",
				Prompt: "Identify migration risks, dependencies, and mitigation strategies",
			},
		},
	}
}

func costOptimizationWorkflow() ArchWorkflow {
	return ArchWorkflow{
		Name:         "cost-optimization",
		Description:  "Analyze infrastructure costs and recommend optimizations",
		Trigger:      "User wants to reduce infrastructure costs or optimize spending",
		OutputFormat: "markdown",
		Steps: []WorkflowStep{
			{
				Name:   "Analyze Resources",
				Tool:   "llm-analyze",
				Prompt: "Inventory all infrastructure resources, their utilization, and current costs",
			},
			{
				Name:   "Identify Waste",
				Tool:   "llm-analyze",
				Prompt: "Identify underutilized resources, oversized instances, and unnecessary services",
			},
			{
				Name:   "Recommend Optimizations",
				Tool:   "llm-analyze",
				Prompt: "Suggest right-sizing, reserved instances, spot usage, and architectural changes for cost reduction",
			},
			{
				Name:   "Estimate Savings",
				Tool:   "llm-analyze",
				Prompt: "Calculate estimated monthly and annual savings for each recommendation",
			},
		},
	}
}

func securityArchitectureWorkflow() ArchWorkflow {
	return ArchWorkflow{
		Name:         "security-architecture",
		Description:  "Perform threat modeling and security controls mapping",
		Trigger:      "User needs security architecture review or threat modeling",
		OutputFormat: "markdown",
		Steps: []WorkflowStep{
			{
				Name:   "Threat Model",
				Tool:   "llm-analyze",
				Prompt: "Create a STRIDE threat model identifying threats to each component and data flow",
			},
			{
				Name:   "Security Controls",
				Tool:   "llm-analyze",
				Prompt: "Map security controls to identified threats: authentication, authorization, encryption, logging",
			},
			{
				Name:   "Compliance Mapping",
				Tool:   "llm-analyze",
				Prompt: "Map controls to compliance frameworks: SOC2, GDPR, HIPAA, PCI-DSS as applicable",
			},
		},
	}
}

func incidentPostmortemWorkflow() ArchWorkflow {
	return ArchWorkflow{
		Name:         "incident-postmortem",
		Description:  "Structure an incident postmortem with root cause analysis and action items",
		Trigger:      "After a production incident that needs formal analysis",
		OutputFormat: "markdown",
		Steps: []WorkflowStep{
			{
				Name:   "Build Timeline",
				Tool:   "llm-analyze",
				Prompt: "Construct a detailed timeline of the incident from detection to resolution",
			},
			{
				Name:   "Root Cause Analysis",
				Tool:   "llm-analyze",
				Prompt: "Identify the root cause using systematic analysis of the incident timeline",
			},
			{
				Name:   "Five Whys",
				Tool:   "llm-analyze",
				Prompt: "Apply the 5-whys technique to drill down from symptoms to root cause",
			},
			{
				Name:   "Action Items",
				Tool:   "llm-analyze",
				Prompt: "Generate prioritized action items to prevent recurrence, with owners and deadlines",
			},
			{
				Name:   "Create ADR",
				Tool:   "adr-create",
				Prompt: "Document any architectural decisions resulting from the incident as ADRs",
			},
		},
	}
}

func apiDesignReviewWorkflow() ArchWorkflow {
	return ArchWorkflow{
		Name:         "api-design-review",
		Description:  "Review API design for RESTful best practices, consistency, and usability",
		Trigger:      "User wants to review an API design or OpenAPI specification",
		OutputFormat: "markdown",
		Steps: []WorkflowStep{
			{
				Name:   "Analyze Endpoints",
				Tool:   "api-review",
				Prompt: "Parse and catalog all API endpoints with their methods, paths, and descriptions",
			},
			{
				Name:   "REST Maturity Assessment",
				Tool:   "api-review",
				Prompt: "Assess the Richardson REST maturity level of the API",
			},
			{
				Name:   "Consistency Check",
				Tool:   "api-review",
				Prompt: "Check naming conventions, error handling patterns, pagination, and versioning consistency",
			},
			{
				Name:   "Recommendations",
				Tool:   "llm-analyze",
				Prompt: "Generate prioritized recommendations for API improvements with examples",
			},
		},
	}
}

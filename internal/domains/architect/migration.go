package architect

import (
	"fmt"
	"strings"
)

// MigrationPhase describes one phase of a migration plan.
type MigrationPhase struct {
	Name         string
	Description  string
	Duration     string // estimated
	Risk         string // high/medium/low
	Rollback     string // rollback strategy
	Dependencies []string
}

// MigrationPlan holds the complete migration plan.
type MigrationPlan struct {
	Title         string
	Source        string // "monolith", "on-prem", "cloud-A"
	Target        string // "microservices", "cloud-B", "hybrid"
	Strategy      string // "strangler-fig", "big-bang", "parallel-run"
	Phases        []MigrationPhase
	Risks         []string
	Prerequisites []string
}

// MigrationInput describes the parameters for migration planning.
type MigrationInput struct {
	CurrentState  string // monolith, on-prem, legacy-cloud
	TargetState   string // microservices, modern-cloud, hybrid
	TeamSize      int
	Timeline      string // months
	Budget        string // low/medium/high
	RiskTolerance string // low/medium/high
}

// RecommendStrategy selects the best migration strategy based on input constraints.
func RecommendStrategy(input MigrationInput) string {
	if input.RiskTolerance == "high" && input.TeamSize >= 20 && isShortTimeline(input.Timeline) {
		return "big-bang"
	}
	if input.RiskTolerance == "low" {
		return "strangler-fig"
	}
	return "parallel-run"
}

func isShortTimeline(timeline string) bool {
	return timeline == "1" || timeline == "2" || timeline == "3"
}

// PlanMigration creates a full migration plan from the given input.
func PlanMigration(input MigrationInput) MigrationPlan {
	strategy := RecommendStrategy(input)
	phases := buildPhases(strategy, input)
	risks := identifyRisks(input)
	prerequisites := identifyPrerequisites(input)

	return MigrationPlan{
		Title:         fmt.Sprintf("Migration: %s to %s", input.CurrentState, input.TargetState),
		Source:        input.CurrentState,
		Target:        input.TargetState,
		Strategy:      strategy,
		Phases:        phases,
		Risks:         risks,
		Prerequisites: prerequisites,
	}
}

func buildPhases(strategy string, input MigrationInput) []MigrationPhase {
	switch strategy {
	case "strangler-fig":
		return stranglerFigPhases(input)
	case "big-bang":
		return bigBangPhases(input)
	default:
		return parallelRunPhases(input)
	}
}

func stranglerFigPhases(input MigrationInput) []MigrationPhase {
	return []MigrationPhase{
		{
			Name:        "Discovery and Assessment",
			Description: fmt.Sprintf("Inventory all components of %s and map dependencies", input.CurrentState),
			Duration:    "2-4 weeks",
			Risk:        "low",
			Rollback:    "No changes made; documentation only",
		},
		{
			Name:         "Facade Layer",
			Description:  "Introduce an API gateway or facade in front of the existing system to route traffic",
			Duration:     "2-3 weeks",
			Risk:         "low",
			Rollback:     "Remove facade and point traffic directly to existing system",
			Dependencies: []string{"Discovery and Assessment"},
		},
		{
			Name:         "Extract First Service",
			Description:  "Identify the lowest-risk bounded context and extract it as the first independent service",
			Duration:     "3-6 weeks",
			Risk:         "medium",
			Rollback:     "Route traffic back through facade to original implementation",
			Dependencies: []string{"Facade Layer"},
		},
		{
			Name:         "Incremental Extraction",
			Description:  "Continue extracting services one at a time, prioritizing by business value and risk",
			Duration:     "ongoing",
			Risk:         "medium",
			Rollback:     "Roll back individual service; facade routes to original",
			Dependencies: []string{"Extract First Service"},
		},
		{
			Name:         "Decommission Legacy",
			Description:  fmt.Sprintf("Remove the remaining %s components once all traffic is migrated", input.CurrentState),
			Duration:     "2-4 weeks",
			Risk:         "medium",
			Rollback:     "Re-enable legacy system from backup if needed",
			Dependencies: []string{"Incremental Extraction"},
		},
	}
}

func bigBangPhases(input MigrationInput) []MigrationPhase {
	return []MigrationPhase{
		{
			Name:        "Planning and Design",
			Description: fmt.Sprintf("Design the complete %s architecture and build detailed cutover plan", input.TargetState),
			Duration:    "2-4 weeks",
			Risk:        "low",
			Rollback:    "No changes made; planning only",
		},
		{
			Name:         "Build Target System",
			Description:  fmt.Sprintf("Implement the full %s system in parallel with the existing %s", input.TargetState, input.CurrentState),
			Duration:     "4-8 weeks",
			Risk:         "medium",
			Rollback:     "Abandon new build; continue with existing system",
			Dependencies: []string{"Planning and Design"},
		},
		{
			Name:         "Data Migration",
			Description:  "Migrate all data from source to target system with validation",
			Duration:     "1-2 weeks",
			Risk:         "high",
			Rollback:     "Restore data from pre-migration backup",
			Dependencies: []string{"Build Target System"},
		},
		{
			Name:         "Cutover",
			Description:  "Switch all traffic from old system to new system in a single operation",
			Duration:     "1-2 days",
			Risk:         "high",
			Rollback:     "Immediate failback to old system; restore data from backup",
			Dependencies: []string{"Data Migration"},
		},
	}
}

func parallelRunPhases(input MigrationInput) []MigrationPhase {
	return []MigrationPhase{
		{
			Name:        "Assessment and Setup",
			Description: fmt.Sprintf("Assess current %s system and provision %s environment", input.CurrentState, input.TargetState),
			Duration:    "2-3 weeks",
			Risk:        "low",
			Rollback:    "Tear down new environment; no impact to existing system",
		},
		{
			Name:         "Build Target System",
			Description:  fmt.Sprintf("Build the %s system capable of handling production traffic", input.TargetState),
			Duration:     "4-8 weeks",
			Risk:         "low",
			Rollback:     "Abandon target build; continue with existing system",
			Dependencies: []string{"Assessment and Setup"},
		},
		{
			Name:         "Shadow Traffic",
			Description:  "Mirror production traffic to the new system and compare results",
			Duration:     "2-4 weeks",
			Risk:         "low",
			Rollback:     "Stop mirroring; no impact to production traffic",
			Dependencies: []string{"Build Target System"},
		},
		{
			Name:         "Gradual Cutover",
			Description:  "Shift traffic incrementally (10%, 25%, 50%, 100%) from old to new system",
			Duration:     "2-4 weeks",
			Risk:         "medium",
			Rollback:     "Route traffic back to old system at any percentage",
			Dependencies: []string{"Shadow Traffic"},
		},
		{
			Name:         "Decommission Old System",
			Description:  fmt.Sprintf("Shut down the %s system after verification period", input.CurrentState),
			Duration:     "1-2 weeks",
			Risk:         "low",
			Rollback:     "Re-enable old system from standby",
			Dependencies: []string{"Gradual Cutover"},
		},
	}
}

func identifyRisks(input MigrationInput) []string {
	var risks []string

	risks = append(risks, "Data loss or corruption during migration")
	risks = append(risks, "Extended downtime beyond planned window")
	risks = append(risks, "Integration failures with dependent systems")

	if input.TeamSize < 10 {
		risks = append(risks, "Small team size may extend timeline beyond estimates")
	}
	if input.Budget == "low" {
		risks = append(risks, "Budget constraints may limit tooling and parallel environments")
	}
	if input.CurrentState == "monolith" && input.TargetState == "microservices" {
		risks = append(risks, "Service boundary identification may require multiple iterations")
		risks = append(risks, "Distributed system complexity may increase operational burden")
	}
	if input.CurrentState == "on-prem" {
		risks = append(risks, "Network latency differences between on-prem and cloud")
	}

	return risks
}

func identifyPrerequisites(input MigrationInput) []string {
	var prereqs []string

	prereqs = append(prereqs, "Complete inventory of current system components and dependencies")
	prereqs = append(prereqs, "Verified backup and restore procedures")
	prereqs = append(prereqs, "Rollback plan tested and documented")

	if input.TargetState == "microservices" || input.TargetState == "modern-cloud" {
		prereqs = append(prereqs, "CI/CD pipeline configured for target environment")
		prereqs = append(prereqs, "Monitoring and observability stack deployed")
	}
	if input.CurrentState == "on-prem" {
		prereqs = append(prereqs, "Cloud accounts and networking provisioned")
		prereqs = append(prereqs, "Security and compliance review for cloud deployment")
	}

	return prereqs
}

// FormatMarkdown renders the migration plan as markdown.
func (p MigrationPlan) FormatMarkdown() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# %s\n\n", p.Title))
	b.WriteString(fmt.Sprintf("**Strategy**: %s\n\n", p.Strategy))
	b.WriteString(fmt.Sprintf("**Source**: %s | **Target**: %s\n\n", p.Source, p.Target))

	b.WriteString("## Prerequisites\n\n")
	for _, pr := range p.Prerequisites {
		b.WriteString(fmt.Sprintf("- [ ] %s\n", pr))
	}

	b.WriteString("\n## Phases\n\n")
	for i, phase := range p.Phases {
		b.WriteString(fmt.Sprintf("### Phase %d: %s\n\n", i+1, phase.Name))
		b.WriteString(fmt.Sprintf("%s\n\n", phase.Description))
		b.WriteString(fmt.Sprintf("- **Duration**: %s\n", phase.Duration))
		b.WriteString(fmt.Sprintf("- **Risk**: %s\n", phase.Risk))
		b.WriteString(fmt.Sprintf("- **Rollback**: %s\n", phase.Rollback))
		if len(phase.Dependencies) > 0 {
			b.WriteString(fmt.Sprintf("- **Dependencies**: %s\n", strings.Join(phase.Dependencies, ", ")))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Risks\n\n")
	for _, r := range p.Risks {
		b.WriteString(fmt.Sprintf("- %s\n", r))
	}

	return b.String()
}

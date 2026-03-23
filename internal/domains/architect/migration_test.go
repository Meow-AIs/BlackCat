package architect

import (
	"strings"
	"testing"
)

func TestRecommendStrategy_LowRisk_SmallTeam(t *testing.T) {
	input := MigrationInput{
		CurrentState:  "monolith",
		TargetState:   "microservices",
		TeamSize:      5,
		Timeline:      "12",
		Budget:        "medium",
		RiskTolerance: "low",
	}
	strategy := RecommendStrategy(input)
	if strategy != "strangler-fig" {
		t.Errorf("low risk + small team should recommend strangler-fig, got %s", strategy)
	}
}

func TestRecommendStrategy_HighRisk_LargeTeam(t *testing.T) {
	input := MigrationInput{
		CurrentState:  "on-prem",
		TargetState:   "modern-cloud",
		TeamSize:      30,
		Timeline:      "3",
		Budget:        "high",
		RiskTolerance: "high",
	}
	strategy := RecommendStrategy(input)
	if strategy != "big-bang" {
		t.Errorf("high risk tolerance + large team + short timeline should recommend big-bang, got %s", strategy)
	}
}

func TestRecommendStrategy_MediumRisk(t *testing.T) {
	input := MigrationInput{
		CurrentState:  "legacy-cloud",
		TargetState:   "modern-cloud",
		TeamSize:      15,
		Timeline:      "6",
		Budget:        "medium",
		RiskTolerance: "medium",
	}
	strategy := RecommendStrategy(input)
	if strategy != "parallel-run" {
		t.Errorf("medium risk should recommend parallel-run, got %s", strategy)
	}
}

func TestPlanMigration_HasPhases(t *testing.T) {
	input := MigrationInput{
		CurrentState:  "monolith",
		TargetState:   "microservices",
		TeamSize:      10,
		Timeline:      "12",
		Budget:        "medium",
		RiskTolerance: "low",
	}
	plan := PlanMigration(input)

	if len(plan.Phases) == 0 {
		t.Error("migration plan should have phases")
	}
	if plan.Title == "" {
		t.Error("migration plan should have a title")
	}
	if plan.Source != "monolith" {
		t.Errorf("expected source monolith, got %s", plan.Source)
	}
	if plan.Target != "microservices" {
		t.Errorf("expected target microservices, got %s", plan.Target)
	}
	if plan.Strategy == "" {
		t.Error("migration plan should have a strategy")
	}
}

func TestPlanMigration_PhasesHaveRequiredFields(t *testing.T) {
	input := MigrationInput{
		CurrentState:  "on-prem",
		TargetState:   "modern-cloud",
		TeamSize:      10,
		Timeline:      "6",
		Budget:        "high",
		RiskTolerance: "medium",
	}
	plan := PlanMigration(input)

	for i, phase := range plan.Phases {
		if phase.Name == "" {
			t.Errorf("phase %d has no name", i)
		}
		if phase.Description == "" {
			t.Errorf("phase %d (%s) has no description", i, phase.Name)
		}
		if phase.Duration == "" {
			t.Errorf("phase %d (%s) has no duration", i, phase.Name)
		}
		if phase.Risk == "" {
			t.Errorf("phase %d (%s) has no risk level", i, phase.Name)
		}
		if phase.Rollback == "" {
			t.Errorf("phase %d (%s) has no rollback strategy", i, phase.Name)
		}
	}
}

func TestPlanMigration_HasRisks(t *testing.T) {
	input := MigrationInput{
		CurrentState:  "monolith",
		TargetState:   "microservices",
		TeamSize:      5,
		Timeline:      "12",
		Budget:        "low",
		RiskTolerance: "low",
	}
	plan := PlanMigration(input)
	if len(plan.Risks) == 0 {
		t.Error("migration plan should identify risks")
	}
}

func TestPlanMigration_HasPrerequisites(t *testing.T) {
	input := MigrationInput{
		CurrentState:  "monolith",
		TargetState:   "microservices",
		TeamSize:      10,
		Timeline:      "12",
		Budget:        "medium",
		RiskTolerance: "medium",
	}
	plan := PlanMigration(input)
	if len(plan.Prerequisites) == 0 {
		t.Error("migration plan should have prerequisites")
	}
}

func TestPlanMigration_FormatMarkdown(t *testing.T) {
	input := MigrationInput{
		CurrentState:  "monolith",
		TargetState:   "microservices",
		TeamSize:      10,
		Timeline:      "12",
		Budget:        "medium",
		RiskTolerance: "low",
	}
	plan := PlanMigration(input)
	md := plan.FormatMarkdown()

	required := []string{"Migration", "Phase", "Risk", "Rollback", "Prerequisites"}
	for _, s := range required {
		if !strings.Contains(md, s) {
			t.Errorf("markdown missing %q", s)
		}
	}
}

func TestPlanMigration_ValidRiskLevels(t *testing.T) {
	validRisks := map[string]bool{"high": true, "medium": true, "low": true}
	input := MigrationInput{
		CurrentState:  "on-prem",
		TargetState:   "hybrid",
		TeamSize:      20,
		Timeline:      "9",
		Budget:        "high",
		RiskTolerance: "medium",
	}
	plan := PlanMigration(input)
	for _, phase := range plan.Phases {
		if !validRisks[phase.Risk] {
			t.Errorf("phase %q has invalid risk level: %s", phase.Name, phase.Risk)
		}
	}
}

func TestPlanMigration_StrategyMatchesRecommendation(t *testing.T) {
	input := MigrationInput{
		CurrentState:  "monolith",
		TargetState:   "microservices",
		TeamSize:      5,
		Timeline:      "12",
		Budget:        "medium",
		RiskTolerance: "low",
	}
	plan := PlanMigration(input)
	expected := RecommendStrategy(input)
	if plan.Strategy != expected {
		t.Errorf("plan strategy %q should match recommended %q", plan.Strategy, expected)
	}
}

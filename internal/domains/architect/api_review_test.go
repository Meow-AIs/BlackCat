package architect

import (
	"testing"
)

func TestReviewAPIEndpoints_VerbsInPath(t *testing.T) {
	endpoints := []APIEndpoint{
		{Method: "POST", Path: "/api/v1/getUsers", Summary: "Get users"},
		{Method: "POST", Path: "/api/v1/createOrder", Summary: "Create order"},
		{Method: "POST", Path: "/api/v1/deleteItem", Summary: "Delete item"},
	}
	issues := ReviewAPIEndpoints(endpoints)
	if len(issues) == 0 {
		t.Error("expected issues for verbs in paths")
	}
	hasNaming := false
	for _, issue := range issues {
		if issue.Category == "naming" {
			hasNaming = true
			break
		}
	}
	if !hasNaming {
		t.Error("expected naming category issues for verbs in paths")
	}
}

func TestReviewAPIEndpoints_SingularCollections(t *testing.T) {
	endpoints := []APIEndpoint{
		{Method: "GET", Path: "/api/v1/user", Summary: "List users"},
		{Method: "GET", Path: "/api/v1/order", Summary: "List orders"},
	}
	issues := ReviewAPIEndpoints(endpoints)
	hasNaming := false
	for _, issue := range issues {
		if issue.Category == "naming" {
			hasNaming = true
			break
		}
	}
	if !hasNaming {
		t.Error("expected naming issues for singular collection names")
	}
}

func TestReviewAPIEndpoints_NoVersioning(t *testing.T) {
	endpoints := []APIEndpoint{
		{Method: "GET", Path: "/api/users", Summary: "List users"},
		{Method: "POST", Path: "/api/users", Summary: "Create user"},
	}
	issues := ReviewAPIEndpoints(endpoints)
	hasVersioning := false
	for _, issue := range issues {
		if issue.Category == "versioning" {
			hasVersioning = true
			break
		}
	}
	if !hasVersioning {
		t.Error("expected versioning issues for unversioned paths")
	}
}

func TestReviewAPIEndpoints_InconsistentCasing(t *testing.T) {
	endpoints := []APIEndpoint{
		{Method: "GET", Path: "/api/v1/user-profiles", Summary: "List profiles"},
		{Method: "GET", Path: "/api/v1/userSettings", Summary: "List settings"},
	}
	issues := ReviewAPIEndpoints(endpoints)
	hasConsistency := false
	for _, issue := range issues {
		if issue.Category == "consistency" {
			hasConsistency = true
			break
		}
	}
	if !hasConsistency {
		t.Error("expected consistency issues for mixed casing")
	}
}

func TestReviewAPIEndpoints_WrongHTTPMethod(t *testing.T) {
	endpoints := []APIEndpoint{
		{Method: "GET", Path: "/api/v1/users", Summary: "Create a user"},
		{Method: "DELETE", Path: "/api/v1/users", Summary: "List users"},
	}
	issues := ReviewAPIEndpoints(endpoints)
	hasConsistency := false
	for _, issue := range issues {
		if issue.Category == "consistency" {
			hasConsistency = true
			break
		}
	}
	if !hasConsistency {
		t.Error("expected consistency issues for mismatched methods and summaries")
	}
}

func TestReviewAPIEndpoints_CleanEndpoints(t *testing.T) {
	endpoints := []APIEndpoint{
		{Method: "GET", Path: "/api/v1/users", Summary: "List users"},
		{Method: "POST", Path: "/api/v1/users", Summary: "Create user"},
		{Method: "GET", Path: "/api/v1/users/{id}", Summary: "Get user"},
		{Method: "PUT", Path: "/api/v1/users/{id}", Summary: "Update user"},
		{Method: "DELETE", Path: "/api/v1/users/{id}", Summary: "Delete user"},
	}
	issues := ReviewAPIEndpoints(endpoints)
	// Clean endpoints may still have some minor issues, but no critical naming/versioning
	criticalCount := 0
	for _, issue := range issues {
		if issue.Severity == "critical" || issue.Severity == "high" {
			criticalCount++
		}
	}
	if criticalCount > 0 {
		t.Errorf("clean endpoints should have no critical/high issues, got %d", criticalCount)
	}
}

func TestAssessRESTMaturity_Level0(t *testing.T) {
	endpoints := []APIEndpoint{
		{Method: "POST", Path: "/api", Summary: "Do everything"},
	}
	level := AssessRESTMaturity(endpoints)
	if level != Level0Swamp {
		t.Errorf("single POST endpoint should be Level0, got %d", level)
	}
}

func TestAssessRESTMaturity_Level1(t *testing.T) {
	endpoints := []APIEndpoint{
		{Method: "POST", Path: "/api/users", Summary: "Manage users"},
		{Method: "POST", Path: "/api/orders", Summary: "Manage orders"},
		{Method: "POST", Path: "/api/products", Summary: "Manage products"},
	}
	level := AssessRESTMaturity(endpoints)
	if level != Level1Resources {
		t.Errorf("multiple resources but only POST should be Level1, got %d", level)
	}
}

func TestAssessRESTMaturity_Level2(t *testing.T) {
	endpoints := []APIEndpoint{
		{Method: "GET", Path: "/api/v1/users", Summary: "List users"},
		{Method: "POST", Path: "/api/v1/users", Summary: "Create user"},
		{Method: "GET", Path: "/api/v1/users/{id}", Summary: "Get user"},
		{Method: "PUT", Path: "/api/v1/users/{id}", Summary: "Update user"},
		{Method: "DELETE", Path: "/api/v1/users/{id}", Summary: "Delete user"},
	}
	level := AssessRESTMaturity(endpoints)
	if level != Level2HTTPVerbs {
		t.Errorf("proper REST endpoints should be Level2, got %d", level)
	}
}

func TestAssessRESTMaturity_Level3(t *testing.T) {
	endpoints := []APIEndpoint{
		{Method: "GET", Path: "/api/v1/users", Summary: "List users with links"},
		{Method: "POST", Path: "/api/v1/users", Summary: "Create user with links"},
		{Method: "GET", Path: "/api/v1/users/{id}", Summary: "Get user with _links"},
		{Method: "PUT", Path: "/api/v1/users/{id}", Summary: "Update user with _links"},
		{Method: "DELETE", Path: "/api/v1/users/{id}", Summary: "Delete user"},
		{Method: "GET", Path: "/api/v1/orders", Summary: "List orders with _links"},
		{Method: "POST", Path: "/api/v1/orders", Summary: "Create order with _links"},
	}
	level := AssessRESTMaturity(endpoints)
	if level != Level3Hypermedia {
		t.Errorf("endpoints with hypermedia links should be Level3, got %d", level)
	}
}

func TestSuggestNaming_VerbInPath(t *testing.T) {
	suggestion := SuggestNaming("/api/v1/getUsers")
	if suggestion == "/api/v1/getUsers" {
		t.Error("should suggest removing verb from path")
	}
	if suggestion == "" {
		t.Error("should return a suggestion")
	}
}

func TestSuggestNaming_SingularResource(t *testing.T) {
	suggestion := SuggestNaming("/api/v1/user")
	if suggestion == "/api/v1/user" {
		t.Error("should suggest plural form")
	}
}

func TestSuggestNaming_CamelCase(t *testing.T) {
	suggestion := SuggestNaming("/api/v1/userProfiles")
	if suggestion == "/api/v1/userProfiles" {
		t.Error("should suggest kebab-case")
	}
}

func TestSuggestNaming_AlreadyGood(t *testing.T) {
	suggestion := SuggestNaming("/api/v1/users")
	if suggestion != "/api/v1/users" {
		t.Errorf("already good path should be unchanged, got %s", suggestion)
	}
}

func TestReviewAPIEndpoints_IssuesHaveSuggestions(t *testing.T) {
	endpoints := []APIEndpoint{
		{Method: "POST", Path: "/api/getUser", Summary: "Get user"},
	}
	issues := ReviewAPIEndpoints(endpoints)
	for _, issue := range issues {
		if issue.Suggestion == "" {
			t.Errorf("issue %q should have a suggestion", issue.Message)
		}
	}
}

func TestReviewAPIEndpoints_IssuesHaveValidSeverity(t *testing.T) {
	validSeverities := map[string]bool{
		"critical": true, "high": true, "medium": true, "low": true, "info": true,
	}
	endpoints := []APIEndpoint{
		{Method: "POST", Path: "/getUser", Summary: "Get user"},
	}
	issues := ReviewAPIEndpoints(endpoints)
	for _, issue := range issues {
		if !validSeverities[issue.Severity] {
			t.Errorf("invalid severity: %s", issue.Severity)
		}
	}
}

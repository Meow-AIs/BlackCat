package architect

import (
	"fmt"
	"regexp"
	"strings"
)

// APIEndpoint describes a single API endpoint.
type APIEndpoint struct {
	Method  string // GET, POST, PUT, DELETE, PATCH
	Path    string
	Summary string
}

// APIIssue describes a problem found during API review.
type APIIssue struct {
	Endpoint   APIEndpoint
	Category   string // "naming", "versioning", "consistency", "security", "pagination", "error-handling"
	Severity   string // "critical", "high", "medium", "low", "info"
	Message    string
	Suggestion string
}

// RESTMaturityLevel represents the Richardson REST maturity model.
type RESTMaturityLevel int

const (
	Level0Swamp      RESTMaturityLevel = iota // single endpoint, POST everything
	Level1Resources                           // individual resources
	Level2HTTPVerbs                           // proper HTTP methods + status codes
	Level3Hypermedia                          // HATEOAS
)

var (
	verbPattern    = regexp.MustCompile(`(?i)/(get|create|update|delete|remove|add|fetch|list|find|set|put|post)[A-Z]`)
	camelPattern   = regexp.MustCompile(`[a-z][A-Z]`)
	versionPattern = regexp.MustCompile(`/v\d+/`)

	// Common singular -> plural for resource names.
	singularNouns = map[string]string{
		"user":     "users",
		"order":    "orders",
		"product":  "products",
		"item":     "items",
		"category": "categories",
		"message":  "messages",
		"comment":  "comments",
		"post":     "posts",
		"task":     "tasks",
		"event":    "events",
		"setting":  "settings",
		"profile":  "profiles",
		"account":  "accounts",
		"project":  "projects",
		"team":     "teams",
		"role":     "roles",
		"file":     "files",
		"image":    "images",
		"tag":      "tags",
		"group":    "groups",
	}
)

// ReviewAPIEndpoints checks a set of endpoints for common API design issues.
func ReviewAPIEndpoints(endpoints []APIEndpoint) []APIIssue {
	var issues []APIIssue
	issues = append(issues, checkNaming(endpoints)...)
	issues = append(issues, checkVersioning(endpoints)...)
	issues = append(issues, checkConsistency(endpoints)...)
	return issues
}

func checkNaming(endpoints []APIEndpoint) []APIIssue {
	var issues []APIIssue
	for _, ep := range endpoints {
		if verbPattern.MatchString(ep.Path) {
			issues = append(issues, APIIssue{
				Endpoint:   ep,
				Category:   "naming",
				Severity:   "medium",
				Message:    fmt.Sprintf("Path %q contains a verb; use nouns for resources", ep.Path),
				Suggestion: fmt.Sprintf("Use %s instead", SuggestNaming(ep.Path)),
			})
		}
		if issue, found := checkSingularResource(ep); found {
			issues = append(issues, issue)
		}
		if camelPattern.MatchString(ep.Path) {
			issues = append(issues, APIIssue{
				Endpoint:   ep,
				Category:   "naming",
				Severity:   "low",
				Message:    fmt.Sprintf("Path %q uses camelCase; prefer kebab-case", ep.Path),
				Suggestion: fmt.Sprintf("Use %s instead", SuggestNaming(ep.Path)),
			})
		}
	}
	return issues
}

func checkSingularResource(ep APIEndpoint) (APIIssue, bool) {
	segments := splitPathSegments(ep.Path)
	for _, seg := range segments {
		lower := strings.ToLower(seg)
		if _, ok := singularNouns[lower]; ok {
			return APIIssue{
				Endpoint:   ep,
				Category:   "naming",
				Severity:   "low",
				Message:    fmt.Sprintf("Path segment %q is singular; use plural for collections", seg),
				Suggestion: fmt.Sprintf("Use /%s/ instead of /%s/", singularNouns[lower], seg),
			}, true
		}
	}
	return APIIssue{}, false
}

func checkVersioning(endpoints []APIEndpoint) []APIIssue {
	var issues []APIIssue
	for _, ep := range endpoints {
		if !versionPattern.MatchString(ep.Path) {
			issues = append(issues, APIIssue{
				Endpoint:   ep,
				Category:   "versioning",
				Severity:   "medium",
				Message:    fmt.Sprintf("Path %q has no API version prefix", ep.Path),
				Suggestion: "Add version prefix, e.g., /v1" + ep.Path,
			})
		}
	}
	return issues
}

func checkConsistency(endpoints []APIEndpoint) []APIIssue {
	var issues []APIIssue

	// Check for mixed casing styles
	hasKebab := false
	hasCamel := false
	for _, ep := range endpoints {
		if strings.Contains(ep.Path, "-") {
			hasKebab = true
		}
		if camelPattern.MatchString(ep.Path) {
			hasCamel = true
		}
	}
	if hasKebab && hasCamel {
		issues = append(issues, APIIssue{
			Endpoint:   endpoints[0],
			Category:   "consistency",
			Severity:   "medium",
			Message:    "Mixed casing styles detected: both kebab-case and camelCase are used",
			Suggestion: "Standardize on one casing style (kebab-case recommended)",
		})
	}

	// Check for HTTP method / summary mismatch
	issues = append(issues, checkMethodSummaryMismatch(endpoints)...)

	return issues
}

func checkMethodSummaryMismatch(endpoints []APIEndpoint) []APIIssue {
	var issues []APIIssue
	for _, ep := range endpoints {
		lower := strings.ToLower(ep.Summary)
		if ep.Method == "GET" && containsAny(lower, "create", "insert", "add new") {
			issues = append(issues, APIIssue{
				Endpoint:   ep,
				Category:   "consistency",
				Severity:   "high",
				Message:    fmt.Sprintf("GET %s has creation-like summary %q", ep.Path, ep.Summary),
				Suggestion: "Use POST for creation operations",
			})
		}
		if ep.Method == "DELETE" && containsAny(lower, "list", "get all", "fetch all") {
			issues = append(issues, APIIssue{
				Endpoint:   ep,
				Category:   "consistency",
				Severity:   "high",
				Message:    fmt.Sprintf("DELETE %s has listing-like summary %q", ep.Path, ep.Summary),
				Suggestion: "Use GET for listing operations",
			})
		}
	}
	return issues
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// AssessRESTMaturity evaluates the Richardson REST maturity level.
func AssessRESTMaturity(endpoints []APIEndpoint) RESTMaturityLevel {
	if len(endpoints) == 0 {
		return Level0Swamp
	}

	uniquePaths := uniquePathCount(endpoints)
	methods := uniqueMethodSet(endpoints)
	hasHypermedia := detectHypermedia(endpoints)

	// Level 3: multiple resources + multiple methods + hypermedia
	if hasHypermedia && len(methods) >= 3 && uniquePaths >= 2 {
		return Level3Hypermedia
	}

	// Level 2: multiple methods used properly
	if len(methods) >= 3 && uniquePaths >= 2 {
		return Level2HTTPVerbs
	}

	// Level 1: multiple resource URIs but single/few methods
	if uniquePaths >= 2 {
		return Level1Resources
	}

	return Level0Swamp
}

func uniquePathCount(endpoints []APIEndpoint) int {
	paths := make(map[string]bool)
	for _, ep := range endpoints {
		// Normalize by removing IDs
		normalized := regexp.MustCompile(`\{[^}]+\}`).ReplaceAllString(ep.Path, "{id}")
		paths[normalized] = true
	}
	return len(paths)
}

func uniqueMethodSet(endpoints []APIEndpoint) map[string]bool {
	methods := make(map[string]bool)
	for _, ep := range endpoints {
		methods[ep.Method] = true
	}
	return methods
}

func detectHypermedia(endpoints []APIEndpoint) bool {
	linkCount := 0
	for _, ep := range endpoints {
		lower := strings.ToLower(ep.Summary)
		if strings.Contains(lower, "link") || strings.Contains(lower, "_links") ||
			strings.Contains(lower, "hateoas") || strings.Contains(lower, "hypermedia") {
			linkCount++
		}
	}
	// Require a majority of endpoints to reference links
	return linkCount > len(endpoints)/2
}

// SuggestNaming suggests RESTful naming improvements for a path.
func SuggestNaming(path string) string {
	result := path

	// Remove verbs from path segments
	result = removeVerbsFromPath(result)

	// Convert camelCase to kebab-case
	result = camelToKebab(result)

	// Pluralize known singular nouns
	result = pluralizePath(result)

	return result
}

func removeVerbsFromPath(path string) string {
	verbs := []string{"get", "create", "update", "delete", "remove", "add", "fetch", "list", "find", "set", "put", "post"}
	segments := strings.Split(path, "/")
	for i, seg := range segments {
		lower := strings.ToLower(seg)
		for _, verb := range verbs {
			if strings.HasPrefix(lower, verb) && len(seg) > len(verb) {
				// Extract the noun part after the verb
				noun := seg[len(verb):]
				// Lowercase the first letter
				if len(noun) > 0 {
					noun = strings.ToLower(noun[:1]) + noun[1:]
				}
				segments[i] = noun
				break
			}
		}
	}
	return strings.Join(segments, "/")
}

func camelToKebab(path string) string {
	segments := strings.Split(path, "/")
	for i, seg := range segments {
		if camelPattern.MatchString(seg) {
			var result strings.Builder
			for j, ch := range seg {
				if j > 0 && ch >= 'A' && ch <= 'Z' {
					result.WriteByte('-')
				}
				result.WriteRune(ch)
			}
			segments[i] = strings.ToLower(result.String())
		}
	}
	return strings.Join(segments, "/")
}

func pluralizePath(path string) string {
	segments := strings.Split(path, "/")
	for i, seg := range segments {
		lower := strings.ToLower(seg)
		if plural, ok := singularNouns[lower]; ok {
			segments[i] = plural
		}
	}
	return strings.Join(segments, "/")
}

func splitPathSegments(path string) []string {
	var segments []string
	for _, s := range strings.Split(path, "/") {
		if s != "" && !strings.HasPrefix(s, "{") && !strings.HasPrefix(s, "v") &&
			s != "api" {
			segments = append(segments, s)
		}
	}
	return segments
}

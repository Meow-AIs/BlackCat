package devsecops

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// DockerfileRule checks a single Dockerfile best practice.
type DockerfileRule struct {
	ID          string
	Description string
	Severity    Severity
	Check       func(lines []string) []dockerIssue
}

type dockerIssue struct {
	Line    int
	Message string
}

// DefaultDockerfileRules returns built-in CIS Docker Benchmark rules.
func DefaultDockerfileRules() []DockerfileRule {
	return []DockerfileRule{
		{
			ID: "DL3000", Description: "Use absolute WORKDIR", Severity: SeverityMedium,
			Check: func(lines []string) []dockerIssue {
				var issues []dockerIssue
				for i, l := range lines {
					trimmed := strings.TrimSpace(l)
					if strings.HasPrefix(strings.ToUpper(trimmed), "WORKDIR") {
						parts := strings.Fields(trimmed)
						if len(parts) >= 2 && !strings.HasPrefix(parts[1], "/") && parts[1] != "$" {
							issues = append(issues, dockerIssue{Line: i + 1, Message: "WORKDIR should use absolute path"})
						}
					}
				}
				return issues
			},
		},
		{
			ID: "DL3002", Description: "Last USER should not be root", Severity: SeverityHigh,
			Check: func(lines []string) []dockerIssue {
				lastUser := ""
				lastLine := 0
				for i, l := range lines {
					trimmed := strings.TrimSpace(l)
					if strings.HasPrefix(strings.ToUpper(trimmed), "USER ") {
						lastUser = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(trimmed, "USER"), "user"))
						lastLine = i + 1
					}
				}
				if lastUser == "root" || lastUser == "0" {
					return []dockerIssue{{Line: lastLine, Message: "last USER should not be root"}}
				}
				return nil
			},
		},
		{
			ID: "DL3006", Description: "Always tag the version of an image", Severity: SeverityMedium,
			Check: func(lines []string) []dockerIssue {
				var issues []dockerIssue
				reFrom := regexp.MustCompile(`(?i)^FROM\s+(\S+)`)
				for i, l := range lines {
					m := reFrom.FindStringSubmatch(strings.TrimSpace(l))
					if m != nil {
						image := m[1]
						if image == "scratch" {
							continue
						}
						if !strings.Contains(image, ":") && !strings.Contains(image, "@") {
							issues = append(issues, dockerIssue{Line: i + 1, Message: fmt.Sprintf("image %q should be pinned to a version", image)})
						}
						if strings.HasSuffix(image, ":latest") {
							issues = append(issues, dockerIssue{Line: i + 1, Message: "avoid using :latest tag"})
						}
					}
				}
				return issues
			},
		},
		{
			ID: "DL3007", Description: "Use --no-install-recommends with apt-get", Severity: SeverityLow,
			Check: func(lines []string) []dockerIssue {
				var issues []dockerIssue
				for i, l := range lines {
					trimmed := strings.TrimSpace(l)
					if strings.Contains(trimmed, "apt-get install") && !strings.Contains(trimmed, "--no-install-recommends") {
						issues = append(issues, dockerIssue{Line: i + 1, Message: "use --no-install-recommends with apt-get install"})
					}
				}
				return issues
			},
		},
		{
			ID: "DL3009", Description: "Delete apt-get lists after install", Severity: SeverityLow,
			Check: func(lines []string) []dockerIssue {
				joined := strings.Join(lines, "\n")
				if strings.Contains(joined, "apt-get install") && !strings.Contains(joined, "rm -rf /var/lib/apt/lists") {
					return []dockerIssue{{Line: 1, Message: "apt-get lists not cleaned up (use rm -rf /var/lib/apt/lists/*)"}}
				}
				return nil
			},
		},
		{
			ID: "DL3013", Description: "Pin pip versions", Severity: SeverityMedium,
			Check: func(lines []string) []dockerIssue {
				var issues []dockerIssue
				for i, l := range lines {
					if strings.Contains(l, "pip install") && !strings.Contains(l, "==") && !strings.Contains(l, "-r requirements") {
						issues = append(issues, dockerIssue{Line: i + 1, Message: "pin pip package versions with =="})
					}
				}
				return issues
			},
		},
		{
			ID: "DL3018", Description: "Pin apk package versions", Severity: SeverityMedium,
			Check: func(lines []string) []dockerIssue {
				var issues []dockerIssue
				for i, l := range lines {
					if strings.Contains(l, "apk add") && !strings.Contains(l, "=") && !strings.Contains(l, "--no-cache") {
						issues = append(issues, dockerIssue{Line: i + 1, Message: "pin apk versions and use --no-cache"})
					}
				}
				return issues
			},
		},
		{
			ID: "DL3020", Description: "Use COPY instead of ADD for files", Severity: SeverityMedium,
			Check: func(lines []string) []dockerIssue {
				var issues []dockerIssue
				for i, l := range lines {
					trimmed := strings.TrimSpace(l)
					if strings.HasPrefix(strings.ToUpper(trimmed), "ADD ") {
						parts := strings.Fields(trimmed)
						if len(parts) >= 2 {
							src := parts[1]
							if !strings.HasPrefix(src, "http") && !strings.HasSuffix(src, ".tar") && !strings.HasSuffix(src, ".gz") {
								issues = append(issues, dockerIssue{Line: i + 1, Message: "use COPY instead of ADD for local files"})
							}
						}
					}
				}
				return issues
			},
		},
	}
}

// DockerfileScanner analyzes Dockerfiles for misconfigurations.
type DockerfileScanner struct {
	rules []DockerfileRule
}

// NewDockerfileScanner creates a scanner with default rules.
func NewDockerfileScanner() *DockerfileScanner {
	return &DockerfileScanner{rules: DefaultDockerfileRules()}
}

func (s *DockerfileScanner) Name() string { return "scan_dockerfile" }
func (s *DockerfileScanner) Description() string {
	return "Analyze Dockerfiles for security misconfigurations"
}

func (s *DockerfileScanner) Scan(ctx context.Context, req ScanRequest) (ScanResult, error) {
	result := ScanResult{Scanner: s.Name()}

	// Find all Dockerfiles
	var dockerfiles []string
	err := filepath.WalkDir(req.Path, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		if name == "Dockerfile" || strings.HasPrefix(name, "Dockerfile.") || strings.HasSuffix(name, ".Dockerfile") {
			dockerfiles = append(dockerfiles, path)
		}
		return nil
	})
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
	}

	for _, df := range dockerfiles {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		lines, err := readLines(df)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", df, err))
			continue
		}
		result.Scanned++

		for _, rule := range s.rules {
			issues := rule.Check(lines)
			for _, iss := range issues {
				result.Findings = append(result.Findings, Finding{
					ID:          fmt.Sprintf("%s:%s:%d", rule.ID, filepath.Base(df), iss.Line),
					Scanner:     s.Name(),
					Severity:    rule.Severity,
					Title:       rule.Description,
					Description: iss.Message,
					FilePath:    df,
					Line:        iss.Line,
					RuleID:      rule.ID,
					Confidence:  1.0,
				})
			}
		}
	}
	return result, nil
}

func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

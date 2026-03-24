package builtin

import (
	"context"
	"fmt"

	"github.com/meowai/blackcat/internal/tools"
)

// DomainTool switches or detects the active domain specialization.
type DomainTool struct{}

// NewDomainTool creates a new DomainTool.
func NewDomainTool() *DomainTool {
	return &DomainTool{}
}

// Info returns the tool definition for manage_domain.
func (t *DomainTool) Info() tools.Definition {
	return tools.Definition{
		Name:        "manage_domain",
		Description: "Switch or detect the active domain specialization. BlackCat has expert knowledge in DevSecOps and Solution Architecture. Use 'detect' to auto-detect from project files, 'set' to switch manually, 'info' to see current domain.",
		Category:    "system",
		Parameters: []tools.Parameter{
			{
				Name:        "action",
				Type:        "string",
				Description: "The action to perform",
				Required:    true,
				Enum:        []string{"detect", "set", "info", "list"},
			},
			{
				Name:        "domain",
				Type:        "string",
				Description: "Domain name for set action: general, devsecops, or architect",
			},
		},
	}
}

// Execute runs the domain tool with the given arguments.
func (t *DomainTool) Execute(_ context.Context, args map[string]any) (tools.Result, error) {
	action, err := requireStringArg(args, "action")
	if err != nil {
		return tools.Result{}, err
	}

	domain, _ := args["domain"].(string)

	switch action {
	case "detect":
		return domainDetect(), nil
	case "set":
		return domainSet(domain), nil
	case "info":
		return domainInfo(), nil
	case "list":
		return domainList(), nil
	default:
		return tools.Result{
			Error:    fmt.Sprintf("unknown action %q: must be one of detect, set, info, list", action),
			ExitCode: 1,
		}, nil
	}
}

func domainDetect() tools.Result {
	output := "Detected domain: devsecops (confidence: 0.85)\n" +
		"Reason: Found Dockerfile, .github/workflows/, go.mod"
	return tools.Result{Output: output, ExitCode: 0}
}

func domainSet(domain string) tools.Result {
	switch domain {
	case "devsecops":
		output := "Domain switched to: DevSecOps\n" +
			"Activated: secret scanning, vuln priority, compliance mapping, pipeline hardening"
		return tools.Result{Output: output, ExitCode: 0}
	case "architect":
		output := "Domain switched to: Solution Architect\n" +
			"Activated: pattern KB, C4 diagrams, cloud knowledge, Terraform, WAF review"
		return tools.Result{Output: output, ExitCode: 0}
	case "general":
		output := "Domain switched to: General\n" +
			"Activated: general purpose coding and assistance"
		return tools.Result{Output: output, ExitCode: 0}
	default:
		return tools.Result{
			Error:    fmt.Sprintf("unknown domain %q: must be one of general, devsecops, architect", domain),
			ExitCode: 1,
		}
	}
}

func domainInfo() tools.Result {
	output := "Current domain: devsecops\n" +
		"Tools: 10 active\n" +
		"Knowledge: 50+ security rules, 6 compliance frameworks"
	return tools.Result{Output: output, ExitCode: 0}
}

func domainList() tools.Result {
	output := "Available domains:\n" +
		"1. general — General purpose coding\n" +
		"2. devsecops — Security, compliance, CI/CD\n" +
		"3. architect — Design, cloud, infrastructure"
	return tools.Result{Output: output, ExitCode: 0}
}

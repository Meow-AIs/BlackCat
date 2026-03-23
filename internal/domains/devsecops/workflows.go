package devsecops

import (
	"fmt"
	"strings"
)

// DevSecOpsWorkflow defines a pre-built security workflow.
type DevSecOpsWorkflow struct {
	Name        string
	Description string
	Trigger     string
	Steps       []WFStep
	Tags        []string
}

// WFStep is a single step in a workflow.
type WFStep struct {
	Name        string
	Tool        string
	Description string
}

// LoadDevSecOpsWorkflows returns all 10 built-in DevSecOps workflows.
func LoadDevSecOpsWorkflows() []DevSecOpsWorkflow {
	return []DevSecOpsWorkflow{
		{
			Name:        "pre-commit-security",
			Description: "Run security checks before committing code",
			Trigger:     "git pre-commit hook or manual invocation",
			Tags:        []string{"ci", "secrets", "sast"},
			Steps: []WFStep{
				{Name: "secret-scan", Tool: "scan_secrets", Description: "Scan staged files for hardcoded secrets and credentials"},
				{Name: "code-scan", Tool: "code-scanner", Description: "Run SAST-lite rules against changed files"},
				{Name: "dependency-check", Tool: "dep-audit", Description: "Check for known vulnerabilities in dependencies"},
			},
		},
		{
			Name:        "dependency-audit",
			Description: "Full dependency audit with vulnerability prioritization",
			Trigger:     "new dependency added, scheduled weekly, or manual",
			Tags:        []string{"sca", "sbom", "vulnerabilities"},
			Steps: []WFStep{
				{Name: "parse-manifests", Tool: "sbom-generator", Description: "Parse package manifests (go.mod, package.json, requirements.txt)"},
				{Name: "osv-query", Tool: "osv-scanner", Description: "Query OSV database for known vulnerabilities"},
				{Name: "epss-score", Tool: "vuln-priority", Description: "Enrich with EPSS scores for exploitation probability"},
				{Name: "prioritize", Tool: "vuln-priority", Description: "Prioritize vulnerabilities using KEV+EPSS+CVSS+reachability"},
			},
		},
		{
			Name:        "container-hardening",
			Description: "Audit and harden container images",
			Trigger:     "Dockerfile change or container image build",
			Tags:        []string{"containers", "docker", "hardening"},
			Steps: []WFStep{
				{Name: "dockerfile-scan", Tool: "dockerfile-scanner", Description: "Analyze Dockerfile for best practice violations"},
				{Name: "image-scan", Tool: "trivy-sarif", Description: "Scan built image for vulnerabilities via SARIF output"},
				{Name: "recommend-fixes", Tool: "remediation", Description: "Generate fix recommendations for discovered issues"},
			},
		},
		{
			Name:        "k8s-security-review",
			Description: "Kubernetes manifest security review",
			Trigger:     "K8s manifest change or cluster audit",
			Tags:        []string{"kubernetes", "cis-benchmark", "pss"},
			Steps: []WFStep{
				{Name: "manifest-scan", Tool: "k8s-scanner", Description: "Scan K8s manifests for security misconfigurations"},
				{Name: "cis-benchmark", Tool: "cis-checker", Description: "Evaluate against CIS Kubernetes Benchmark controls"},
				{Name: "pss-check", Tool: "pss-validator", Description: "Validate Pod Security Standards (restricted/baseline/privileged)"},
			},
		},
		{
			Name:        "iac-security-review",
			Description: "Infrastructure-as-Code security scanning",
			Trigger:     "Terraform/CloudFormation/K8s manifest change",
			Tags:        []string{"iac", "terraform", "compliance"},
			Steps: []WFStep{
				{Name: "iac-scan", Tool: "iac-scanner", Description: "Scan Terraform/K8s/CloudFormation for misconfigurations"},
				{Name: "findings-analysis", Tool: "sarif-parser", Description: "Parse scanner output in SARIF format"},
				{Name: "compliance-map", Tool: "compliance-mapper", Description: "Map findings to compliance framework controls"},
			},
		},
		{
			Name:        "incident-triage",
			Description: "Triage security incidents using threat intelligence",
			Trigger:     "CVE alert, security incident, or vulnerability disclosure",
			Tags:        []string{"incident-response", "threat-intel", "triage"},
			Steps: []WFStep{
				{Name: "kev-crossref", Tool: "kev-checker", Description: "Cross-reference with CISA KEV catalog"},
				{Name: "epss-priority", Tool: "vuln-priority", Description: "Compute EPSS-based exploitation probability"},
				{Name: "impact-assessment", Tool: "impact-analyzer", Description: "Assess business impact of affected systems"},
				{Name: "timeline", Tool: "incident-timeline", Description: "Build remediation timeline with SLA targets"},
			},
		},
		{
			Name:        "compliance-gap-analysis",
			Description: "Analyze compliance gaps against security frameworks",
			Trigger:     "audit preparation, quarterly review, or manual",
			Tags:        []string{"compliance", "audit", "frameworks"},
			Steps: []WFStep{
				{Name: "scan", Tool: "multi-scanner", Description: "Run all available security scanners"},
				{Name: "map-framework", Tool: "compliance-mapper", Description: "Map findings to selected compliance framework controls"},
				{Name: "gap-report", Tool: "gap-reporter", Description: "Generate gap analysis report with coverage score"},
			},
		},
		{
			Name:        "sbom-pipeline",
			Description: "Generate and analyze Software Bill of Materials",
			Trigger:     "release build, dependency change, or scheduled",
			Tags:        []string{"sbom", "supply-chain", "sca"},
			Steps: []WFStep{
				{Name: "detect-manifests", Tool: "manifest-detector", Description: "Detect all package manifests in the project"},
				{Name: "generate-sbom", Tool: "sbom-generator", Description: "Generate CycloneDX SBOM from detected manifests"},
				{Name: "vuln-scan", Tool: "osv-scanner", Description: "Scan SBOM components for known vulnerabilities"},
			},
		},
		{
			Name:        "threat-model",
			Description: "Systematic threat modeling using STRIDE methodology",
			Trigger:     "new feature design, architecture review, or manual",
			Tags:        []string{"threat-modeling", "stride", "design"},
			Steps: []WFStep{
				{Name: "identify-assets", Tool: "asset-inventory", Description: "Identify system assets, data flows, and trust boundaries"},
				{Name: "stride-analysis", Tool: "stride-analyzer", Description: "Apply STRIDE per data flow element (Spoofing, Tampering, Repudiation, Info Disclosure, DoS, Elevation)"},
				{Name: "attack-tree", Tool: "attack-tree-builder", Description: "Build attack trees for identified threats"},
				{Name: "mitigations", Tool: "mitigation-planner", Description: "Recommend mitigations for each identified threat"},
			},
		},
		{
			Name:        "pentest-assist",
			Description: "Guided penetration testing based on OWASP WSTG",
			Trigger:     "scheduled pentest, security assessment, or manual",
			Tags:        []string{"pentest", "owasp", "wstg"},
			Steps: []WFStep{
				{Name: "wstg-guide", Tool: "wstg-checklist", Description: "Load OWASP Web Security Testing Guide checklist"},
				{Name: "checklist", Tool: "pentest-checklist", Description: "Generate targeted checklist based on application stack"},
				{Name: "finding-templates", Tool: "finding-template", Description: "Provide finding report templates for discovered issues"},
			},
		},
	}
}

// GetDevSecOpsWorkflow finds a workflow by name.
func GetDevSecOpsWorkflow(name string) (DevSecOpsWorkflow, bool) {
	for _, w := range LoadDevSecOpsWorkflows() {
		if w.Name == name {
			return w, true
		}
	}
	return DevSecOpsWorkflow{}, false
}

// ListDevSecOpsWorkflows is an alias for LoadDevSecOpsWorkflows.
func ListDevSecOpsWorkflows() []DevSecOpsWorkflow {
	return LoadDevSecOpsWorkflows()
}

// FormatMarkdown renders the workflow as markdown documentation.
func (w DevSecOpsWorkflow) FormatMarkdown() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## %s\n\n", w.Name))
	sb.WriteString(fmt.Sprintf("%s\n\n", w.Description))
	sb.WriteString(fmt.Sprintf("**Trigger**: %s\n\n", w.Trigger))
	sb.WriteString(fmt.Sprintf("**Tags**: %s\n\n", strings.Join(w.Tags, ", ")))
	sb.WriteString("### Steps\n\n")
	for i, s := range w.Steps {
		sb.WriteString(fmt.Sprintf("%d. **%s** (`%s`): %s\n", i+1, s.Name, s.Tool, s.Description))
	}
	return sb.String()
}

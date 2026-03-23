package devsecops

import (
	"fmt"
	"strings"
)

// ComplianceFramework identifies a security/compliance standard.
type ComplianceFramework string

const (
	FrameworkSOC2     ComplianceFramework = "SOC2"
	FrameworkISO27001 ComplianceFramework = "ISO27001"
	FrameworkPCIDSS   ComplianceFramework = "PCI-DSS"
	FrameworkCIS      ComplianceFramework = "CIS"
	FrameworkNIST     ComplianceFramework = "NIST-CSF"
	FrameworkHIPAA    ComplianceFramework = "HIPAA"
)

// ComplianceControl represents a single control within a framework.
type ComplianceControl struct {
	ID          string
	Framework   ComplianceFramework
	Title       string
	Description string
	Category    string // "access-control", "encryption", "logging", etc.
}

// ComplianceMapping links a finding category to relevant controls.
type ComplianceMapping struct {
	FindingCategory string
	Controls        []ComplianceControl
}

// ComplianceGapReport summarizes coverage for a framework.
type ComplianceGapReport struct {
	Framework     ComplianceFramework
	TotalControls int
	Covered       int
	Gaps          []ComplianceControl
	Score         float64 // 0-100
}

// ListFrameworks returns all supported compliance frameworks.
func ListFrameworks() []ComplianceFramework {
	return []ComplianceFramework{
		FrameworkSOC2, FrameworkISO27001, FrameworkPCIDSS,
		FrameworkCIS, FrameworkNIST, FrameworkHIPAA,
	}
}

// LoadComplianceMappings returns the built-in set of finding-to-control mappings.
func LoadComplianceMappings() []ComplianceMapping {
	return []ComplianceMapping{
		{
			FindingCategory: "hardcoded-secret",
			Controls: []ComplianceControl{
				{ID: "SOC2-CC6.1", Framework: FrameworkSOC2, Title: "Logical Access Security", Description: "Restrict access to information assets", Category: "access-control"},
				{ID: "ISO-A.9.4.3", Framework: FrameworkISO27001, Title: "Password Management", Description: "Manage passwords and secrets securely", Category: "access-control"},
				{ID: "PCI-6.5.3", Framework: FrameworkPCIDSS, Title: "Insecure Cryptographic Storage", Description: "Protect stored secrets", Category: "encryption"},
				{ID: "CIS-5.2", Framework: FrameworkCIS, Title: "Secret Management", Description: "Use a secrets management solution", Category: "access-control"},
				{ID: "NIST-PR.AC-1", Framework: FrameworkNIST, Title: "Access Control", Description: "Identities and credentials are issued and managed", Category: "access-control"},
				{ID: "HIPAA-164.312(a)", Framework: FrameworkHIPAA, Title: "Access Control", Description: "Implement technical policies for electronic information", Category: "access-control"},
			},
		},
		{
			FindingCategory: "no-encryption",
			Controls: []ComplianceControl{
				{ID: "SOC2-CC6.7", Framework: FrameworkSOC2, Title: "Encryption in Transit", Description: "Data encrypted during transmission", Category: "encryption"},
				{ID: "ISO-A.10.1.1", Framework: FrameworkISO27001, Title: "Cryptographic Controls", Description: "Policy on the use of cryptographic controls", Category: "encryption"},
				{ID: "PCI-4.1", Framework: FrameworkPCIDSS, Title: "Strong Cryptography", Description: "Protect cardholder data with strong cryptography during transmission", Category: "encryption"},
				{ID: "CIS-14.4", Framework: FrameworkCIS, Title: "Encrypt All Sensitive Data in Transit", Description: "Encrypt all sensitive data in transit", Category: "encryption"},
				{ID: "NIST-PR.DS-2", Framework: FrameworkNIST, Title: "Data-in-Transit Protection", Description: "Data in transit is protected", Category: "encryption"},
				{ID: "HIPAA-164.312(e)", Framework: FrameworkHIPAA, Title: "Transmission Security", Description: "Guard against unauthorized access to ePHI in transit", Category: "encryption"},
			},
		},
		{
			FindingCategory: "public-bucket",
			Controls: []ComplianceControl{
				{ID: "SOC2-CC6.3", Framework: FrameworkSOC2, Title: "Access Restrictions", Description: "Limit access based on authorization", Category: "access-control"},
				{ID: "ISO-A.9.1.2", Framework: FrameworkISO27001, Title: "Access to Networks", Description: "Restrict access to network services", Category: "access-control"},
				{ID: "CIS-3.3", Framework: FrameworkCIS, Title: "Cloud Storage Access", Description: "Ensure cloud storage is not publicly accessible", Category: "access-control"},
				{ID: "NIST-PR.AC-4", Framework: FrameworkNIST, Title: "Access Permissions", Description: "Access permissions are managed with least privilege", Category: "access-control"},
			},
		},
		{
			FindingCategory: "no-logging",
			Controls: []ComplianceControl{
				{ID: "SOC2-CC7.2", Framework: FrameworkSOC2, Title: "System Monitoring", Description: "Monitor system components for anomalies", Category: "logging"},
				{ID: "ISO-A.12.4.1", Framework: FrameworkISO27001, Title: "Event Logging", Description: "Record user activities, exceptions, and events", Category: "logging"},
				{ID: "PCI-10.2", Framework: FrameworkPCIDSS, Title: "Audit Trail", Description: "Implement automated audit trails for all system components", Category: "logging"},
				{ID: "CIS-8.2", Framework: FrameworkCIS, Title: "Audit Log Management", Description: "Collect and manage audit logs", Category: "logging"},
				{ID: "NIST-DE.AE-3", Framework: FrameworkNIST, Title: "Event Data Aggregation", Description: "Event data are collected and correlated", Category: "logging"},
				{ID: "HIPAA-164.312(b)", Framework: FrameworkHIPAA, Title: "Audit Controls", Description: "Record and examine access and activity", Category: "logging"},
			},
		},
		{
			FindingCategory: "weak-auth",
			Controls: []ComplianceControl{
				{ID: "SOC2-CC6.1", Framework: FrameworkSOC2, Title: "Logical Access Security", Description: "Restrict access to information assets", Category: "access-control"},
				{ID: "ISO-A.9.4.2", Framework: FrameworkISO27001, Title: "Secure Log-on", Description: "Controlled by secure log-on procedures", Category: "access-control"},
				{ID: "PCI-8.2", Framework: FrameworkPCIDSS, Title: "User Authentication", Description: "Employ at least one method to authenticate users", Category: "access-control"},
				{ID: "CIS-4.1", Framework: FrameworkCIS, Title: "Secure Authentication", Description: "Establish and maintain a secure authentication process", Category: "access-control"},
				{ID: "NIST-PR.AC-7", Framework: FrameworkNIST, Title: "Authentication", Description: "Users and devices are authenticated", Category: "access-control"},
				{ID: "HIPAA-164.312(d)", Framework: FrameworkHIPAA, Title: "Person Authentication", Description: "Verify identity of persons seeking ePHI access", Category: "access-control"},
			},
		},
		{
			FindingCategory: "no-mfa",
			Controls: []ComplianceControl{
				{ID: "SOC2-CC6.1", Framework: FrameworkSOC2, Title: "Multi-factor Authentication", Description: "MFA for access to sensitive systems", Category: "access-control"},
				{ID: "ISO-A.9.4.2", Framework: FrameworkISO27001, Title: "Multi-factor Authentication", Description: "Use MFA where appropriate", Category: "access-control"},
				{ID: "PCI-8.3", Framework: FrameworkPCIDSS, Title: "MFA", Description: "Secure all individual non-console admin access with MFA", Category: "access-control"},
				{ID: "CIS-4.5", Framework: FrameworkCIS, Title: "MFA", Description: "Implement MFA for all remote access", Category: "access-control"},
				{ID: "NIST-PR.AC-7", Framework: FrameworkNIST, Title: "Strong Authentication", Description: "Multi-factor authentication for access", Category: "access-control"},
			},
		},
		{
			FindingCategory: "missing-backup",
			Controls: []ComplianceControl{
				{ID: "SOC2-A1.2", Framework: FrameworkSOC2, Title: "Recovery Procedures", Description: "Environmental protections and recovery procedures", Category: "incident-response"},
				{ID: "ISO-A.12.3.1", Framework: FrameworkISO27001, Title: "Information Backup", Description: "Backup copies of information shall be taken and tested", Category: "incident-response"},
				{ID: "CIS-11.2", Framework: FrameworkCIS, Title: "Automated Backups", Description: "Perform automated backups", Category: "incident-response"},
				{ID: "NIST-PR.IP-4", Framework: FrameworkNIST, Title: "Backups", Description: "Backups of information are conducted and tested", Category: "incident-response"},
			},
		},
		{
			FindingCategory: "sql-injection",
			Controls: []ComplianceControl{
				{ID: "SOC2-CC7.1", Framework: FrameworkSOC2, Title: "Vulnerability Detection", Description: "Detect and address software vulnerabilities", Category: "secure-coding"},
				{ID: "ISO-A.14.2.1", Framework: FrameworkISO27001, Title: "Secure Development", Description: "Rules for development of software shall be established", Category: "secure-coding"},
				{ID: "PCI-6.5.1", Framework: FrameworkPCIDSS, Title: "Injection Flaws", Description: "Prevent injection flaws (SQL, OS, LDAP)", Category: "secure-coding"},
				{ID: "NIST-PR.IP-12", Framework: FrameworkNIST, Title: "Vulnerability Management", Description: "Technical vulnerability management implemented", Category: "secure-coding"},
			},
		},
		{
			FindingCategory: "xss",
			Controls: []ComplianceControl{
				{ID: "SOC2-CC7.1", Framework: FrameworkSOC2, Title: "Vulnerability Detection", Description: "Detect and address software vulnerabilities", Category: "secure-coding"},
				{ID: "PCI-6.5.7", Framework: FrameworkPCIDSS, Title: "XSS Prevention", Description: "Address cross-site scripting (XSS) vulnerabilities", Category: "secure-coding"},
				{ID: "CIS-16.5", Framework: FrameworkCIS, Title: "Input Validation", Description: "Validate all input", Category: "secure-coding"},
			},
		},
		{
			FindingCategory: "command-injection",
			Controls: []ComplianceControl{
				{ID: "PCI-6.5.1", Framework: FrameworkPCIDSS, Title: "OS Command Injection", Description: "Prevent OS command injection", Category: "secure-coding"},
				{ID: "NIST-PR.IP-12", Framework: FrameworkNIST, Title: "Secure Coding", Description: "Secure software development practices", Category: "secure-coding"},
				{ID: "ISO-A.14.2.5", Framework: FrameworkISO27001, Title: "Secure System Engineering", Description: "Principles for engineering secure systems", Category: "secure-coding"},
			},
		},
		{
			FindingCategory: "insecure-deserialization",
			Controls: []ComplianceControl{
				{ID: "PCI-6.5.8", Framework: FrameworkPCIDSS, Title: "Insecure Deserialization", Description: "Prevent insecure deserialization", Category: "secure-coding"},
				{ID: "NIST-PR.DS-5", Framework: FrameworkNIST, Title: "Data Leak Prevention", Description: "Protections against data leaks are implemented", Category: "secure-coding"},
			},
		},
		{
			FindingCategory: "weak-crypto",
			Controls: []ComplianceControl{
				{ID: "SOC2-CC6.7", Framework: FrameworkSOC2, Title: "Strong Cryptography", Description: "Use strong cryptographic algorithms", Category: "encryption"},
				{ID: "ISO-A.10.1.1", Framework: FrameworkISO27001, Title: "Crypto Policy", Description: "Use approved cryptographic algorithms", Category: "encryption"},
				{ID: "PCI-4.1", Framework: FrameworkPCIDSS, Title: "Strong Cryptography", Description: "Use strong cryptography protocols", Category: "encryption"},
				{ID: "NIST-PR.DS-5", Framework: FrameworkNIST, Title: "Cryptographic Standards", Description: "Apply cryptographic standards", Category: "encryption"},
				{ID: "HIPAA-164.312(a)(2)(iv)", Framework: FrameworkHIPAA, Title: "Encryption", Description: "Encrypt ePHI as appropriate", Category: "encryption"},
			},
		},
		{
			FindingCategory: "open-redirect",
			Controls: []ComplianceControl{
				{ID: "PCI-6.5.10", Framework: FrameworkPCIDSS, Title: "Unvalidated Redirects", Description: "Prevent unvalidated redirects and forwards", Category: "secure-coding"},
				{ID: "CIS-16.5", Framework: FrameworkCIS, Title: "Input Validation", Description: "Validate redirect targets", Category: "secure-coding"},
			},
		},
		{
			FindingCategory: "ssrf",
			Controls: []ComplianceControl{
				{ID: "PCI-6.5.9", Framework: FrameworkPCIDSS, Title: "SSRF Prevention", Description: "Prevent server-side request forgery", Category: "secure-coding"},
				{ID: "NIST-PR.AC-5", Framework: FrameworkNIST, Title: "Network Integrity", Description: "Network integrity is protected", Category: "access-control"},
			},
		},
		{
			FindingCategory: "no-rate-limiting",
			Controls: []ComplianceControl{
				{ID: "SOC2-CC6.1", Framework: FrameworkSOC2, Title: "Denial of Service Protection", Description: "Implement rate limiting", Category: "access-control"},
				{ID: "CIS-13.4", Framework: FrameworkCIS, Title: "Rate Limiting", Description: "Apply rate limiting to prevent abuse", Category: "access-control"},
				{ID: "NIST-PR.PT-4", Framework: FrameworkNIST, Title: "Communication Protection", Description: "Protect availability of communications", Category: "access-control"},
			},
		},
		{
			FindingCategory: "excessive-permissions",
			Controls: []ComplianceControl{
				{ID: "SOC2-CC6.3", Framework: FrameworkSOC2, Title: "Least Privilege", Description: "Enforce least privilege access", Category: "access-control"},
				{ID: "ISO-A.9.2.3", Framework: FrameworkISO27001, Title: "Privilege Management", Description: "Restrict and control allocation of privileges", Category: "access-control"},
				{ID: "PCI-7.1", Framework: FrameworkPCIDSS, Title: "Least Privilege", Description: "Limit access to system components", Category: "access-control"},
				{ID: "NIST-PR.AC-4", Framework: FrameworkNIST, Title: "Least Privilege", Description: "Manage access permissions incorporating least privilege", Category: "access-control"},
			},
		},
	}
}

// MapFindingToControls returns compliance controls for a given finding category.
func MapFindingToControls(findingCategory string) []ComplianceControl {
	for _, m := range LoadComplianceMappings() {
		if m.FindingCategory == findingCategory {
			result := make([]ComplianceControl, len(m.Controls))
			copy(result, m.Controls)
			return result
		}
	}
	return nil
}

// GenerateGapReport evaluates which controls in a framework are addressed by
// findings and which remain gaps.
func GenerateGapReport(framework ComplianceFramework, findings []Finding) ComplianceGapReport {
	// Collect all controls for this framework
	allControls := controlsForFramework(framework)

	// Determine which controls are covered by the findings
	coveredIDs := map[string]bool{}
	for _, f := range findings {
		category := findingCategory(f)
		if category == "" {
			continue
		}
		for _, c := range MapFindingToControls(category) {
			if c.Framework == framework {
				coveredIDs[c.ID] = true
			}
		}
	}

	var gaps []ComplianceControl
	for _, c := range allControls {
		if !coveredIDs[c.ID] {
			gaps = append(gaps, c)
		}
	}

	total := len(allControls)
	covered := total - len(gaps)
	var score float64
	if total > 0 {
		score = float64(covered) / float64(total) * 100
	}

	return ComplianceGapReport{
		Framework:     framework,
		TotalControls: total,
		Covered:       covered,
		Gaps:          gaps,
		Score:         score,
	}
}

// FormatMarkdown renders the gap report as markdown.
func (r ComplianceGapReport) FormatMarkdown() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Compliance Gap Report: %s\n\n", r.Framework))
	sb.WriteString(fmt.Sprintf("- **Score**: %.1f%%\n", r.Score))
	sb.WriteString(fmt.Sprintf("- **Covered**: %d / %d controls\n", r.Covered, r.TotalControls))
	sb.WriteString(fmt.Sprintf("- **Gaps**: %d controls\n\n", len(r.Gaps)))

	if len(r.Gaps) > 0 {
		sb.WriteString("## Gap Details\n\n")
		sb.WriteString("| Control | Title | Category |\n")
		sb.WriteString("|---------|-------|----------|\n")
		for _, g := range r.Gaps {
			sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n", g.ID, g.Title, g.Category))
		}
	}
	return sb.String()
}

// controlsForFramework collects all unique controls for a framework from all mappings.
func controlsForFramework(framework ComplianceFramework) []ComplianceControl {
	seen := map[string]bool{}
	var controls []ComplianceControl
	for _, m := range LoadComplianceMappings() {
		for _, c := range m.Controls {
			if c.Framework == framework && !seen[c.ID] {
				seen[c.ID] = true
				controls = append(controls, c)
			}
		}
	}
	return controls
}

// findingCategory extracts the category from a finding's metadata.
func findingCategory(f Finding) string {
	if f.Metadata != nil {
		if cat, ok := f.Metadata["category"]; ok {
			return cat
		}
	}
	return ""
}

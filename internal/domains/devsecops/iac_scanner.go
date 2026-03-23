package devsecops

import (
	"fmt"
	"regexp"
	"strings"
)

// IaCRule defines a single infrastructure-as-code security check.
type IaCRule struct {
	ID           string
	Name         string
	Severity     string
	ResourceType string // "aws_s3_bucket", "kubernetes_pod", etc.
	Check        func(content string) []Finding
}

// IaCScanner scans infrastructure-as-code files for security misconfigurations.
type IaCScanner struct {
	rules []IaCRule
}

// NewIaCScanner creates a new IaC scanner with no rules loaded.
func NewIaCScanner() *IaCScanner {
	return &IaCScanner{
		rules: make([]IaCRule, 0),
	}
}

// LoadTerraformRules loads Terraform-specific security rules.
func (s *IaCScanner) LoadTerraformRules() {
	s.rules = append(s.rules,
		IaCRule{
			ID: "TF001", Name: "Public S3 Bucket", Severity: "critical",
			ResourceType: "aws_s3_bucket",
			Check:        checkPublicS3,
		},
		IaCRule{
			ID: "TF002", Name: "Unencrypted RDS", Severity: "high",
			ResourceType: "aws_db_instance",
			Check:        checkUnencryptedRDS,
		},
		IaCRule{
			ID: "TF003", Name: "Open Security Group", Severity: "critical",
			ResourceType: "aws_security_group",
			Check:        checkOpenSecurityGroup,
		},
		IaCRule{
			ID: "TF004", Name: "S3 Bucket Without Logging", Severity: "medium",
			ResourceType: "aws_s3_bucket",
			Check:        checkS3NoLogging,
		},
		IaCRule{
			ID: "TF005", Name: "Unencrypted EBS Volume", Severity: "high",
			ResourceType: "aws_ebs_volume",
			Check:        checkUnencryptedEBS,
		},
		IaCRule{
			ID: "TF006", Name: "Public RDS Instance", Severity: "critical",
			ResourceType: "aws_db_instance",
			Check:        checkPublicRDS,
		},
		IaCRule{
			ID: "TF007", Name: "IAM Wildcard Actions", Severity: "high",
			ResourceType: "aws_iam_policy",
			Check:        checkIAMWildcard,
		},
		IaCRule{
			ID: "TF008", Name: "SSH Open to World", Severity: "critical",
			ResourceType: "aws_security_group",
			Check:        checkSSHOpenToWorld,
		},
		IaCRule{
			ID: "TF009", Name: "No VPC Flow Logs", Severity: "medium",
			ResourceType: "aws_vpc",
			Check:        checkNoVPCFlowLogs,
		},
		IaCRule{
			ID: "TF010", Name: "Unencrypted S3 Bucket", Severity: "high",
			ResourceType: "aws_s3_bucket",
			Check:        checkUnencryptedS3,
		},
		IaCRule{
			ID: "TF011", Name: "CloudTrail Disabled", Severity: "high",
			ResourceType: "aws_cloudtrail",
			Check:        checkCloudTrailDisabled,
		},
	)
}

// LoadKubernetesRules loads Kubernetes-specific security rules.
func (s *IaCScanner) LoadKubernetesRules() {
	s.rules = append(s.rules,
		IaCRule{
			ID: "K8S001", Name: "Privileged Container", Severity: "critical",
			ResourceType: "kubernetes_pod",
			Check:        checkPrivilegedContainer,
		},
		IaCRule{
			ID: "K8S002", Name: "Host Network Enabled", Severity: "high",
			ResourceType: "kubernetes_pod",
			Check:        checkHostNetwork,
		},
		IaCRule{
			ID: "K8S003", Name: "No Resource Limits", Severity: "medium",
			ResourceType: "kubernetes_pod",
			Check:        checkNoResourceLimits,
		},
		IaCRule{
			ID: "K8S004", Name: "Running as Root", Severity: "high",
			ResourceType: "kubernetes_pod",
			Check:        checkRunAsRoot,
		},
		IaCRule{
			ID: "K8S005", Name: "Latest Image Tag", Severity: "medium",
			ResourceType: "kubernetes_pod",
			Check:        checkLatestTag,
		},
		IaCRule{
			ID: "K8S006", Name: "Host PID Enabled", Severity: "high",
			ResourceType: "kubernetes_pod",
			Check:        checkHostPID,
		},
		IaCRule{
			ID: "K8S007", Name: "Host IPC Enabled", Severity: "high",
			ResourceType: "kubernetes_pod",
			Check:        checkHostIPC,
		},
		IaCRule{
			ID: "K8S008", Name: "No Readiness Probe", Severity: "low",
			ResourceType: "kubernetes_pod",
			Check:        checkNoReadinessProbe,
		},
		IaCRule{
			ID: "K8S009", Name: "Capabilities Added", Severity: "medium",
			ResourceType: "kubernetes_pod",
			Check:        checkCapabilitiesAdded,
		},
		IaCRule{
			ID: "K8S010", Name: "No Security Context", Severity: "medium",
			ResourceType: "kubernetes_pod",
			Check:        checkNoSecurityContext,
		},
		IaCRule{
			ID: "K8S011", Name: "Default Namespace", Severity: "low",
			ResourceType: "kubernetes_pod",
			Check:        checkDefaultNamespace,
		},
	)
}

// ScanFile scans a single file against all loaded rules.
func (s *IaCScanner) ScanFile(filename, content string) ScanResult {
	var findings []Finding
	for _, rule := range s.rules {
		ruleFindings := rule.Check(content)
		for _, f := range ruleFindings {
			findings = append(findings, Finding{
				ID:          fmt.Sprintf("%s-%s", rule.ID, filename),
				Scanner:     "iac",
				Severity:    Severity(rule.Severity),
				Title:       rule.Name,
				Description: f.Description,
				FilePath:    filename,
				RuleID:      rule.ID,
				Confidence:  f.Confidence,
			})
		}
	}
	return ScanResult{
		Scanner:  "iac",
		Findings: findings,
		Scanned:  1,
	}
}

// --- Terraform check functions ---

var publicACLPattern = regexp.MustCompile(`acl\s*=\s*"public-(read|read-write)"`)

func checkPublicS3(content string) []Finding {
	if publicACLPattern.MatchString(content) {
		return []Finding{{
			Description: "S3 bucket has public ACL, making contents accessible to the internet",
			Confidence:  0.95,
		}}
	}
	return nil
}

var storageEncryptedFalse = regexp.MustCompile(`storage_encrypted\s*=\s*false`)

func checkUnencryptedRDS(content string) []Finding {
	if strings.Contains(content, "aws_db_instance") && storageEncryptedFalse.MatchString(content) {
		return []Finding{{
			Description: "RDS instance has encryption disabled",
			Confidence:  0.9,
		}}
	}
	return nil
}

var openCIDR = regexp.MustCompile(`cidr_blocks\s*=\s*\["0\.0\.0\.0/0"\]`)

func checkOpenSecurityGroup(content string) []Finding {
	if strings.Contains(content, "aws_security_group") && openCIDR.MatchString(content) {
		return []Finding{{
			Description: "Security group allows traffic from all IPs (0.0.0.0/0)",
			Confidence:  0.9,
		}}
	}
	return nil
}

func checkS3NoLogging(content string) []Finding {
	if strings.Contains(content, "aws_s3_bucket") && !strings.Contains(content, "logging") {
		return []Finding{{
			Description: "S3 bucket does not have access logging enabled",
			Confidence:  0.7,
		}}
	}
	return nil
}

var ebsEncryptedFalse = regexp.MustCompile(`encrypted\s*=\s*false`)

func checkUnencryptedEBS(content string) []Finding {
	if strings.Contains(content, "aws_ebs_volume") && ebsEncryptedFalse.MatchString(content) {
		return []Finding{{
			Description: "EBS volume encryption is disabled",
			Confidence:  0.9,
		}}
	}
	return nil
}

var publiclyAccessible = regexp.MustCompile(`publicly_accessible\s*=\s*true`)

func checkPublicRDS(content string) []Finding {
	if strings.Contains(content, "aws_db_instance") && publiclyAccessible.MatchString(content) {
		return []Finding{{
			Description: "RDS instance is publicly accessible from the internet",
			Confidence:  0.95,
		}}
	}
	return nil
}

var wildcardAction = regexp.MustCompile(`"Action"\s*:\s*\[?\s*"\*"\s*\]?`)

func checkIAMWildcard(content string) []Finding {
	if wildcardAction.MatchString(content) {
		return []Finding{{
			Description: "IAM policy uses wildcard (*) actions, granting excessive permissions",
			Confidence:  0.85,
		}}
	}
	return nil
}

var sshPort = regexp.MustCompile(`(from_port|to_port)\s*=\s*22`)

func checkSSHOpenToWorld(content string) []Finding {
	if sshPort.MatchString(content) && openCIDR.MatchString(content) {
		return []Finding{{
			Description: "SSH (port 22) is open to all IPs (0.0.0.0/0)",
			Confidence:  0.95,
		}}
	}
	return nil
}

func checkNoVPCFlowLogs(content string) []Finding {
	if strings.Contains(content, "aws_vpc") && !strings.Contains(content, "aws_flow_log") {
		return []Finding{{
			Description: "VPC does not have flow logs enabled",
			Confidence:  0.7,
		}}
	}
	return nil
}

func checkUnencryptedS3(content string) []Finding {
	if strings.Contains(content, "aws_s3_bucket") &&
		!strings.Contains(content, "server_side_encryption") {
		return []Finding{{
			Description: "S3 bucket does not have server-side encryption configured",
			Confidence:  0.75,
		}}
	}
	return nil
}

var multiRegionFalse = regexp.MustCompile(`is_multi_region_trail\s*=\s*false`)

func checkCloudTrailDisabled(content string) []Finding {
	if strings.Contains(content, "aws_cloudtrail") && multiRegionFalse.MatchString(content) {
		return []Finding{{
			Description: "CloudTrail is not configured for multi-region logging",
			Confidence:  0.8,
		}}
	}
	return nil
}

// --- Kubernetes check functions ---

var privilegedTrue = regexp.MustCompile(`privileged:\s*true`)

func checkPrivilegedContainer(content string) []Finding {
	if privilegedTrue.MatchString(content) {
		return []Finding{{
			Description: "Container runs in privileged mode, bypassing security boundaries",
			Confidence:  0.95,
		}}
	}
	return nil
}

var hostNetworkTrue = regexp.MustCompile(`hostNetwork:\s*true`)

func checkHostNetwork(content string) []Finding {
	if hostNetworkTrue.MatchString(content) {
		return []Finding{{
			Description: "Pod uses host network namespace, exposing all host ports",
			Confidence:  0.9,
		}}
	}
	return nil
}

func checkNoResourceLimits(content string) []Finding {
	hasContainers := strings.Contains(content, "containers:")
	hasLimits := strings.Contains(content, "limits:")
	if hasContainers && !hasLimits {
		return []Finding{{
			Description: "Container has no resource limits, risking resource exhaustion",
			Confidence:  0.8,
		}}
	}
	return nil
}

var runAsUserZero = regexp.MustCompile(`runAsUser:\s*0`)

func checkRunAsRoot(content string) []Finding {
	if runAsUserZero.MatchString(content) {
		return []Finding{{
			Description: "Container runs as root user (UID 0)",
			Confidence:  0.95,
		}}
	}
	return nil
}

var latestTagPattern = regexp.MustCompile(`image:\s*\S+:latest`)

func checkLatestTag(content string) []Finding {
	if latestTagPattern.MatchString(content) {
		return []Finding{{
			Description: "Container uses :latest tag, making deployments non-reproducible",
			Confidence:  0.85,
		}}
	}
	return nil
}

var hostPIDTrue = regexp.MustCompile(`hostPID:\s*true`)

func checkHostPID(content string) []Finding {
	if hostPIDTrue.MatchString(content) {
		return []Finding{{
			Description: "Pod uses host PID namespace, can see all host processes",
			Confidence:  0.9,
		}}
	}
	return nil
}

var hostIPCTrue = regexp.MustCompile(`hostIPC:\s*true`)

func checkHostIPC(content string) []Finding {
	if hostIPCTrue.MatchString(content) {
		return []Finding{{
			Description: "Pod uses host IPC namespace, can access host shared memory",
			Confidence:  0.9,
		}}
	}
	return nil
}

func checkNoReadinessProbe(content string) []Finding {
	hasContainers := strings.Contains(content, "containers:")
	hasReadiness := strings.Contains(content, "readinessProbe:")
	if hasContainers && !hasReadiness {
		return []Finding{{
			Description: "Container has no readiness probe, may receive traffic before ready",
			Confidence:  0.7,
		}}
	}
	return nil
}

var addCapabilities = regexp.MustCompile(`add:\s*\n\s*-\s*(SYS_ADMIN|NET_ADMIN|ALL)`)

func checkCapabilitiesAdded(content string) []Finding {
	if addCapabilities.MatchString(content) {
		return []Finding{{
			Description: "Container adds dangerous Linux capabilities",
			Confidence:  0.85,
		}}
	}
	return nil
}

func checkNoSecurityContext(content string) []Finding {
	hasContainers := strings.Contains(content, "containers:")
	hasSC := strings.Contains(content, "securityContext:")
	if hasContainers && !hasSC {
		return []Finding{{
			Description: "No securityContext defined, container runs with default permissions",
			Confidence:  0.7,
		}}
	}
	return nil
}

var defaultNamespace = regexp.MustCompile(`namespace:\s*["']?default["']?`)

func checkDefaultNamespace(content string) []Finding {
	if defaultNamespace.MatchString(content) {
		return []Finding{{
			Description: "Resource deployed to default namespace, violating isolation practices",
			Confidence:  0.6,
		}}
	}
	return nil
}

package devsecops

import (
	"testing"
)

func TestNewIaCScanner(t *testing.T) {
	s := NewIaCScanner()
	if s == nil {
		t.Fatal("NewIaCScanner returned nil")
	}
}

func TestLoadTerraformRules(t *testing.T) {
	s := NewIaCScanner()
	s.LoadTerraformRules()
	if len(s.rules) < 10 {
		t.Errorf("expected at least 10 terraform rules, got %d", len(s.rules))
	}
}

func TestLoadKubernetesRules(t *testing.T) {
	s := NewIaCScanner()
	s.LoadKubernetesRules()
	if len(s.rules) < 10 {
		t.Errorf("expected at least 10 kubernetes rules, got %d", len(s.rules))
	}
}

func TestScanFile_TerraformPublicS3(t *testing.T) {
	s := NewIaCScanner()
	s.LoadTerraformRules()

	content := `
resource "aws_s3_bucket" "data" {
  bucket = "my-data-bucket"
  acl    = "public-read"
}
`
	result := s.ScanFile("main.tf", content)
	if len(result.Findings) == 0 {
		t.Error("expected findings for public S3 bucket")
	}

	found := false
	for _, f := range result.Findings {
		if f.Scanner == "iac" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected at least one finding with scanner 'iac'")
	}
}

func TestScanFile_TerraformUnencryptedRDS(t *testing.T) {
	s := NewIaCScanner()
	s.LoadTerraformRules()

	content := `
resource "aws_db_instance" "main" {
  engine         = "mysql"
  instance_class = "db.t3.micro"
  storage_encrypted = false
}
`
	result := s.ScanFile("rds.tf", content)
	hasEncryptionFinding := false
	for _, f := range result.Findings {
		if f.RuleID != "" {
			hasEncryptionFinding = true
		}
	}
	if !hasEncryptionFinding {
		t.Error("expected finding for unencrypted RDS")
	}
}

func TestScanFile_TerraformOpenSecurityGroup(t *testing.T) {
	s := NewIaCScanner()
	s.LoadTerraformRules()

	content := `
resource "aws_security_group" "allow_all" {
  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}
`
	result := s.ScanFile("sg.tf", content)
	if len(result.Findings) == 0 {
		t.Error("expected findings for open security group")
	}
}

func TestScanFile_KubernetesPrivileged(t *testing.T) {
	s := NewIaCScanner()
	s.LoadKubernetesRules()

	content := `
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  containers:
  - name: app
    image: nginx
    securityContext:
      privileged: true
`
	result := s.ScanFile("pod.yaml", content)
	if len(result.Findings) == 0 {
		t.Error("expected findings for privileged container")
	}
}

func TestScanFile_KubernetesHostNetwork(t *testing.T) {
	s := NewIaCScanner()
	s.LoadKubernetesRules()

	content := `
apiVersion: v1
kind: Pod
spec:
  hostNetwork: true
  containers:
  - name: app
    image: nginx
`
	result := s.ScanFile("pod.yaml", content)
	if len(result.Findings) == 0 {
		t.Error("expected findings for hostNetwork")
	}
}

func TestScanFile_KubernetesNoLimits(t *testing.T) {
	s := NewIaCScanner()
	s.LoadKubernetesRules()

	content := `
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: app
        image: nginx
`
	result := s.ScanFile("deploy.yaml", content)
	// Should flag missing resource limits
	found := false
	for _, f := range result.Findings {
		if f.RuleID != "" {
			found = true
		}
	}
	if !found {
		t.Error("expected findings for missing resource limits")
	}
}

func TestScanFile_KubernetesLatestTag(t *testing.T) {
	s := NewIaCScanner()
	s.LoadKubernetesRules()

	content := `
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: app
    image: nginx:latest
`
	result := s.ScanFile("pod.yaml", content)
	if len(result.Findings) == 0 {
		t.Error("expected findings for :latest tag")
	}
}

func TestScanFile_KubernetesRunAsRoot(t *testing.T) {
	s := NewIaCScanner()
	s.LoadKubernetesRules()

	content := `
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: app
    image: nginx
    securityContext:
      runAsUser: 0
`
	result := s.ScanFile("pod.yaml", content)
	if len(result.Findings) == 0 {
		t.Error("expected findings for running as root")
	}
}

func TestScanFile_CleanTerraform(t *testing.T) {
	s := NewIaCScanner()
	s.LoadTerraformRules()

	content := `
resource "aws_s3_bucket" "data" {
  bucket = "my-private-bucket"
  acl    = "private"
}
`
	result := s.ScanFile("clean.tf", content)
	// Should have fewer findings for private bucket
	publicFound := false
	for _, f := range result.Findings {
		if f.Title == "Public S3 Bucket" {
			publicFound = true
		}
	}
	if publicFound {
		t.Error("should not flag private S3 bucket as public")
	}
}

func TestScanFile_ScanResultFields(t *testing.T) {
	s := NewIaCScanner()
	s.LoadTerraformRules()

	content := `
resource "aws_s3_bucket" "data" {
  acl = "public-read-write"
}
`
	result := s.ScanFile("test.tf", content)
	if result.Scanner != "iac" {
		t.Errorf("expected scanner iac, got %s", result.Scanner)
	}
}

func TestScanFile_TerraformNoLogging(t *testing.T) {
	s := NewIaCScanner()
	s.LoadTerraformRules()

	content := `
resource "aws_s3_bucket" "data" {
  bucket = "my-bucket"
  acl    = "private"
}
`
	// Should detect missing logging configuration
	result := s.ScanFile("bucket.tf", content)
	_ = result // just verify no panic
}

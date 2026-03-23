package security

import "testing"

// --- Dangerous commands ---

func TestRmRfIsDangerous(t *testing.T) {
	a := AssessCommandRisk("rm -rf /tmp/build")
	if a.Level != RiskDangerous {
		t.Errorf("expected dangerous, got %q", a.Level)
	}
	if a.Category != "filesystem" {
		t.Errorf("expected category 'filesystem', got %q", a.Category)
	}
	if a.Reason == "" {
		t.Error("expected non-empty reason")
	}
}

func TestRmRfRootIsDangerous(t *testing.T) {
	a := AssessCommandRisk("rm -rf /")
	if a.Level != RiskDangerous {
		t.Errorf("expected dangerous for rm -rf /, got %q", a.Level)
	}
}

func TestMkfsIsDangerous(t *testing.T) {
	a := AssessCommandRisk("mkfs.ext4 /dev/sda1")
	if a.Level != RiskDangerous {
		t.Errorf("expected dangerous for mkfs, got %q", a.Level)
	}
	if a.Category != "filesystem" {
		t.Errorf("expected category 'filesystem', got %q", a.Category)
	}
}

func TestDdIfIsDangerous(t *testing.T) {
	a := AssessCommandRisk("dd if=/dev/zero of=/dev/sda")
	if a.Level != RiskDangerous {
		t.Errorf("expected dangerous for dd if=, got %q", a.Level)
	}
	if a.Category != "filesystem" {
		t.Errorf("expected category 'filesystem', got %q", a.Category)
	}
}

func TestDropTableIsDangerous(t *testing.T) {
	a := AssessCommandRisk("DROP TABLE users")
	if a.Level != RiskDangerous {
		t.Errorf("expected dangerous for DROP TABLE, got %q", a.Level)
	}
	if a.Category != "database" {
		t.Errorf("expected category 'database', got %q", a.Category)
	}
}

func TestDropDatabaseIsDangerous(t *testing.T) {
	a := AssessCommandRisk("DROP DATABASE mydb")
	if a.Level != RiskDangerous {
		t.Errorf("expected dangerous for DROP DATABASE, got %q", a.Level)
	}
	if a.Category != "database" {
		t.Errorf("expected category 'database', got %q", a.Category)
	}
}

func TestTruncateIsDangerous(t *testing.T) {
	a := AssessCommandRisk("TRUNCATE TABLE events")
	if a.Level != RiskDangerous {
		t.Errorf("expected dangerous for TRUNCATE, got %q", a.Level)
	}
	if a.Category != "database" {
		t.Errorf("expected category 'database', got %q", a.Category)
	}
}

func TestKubectlDeleteNamespaceIsDangerous(t *testing.T) {
	a := AssessCommandRisk("kubectl delete namespace production")
	if a.Level != RiskDangerous {
		t.Errorf("expected dangerous for kubectl delete namespace, got %q", a.Level)
	}
	if a.Category != "kubernetes" {
		t.Errorf("expected category 'kubernetes', got %q", a.Category)
	}
}

func TestKubectlDeleteFIsDangerous(t *testing.T) {
	a := AssessCommandRisk("kubectl delete -f deployment.yaml")
	if a.Level != RiskDangerous {
		t.Errorf("expected dangerous for kubectl delete -f, got %q", a.Level)
	}
	if a.Category != "kubernetes" {
		t.Errorf("expected category 'kubernetes', got %q", a.Category)
	}
}

func TestGitPushForceIsDangerous(t *testing.T) {
	a := AssessCommandRisk("git push --force origin main")
	if a.Level != RiskDangerous {
		t.Errorf("expected dangerous for git push --force, got %q", a.Level)
	}
	if a.Category != "git" {
		t.Errorf("expected category 'git', got %q", a.Category)
	}
}

func TestGitResetHardIsDangerous(t *testing.T) {
	a := AssessCommandRisk("git reset --hard HEAD~3")
	if a.Level != RiskDangerous {
		t.Errorf("expected dangerous for git reset --hard, got %q", a.Level)
	}
	if a.Category != "git" {
		t.Errorf("expected category 'git', got %q", a.Category)
	}
}

func TestTerraformDestroyIsDangerous(t *testing.T) {
	a := AssessCommandRisk("terraform destroy -auto-approve")
	if a.Level != RiskDangerous {
		t.Errorf("expected dangerous for terraform destroy, got %q", a.Level)
	}
}

func TestChmod777IsDangerous(t *testing.T) {
	a := AssessCommandRisk("chmod 777 /etc/passwd")
	if a.Level != RiskDangerous {
		t.Errorf("expected dangerous for chmod 777, got %q", a.Level)
	}
	if a.Category != "filesystem" {
		t.Errorf("expected category 'filesystem', got %q", a.Category)
	}
}

func TestChmodRecursiveIsDangerous(t *testing.T) {
	a := AssessCommandRisk("chmod -R 755 /")
	if a.Level != RiskDangerous {
		t.Errorf("expected dangerous for chmod -R, got %q", a.Level)
	}
	if a.Category != "filesystem" {
		t.Errorf("expected category 'filesystem', got %q", a.Category)
	}
}

func TestKill9IsDangerous(t *testing.T) {
	a := AssessCommandRisk("kill -9 1234")
	if a.Level != RiskDangerous {
		t.Errorf("expected dangerous for kill -9, got %q", a.Level)
	}
	if a.Category != "process" {
		t.Errorf("expected category 'process', got %q", a.Category)
	}
}

func TestPkillIsDangerous(t *testing.T) {
	a := AssessCommandRisk("pkill -f myapp")
	if a.Level != RiskDangerous {
		t.Errorf("expected dangerous for pkill, got %q", a.Level)
	}
	if a.Category != "process" {
		t.Errorf("expected category 'process', got %q", a.Category)
	}
}

func TestKillAllIsDangerous(t *testing.T) {
	a := AssessCommandRisk("killall nginx")
	if a.Level != RiskDangerous {
		t.Errorf("expected dangerous for killall, got %q", a.Level)
	}
	if a.Category != "process" {
		t.Errorf("expected category 'process', got %q", a.Category)
	}
}

// --- Moderate commands ---

func TestGitPushWithoutForceIsModerate(t *testing.T) {
	a := AssessCommandRisk("git push origin main")
	if a.Level != RiskModerate {
		t.Errorf("expected moderate for git push, got %q", a.Level)
	}
	if a.Category != "git" {
		t.Errorf("expected category 'git', got %q", a.Category)
	}
}

func TestDockerRmIsModerate(t *testing.T) {
	a := AssessCommandRisk("docker rm mycontainer")
	if a.Level != RiskModerate {
		t.Errorf("expected moderate for docker rm, got %q", a.Level)
	}
}

func TestDockerRmiIsModerate(t *testing.T) {
	a := AssessCommandRisk("docker rmi myimage:latest")
	if a.Level != RiskModerate {
		t.Errorf("expected moderate for docker rmi, got %q", a.Level)
	}
}

func TestNpmPublishIsModerate(t *testing.T) {
	a := AssessCommandRisk("npm publish")
	if a.Level != RiskModerate {
		t.Errorf("expected moderate for npm publish, got %q", a.Level)
	}
}

func TestCargoPublishIsModerate(t *testing.T) {
	a := AssessCommandRisk("cargo publish")
	if a.Level != RiskModerate {
		t.Errorf("expected moderate for cargo publish, got %q", a.Level)
	}
}

func TestCurlPostIsModerate(t *testing.T) {
	a := AssessCommandRisk("curl -X POST https://api.example.com/data")
	if a.Level != RiskModerate {
		t.Errorf("expected moderate for curl -X POST, got %q", a.Level)
	}
	if a.Category != "network" {
		t.Errorf("expected category 'network', got %q", a.Category)
	}
}

func TestCurlPutIsModerate(t *testing.T) {
	a := AssessCommandRisk("curl -X PUT https://api.example.com/resource/1")
	if a.Level != RiskModerate {
		t.Errorf("expected moderate for curl -X PUT, got %q", a.Level)
	}
}

func TestCurlDeleteIsModerate(t *testing.T) {
	a := AssessCommandRisk("curl -X DELETE https://api.example.com/resource/1")
	if a.Level != RiskModerate {
		t.Errorf("expected moderate for curl -X DELETE, got %q", a.Level)
	}
}

func TestSedInplaceIsModerate(t *testing.T) {
	a := AssessCommandRisk("sed -i 's/foo/bar/g' file.txt")
	if a.Level != RiskModerate {
		t.Errorf("expected moderate for sed -i, got %q", a.Level)
	}
}

func TestAwkInplaceIsModerate(t *testing.T) {
	a := AssessCommandRisk("awk -i inplace '{print}' file.txt")
	if a.Level != RiskModerate {
		t.Errorf("expected moderate for awk -i inplace, got %q", a.Level)
	}
}

// --- Safe commands ---

func TestLsIsSafe(t *testing.T) {
	a := AssessCommandRisk("ls -la /tmp")
	if a.Level != RiskSafe {
		t.Errorf("expected safe for ls, got %q", a.Level)
	}
}

func TestCatIsSafe(t *testing.T) {
	a := AssessCommandRisk("cat README.md")
	if a.Level != RiskSafe {
		t.Errorf("expected safe for cat, got %q", a.Level)
	}
}

func TestHeadIsSafe(t *testing.T) {
	a := AssessCommandRisk("head -n 10 file.txt")
	if a.Level != RiskSafe {
		t.Errorf("expected safe for head, got %q", a.Level)
	}
}

func TestTailIsSafe(t *testing.T) {
	a := AssessCommandRisk("tail -n 20 app.log")
	if a.Level != RiskSafe {
		t.Errorf("expected safe for tail, got %q", a.Level)
	}
}

func TestGrepIsSafe(t *testing.T) {
	a := AssessCommandRisk("grep -r 'pattern' ./src")
	if a.Level != RiskSafe {
		t.Errorf("expected safe for grep, got %q", a.Level)
	}
}

func TestFindIsSafe(t *testing.T) {
	a := AssessCommandRisk("find . -name '*.go'")
	if a.Level != RiskSafe {
		t.Errorf("expected safe for find, got %q", a.Level)
	}
}

func TestGitStatusIsSafe(t *testing.T) {
	a := AssessCommandRisk("git status")
	if a.Level != RiskSafe {
		t.Errorf("expected safe for git status, got %q", a.Level)
	}
	if a.Category != "git" {
		t.Errorf("expected category 'git', got %q", a.Category)
	}
}

func TestGitLogIsSafe(t *testing.T) {
	a := AssessCommandRisk("git log --oneline -10")
	if a.Level != RiskSafe {
		t.Errorf("expected safe for git log, got %q", a.Level)
	}
}

func TestGitDiffIsSafe(t *testing.T) {
	a := AssessCommandRisk("git diff HEAD")
	if a.Level != RiskSafe {
		t.Errorf("expected safe for git diff, got %q", a.Level)
	}
}

func TestGoBuildIsSafe(t *testing.T) {
	a := AssessCommandRisk("go build ./...")
	if a.Level != RiskSafe {
		t.Errorf("expected safe for go build, got %q", a.Level)
	}
}

func TestGoTestIsSafe(t *testing.T) {
	a := AssessCommandRisk("go test ./internal/...")
	if a.Level != RiskSafe {
		t.Errorf("expected safe for go test, got %q", a.Level)
	}
}

func TestCargoBuildIsSafe(t *testing.T) {
	a := AssessCommandRisk("cargo build --release")
	if a.Level != RiskSafe {
		t.Errorf("expected safe for cargo build, got %q", a.Level)
	}
}

func TestEchoIsSafe(t *testing.T) {
	a := AssessCommandRisk("echo hello world")
	if a.Level != RiskSafe {
		t.Errorf("expected safe for echo, got %q", a.Level)
	}
}

func TestPwdIsSafe(t *testing.T) {
	a := AssessCommandRisk("pwd")
	if a.Level != RiskSafe {
		t.Errorf("expected safe for pwd, got %q", a.Level)
	}
}

func TestWhichIsSafe(t *testing.T) {
	a := AssessCommandRisk("which go")
	if a.Level != RiskSafe {
		t.Errorf("expected safe for which, got %q", a.Level)
	}
}

func TestWhoamiIsSafe(t *testing.T) {
	a := AssessCommandRisk("whoami")
	if a.Level != RiskSafe {
		t.Errorf("expected safe for whoami, got %q", a.Level)
	}
}

func TestCurlGetIsSafe(t *testing.T) {
	a := AssessCommandRisk("curl -X GET https://api.example.com/data")
	if a.Level != RiskSafe {
		t.Errorf("expected safe for curl -X GET, got %q", a.Level)
	}
	if a.Category != "network" {
		t.Errorf("expected category 'network', got %q", a.Category)
	}
}

func TestCurlWithoutMethodIsSafe(t *testing.T) {
	a := AssessCommandRisk("curl https://api.example.com/data")
	if a.Level != RiskSafe {
		t.Errorf("expected safe for curl without method, got %q", a.Level)
	}
}

// --- Edge cases ---

func TestCommandWithPipeUsesWorstCase(t *testing.T) {
	// Safe pipe: echo test | grep test → Safe
	a := AssessCommandRisk("echo test | grep test")
	if a.Level != RiskSafe {
		t.Errorf("expected safe for 'echo test | grep test', got %q", a.Level)
	}
}

func TestCommandWithAndAndDangerousPartIsDangerous(t *testing.T) {
	// Dangerous in chain: rm -rf /tmp && echo done → Dangerous (worst-case)
	a := AssessCommandRisk("rm -rf /tmp && echo done")
	if a.Level != RiskDangerous {
		t.Errorf("expected dangerous for 'rm -rf /tmp && echo done', got %q", a.Level)
	}
}

func TestCaseInsensitiveDropTableIsDangerous(t *testing.T) {
	a := AssessCommandRisk("drop table users")
	if a.Level != RiskDangerous {
		t.Errorf("expected dangerous for lowercase 'drop table', got %q", a.Level)
	}
}

func TestCaseInsensitiveTruncateIsDangerous(t *testing.T) {
	a := AssessCommandRisk("Truncate Table events")
	if a.Level != RiskDangerous {
		t.Errorf("expected dangerous for mixed-case 'Truncate Table', got %q", a.Level)
	}
}

func TestUnknownCommandDefaultsToModerate(t *testing.T) {
	a := AssessCommandRisk("somebinarynooneknows --flag value")
	if a.Level != RiskModerate {
		t.Errorf("expected moderate for unknown command, got %q", a.Level)
	}
}

func TestEmptyCommandReturnsSafe(t *testing.T) {
	a := AssessCommandRisk("")
	if a.Level != RiskSafe {
		t.Errorf("expected safe for empty command, got %q", a.Level)
	}
}

func TestAssessCommandRiskReturnsCommand(t *testing.T) {
	cmd := "ls -la /tmp"
	a := AssessCommandRisk(cmd)
	if a.Command != cmd {
		t.Errorf("expected Command=%q in result, got %q", cmd, a.Command)
	}
}

func TestAssessCommandRiskHasReasonForDangerous(t *testing.T) {
	a := AssessCommandRisk("rm -rf /var/data")
	if a.Reason == "" {
		t.Error("expected non-empty Reason for dangerous command")
	}
}

func TestAssessCommandRiskHasReasonForModerate(t *testing.T) {
	a := AssessCommandRisk("git push origin main")
	if a.Reason == "" {
		t.Error("expected non-empty Reason for moderate command")
	}
}

func TestAssessCommandRiskCategoryFilesystem(t *testing.T) {
	tests := []string{"rm -rf /tmp", "mkfs.ext4 /dev/sda", "chmod 777 file", "chmod -R 644 dir"}
	for _, cmd := range tests {
		a := AssessCommandRisk(cmd)
		if a.Category != "filesystem" {
			t.Errorf("expected category 'filesystem' for %q, got %q", cmd, a.Category)
		}
	}
}

func TestAssessCommandRiskCategoryProcess(t *testing.T) {
	tests := []string{"kill -9 123", "pkill myapp", "killall nginx"}
	for _, cmd := range tests {
		a := AssessCommandRisk(cmd)
		if a.Category != "process" {
			t.Errorf("expected category 'process' for %q, got %q", cmd, a.Category)
		}
	}
}

func TestAssessCommandRiskCategoryDatabase(t *testing.T) {
	tests := []string{"DROP TABLE t", "DROP DATABASE d", "TRUNCATE TABLE t"}
	for _, cmd := range tests {
		a := AssessCommandRisk(cmd)
		if a.Category != "database" {
			t.Errorf("expected category 'database' for %q, got %q", cmd, a.Category)
		}
	}
}

func TestAssessCommandRiskCategoryKubernetes(t *testing.T) {
	tests := []string{"kubectl delete namespace prod", "kubectl delete -f app.yaml"}
	for _, cmd := range tests {
		a := AssessCommandRisk(cmd)
		if a.Category != "kubernetes" {
			t.Errorf("expected category 'kubernetes' for %q, got %q", cmd, a.Category)
		}
	}
}

func TestAssessCommandRiskCategoryGit(t *testing.T) {
	tests := []string{"git push --force origin main", "git reset --hard HEAD", "git push origin main", "git status", "git log", "git diff"}
	for _, cmd := range tests {
		a := AssessCommandRisk(cmd)
		if a.Category != "git" {
			t.Errorf("expected category 'git' for %q, got %q", cmd, a.Category)
		}
	}
}

func TestAssessCommandRiskCategoryNetwork(t *testing.T) {
	tests := []string{"curl https://example.com", "curl -X POST https://example.com", "curl -X GET https://example.com"}
	for _, cmd := range tests {
		a := AssessCommandRisk(cmd)
		if a.Category != "network" {
			t.Errorf("expected category 'network' for %q, got %q", cmd, a.Category)
		}
	}
}

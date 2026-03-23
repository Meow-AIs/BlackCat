package security

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestSandboxExecuteSimpleCommand(t *testing.T) {
	sb := NewSandbox(SandboxConfig{Timeout: 5 * time.Second})

	cmd := "echo hello"
	if runtime.GOOS == "windows" {
		cmd = "cmd /c echo hello"
	}

	result, err := sb.Execute(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(result.Output, "hello") {
		t.Errorf("expected output containing 'hello', got %q", result.Output)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
}

func TestSandboxExecuteNonZeroExit(t *testing.T) {
	sb := NewSandbox(SandboxConfig{Timeout: 5 * time.Second})

	cmd := "exit 42"
	if runtime.GOOS == "windows" {
		cmd = "cmd /c exit 42"
	}

	result, err := sb.Execute(context.Background(), cmd)
	// Non-zero exit is not a Go error — it's captured in ExitCode
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.ExitCode != 42 {
		t.Errorf("expected exit code 42, got %d", result.ExitCode)
	}
}

func TestSandboxExecuteTimeout(t *testing.T) {
	sb := NewSandbox(SandboxConfig{Timeout: 500 * time.Millisecond})

	// Command that runs longer than timeout
	cmd := "sleep 10"
	if runtime.GOOS == "windows" {
		cmd = "cmd /c ping -n 11 127.0.0.1"
	}

	result, err := sb.Execute(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code for timed-out command")
	}
	if !result.TimedOut {
		t.Error("expected TimedOut to be true")
	}
}

func TestSandboxExecuteOutputCapture(t *testing.T) {
	sb := NewSandbox(SandboxConfig{Timeout: 5 * time.Second})

	cmd := "echo stdout_text && echo stderr_text >&2"
	if runtime.GOOS == "windows" {
		cmd = "cmd /c echo stdout_text & echo stderr_text >&2"
	}

	result, err := sb.Execute(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(result.Output, "stdout_text") {
		t.Errorf("expected stdout captured, got %q", result.Output)
	}
}

func TestSandboxConfigDefaults(t *testing.T) {
	cfg := DefaultSandboxConfig()
	if cfg.Timeout != 120*time.Second {
		t.Errorf("expected default timeout 120s, got %v", cfg.Timeout)
	}
	if cfg.MaxOutputBytes != 1024*1024 {
		t.Errorf("expected max output 1MB, got %d", cfg.MaxOutputBytes)
	}
}

// --- P0 Fix 2: Environment Filtering Tests ---

func TestFilterEnvironmentStripsSecretVars(t *testing.T) {
	sensitiveNames := []string{
		"MY_SECRET",
		"API_KEY",
		"AWS_ACCESS_KEY",
		"DB_PASSWORD",
		"GITHUB_TOKEN",
		"PRIVATE_KEY",
		"AUTH_TOKEN",
		"SIGNING_SECRET",
		"PASSWD",
		"CREDENTIAL_FILE",
		"OPENAI_API_KEY",
	}

	filtered := filterEnvironment()
	filteredMap := make(map[string]bool)
	for _, env := range filtered {
		parts := strings.SplitN(env, "=", 2)
		filteredMap[parts[0]] = true
	}

	for _, name := range sensitiveNames {
		if filteredMap[name] {
			t.Errorf("expected sensitive var %q to be stripped, but it was present", name)
		}
	}
}

func TestFilterEnvironmentPreservesSafeVars(t *testing.T) {
	// These vars may not all exist in every environment, but filterEnvironment
	// must not strip them when they do exist. We inject them via t.Setenv and
	// verify they survive the filter.
	safeVars := map[string]string{
		"PATH":    "/usr/bin:/bin",
		"HOME":    "/home/testuser",
		"USER":    "testuser",
		"TMPDIR":  "/tmp",
		"GOPATH":  "/home/testuser/go",
		"GOROOT":  "/usr/local/go",
		"TERM":    "xterm-256color",
		"LANG":    "en_US.UTF-8",
	}

	for k, v := range safeVars {
		t.Setenv(k, v)
	}

	filtered := filterEnvironment()
	filteredMap := make(map[string]string)
	for _, env := range filtered {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			filteredMap[parts[0]] = parts[1]
		}
	}

	for k, expected := range safeVars {
		got, ok := filteredMap[k]
		if !ok {
			t.Errorf("expected safe var %q to be preserved, but it was stripped", k)
			continue
		}
		if got != expected {
			t.Errorf("expected %q=%q, got %q=%q", k, expected, k, got)
		}
	}
}

func TestSandboxDoesNotLeakSecretEnvVar(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("env var echo test not reliable on windows cmd")
	}

	// Set a secret-looking env var in the current process
	t.Setenv("BLACKCAT_TEST_SECRET_TOKEN", "super_secret_value_12345")

	sb := NewSandbox(SandboxConfig{Timeout: 5 * time.Second})
	// The shell should not see BLACKCAT_TEST_SECRET_TOKEN because it contains TOKEN
	result, err := sb.Execute(context.Background(), `echo "val=$BLACKCAT_TEST_SECRET_TOKEN"`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if strings.Contains(result.Output, "super_secret_value_12345") {
		t.Error("sandbox leaked secret env var to child process")
	}
}

// TestFilterEnvironmentSecretNotInFilteredList verifies at the unit level that
// a secret variable injected into the process environment is absent from the
// filtered list returned by filterEnvironment — runs on all platforms.
func TestFilterEnvironmentSecretNotInFilteredList(t *testing.T) {
	t.Setenv("BLACKCAT_TEST_SECRET_TOKEN", "super_secret_value_12345")

	filtered := filterEnvironment()
	for _, entry := range filtered {
		if strings.Contains(entry, "super_secret_value_12345") {
			t.Errorf("secret value found in filtered environment: %q", entry)
		}
		parts := strings.SplitN(entry, "=", 2)
		if strings.EqualFold(parts[0], "BLACKCAT_TEST_SECRET_TOKEN") {
			t.Errorf("secret env var name %q found in filtered environment", parts[0])
		}
	}
}

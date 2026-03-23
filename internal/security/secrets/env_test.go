package secrets

import (
	"slices"
	"strings"
	"testing"
)

// TestFilterEnvironment_KeepSafeVars verifies that safe variables are preserved.
func TestFilterEnvironment_KeepSafeVars(t *testing.T) {
	input := []string{
		"PATH=/usr/bin:/bin",
		"HOME=/home/user",
		"USER=alice",
		"LANG=en_US.UTF-8",
		"GOPATH=/home/user/go",
		"GOROOT=/usr/local/go",
		"TERM=xterm-256color",
		"SHELL=/bin/bash",
		"PWD=/home/user/projects",
		"TMPDIR=/tmp",
	}

	result := FilterEnvironment(input)

	for _, want := range input {
		if !slices.Contains(result, want) {
			t.Errorf("FilterEnvironment preserved expected var, missing: %q", want)
		}
	}
}

// TestFilterEnvironment_StripAPIKeys verifies API key variables are stripped.
func TestFilterEnvironment_StripAPIKeys(t *testing.T) {
	input := []string{
		"PATH=/usr/bin",
		"OPENAI_API_KEY=sk-proj-abc123",
		"AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		"ANTHROPIC_API_KEY=sk-ant-api03-XXXXXXXX",
		"DB_PASSWORD=supersecret",
		"DATABASE_PASSWORD=hunter2",
		"AUTH_TOKEN=mytoken123",
		"SECRET_KEY=django-insecure-xxx",
		"HOME=/home/user",
	}

	result := FilterEnvironment(input)

	secretVars := []string{
		"OPENAI_API_KEY",
		"AWS_SECRET_ACCESS_KEY",
		"ANTHROPIC_API_KEY",
		"DB_PASSWORD",
		"DATABASE_PASSWORD",
		"AUTH_TOKEN",
		"SECRET_KEY",
	}

	for _, secretVar := range secretVars {
		for _, entry := range result {
			if strings.HasPrefix(entry, secretVar+"=") {
				t.Errorf("FilterEnvironment kept secret var %q in result: %q", secretVar, entry)
			}
		}
	}

	// Safe vars must still be present.
	if !slices.Contains(result, "PATH=/usr/bin") {
		t.Error("FilterEnvironment removed safe var PATH")
	}
	if !slices.Contains(result, "HOME=/home/user") {
		t.Error("FilterEnvironment removed safe var HOME")
	}
}

// TestFilterEnvironment_BlackcatSecretPrefix verifies BLACKCAT_SECRET_* prefix is stripped.
func TestFilterEnvironment_BlackcatSecretPrefix(t *testing.T) {
	input := []string{
		"PATH=/usr/bin",
		"BLACKCAT_SECRET_TOKEN=my-private-token",
		"BLACKCAT_SECRET_API_KEY=my-api-key",
		"BLACKCAT_MODEL=claude-3",
	}

	result := FilterEnvironment(input)

	for _, entry := range result {
		if strings.HasPrefix(entry, "BLACKCAT_SECRET_") {
			t.Errorf("FilterEnvironment kept BLACKCAT_SECRET_* var: %q", entry)
		}
	}

	// Non-secret BLACKCAT_ var should be kept.
	if !slices.Contains(result, "BLACKCAT_MODEL=claude-3") {
		t.Error("FilterEnvironment removed non-secret BLACKCAT_ var")
	}
}

// TestFilterEnvironment_AdditionalSecretPatterns verifies other common secret var names are stripped.
func TestFilterEnvironment_AdditionalSecretPatterns(t *testing.T) {
	secretVars := []string{
		"GITHUB_TOKEN=ghp_XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
		"GITLAB_TOKEN=glpat-XXXXXXXXXXXXXXXXXXXX",
		"NPM_TOKEN=npm_XXXXXXXXXXXXXXXXXXXX",
		"STRIPE_SECRET_KEY=sk_live_XXXXXXXXXXXX",
		"TWILIO_AUTH_TOKEN=ACXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
		"SENDGRID_API_KEY=SG.XXXXXXXXXX.XXXXXXXXXX",
		"REDIS_PASSWORD=redis_secret",
		"POSTGRES_PASSWORD=pg_secret",
		"MYSQL_ROOT_PASSWORD=mysql_secret",
		"JWT_SECRET=jwt_signing_secret",
	}

	input := append([]string{"PATH=/usr/bin"}, secretVars...)
	result := FilterEnvironment(input)

	for _, entry := range secretVars {
		key := strings.SplitN(entry, "=", 2)[0]
		for _, r := range result {
			if strings.HasPrefix(r, key+"=") {
				t.Errorf("FilterEnvironment kept secret var %q", key)
			}
		}
	}
}

// TestFilterEnvironment_EmptyInput handles empty slice without panic.
func TestFilterEnvironment_EmptyInput(t *testing.T) {
	result := FilterEnvironment([]string{})
	if result == nil {
		t.Error("FilterEnvironment(empty) returned nil, want empty slice")
	}
	if len(result) != 0 {
		t.Errorf("FilterEnvironment(empty) returned %d entries, want 0", len(result))
	}
}

// TestFilterEnvironment_NilInput handles nil input without panic.
func TestFilterEnvironment_NilInput(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("FilterEnvironment(nil) panicked: %v", r)
		}
	}()
	result := FilterEnvironment(nil)
	if result == nil {
		t.Error("FilterEnvironment(nil) returned nil, want empty slice")
	}
}

// TestFilterEnvironment_MalformedEntry handles entries without '=' without panic.
func TestFilterEnvironment_MalformedEntry(t *testing.T) {
	input := []string{
		"VALID_KEY=value",
		"MALFORMED_NO_EQUALS",
		"PATH=/usr/bin",
	}
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("FilterEnvironment panicked on malformed entry: %v", r)
		}
	}()
	result := FilterEnvironment(input)
	_ = result // Result content is implementation-defined for malformed entries.
}

// TestFilterEnvironment_DoesNotMutateInput verifies the input slice is not mutated.
func TestFilterEnvironment_DoesNotMutateInput(t *testing.T) {
	input := []string{
		"PATH=/usr/bin",
		"OPENAI_API_KEY=secret",
		"HOME=/home/user",
	}
	original := make([]string, len(input))
	copy(original, input)

	FilterEnvironment(input)

	for i, v := range input {
		if v != original[i] {
			t.Errorf("FilterEnvironment mutated input[%d]: got %q, want %q", i, v, original[i])
		}
	}
}

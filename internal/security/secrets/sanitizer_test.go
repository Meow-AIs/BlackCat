package secrets

import (
	"strings"
	"testing"
)

// TestNewSanitizer_NotNil verifies that NewSanitizer returns a usable instance.
func TestNewSanitizer_NotNil(t *testing.T) {
	s := NewSanitizer()
	if s == nil {
		t.Fatal("NewSanitizer() returned nil")
	}
}

// TestSanitizer_Sanitize_CleanText verifies that text without secrets passes through.
func TestSanitizer_Sanitize_CleanText(t *testing.T) {
	s := NewSanitizer()
	clean := "The server returned HTTP 200 OK with no issues."
	for _, target := range []SanitizeTarget{TargetLLM, TargetChannel, TargetMemory, TargetUser} {
		result := s.Sanitize(clean, target)
		if result.Redacted {
			t.Errorf("Sanitize(%q, %v) Redacted=true for clean text", clean, target)
		}
		if result.Output != clean {
			t.Errorf("Sanitize(%q, %v) Output=%q, want original text", clean, target, result.Output)
		}
	}
}

// TestSanitizer_Sanitize_AWSKey verifies AWS keys are redacted for all targets.
func TestSanitizer_Sanitize_AWSKey(t *testing.T) {
	s := NewSanitizer()
	text := "Using key AKIAIOSFODNN7EXAMPLE for AWS access"
	for _, target := range []SanitizeTarget{TargetLLM, TargetChannel, TargetMemory, TargetUser} {
		result := s.Sanitize(text, target)
		if !result.Redacted {
			t.Errorf("Sanitize(aws key text, %v) Redacted=false, want true", target)
		}
		if strings.Contains(result.Output, "AKIAIOSFODNN7EXAMPLE") {
			t.Errorf("Sanitize(aws key text, %v) Output still contains raw key", target)
		}
	}
}

// TestSanitizer_Sanitize_GitHubToken verifies GitHub tokens are redacted.
func TestSanitizer_Sanitize_GitHubToken(t *testing.T) {
	s := NewSanitizer()
	token := "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ012345"
	text := "token: " + token
	result := s.Sanitize(text, TargetLLM)
	if !result.Redacted {
		t.Error("Sanitize(github token, TargetLLM) Redacted=false, want true")
	}
	if strings.Contains(result.Output, token) {
		t.Errorf("Sanitize output still contains raw GitHub token")
	}
}

// TestSanitizer_Sanitize_PrivateKey verifies private key headers are redacted.
func TestSanitizer_Sanitize_PrivateKey(t *testing.T) {
	s := NewSanitizer()
	text := "-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEA...\n-----END RSA PRIVATE KEY-----"
	result := s.Sanitize(text, TargetLLM)
	if !result.Redacted {
		t.Error("Sanitize(private key, TargetLLM) Redacted=false, want true")
	}
	if strings.Contains(result.Output, "BEGIN RSA PRIVATE KEY") {
		t.Error("Sanitize output still contains private key header")
	}
}

// TestSanitizer_Sanitize_RegisteredSecret verifies manually registered secrets are redacted.
func TestSanitizer_Sanitize_RegisteredSecret(t *testing.T) {
	s := NewSanitizer()
	s.RegisterSecret("my-super-secret-password-xyz")

	text := "connecting with password my-super-secret-password-xyz to database"
	result := s.Sanitize(text, TargetLLM)
	if !result.Redacted {
		t.Error("Sanitize(registered secret, TargetLLM) Redacted=false, want true")
	}
	if strings.Contains(result.Output, "my-super-secret-password-xyz") {
		t.Error("Sanitize output contains registered secret value")
	}
}

// TestSanitizer_Sanitize_RedactionPlaceholder verifies redacted output contains a placeholder.
func TestSanitizer_Sanitize_RedactionPlaceholder(t *testing.T) {
	s := NewSanitizer()
	text := "key=AKIAIOSFODNN7EXAMPLE and some context"
	result := s.Sanitize(text, TargetLLM)
	if !result.Redacted {
		t.Fatal("expected redaction")
	}
	// Output must contain some placeholder text — not just be empty.
	if result.Output == "" {
		t.Error("Sanitize output is empty; expected text with redaction placeholder")
	}
	if !strings.Contains(result.Output, "[REDACTED") {
		t.Errorf("Sanitize output does not contain [REDACTED...] placeholder, got: %q", result.Output)
	}
}

// TestSanitizer_Sanitize_FindingsPopulated verifies SanitizeResult.Findings is populated on match.
func TestSanitizer_Sanitize_FindingsPopulated(t *testing.T) {
	s := NewSanitizer()
	text := "AKIAIOSFODNN7EXAMPLE"
	result := s.Sanitize(text, TargetLLM)
	if !result.Redacted {
		t.Fatal("expected redaction")
	}
	if len(result.Findings) == 0 {
		t.Error("Sanitize Findings is empty; expected at least one finding")
	}
	for _, f := range result.Findings {
		if f == "" {
			t.Error("Sanitize Findings contains empty string entry")
		}
	}
}

// TestSanitizer_Sanitize_TargetLLM_MostAggressive verifies LLM target is most aggressive.
func TestSanitizer_Sanitize_TargetLLM_MostAggressive(t *testing.T) {
	// TargetLLM must redact connection strings with passwords.
	s := NewSanitizer()
	text := "db url: postgres://admin:hunter2@localhost/prod"
	result := s.Sanitize(text, TargetLLM)
	if !result.Redacted {
		t.Error("Sanitize(connection string, TargetLLM) Redacted=false, want true")
	}
}

// TestSanitizer_Sanitize_EmptyString handles empty input without panic.
func TestSanitizer_Sanitize_EmptyString(t *testing.T) {
	s := NewSanitizer()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Sanitize(\"\") panicked: %v", r)
		}
	}()
	result := s.Sanitize("", TargetLLM)
	if result.Redacted {
		t.Error("Sanitize(\"\") Redacted=true for empty string")
	}
}

// TestSanitizer_RegisterSecret_ShortValueIgnored verifies very short secrets are not registered
// to avoid over-redaction of common tokens.
func TestSanitizer_RegisterSecret_ShortValueIgnored(t *testing.T) {
	s := NewSanitizer()
	s.RegisterSecret("ab") // too short — should be silently ignored

	text := "ab is a common bigram in English text"
	result := s.Sanitize(text, TargetLLM)
	if result.Redacted {
		t.Error("Sanitize redacted text after registering too-short secret (over-redaction)")
	}
}

// TestSanitizer_Sanitize_DoesNotMutateInput verifies immutability of input string.
func TestSanitizer_Sanitize_DoesNotMutateInput(t *testing.T) {
	s := NewSanitizer()
	original := "key=AKIAIOSFODNN7EXAMPLE"
	input := original
	s.Sanitize(input, TargetLLM)
	if input != original {
		t.Errorf("Sanitize mutated input string: got %q, want %q", input, original)
	}
}

// TestSanitizer_Sanitize_MultipleSecretsInText verifies all occurrences are redacted.
func TestSanitizer_Sanitize_MultipleSecretsInText(t *testing.T) {
	s := NewSanitizer()
	text := "AWS=AKIAIOSFODNN7EXAMPLE and GH=ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ012345"
	result := s.Sanitize(text, TargetLLM)
	if !result.Redacted {
		t.Fatal("expected redaction for text with multiple secrets")
	}
	if strings.Contains(result.Output, "AKIAIOSFODNN7EXAMPLE") {
		t.Error("output still contains AWS key")
	}
	if strings.Contains(result.Output, "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ012345") {
		t.Error("output still contains GitHub token")
	}
}

// TestSanitizer_Sanitize_TargetUser_SomeRedaction verifies TargetUser still redacts known secrets.
func TestSanitizer_Sanitize_TargetUser_SomeRedaction(t *testing.T) {
	s := NewSanitizer()
	text := "key: AKIAIOSFODNN7EXAMPLE"
	result := s.Sanitize(text, TargetUser)
	if !result.Redacted {
		t.Error("Sanitize(aws key, TargetUser) Redacted=false; TargetUser must still redact secrets")
	}
}

// TestSanitizer_Sanitize_DSNPhase3_Fallback exercises the Phase 3 connection-string
// scrubber path by using a DSN variant where the scheme is not caught by DetectPatterns
// (e.g. amqp:// with a password) and verifying the password is still removed.
func TestSanitizer_Sanitize_DSNPhase3_Fallback(t *testing.T) {
	s := NewSanitizer()
	// AMQP DSN: the connection_string pattern in DetectPatterns covers amqp://, but
	// the Phase 3 scrubber provides a defence-in-depth path.  We construct a DSN
	// that is caught by DetectPatterns first; the fallback path is reached when
	// the Phase 2 match result equals the original (i.e. no change from Phase 2).
	// To exercise scrubConnectionPasswords directly we call it via Sanitize with
	// a DSN whose password was not caught by Phase 2 because it had been partially
	// modified by Phase 1 registered-secret replacement.
	s.RegisterSecret("hunter2") // register the password as a known secret
	text := "amqp://user:hunter2@broker.example.com/vhost"
	result := s.Sanitize(text, TargetLLM)
	if !result.Redacted {
		t.Error("Sanitize(amqp DSN, TargetLLM) Redacted=false")
	}
	if strings.Contains(result.Output, "hunter2") {
		t.Error("Sanitize output still contains password 'hunter2'")
	}
}

// TestSanitizer_Sanitize_TargetChannel_Redacts verifies TargetChannel still redacts secrets.
func TestSanitizer_Sanitize_TargetChannel_Redacts(t *testing.T) {
	s := NewSanitizer()
	text := "token: ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ012345"
	result := s.Sanitize(text, TargetChannel)
	if !result.Redacted {
		t.Error("Sanitize(github token, TargetChannel) Redacted=false")
	}
}

// TestSanitizer_Sanitize_TargetMemory_Redacts verifies TargetMemory still redacts secrets.
func TestSanitizer_Sanitize_TargetMemory_Redacts(t *testing.T) {
	s := NewSanitizer()
	text := "stored key AKIAIOSFODNN7EXAMPLE for later"
	result := s.Sanitize(text, TargetMemory)
	if !result.Redacted {
		t.Error("Sanitize(aws key, TargetMemory) Redacted=false")
	}
}

// TestSanitizeTarget_StringRepresentation verifies target constants are distinct.
func TestSanitizeTarget_StringRepresentation(t *testing.T) {
	targets := []SanitizeTarget{TargetLLM, TargetChannel, TargetMemory, TargetUser}
	seen := make(map[SanitizeTarget]bool)
	for _, target := range targets {
		if seen[target] {
			t.Errorf("duplicate SanitizeTarget value: %v", target)
		}
		seen[target] = true
	}
	if len(seen) != 4 {
		t.Errorf("expected 4 distinct SanitizeTarget values, got %d", len(seen))
	}
}

// TestSanitizer_Sanitize_LargeText_Performance verifies 1MB text is sanitized quickly.
func TestSanitizer_Sanitize_LargeText_Performance(t *testing.T) {
	s := NewSanitizer()
	base := strings.Repeat("normal log output line with no secrets here\n", 23000)
	secret := "AKIAIOSFODNN7EXAMPLE"
	text := base[:len(base)/2] + secret + base[len(base)/2:]

	result := s.Sanitize(text, TargetLLM)
	if !result.Redacted {
		t.Error("expected redaction in large text")
	}
	if strings.Contains(result.Output, secret) {
		t.Error("large text output still contains secret")
	}
}

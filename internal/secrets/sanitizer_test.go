package secrets

import (
	"strings"
	"sync"
	"testing"
)

// ---------------------------------------------------------------------------
// Sanitizer.Register / Unregister
// ---------------------------------------------------------------------------

func TestSanitizer_Register_IgnoresShortValues(t *testing.T) {
	s := NewSanitizer()
	tests := []struct {
		value string
	}{
		{""},
		{"a"},
		{"ab"},
		{"abc"},
	}
	for _, tt := range tests {
		t.Run("value="+tt.value, func(t *testing.T) {
			s.Register(tt.value, "label")
			if s.Count() != 0 {
				t.Errorf("short value %q should be ignored, but count is %d", tt.value, s.Count())
			}
		})
	}
}

func TestSanitizer_Register_AcceptsMinimumLength(t *testing.T) {
	s := NewSanitizer()
	s.Register("abcd", "test-label") // exactly 4 chars
	if s.Count() != 1 {
		t.Errorf("expected count 1, got %d", s.Count())
	}
}

func TestSanitizer_Unregister(t *testing.T) {
	s := NewSanitizer()
	s.Register("my-api-key-1234", "api_key")
	if s.Count() != 1 {
		t.Fatalf("expected 1 pattern after register")
	}
	s.Unregister("my-api-key-1234")
	if s.Count() != 0 {
		t.Errorf("expected 0 patterns after unregister, got %d", s.Count())
	}
}

func TestSanitizer_Unregister_NonexistentValue(t *testing.T) {
	// Should not panic.
	s := NewSanitizer()
	s.Unregister("never-registered")
}

// ---------------------------------------------------------------------------
// Sanitizer.Redact
// ---------------------------------------------------------------------------

func TestSanitizer_Redact_ReplacesKnownSecret(t *testing.T) {
	s := NewSanitizer()
	s.Register("sk-abc123def456", "openai_key")

	text := "Authorization: sk-abc123def456 in use"
	got, redacted := s.Redact(text)
	if !redacted {
		t.Error("expected redaction to occur")
	}
	if strings.Contains(got, "sk-abc123def456") {
		t.Errorf("secret still present in output: %q", got)
	}
	if !strings.Contains(got, "[REDACTED:openai_key]") {
		t.Errorf("expected redaction placeholder, got %q", got)
	}
}

func TestSanitizer_Redact_NoPatterns(t *testing.T) {
	s := NewSanitizer()
	text := "nothing to redact here"
	got, redacted := s.Redact(text)
	if redacted {
		t.Error("expected no redaction for empty sanitizer")
	}
	if got != text {
		t.Errorf("text modified when no patterns registered: %q", got)
	}
}

func TestSanitizer_Redact_NoMatchReturnsOriginal(t *testing.T) {
	s := NewSanitizer()
	s.Register("secret-value-xyz", "label")
	text := "this text does not contain the secret"
	got, redacted := s.Redact(text)
	if redacted {
		t.Error("expected no redaction")
	}
	if got != text {
		t.Errorf("text should be unchanged: %q", got)
	}
}

func TestSanitizer_Redact_MultipleOccurrences(t *testing.T) {
	s := NewSanitizer()
	s.Register("token-1234", "my_token")
	text := "use token-1234 here and also token-1234 there"
	got, _ := s.Redact(text)
	if strings.Contains(got, "token-1234") {
		t.Errorf("all occurrences should be redacted, got %q", got)
	}
	if strings.Count(got, "[REDACTED:my_token]") != 2 {
		t.Errorf("expected 2 redactions, got %q", got)
	}
}

func TestSanitizer_Redact_MultipleSecrets(t *testing.T) {
	s := NewSanitizer()
	s.Register("openai-key-abc", "openai")
	s.Register("github-token-xyz", "github")
	text := "openai-key-abc and github-token-xyz are both secret"
	got, redacted := s.Redact(text)
	if !redacted {
		t.Error("expected redaction")
	}
	if strings.Contains(got, "openai-key-abc") || strings.Contains(got, "github-token-xyz") {
		t.Errorf("secrets still present: %q", got)
	}
}

func TestSanitizer_Redact_EmptyText(t *testing.T) {
	s := NewSanitizer()
	s.Register("secret-1234", "label")
	got, redacted := s.Redact("")
	if redacted {
		t.Error("empty text should have no redactions")
	}
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestSanitizer_RedactBytes(t *testing.T) {
	s := NewSanitizer()
	s.Register("bytes-secret-key", "bytes_label")
	data := []byte("the bytes-secret-key is here")
	got, redacted := s.RedactBytes(data)
	if !redacted {
		t.Error("expected redaction in bytes")
	}
	if strings.Contains(string(got), "bytes-secret-key") {
		t.Errorf("secret still in bytes output: %q", got)
	}
}

func TestSanitizer_RedactBytes_NoRedaction(t *testing.T) {
	s := NewSanitizer()
	s.Register("not-here-secret", "label")
	data := []byte("clean text")
	got, redacted := s.RedactBytes(data)
	if redacted {
		t.Error("expected no redaction")
	}
	// Should return the original slice unchanged.
	if &got[0] != &data[0] {
		// They might differ if a copy was made — that is fine, but the content must match.
		if string(got) != string(data) {
			t.Errorf("content changed unexpectedly: %q", got)
		}
	}
}

// ---------------------------------------------------------------------------
// Sanitizer.Clear
// ---------------------------------------------------------------------------

func TestSanitizer_Clear(t *testing.T) {
	s := NewSanitizer()
	s.Register("secret-one-aaa", "a")
	s.Register("secret-two-bbb", "b")
	s.Clear()
	if s.Count() != 0 {
		t.Errorf("expected 0 after clear, got %d", s.Count())
	}
}

// ---------------------------------------------------------------------------
// Concurrent access
// ---------------------------------------------------------------------------

func TestSanitizer_ConcurrentAccess(t *testing.T) {
	s := NewSanitizer()
	var wg sync.WaitGroup
	const goroutines = 50

	// Concurrent Register + Redact.
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			secret := "concurrent-secret-" + string(rune('a'+n%26)) + "1234"
			s.Register(secret, "label")
			s.Redact("this text contains " + secret)
		}(i)
	}
	wg.Wait()

	// Concurrent Unregister.
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			s.Unregister("concurrent-secret-" + string(rune('a'+n%26)) + "1234")
		}(i)
	}
	wg.Wait()
}

// ---------------------------------------------------------------------------
// SanitizeForLLM — pattern-based redaction
// ---------------------------------------------------------------------------

func TestSanitizeForLLM_RedactsRegisteredSecret(t *testing.T) {
	s := NewSanitizer()
	s.Register("my-registered-secret-key", "my_key")
	result := SanitizeForLLM("value is my-registered-secret-key here", s)
	if strings.Contains(result, "my-registered-secret-key") {
		t.Errorf("registered secret should be redacted: %q", result)
	}
}

func TestSanitizeForLLM_RedactsBearerToken(t *testing.T) {
	s := NewSanitizer()
	text := "Authorization: Bearer eyJhbGciOiJSUzI1NiJ9.payload.sig"
	result := SanitizeForLLM(text, s)
	if strings.Contains(result, "eyJhbGciOiJSUzI1NiJ9") {
		t.Errorf("Bearer token should be redacted: %q", result)
	}
	if !strings.Contains(result, "[REDACTED]") {
		t.Errorf("expected [REDACTED] placeholder: %q", result)
	}
}

func TestSanitizeForLLM_RedactsApiKeyParam(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{"api_key=", "https://api.example.com?api_key=sk-abc12345678"},
		{"apikey=", "https://api.example.com?apikey=sk-abc12345678"},
		{"token=", "https://api.example.com?token=ghp_abc12345678"},
		{"password=", "db://host?password=mysecretpass"},
		{"secret=", "config: secret=shhh-dont-tell"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSanitizer()
			result := SanitizeForLLM(tt.text, s)
			// The value after the prefix should be redacted.
			if strings.Contains(result, "sk-abc12345678") ||
				strings.Contains(result, "ghp_abc12345678") ||
				strings.Contains(result, "mysecretpass") ||
				strings.Contains(result, "shhh-dont-tell") {
				t.Errorf("secret value not redacted: %q", result)
			}
		})
	}
}

func TestSanitizeForLLM_ShortValuesNotRedacted(t *testing.T) {
	// Values shorter than 8 chars after the prefix should not be redacted.
	s := NewSanitizer()
	// "api_key=short" — "short" is 5 chars, under the 8-char threshold.
	text := "api_key=short"
	result := SanitizeForLLM(text, s)
	if strings.Contains(result, "[REDACTED]") {
		t.Errorf("short values should not be redacted: %q", result)
	}
}

func TestSanitizeForLLM_EmptyText(t *testing.T) {
	s := NewSanitizer()
	result := SanitizeForLLM("", s)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

// ---------------------------------------------------------------------------
// redactAfterPrefix — exported indirectly via SanitizeForLLM
// ---------------------------------------------------------------------------

func TestRedactAfterPrefix_MultipleOccurrences(t *testing.T) {
	// Two Bearer tokens in the same text.
	text := "Bearer eyJhbGciOiJSUzI1NiJ9token1 ... Bearer eyJhbGciOiJSUzI1NiJ9token2"
	result := redactAfterPrefix(text, "Bearer ", " ", "\n")
	if strings.Contains(result, "eyJhbGciOiJSUzI1NiJ9") {
		t.Errorf("both tokens should be redacted: %q", result)
	}
}

func TestRedactAfterPrefix_AtEndOfString(t *testing.T) {
	// Token at the end of the string with no trailing delimiter.
	text := "Bearer eyJhbGciOiJSUzI1NiJ9longtoken"
	result := redactAfterPrefix(text, "Bearer ", " ")
	if strings.Contains(result, "eyJhbGciOiJSUzI1NiJ9") {
		t.Errorf("end-of-string token should be redacted: %q", result)
	}
}

func TestRedactAfterPrefix_PrefixNotPresent(t *testing.T) {
	text := "no bearer token here"
	result := redactAfterPrefix(text, "Bearer ", " ")
	if result != text {
		t.Errorf("text should be unchanged: %q", result)
	}
}

func TestRedactAfterPrefix_PrefixAtEndOfString(t *testing.T) {
	// Prefix present but no value after it — should not panic.
	text := "Authorization: "
	result := redactAfterPrefix(text, "Authorization: ", "\n")
	// No value to redact, text may or may not change.
	_ = result
}

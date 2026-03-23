package secrets

import (
	"strings"
	"testing"
)

// TestDetectPatterns_AWSAccessKey verifies AWS AKIA-prefix access key detection.
func TestDetectPatterns_AWSAccessKey(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		wantHit bool
	}{
		{
			name:    "canonical example key",
			text:    "AKIAIOSFODNN7EXAMPLE",
			wantHit: true,
		},
		{
			name:    "key embedded in longer line",
			text:    "export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE",
			wantHit: true,
		},
		{
			name:    "key in JSON",
			text:    `{"access_key": "AKIAJ3FOOBAR12345678"}`,
			wantHit: true,
		},
		{
			name:    "AKIA prefix too short",
			text:    "AKIASHORT",
			wantHit: false,
		},
		{
			name:    "not an AWS key — random caps",
			text:    "THIS_IS_NOT_A_KEY",
			wantHit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := DetectPatterns(tt.text)
			got := len(matches) > 0
			if got != tt.wantHit {
				t.Errorf("DetectPatterns(%q) hit=%v, want=%v (matches: %v)", tt.text, got, tt.wantHit, matches)
			}
			if tt.wantHit && got {
				found := false
				for _, m := range matches {
					if m.PatternName == "aws_access_key" {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected pattern name 'aws_access_key', got: %v", matches)
				}
			}
		})
	}
}

// TestDetectPatterns_GitHubTokens verifies all GitHub token prefix variants.
func TestDetectPatterns_GitHubTokens(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		wantHit bool
	}{
		{
			name:    "ghp_ classic personal access token",
			text:    "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ012345",
			wantHit: true,
		},
		{
			name:    "gho_ OAuth token",
			text:    "gho_ABCDEFGHIJKLMNOPQRSTUVWXYZ012345",
			wantHit: true,
		},
		{
			name:    "ghs_ server-to-server token",
			text:    "ghs_ABCDEFGHIJKLMNOPQRSTUVWXYZ012345",
			wantHit: true,
		},
		{
			name:    "ghr_ refresh token",
			text:    "ghr_ABCDEFGHIJKLMNOPQRSTUVWXYZ012345",
			wantHit: true,
		},
		{
			name:    "github_pat_ fine-grained PAT",
			text:    "github_pat_ABCDEFGHIJKLMNOPQRSTUVWXYZ01234567890abcdef",
			wantHit: true,
		},
		{
			name:    "ghp_ token in env assignment",
			text:    "GITHUB_TOKEN=ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ012345",
			wantHit: true,
		},
		{
			name:    "too short — not a real token",
			text:    "ghp_SHORT",
			wantHit: false,
		},
		{
			name:    "random word starting with gh",
			text:    "ghost_hunting",
			wantHit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := DetectPatterns(tt.text)
			got := len(matches) > 0
			if got != tt.wantHit {
				t.Errorf("DetectPatterns(%q) hit=%v, want=%v (matches: %v)", tt.text, got, tt.wantHit, matches)
			}
		})
	}
}

// TestDetectPatterns_AnthropicKeys verifies Anthropic sk-ant- key detection.
func TestDetectPatterns_AnthropicKeys(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		wantHit bool
	}{
		{
			name:    "typical anthropic key",
			text:    "sk-ant-api03-ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abcdefghijklmnopqrstuvwxyz",
			wantHit: true,
		},
		{
			name:    "anthropic key in config",
			text:    `anthropic_api_key: "sk-ant-api03-xxxxxxxxxxxxxxxxxxx"`,
			wantHit: true,
		},
		{
			name:    "sk-ant- too short",
			text:    "sk-ant-x",
			wantHit: false,
		},
		{
			name:    "sk prefix without ant",
			text:    "sk-something-else-entirely",
			wantHit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := DetectPatterns(tt.text)
			got := len(matches) > 0
			if got != tt.wantHit {
				t.Errorf("DetectPatterns(%q) hit=%v, want=%v (matches: %v)", tt.text, got, tt.wantHit, matches)
			}
			if tt.wantHit && got {
				found := false
				for _, m := range matches {
					if m.PatternName == "anthropic_api_key" {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected pattern name 'anthropic_api_key', got: %v", matches)
				}
			}
		})
	}
}

// TestDetectPatterns_OpenAIKeys verifies OpenAI sk-proj- key detection.
func TestDetectPatterns_OpenAIKeys(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		wantHit bool
	}{
		{
			name:    "openai project key",
			text:    "sk-proj-ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abcdefghijklmnop",
			wantHit: true,
		},
		{
			name:    "legacy openai sk- key (long enough)",
			text:    "sk-ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abcdefghij",
			wantHit: true,
		},
		{
			name:    "short sk- not a key",
			text:    "sk-short",
			wantHit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := DetectPatterns(tt.text)
			got := len(matches) > 0
			if got != tt.wantHit {
				t.Errorf("DetectPatterns(%q) hit=%v, want=%v (matches: %v)", tt.text, got, tt.wantHit, matches)
			}
		})
	}
}

// TestDetectPatterns_SlackTokens verifies Slack xox* token detection.
func TestDetectPatterns_SlackTokens(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		wantHit bool
	}{
		{
			name:    "xoxb bot token",
			text:    "xoxb" + "-12345678901-12345678901-ABCDEFGHIJKLMNOPQRSTUVWX",
			wantHit: true,
		},
		{
			name:    "xoxp user token",
			text:    "xoxp-12345678901-12345678901-12345678901-ABCDEF1234567890abcdef",
			wantHit: true,
		},
		{
			name:    "xoxa app-level token",
			text:    "xoxa-2-ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abcdefghij",
			wantHit: true,
		},
		{
			name:    "xoxs workspace token",
			text:    "xoxs-12345678901-ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789ab",
			wantHit: true,
		},
		{
			name:    "not a slack token",
			text:    "xox_something_short",
			wantHit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := DetectPatterns(tt.text)
			got := len(matches) > 0
			if got != tt.wantHit {
				t.Errorf("DetectPatterns(%q) hit=%v, want=%v (matches: %v)", tt.text, got, tt.wantHit, matches)
			}
		})
	}
}

// TestDetectPatterns_PrivateKeyHeaders verifies PEM private key header detection.
func TestDetectPatterns_PrivateKeyHeaders(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		wantHit bool
	}{
		{
			name:    "RSA private key header",
			text:    "-----BEGIN RSA PRIVATE KEY-----",
			wantHit: true,
		},
		{
			name:    "generic private key header",
			text:    "-----BEGIN PRIVATE KEY-----",
			wantHit: true,
		},
		{
			name:    "EC private key header",
			text:    "-----BEGIN EC PRIVATE KEY-----",
			wantHit: true,
		},
		{
			name:    "OpenSSH private key header",
			text:    "-----BEGIN OPENSSH PRIVATE KEY-----",
			wantHit: true,
		},
		{
			name:    "public certificate header — not secret",
			text:    "-----BEGIN CERTIFICATE-----",
			wantHit: false,
		},
		{
			name:    "public key header — not secret",
			text:    "-----BEGIN PUBLIC KEY-----",
			wantHit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := DetectPatterns(tt.text)
			got := len(matches) > 0
			if got != tt.wantHit {
				t.Errorf("DetectPatterns(%q) hit=%v, want=%v (matches: %v)", tt.text, got, tt.wantHit, matches)
			}
			if tt.wantHit && got {
				found := false
				for _, m := range matches {
					if m.PatternName == "private_key_header" {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected pattern name 'private_key_header', got: %v", matches)
				}
			}
		})
	}
}

// TestDetectPatterns_ConnectionStrings verifies database connection string detection.
func TestDetectPatterns_ConnectionStrings(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		wantHit bool
	}{
		{
			name:    "postgres connection string with password",
			text:    "postgres://user:secretpass@localhost:5432/mydb",
			wantHit: true,
		},
		{
			name:    "postgresql:// variant",
			text:    "postgresql://admin:hunter2@db.example.com/prod",
			wantHit: true,
		},
		{
			name:    "mysql connection string",
			text:    "mysql://root:p@ssw0rd@127.0.0.1:3306/appdb",
			wantHit: true,
		},
		{
			name:    "connection string in config file content",
			text:    `DATABASE_URL="postgres://app_user:s3cr3t@rds.example.com:5432/myapp"`,
			wantHit: true,
		},
		{
			name:    "postgres without password (no colon before @)",
			text:    "postgres://user@localhost/mydb",
			wantHit: false,
		},
		{
			name:    "plain URL — not a DSN",
			text:    "https://example.com/path",
			wantHit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := DetectPatterns(tt.text)
			got := len(matches) > 0
			if got != tt.wantHit {
				t.Errorf("DetectPatterns(%q) hit=%v, want=%v (matches: %v)", tt.text, got, tt.wantHit, matches)
			}
		})
	}
}

// TestDetectPatterns_BearerTokens verifies Bearer token header detection.
func TestDetectPatterns_BearerTokens(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		wantHit bool
	}{
		{
			name:    "Authorization header with Bearer token",
			text:    "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.payload.sig",
			wantHit: true,
		},
		{
			name:    "lowercase bearer",
			text:    "authorization: bearer eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ1c2VyIn0.ABC123",
			wantHit: true,
		},
		{
			name:    "Bearer with short token — not a real token",
			text:    "Authorization: Bearer abc",
			wantHit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := DetectPatterns(tt.text)
			got := len(matches) > 0
			if got != tt.wantHit {
				t.Errorf("DetectPatterns(%q) hit=%v, want=%v (matches: %v)", tt.text, got, tt.wantHit, matches)
			}
		})
	}
}

// TestDetectPatterns_GenericAPIKeyAssignment verifies generic api_key= / api-key= detection.
func TestDetectPatterns_GenericAPIKeyAssignment(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		wantHit bool
	}{
		{
			name:    "api_key= assignment with long value",
			text:    "api_key=abcdefghijklmnopqrstuvwxyz0123456789ABCDEF",
			wantHit: true,
		},
		{
			name:    "API_KEY= in env",
			text:    "API_KEY=aBcDeFgHiJkLmNoPqRsTuVwXyZ01234567",
			wantHit: true,
		},
		{
			name:    "apikey in JSON",
			text:    `{"apikey": "abcdefghijklmnopqrstuvwxyz0123456789"}`,
			wantHit: true,
		},
		{
			name:    "api_key= too short value",
			text:    "api_key=short",
			wantHit: false,
		},
		{
			name:    "api_key in a comment",
			text:    "// set api_key before running",
			wantHit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := DetectPatterns(tt.text)
			got := len(matches) > 0
			if got != tt.wantHit {
				t.Errorf("DetectPatterns(%q) hit=%v, want=%v (matches: %v)", tt.text, got, tt.wantHit, matches)
			}
		})
	}
}

// TestDetectPatterns_FalsePositives verifies that safe text does NOT trigger detection.
func TestDetectPatterns_FalsePositives(t *testing.T) {
	safeCases := []struct {
		name string
		text string
	}{
		{
			name: "regular English prose",
			text: "The quick brown fox jumps over the lazy dog.",
		},
		{
			name: "short alphanumeric string",
			text: "abc123",
		},
		{
			name: "git commit hash (40-char hex)",
			text: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
		},
		{
			name: "UUID",
			text: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name: "Go import path",
			text: "import \"github.com/meowai/blackcat/internal/memory\"",
		},
		{
			name: "empty string",
			text: "",
		},
		{
			name: "source code line with variable named key",
			text: "keyFile := path.Join(home, \".ssh\", \"known_hosts\")",
		},
		{
			name: "URL without credentials",
			text: "https://api.example.com/v1/users",
		},
	}

	for _, tt := range safeCases {
		t.Run(tt.name, func(t *testing.T) {
			matches := DetectPatterns(tt.text)
			if len(matches) > 0 {
				t.Errorf("DetectPatterns(%q) produced false positive matches: %v", tt.text, matches)
			}
		})
	}
}

// TestPatternMatch_Fields verifies PatternMatch struct fields are populated.
func TestPatternMatch_Fields(t *testing.T) {
	text := "AKIAIOSFODNN7EXAMPLE"
	matches := DetectPatterns(text)
	if len(matches) == 0 {
		t.Fatal("expected at least one match for AWS access key")
	}
	m := matches[0]
	if m.PatternName == "" {
		t.Error("PatternMatch.PatternName must not be empty")
	}
	if m.Value == "" {
		t.Error("PatternMatch.Value must not be empty")
	}
	if !strings.Contains(text, m.Value) {
		t.Errorf("PatternMatch.Value %q not found in original text %q", m.Value, text)
	}
	if m.Start < 0 {
		t.Error("PatternMatch.Start must be >= 0")
	}
	if m.End <= m.Start {
		t.Error("PatternMatch.End must be > Start")
	}
}

// TestDetectPatterns_MultipleSecretsInOneString checks that multiple patterns are found.
func TestDetectPatterns_MultipleSecretsInOneString(t *testing.T) {
	text := "AWS_KEY=AKIAIOSFODNN7EXAMPLE and GITHUB=ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ012345"
	matches := DetectPatterns(text)
	if len(matches) < 2 {
		t.Errorf("expected >= 2 matches for text with 2 secrets, got %d: %v", len(matches), matches)
	}
}

// TestDetectPatterns_LargeText verifies performance is acceptable for 1MB input.
func TestDetectPatterns_LargeText(t *testing.T) {
	// Build a 1MB string with a secret buried in it.
	base := strings.Repeat("the quick brown fox jumps over the lazy dog\n", 23000)
	secret := "AKIAIOSFODNN7EXAMPLE"
	text := base[:len(base)/2] + secret + base[len(base)/2:]

	matches := DetectPatterns(text)
	found := false
	for _, m := range matches {
		if m.PatternName == "aws_access_key" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find aws_access_key in large text")
	}
}

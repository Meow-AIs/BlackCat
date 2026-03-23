package remote

import (
	"strings"
	"testing"
)

func TestSanitizeRemoteOutput_RedactsIPAddresses(t *testing.T) {
	input := "Connected to 192.168.1.100 via gateway 10.0.0.1"
	output := SanitizeRemoteOutput(input)

	if strings.Contains(output, "192.168.1.100") {
		t.Error("expected IP 192.168.1.100 to be redacted")
	}
	if strings.Contains(output, "10.0.0.1") {
		t.Error("expected IP 10.0.0.1 to be redacted")
	}
	// Should keep first octet
	if !strings.Contains(output, "192.X.X.X") {
		t.Errorf("expected redacted IP to keep first octet, got: %s", output)
	}
	if !strings.Contains(output, "10.X.X.X") {
		t.Errorf("expected redacted IP to keep first octet, got: %s", output)
	}
}

func TestSanitizeRemoteOutput_PreservesNonIPText(t *testing.T) {
	input := "Hello world, version 2.0"
	output := SanitizeRemoteOutput(input)
	if output != input {
		t.Errorf("expected unchanged output, got: %s", output)
	}
}

func TestSanitizeRemoteOutput_RedactsSSHKeys(t *testing.T) {
	input := `Some output
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA1234567890abcdef
more key data here
-----END RSA PRIVATE KEY-----
more output`

	output := SanitizeRemoteOutput(input)
	if strings.Contains(output, "MIIEpAIBAAKCAQEA") {
		t.Error("expected SSH key content to be redacted")
	}
	if !strings.Contains(output, "[REDACTED SSH KEY]") {
		t.Error("expected [REDACTED SSH KEY] placeholder")
	}
	if !strings.Contains(output, "Some output") {
		t.Error("expected surrounding text to be preserved")
	}
	if !strings.Contains(output, "more output") {
		t.Error("expected surrounding text to be preserved")
	}
}

func TestSanitizeRemoteOutput_RedactsOpenSSHKeys(t *testing.T) {
	input := `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmU
-----END OPENSSH PRIVATE KEY-----`

	output := SanitizeRemoteOutput(input)
	if strings.Contains(output, "b3BlbnNzaC1rZXktdjEAAAAABG5vbmU") {
		t.Error("expected OpenSSH key to be redacted")
	}
	if !strings.Contains(output, "[REDACTED SSH KEY]") {
		t.Error("expected [REDACTED SSH KEY] placeholder")
	}
}

func TestSanitizeRemoteOutput_RedactsTokenPatterns(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"bearer token", "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.abc.def"},
		{"password field", `password: "s3cretP@ss!"`},
		{"password equals", `PASSWORD=mysecret123`},
		{"token field", `token: "ghp_1234567890abcdef"`},
		{"api_key field", `api_key: "sk-1234567890"`},
		{"secret field", `secret: "my-secret-value"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := SanitizeRemoteOutput(tt.input)
			if output == tt.input {
				t.Errorf("expected %q to have redactions, but output was unchanged", tt.name)
			}
			if !strings.Contains(output, "[REDACTED]") {
				t.Errorf("expected [REDACTED] placeholder in output: %s", output)
			}
		})
	}
}

func TestSanitizeRemoteOutput_EmptyInput(t *testing.T) {
	output := SanitizeRemoteOutput("")
	if output != "" {
		t.Errorf("expected empty output, got: %s", output)
	}
}

func TestSanitizeRemoteOutput_MultipleRedactions(t *testing.T) {
	input := "Server 192.168.1.1 password: secret123 connected to 10.0.0.5"
	output := SanitizeRemoteOutput(input)

	if strings.Contains(output, "192.168.1.1") {
		t.Error("expected first IP to be redacted")
	}
	if strings.Contains(output, "10.0.0.5") {
		t.Error("expected second IP to be redacted")
	}
	if strings.Contains(output, "secret123") {
		t.Error("expected password to be redacted")
	}
}

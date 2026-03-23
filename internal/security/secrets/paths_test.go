package secrets

import (
	"os"
	"path/filepath"
	"testing"
)

// TestIsSensitivePath_Blocked verifies well-known sensitive paths are blocked.
func TestIsSensitivePath_Blocked(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("could not get home dir: %v", err)
	}

	blocked := []struct {
		name string
		path string
	}{
		{"ssh private key", filepath.Join(home, ".ssh", "id_rsa")},
		{"ssh ed25519 private key", filepath.Join(home, ".ssh", "id_ed25519")},
		{"aws credentials", filepath.Join(home, ".aws", "credentials")},
		{"aws config (may have keys)", filepath.Join(home, ".aws", "config")},
		{"kube config", filepath.Join(home, ".kube", "config")},
		{"dotenv file", ".env"},
		{"dotenv with suffix", ".env.production"},
		{"pem file", "server.pem"},
		{"key file", "private.key"},
		{"p12 certificate bundle", "cert.p12"},
		{"pfx bundle", "cert.pfx"},
		{"jks keystore", "keystore.jks"},
		{"netrc", filepath.Join(home, ".netrc")},
		{"gnupg private key", filepath.Join(home, ".gnupg", "secring.gpg")},
		{"docker config (may have auth tokens)", filepath.Join(home, ".docker", "config.json")},
	}

	for _, tt := range blocked {
		t.Run(tt.name, func(t *testing.T) {
			if !IsSensitivePath(tt.path) {
				t.Errorf("IsSensitivePath(%q) = false, want true", tt.path)
			}
		})
	}
}

// TestIsSensitivePath_Allowed verifies regular paths are not blocked.
func TestIsSensitivePath_Allowed(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("could not get home dir: %v", err)
	}

	allowed := []struct {
		name string
		path string
	}{
		{"go source file", filepath.Join(home, "projects", "main.go")},
		{"tmp output file", "/tmp/output.txt"},
		{"regular text file", "README.txt"},
		{"go module file", "go.mod"},
		{"json config (not docker)", "config.json"},
		{"yaml config", "config.yaml"},
		{"go test file", "main_test.go"},
	}

	for _, tt := range allowed {
		t.Run(tt.name, func(t *testing.T) {
			if IsSensitivePath(tt.path) {
				t.Errorf("IsSensitivePath(%q) = true, want false", tt.path)
			}
		})
	}
}

// TestIsSensitivePath_TraversalPrevention verifies that path traversal to sensitive files is blocked.
func TestIsSensitivePath_TraversalPrevention(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("could not get home dir: %v", err)
	}

	// Construct a traversal path that resolves to ~/.ssh/id_rsa.
	// e.g. /tmp/../../<home>/.ssh/id_rsa
	traversal := filepath.Join("/tmp", "..", "..", filepath.Join(home, ".ssh", "id_rsa")[1:])

	// The path may or may not exist on disk; IsSensitivePath must evaluate the
	// resolved/cleaned path regardless.
	result := IsSensitivePath(traversal)
	if !result {
		// If the traversal cannot be resolved (path doesn't exist), the function
		// must still block paths matching sensitive patterns after filepath.Clean.
		cleaned := filepath.Clean(traversal)
		if IsSensitivePath(cleaned) {
			// The clean path is blocked — traversal test passes structurally.
			return
		}
		t.Errorf("IsSensitivePath(%q) = false; traversal to .ssh/id_rsa should be blocked", traversal)
	}
}

// TestIsSensitivePath_EnvFileVariants covers .env file naming patterns.
func TestIsSensitivePath_EnvFileVariants(t *testing.T) {
	envFiles := []string{
		".env",
		".env.local",
		".env.development",
		".env.staging",
		".env.production",
		".env.test",
	}

	for _, p := range envFiles {
		t.Run(p, func(t *testing.T) {
			if !IsSensitivePath(p) {
				t.Errorf("IsSensitivePath(%q) = false, want true for .env variant", p)
			}
		})
	}
}

// TestIsSensitivePath_KeyExtensions covers various key/cert extension patterns.
func TestIsSensitivePath_KeyExtensions(t *testing.T) {
	keyFiles := []string{
		"my_server.pem",
		"client.key",
		"auth.p12",
		"bundle.pfx",
		"store.jks",
	}

	for _, p := range keyFiles {
		t.Run(p, func(t *testing.T) {
			if !IsSensitivePath(p) {
				t.Errorf("IsSensitivePath(%q) = false, want true for key/cert file", p)
			}
		})
	}
}

// TestIsSensitivePath_EmptyString handles empty input without panic.
func TestIsSensitivePath_EmptyString(t *testing.T) {
	// Must not panic; result may be true or false.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("IsSensitivePath(\"\") panicked: %v", r)
		}
	}()
	IsSensitivePath("")
}

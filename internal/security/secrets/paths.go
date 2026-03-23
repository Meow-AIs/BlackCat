package secrets

import (
	"os"
	"path/filepath"
	"strings"
)

// sensitiveExactNames is the set of exact base names that are always sensitive.
var sensitiveExactNames = map[string]bool{
	".netrc":      true,
	"credentials": true, // ~/.aws/credentials, etc.
}

// sensitiveDirSegments are directory path segments that indicate a sensitive file
// when combined with known sub-paths.
var sensitiveSuffixes = []string{
	// SSH keys
	".ssh/id_rsa",
	".ssh/id_dsa",
	".ssh/id_ecdsa",
	".ssh/id_ed25519",
	".ssh/id_xmss",
	// AWS
	".aws/credentials",
	".aws/config",
	// Kubernetes
	".kube/config",
	// GnuPG
	".gnupg/secring.gpg",
	".gnupg/private-keys-v1.d",
	// Docker (contains auth tokens)
	".docker/config.json",
}

// sensitiveExtensions are file extensions that indicate cryptographic material.
var sensitiveExtensions = map[string]bool{
	".pem": true,
	".key": true,
	".p12": true,
	".pfx": true,
	".jks": true,
	".pkcs12": true,
}

// IsSensitivePath reports whether the given path points to a file that may
// contain credentials or private key material.
//
// The function resolves path traversal sequences via filepath.Clean (and
// filepath.EvalSymlinks when the file exists) before applying any checks,
// so callers cannot bypass the denylist with ../../ tricks.
func IsSensitivePath(path string) bool {
	if path == "" {
		return false
	}

	// Resolve symlinks when the path exists on disk; otherwise just clean it.
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		resolved = filepath.Clean(path)
	}

	// Normalise to forward slashes for uniform matching on all platforms.
	normalised := filepath.ToSlash(resolved)

	// 1. Check by known sensitive suffixes (substring match on the cleaned path).
	home, _ := os.UserHomeDir()
	homeSlash := filepath.ToSlash(home)
	for _, suffix := range sensitiveSuffixes {
		// Match absolute home-relative paths.
		if strings.HasSuffix(normalised, "/"+suffix) {
			return true
		}
		if homeSlash != "" && normalised == homeSlash+"/"+suffix {
			return true
		}
	}

	// 2. Check the base name.
	base := filepath.Base(resolved)

	// Exact base name hits.
	if sensitiveExactNames[base] {
		return true
	}

	// .env files: exact ".env" or ".env.<anything>".
	if base == ".env" || strings.HasPrefix(base, ".env.") {
		return true
	}

	// 3. Check extension.
	ext := strings.ToLower(filepath.Ext(base))
	if sensitiveExtensions[ext] {
		return true
	}

	return false
}

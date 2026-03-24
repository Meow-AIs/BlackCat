package secrets

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ImportSource identifies the origin format for imported secrets.
type ImportSource string

const (
	ImportDotEnv         ImportSource = "dotenv"
	ImportAWSCredentials ImportSource = "aws_credentials"
	ImportGitHubToken    ImportSource = "github_token"
	ImportKubeConfig     ImportSource = "kube_config"
	ImportOnePasswordCLI ImportSource = "1password_cli"
	ImportJSON           ImportSource = "json"
)

// ImportResult summarizes the outcome of an import operation.
type ImportResult struct {
	Source   ImportSource `json:"source"`
	Imported int          `json:"imported"`
	Skipped  int          `json:"skipped"`
	Errors   []string     `json:"errors,omitempty"`
}

// Importer handles importing secrets from external formats.
type Importer struct {
	manager *Manager
}

// NewImporter creates a new Importer.
func NewImporter(manager *Manager) *Importer {
	return &Importer{manager: manager}
}

// ImportDotEnvFile reads a .env file and imports all key-value pairs as secrets.
// Lines starting with # are comments. Empty lines are skipped.
// Keys are lowercased and used as secret names. The original key becomes the EnvVar.
func (imp *Importer) ImportDotEnvFile(ctx context.Context, path string, scope Scope) (ImportResult, error) {
	result := ImportResult{Source: ImportDotEnv}

	f, err := os.Open(path)
	if err != nil {
		return result, fmt.Errorf("open .env file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments.
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			result.Skipped++
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Strip surrounding quotes from value.
		value = strings.Trim(value, "\"'")

		secretName := strings.ToLower(strings.ReplaceAll(key, "-", "_"))

		meta := SecretMetadata{
			Name:         secretName,
			Type:         guessSecretType(key),
			Scope:        scope,
			EnvVar:       key,
			Description:  fmt.Sprintf("Imported from .env file: %s", filepath.Base(path)),
			ImportedFrom: string(ImportDotEnv),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		if err := imp.manager.Set(ctx, meta, []byte(value)); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", key, err))
			continue
		}
		result.Imported++
	}

	if err := scanner.Err(); err != nil {
		return result, fmt.Errorf("read .env file: %w", err)
	}

	return result, nil
}

// ImportAWSCredentialsFile reads ~/.aws/credentials and imports profiles as secrets.
// Each profile's access key ID and secret access key become separate secrets named
// "aws_<profile>_access_key_id" and "aws_<profile>_secret_access_key".
func (imp *Importer) ImportAWSCredentialsFile(ctx context.Context, path string) (ImportResult, error) {
	result := ImportResult{Source: ImportAWSCredentials}

	f, err := os.Open(path)
	if err != nil {
		return result, fmt.Errorf("open AWS credentials file: %w", err)
	}
	defer f.Close()

	var currentProfile string
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Profile header: [profile-name]
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentProfile = strings.Trim(line, "[]")
			continue
		}

		if currentProfile == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if key != "aws_access_key_id" && key != "aws_secret_access_key" && key != "aws_session_token" {
			continue
		}

		secretName := fmt.Sprintf("aws_%s_%s", strings.ToLower(currentProfile), key)
		envVar := strings.ToUpper(key)
		if currentProfile != "default" {
			envVar = "AWS_" + strings.ToUpper(currentProfile) + "_" + strings.ToUpper(strings.TrimPrefix(key, "aws_"))
		}

		meta := SecretMetadata{
			Name:         secretName,
			Type:         TypeCloudCred,
			Scope:        ScopeGlobal,
			EnvVar:       envVar,
			Tags:         []string{"aws", currentProfile},
			Description:  fmt.Sprintf("AWS %s for profile %q", key, currentProfile),
			ImportedFrom: string(ImportAWSCredentials),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		if err := imp.manager.Set(ctx, meta, []byte(value)); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s/%s: %v", currentProfile, key, err))
			continue
		}
		result.Imported++
	}

	return result, nil
}

// ImportJSON imports secrets from a JSON file with the format:
//
//	{"secrets": [{"name": "...", "value": "...", "type": "...", "env_var": "..."}]}
//
// This is used for importing exported backups (after decryption).
func (imp *Importer) ImportJSON(ctx context.Context, data []byte, scope Scope) (ImportResult, error) {
	result := ImportResult{Source: ImportJSON}

	var payload struct {
		Secrets []struct {
			Name   string   `json:"name"`
			Value  string   `json:"value"`
			Type   string   `json:"type"`
			EnvVar string   `json:"env_var"`
			Tags   []string `json:"tags,omitempty"`
		} `json:"secrets"`
	}

	if err := json.Unmarshal(data, &payload); err != nil {
		return result, fmt.Errorf("parse JSON import: %w", err)
	}

	for _, s := range payload.Secrets {
		meta := SecretMetadata{
			Name:         s.Name,
			Type:         SecretType(s.Type),
			Scope:        scope,
			EnvVar:       s.EnvVar,
			Tags:         s.Tags,
			ImportedFrom: string(ImportJSON),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		if err := imp.manager.Set(ctx, meta, []byte(s.Value)); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", s.Name, err))
			continue
		}
		result.Imported++
	}

	return result, nil
}

// ExportEncrypted exports all secrets in the given scope as an encrypted JSON blob.
// The passphrase protects the export; it can be imported on another machine
// using ImportJSON after decryption.
func (imp *Importer) ExportEncrypted(ctx context.Context, scope Scope, passphrase []byte) ([]byte, error) {
	metas, err := imp.manager.List(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("list secrets for export: %w", err)
	}

	type exportEntry struct {
		Name   string   `json:"name"`
		Value  string   `json:"value"`
		Type   string   `json:"type"`
		EnvVar string   `json:"env_var"`
		Tags   []string `json:"tags,omitempty"`
	}

	entries := make([]exportEntry, 0, len(metas))
	for _, meta := range metas {
		val, err := imp.manager.Get(ctx, meta.Name, meta.Scope)
		if err != nil {
			continue // Skip inaccessible secrets.
		}
		entries = append(entries, exportEntry{
			Name:   meta.Name,
			Value:  string(val),
			Type:   string(meta.Type),
			EnvVar: meta.EnvVar,
			Tags:   meta.Tags,
		})
		SecureWipe(val)
	}

	payload := struct {
		Secrets  []exportEntry `json:"secrets"`
		Exported time.Time     `json:"exported_at"`
	}{
		Secrets:  entries,
		Exported: time.Now(),
	}

	plaintext, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal export: %w", err)
	}

	encrypted, err := EncryptWithPassphrase(passphrase, plaintext)
	SecureWipe(plaintext)
	if err != nil {
		return nil, fmt.Errorf("encrypt export: %w", err)
	}

	return encrypted, nil
}

// guessSecretType infers the secret type from the environment variable name.
func guessSecretType(envVar string) SecretType {
	upper := strings.ToUpper(envVar)

	switch {
	case strings.Contains(upper, "OPENAI") ||
		strings.Contains(upper, "ANTHROPIC") ||
		strings.Contains(upper, "GOOGLE_AI") ||
		strings.Contains(upper, "OPENROUTER") ||
		strings.Contains(upper, "API_KEY") ||
		strings.Contains(upper, "APIKEY"):
		return TypeAPIKey

	case strings.Contains(upper, "AWS_") ||
		strings.Contains(upper, "GCP_") ||
		strings.Contains(upper, "AZURE_"):
		return TypeCloudCred

	case strings.Contains(upper, "GITHUB") ||
		strings.Contains(upper, "GITLAB") ||
		strings.Contains(upper, "GIT_TOKEN"):
		return TypeGitToken

	case strings.Contains(upper, "DATABASE") ||
		strings.Contains(upper, "DB_") ||
		strings.Contains(upper, "POSTGRES") ||
		strings.Contains(upper, "MYSQL") ||
		strings.Contains(upper, "MONGO") ||
		strings.Contains(upper, "REDIS"):
		return TypeDBCred

	case strings.Contains(upper, "SSH"):
		return TypeSSHKey

	case strings.Contains(upper, "KUBE") ||
		strings.Contains(upper, "K8S"):
		return TypeKubeConfig

	case strings.Contains(upper, "VPN") ||
		strings.Contains(upper, "WIREGUARD") ||
		strings.Contains(upper, "OPENVPN"):
		return TypeVPN

	default:
		return TypeCustom
	}
}

package secrets

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Shared test helpers used by importer_test and injection_test
// ---------------------------------------------------------------------------

// memBackend is an in-memory Backend for testing — no OS or file-system deps.
type memBackend struct {
	data map[string][]byte
}

func newMemBackend() *memBackend {
	return &memBackend{data: make(map[string][]byte)}
}

func (m *memBackend) Name() string    { return "mem" }
func (m *memBackend) Available() bool { return true }

func (m *memBackend) Get(_ context.Context, key string) ([]byte, error) {
	v, ok := m.data[key]
	if !ok {
		return nil, ErrNotFound
	}
	cp := make([]byte, len(v))
	copy(cp, v)
	return cp, nil
}

func (m *memBackend) Set(_ context.Context, key string, value []byte) error {
	cp := make([]byte, len(value))
	copy(cp, value)
	m.data[key] = cp
	return nil
}

func (m *memBackend) Delete(_ context.Context, key string) error {
	if _, ok := m.data[key]; !ok {
		return ErrNotFound
	}
	delete(m.data, key)
	return nil
}

func (m *memBackend) List(_ context.Context) ([]string, error) {
	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	return keys, nil
}

// newTestManager creates a Manager backed by an in-memory backend and
// in-memory metadata store. No CGo/SQLite dependency.
func newTestManager(t *testing.T) *Manager {
	t.Helper()
	mgr, err := NewManager(ManagerOpts{
		Backends:      []Backend{newMemBackend()},
		MetadataStore: newMemMetaStore(),
		AuditLog:      &memAuditLog{},
	})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	return mgr
}

// writeTempFile creates a temp file with the given content and returns its path.
func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}

// simpleMeta builds a minimal SecretMetadata for test inserts.
func simpleMeta(name string, scope Scope) SecretMetadata {
	now := time.Now().UTC().Truncate(time.Second)
	return SecretMetadata{
		Name:      name,
		Type:      TypeAPIKey,
		Scope:     scope,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// ---------------------------------------------------------------------------
// ImportDotEnvFile
// ---------------------------------------------------------------------------

func TestImporter_ImportDotEnvFile_Basic(t *testing.T) {
	mgr := newTestManager(t)
	imp := NewImporter(mgr)
	ctx := context.Background()

	dir := t.TempDir()
	path := writeTempFile(t, dir, ".env", `
# This is a comment
OPENAI_API_KEY=sk-abcdef1234567890

DATABASE_URL=postgres://user:pass@localhost/db
EMPTY_LINE_ABOVE=yes
`)

	result, err := imp.ImportDotEnvFile(ctx, path, ScopeGlobal)
	if err != nil {
		t.Fatalf("ImportDotEnvFile: %v", err)
	}
	if result.Imported != 3 {
		t.Errorf("Imported: want 3, got %d", result.Imported)
	}
	if result.Skipped != 0 {
		t.Errorf("Skipped: want 0, got %d", result.Skipped)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Errors: want none, got %v", result.Errors)
	}
	if result.Source != ImportDotEnv {
		t.Errorf("Source: want dotenv, got %q", result.Source)
	}

	// Verify a value was stored.
	val, err := mgr.Get(ctx, "openai_api_key", ScopeGlobal)
	if err != nil {
		t.Fatalf("Get openai_api_key: %v", err)
	}
	if string(val) != "sk-abcdef1234567890" {
		t.Errorf("value mismatch: want sk-abcdef1234567890, got %q", val)
	}
}

func TestImporter_ImportDotEnvFile_CommentsAndEmptyLinesSkipped(t *testing.T) {
	mgr := newTestManager(t)
	imp := NewImporter(mgr)
	ctx := context.Background()

	dir := t.TempDir()
	path := writeTempFile(t, dir, ".env", `
# full-line comment
  # indented comment

KEY_A=value_a
`)

	result, err := imp.ImportDotEnvFile(ctx, path, ScopeGlobal)
	if err != nil {
		t.Fatalf("ImportDotEnvFile: %v", err)
	}
	if result.Imported != 1 {
		t.Errorf("Imported: want 1, got %d", result.Imported)
	}
}

func TestImporter_ImportDotEnvFile_QuotedValues(t *testing.T) {
	mgr := newTestManager(t)
	imp := NewImporter(mgr)
	ctx := context.Background()

	dir := t.TempDir()
	path := writeTempFile(t, dir, ".env", `
DOUBLE_QUOTED="my secret value"
SINGLE_QUOTED='another secret'
`)

	result, err := imp.ImportDotEnvFile(ctx, path, ScopeGlobal)
	if err != nil {
		t.Fatalf("ImportDotEnvFile: %v", err)
	}
	if result.Imported != 2 {
		t.Fatalf("Imported: want 2, got %d", result.Imported)
	}

	val, err := mgr.Get(ctx, "double_quoted", ScopeGlobal)
	if err != nil {
		t.Fatalf("Get double_quoted: %v", err)
	}
	if string(val) != "my secret value" {
		t.Errorf("double-quoted value: want %q, got %q", "my secret value", val)
	}

	val2, err := mgr.Get(ctx, "single_quoted", ScopeGlobal)
	if err != nil {
		t.Fatalf("Get single_quoted: %v", err)
	}
	if string(val2) != "another secret" {
		t.Errorf("single-quoted value: want %q, got %q", "another secret", val2)
	}
}

func TestImporter_ImportDotEnvFile_KeyWithNoEquals_Skipped(t *testing.T) {
	mgr := newTestManager(t)
	imp := NewImporter(mgr)
	ctx := context.Background()

	dir := t.TempDir()
	path := writeTempFile(t, dir, ".env", `
VALID_KEY=valid
INVALID_LINE_NO_EQUALS
ANOTHER_VALID=also_valid
`)

	result, err := imp.ImportDotEnvFile(ctx, path, ScopeGlobal)
	if err != nil {
		t.Fatalf("ImportDotEnvFile: %v", err)
	}
	if result.Imported != 2 {
		t.Errorf("Imported: want 2, got %d", result.Imported)
	}
	if result.Skipped != 1 {
		t.Errorf("Skipped: want 1, got %d", result.Skipped)
	}
}

func TestImporter_ImportDotEnvFile_NonexistentFile(t *testing.T) {
	mgr := newTestManager(t)
	imp := NewImporter(mgr)
	ctx := context.Background()

	_, err := imp.ImportDotEnvFile(ctx, "/no/such/file.env", ScopeGlobal)
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestImporter_ImportDotEnvFile_KeyLowercased(t *testing.T) {
	mgr := newTestManager(t)
	imp := NewImporter(mgr)
	ctx := context.Background()

	dir := t.TempDir()
	path := writeTempFile(t, dir, ".env", "UPPER_CASE_KEY=hello\n")

	result, err := imp.ImportDotEnvFile(ctx, path, ScopeGlobal)
	if err != nil {
		t.Fatalf("ImportDotEnvFile: %v", err)
	}
	if result.Imported != 1 {
		t.Fatalf("Imported: want 1, got %d", result.Imported)
	}

	// Secret name must be lowercased.
	_, err = mgr.Get(ctx, "upper_case_key", ScopeGlobal)
	if err != nil {
		t.Errorf("Get lower-cased key: %v", err)
	}
}

func TestImporter_ImportDotEnvFile_ValueWithEquals(t *testing.T) {
	mgr := newTestManager(t)
	imp := NewImporter(mgr)
	ctx := context.Background()

	dir := t.TempDir()
	// Value contains '=' characters (e.g., base64 padding).
	path := writeTempFile(t, dir, ".env", "ENCODED_KEY=abc=def==\n")

	result, err := imp.ImportDotEnvFile(ctx, path, ScopeGlobal)
	if err != nil {
		t.Fatalf("ImportDotEnvFile: %v", err)
	}
	if result.Imported != 1 {
		t.Fatalf("Imported: want 1, got %d", result.Imported)
	}

	val, err := mgr.Get(ctx, "encoded_key", ScopeGlobal)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(val) != "abc=def==" {
		t.Errorf("value with '=': want %q, got %q", "abc=def==", val)
	}
}

func TestImporter_ImportDotEnvFile_TypeInferredFromKey(t *testing.T) {
	mgr := newTestManager(t)
	imp := NewImporter(mgr)
	ctx := context.Background()

	dir := t.TempDir()
	path := writeTempFile(t, dir, ".env", "OPENAI_API_KEY=sk-test\n")

	result, err := imp.ImportDotEnvFile(ctx, path, ScopeGlobal)
	if err != nil {
		t.Fatalf("ImportDotEnvFile: %v", err)
	}
	if result.Imported != 1 {
		t.Fatalf("Imported: want 1, got %d", result.Imported)
	}

	meta, err := mgr.meta.GetMeta(ctx, "openai_api_key", ScopeGlobal)
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if meta.Type != TypeAPIKey {
		t.Errorf("Type: want api_key (inferred), got %q", meta.Type)
	}
}

// ---------------------------------------------------------------------------
// ImportAWSCredentialsFile
// ---------------------------------------------------------------------------

func TestImporter_ImportAWSCredentials_Default(t *testing.T) {
	mgr := newTestManager(t)
	imp := NewImporter(mgr)
	ctx := context.Background()

	dir := t.TempDir()
	path := writeTempFile(t, dir, "credentials", `[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
`)

	result, err := imp.ImportAWSCredentialsFile(ctx, path)
	if err != nil {
		t.Fatalf("ImportAWSCredentialsFile: %v", err)
	}
	if result.Imported != 2 {
		t.Errorf("Imported: want 2, got %d; errors: %v", result.Imported, result.Errors)
	}
	if result.Source != ImportAWSCredentials {
		t.Errorf("Source: want aws_credentials, got %q", result.Source)
	}

	// Secret name format: aws_<profile>_<key>
	val, err := mgr.Get(ctx, "aws_default_aws_access_key_id", ScopeGlobal)
	if err != nil {
		t.Fatalf("Get aws key id: %v", err)
	}
	if string(val) != "AKIAIOSFODNN7EXAMPLE" {
		t.Errorf("access key id: want AKIAIOSFODNN7EXAMPLE, got %q", val)
	}
}

func TestImporter_ImportAWSCredentials_MultipleProfiles(t *testing.T) {
	mgr := newTestManager(t)
	imp := NewImporter(mgr)
	ctx := context.Background()

	dir := t.TempDir()
	path := writeTempFile(t, dir, "credentials", `[default]
aws_access_key_id = KEY_DEFAULT
aws_secret_access_key = SECRET_DEFAULT

[staging]
aws_access_key_id = KEY_STAGING
aws_secret_access_key = SECRET_STAGING
`)

	result, err := imp.ImportAWSCredentialsFile(ctx, path)
	if err != nil {
		t.Fatalf("ImportAWSCredentialsFile: %v", err)
	}
	if result.Imported != 4 {
		t.Errorf("Imported: want 4, got %d; errors: %v", result.Imported, result.Errors)
	}

	// Check staging profile key exists.
	_, err = mgr.Get(ctx, "aws_staging_aws_access_key_id", ScopeGlobal)
	if err != nil {
		t.Errorf("Get staging access key: %v", err)
	}
}

func TestImporter_ImportAWSCredentials_IgnoresUnknownKeys(t *testing.T) {
	mgr := newTestManager(t)
	imp := NewImporter(mgr)
	ctx := context.Background()

	dir := t.TempDir()
	path := writeTempFile(t, dir, "credentials", `[default]
aws_access_key_id = KEY
aws_secret_access_key = SECRET
region = us-east-1
output = json
`)

	result, err := imp.ImportAWSCredentialsFile(ctx, path)
	if err != nil {
		t.Fatalf("ImportAWSCredentialsFile: %v", err)
	}
	// region and output must not be imported.
	if result.Imported != 2 {
		t.Errorf("Imported: want 2 (region/output ignored), got %d", result.Imported)
	}
}

func TestImporter_ImportAWSCredentials_EmptyFile(t *testing.T) {
	mgr := newTestManager(t)
	imp := NewImporter(mgr)
	ctx := context.Background()

	dir := t.TempDir()
	path := writeTempFile(t, dir, "credentials", "")

	result, err := imp.ImportAWSCredentialsFile(ctx, path)
	if err != nil {
		t.Fatalf("ImportAWSCredentialsFile empty: %v", err)
	}
	if result.Imported != 0 {
		t.Errorf("expected 0 imports from empty file, got %d", result.Imported)
	}
}

func TestImporter_ImportAWSCredentials_NonexistentFile(t *testing.T) {
	mgr := newTestManager(t)
	imp := NewImporter(mgr)
	ctx := context.Background()

	_, err := imp.ImportAWSCredentialsFile(ctx, "/no/such/credentials")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

// ---------------------------------------------------------------------------
// ImportJSON
// ---------------------------------------------------------------------------

func TestImporter_ImportJSON_Basic(t *testing.T) {
	mgr := newTestManager(t)
	imp := NewImporter(mgr)
	ctx := context.Background()

	payload := map[string]interface{}{
		"secrets": []map[string]interface{}{
			{
				"name":    "api_token",
				"value":   "tok-123",
				"type":    "api_key",
				"env_var": "API_TOKEN",
				"tags":    []string{"prod"},
			},
			{
				"name":    "db_password",
				"value":   "s3cr3t",
				"type":    "db_credential",
				"env_var": "DB_PASSWORD",
			},
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	result, err := imp.ImportJSON(ctx, data, ScopeGlobal)
	if err != nil {
		t.Fatalf("ImportJSON: %v", err)
	}
	if result.Imported != 2 {
		t.Errorf("Imported: want 2, got %d; errors: %v", result.Imported, result.Errors)
	}

	val, err := mgr.Get(ctx, "api_token", ScopeGlobal)
	if err != nil {
		t.Fatalf("Get api_token: %v", err)
	}
	if string(val) != "tok-123" {
		t.Errorf("api_token value: want tok-123, got %q", val)
	}
}

func TestImporter_ImportJSON_InvalidJSON(t *testing.T) {
	mgr := newTestManager(t)
	imp := NewImporter(mgr)
	ctx := context.Background()

	_, err := imp.ImportJSON(ctx, []byte("not-json{{{"), ScopeGlobal)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestImporter_ImportJSON_EmptySecrets(t *testing.T) {
	mgr := newTestManager(t)
	imp := NewImporter(mgr)
	ctx := context.Background()

	data := []byte(`{"secrets": []}`)
	result, err := imp.ImportJSON(ctx, data, ScopeGlobal)
	if err != nil {
		t.Fatalf("ImportJSON: %v", err)
	}
	if result.Imported != 0 {
		t.Errorf("Imported: want 0, got %d", result.Imported)
	}
}

func TestImporter_ImportJSON_SourceLabel(t *testing.T) {
	mgr := newTestManager(t)
	imp := NewImporter(mgr)
	ctx := context.Background()

	data := []byte(`{"secrets": [{"name": "my_key", "value": "v", "type": "custom"}]}`)
	result, err := imp.ImportJSON(ctx, data, ScopeGlobal)
	if err != nil {
		t.Fatalf("ImportJSON: %v", err)
	}
	if result.Source != ImportJSON {
		t.Errorf("Source: want json, got %q", result.Source)
	}
	_ = result
}

// ---------------------------------------------------------------------------
// Export and re-import round-trip
// ---------------------------------------------------------------------------

func TestImporter_ExportAndReImport(t *testing.T) {
	mgr := newTestManager(t)
	imp := NewImporter(mgr)
	ctx := context.Background()

	type fixture struct {
		name  string
		value string
	}
	fixtures := []fixture{
		{"export_key_a", "value-aaa"},
		{"export_key_b", "value-bbb"},
	}

	now := time.Now().UTC().Truncate(time.Second)
	for _, f := range fixtures {
		meta := SecretMetadata{
			Name:      f.name,
			Type:      TypeAPIKey,
			Scope:     ScopeGlobal,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := mgr.Set(ctx, meta, []byte(f.value)); err != nil {
			t.Fatalf("Set %s: %v", f.name, err)
		}
	}

	passphrase := []byte("export-passphrase")
	encrypted, err := imp.ExportEncrypted(ctx, ScopeGlobal, passphrase)
	if err != nil {
		t.Fatalf("ExportEncrypted: %v", err)
	}
	if len(encrypted) == 0 {
		t.Fatal("ExportEncrypted returned empty blob")
	}

	// Decrypt to get the plaintext export envelope.
	plaintext, err := DecryptWithPassphrase(passphrase, encrypted)
	if err != nil {
		t.Fatalf("DecryptWithPassphrase: %v", err)
	}

	// Parse envelope to extract secrets array.
	var envelope struct {
		Secrets []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"secrets"`
	}
	if err := json.Unmarshal(plaintext, &envelope); err != nil {
		t.Fatalf("unmarshal export: %v", err)
	}

	// Verify all secrets are present.
	found := map[string]string{}
	for _, s := range envelope.Secrets {
		found[s.Name] = s.Value
	}
	for _, want := range fixtures {
		if v, ok := found[want.name]; !ok {
			t.Errorf("secret %q missing from export", want.name)
		} else if v != want.value {
			t.Errorf("secret %q: want %q, got %q", want.name, want.value, v)
		}
	}

	// Re-import into a fresh manager.
	mgr2 := newTestManager(t)
	imp2 := NewImporter(mgr2)

	importData, err := json.Marshal(map[string]interface{}{"secrets": envelope.Secrets})
	if err != nil {
		t.Fatalf("marshal import data: %v", err)
	}

	result, err := imp2.ImportJSON(ctx, importData, ScopeGlobal)
	if err != nil {
		t.Fatalf("ImportJSON: %v", err)
	}
	if result.Imported != len(fixtures) {
		t.Errorf("re-import: want %d, got %d; errors: %v", len(fixtures), result.Imported, result.Errors)
	}

	for _, want := range fixtures {
		val, err := mgr2.Get(ctx, want.name, ScopeGlobal)
		if err != nil {
			t.Fatalf("Get %s after re-import: %v", want.name, err)
		}
		if !bytes.Equal(val, []byte(want.value)) {
			t.Errorf("re-imported %s: want %q, got %q", want.name, want.value, val)
		}
	}
}

func TestImporter_ExportEncrypted_WrongPassphraseCannotDecrypt(t *testing.T) {
	mgr := newTestManager(t)
	imp := NewImporter(mgr)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	meta := SecretMetadata{
		Name:      "test_secret",
		Type:      TypeCustom,
		Scope:     ScopeGlobal,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := mgr.Set(ctx, meta, []byte("value")); err != nil {
		t.Fatalf("Set: %v", err)
	}

	encrypted, err := imp.ExportEncrypted(ctx, ScopeGlobal, []byte("correct-pass"))
	if err != nil {
		t.Fatalf("ExportEncrypted: %v", err)
	}

	_, err = DecryptWithPassphrase([]byte("wrong-pass"), encrypted)
	if err == nil {
		t.Error("expected decryption failure with wrong passphrase")
	}
}

func TestImporter_ExportEncrypted_EmptyScope(t *testing.T) {
	mgr := newTestManager(t)
	imp := NewImporter(mgr)
	ctx := context.Background()

	// Nothing stored — export should return a valid (but empty secrets list) blob.
	encrypted, err := imp.ExportEncrypted(ctx, ScopeGlobal, []byte("pass"))
	if err != nil {
		t.Fatalf("ExportEncrypted empty: %v", err)
	}
	if len(encrypted) == 0 {
		t.Fatal("expected non-empty ciphertext even for empty export")
	}

	plaintext, err := DecryptWithPassphrase([]byte("pass"), encrypted)
	if err != nil {
		t.Fatalf("DecryptWithPassphrase: %v", err)
	}

	var envelope struct {
		Secrets []interface{} `json:"secrets"`
	}
	if err := json.Unmarshal(plaintext, &envelope); err != nil {
		t.Fatalf("unmarshal empty export: %v", err)
	}
	if len(envelope.Secrets) != 0 {
		t.Errorf("expected 0 secrets, got %d", len(envelope.Secrets))
	}
}

// ---------------------------------------------------------------------------
// SecretMetadata MarshalJSON (types.go)
// ---------------------------------------------------------------------------

func TestSecretMetadata_MarshalJSON(t *testing.T) {
	meta := simpleMeta("my_key", ScopeGlobal)
	meta.Type = TypeAPIKey
	meta.Description = "test"

	data, err := meta.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if len(data) == 0 {
		t.Error("MarshalJSON returned empty data")
	}
	// Must contain the name field.
	if !strings.Contains(string(data), "my_key") {
		t.Errorf("marshalled JSON does not contain name: %s", data)
	}
}

// ---------------------------------------------------------------------------
// guessSecretType — type inference from env var names
// ---------------------------------------------------------------------------

func TestGuessSecretType(t *testing.T) {
	tests := []struct {
		envVar string
		want   SecretType
	}{
		{"OPENAI_API_KEY", TypeAPIKey},
		{"ANTHROPIC_API_KEY", TypeAPIKey},
		{"MY_APIKEY", TypeAPIKey},
		{"GOOGLE_AI_TOKEN", TypeAPIKey},
		{"OPENROUTER_API_KEY", TypeAPIKey},
		{"AWS_ACCESS_KEY_ID", TypeCloudCred},
		{"GCP_PROJECT_ID", TypeCloudCred},
		{"AZURE_CLIENT_SECRET", TypeCloudCred},
		{"GITHUB_TOKEN", TypeGitToken},
		{"GITLAB_TOKEN", TypeGitToken},
		{"GIT_TOKEN_PERSONAL", TypeGitToken},
		{"DATABASE_URL", TypeDBCred},
		{"DB_PASSWORD", TypeDBCred},
		{"POSTGRES_URL", TypeDBCred},
		{"MYSQL_HOST", TypeDBCred},
		{"MONGO_URI", TypeDBCred},
		{"REDIS_URL", TypeDBCred},
		{"SSH_PRIVATE_KEY", TypeSSHKey},
		{"KUBECONFIG", TypeKubeConfig},
		{"K8S_TOKEN", TypeKubeConfig},
		{"VPN_KEY", TypeVPN},
		{"WIREGUARD_PRIVATE_KEY", TypeVPN},
		{"OPENVPN_CONFIG", TypeVPN},
		{"MY_CUSTOM_SECRET", TypeCustom},
		{"APP_PASSWORD", TypeCustom},
	}

	for _, tt := range tests {
		t.Run(tt.envVar, func(t *testing.T) {
			got := guessSecretType(tt.envVar)
			if got != tt.want {
				t.Errorf("guessSecretType(%q): want %q, got %q", tt.envVar, tt.want, got)
			}
		})
	}
}

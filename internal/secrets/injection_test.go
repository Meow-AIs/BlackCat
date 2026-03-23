package secrets

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Test helpers (Manager + SecureManager + Injector chain)
// ---------------------------------------------------------------------------

// newTestInjector builds a complete Injector stack:
//
//	Manager (memBackend + SQLite meta) → DefaultAccessPolicy → SecureManager → Injector
func newTestInjector(t *testing.T) (*Injector, *Manager) {
	t.Helper()
	mgr := newTestManager(t)
	policy := NewDefaultAccessPolicy()
	secureMgr := NewSecureManager(mgr, policy)
	inj := NewInjector(secureMgr)
	return inj, mgr
}

// storeSecret is a convenience helper that saves a secret via the Manager.
func storeSecret(t *testing.T, mgr *Manager, name string, scope Scope, envVar string, value string) SecretMetadata {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	meta := SecretMetadata{
		Name:      name,
		Type:      TypeAPIKey,
		Scope:     scope,
		EnvVar:    envVar,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := mgr.Set(context.Background(), meta, []byte(value)); err != nil {
		t.Fatalf("storeSecret %s: %v", name, err)
	}
	return meta
}

// primaryCtx returns an AccessContext for the primary agent.
func primaryCtx() AccessContext {
	return AccessContext{AgentID: "primary"}
}

// ---------------------------------------------------------------------------
// InjectIntoCmd
// ---------------------------------------------------------------------------

func TestInjector_InjectIntoCmd_AddsSecretAsEnvVar(t *testing.T) {
	inj, mgr := newTestInjector(t)
	ctx := context.Background()

	storeSecret(t, mgr, "openai_api_key", ScopeGlobal, "OPENAI_API_KEY", "sk-test-key")

	cmd := exec.Command("echo")
	req := InjectionRequest{
		Secrets: map[string]SecretRef{
			"OPENAI_API_KEY": {Name: "openai_api_key", Scope: ScopeGlobal},
		},
		Requester: primaryCtx(),
	}

	if err := inj.InjectIntoCmd(ctx, cmd, req); err != nil {
		t.Fatalf("InjectIntoCmd: %v", err)
	}

	// The env var must appear in cmd.Env.
	found := false
	for _, e := range cmd.Env {
		if e == "OPENAI_API_KEY=sk-test-key" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("OPENAI_API_KEY not found in cmd.Env: %v", cmd.Env)
	}
}

func TestInjector_InjectIntoCmd_MultipleSecrets(t *testing.T) {
	inj, mgr := newTestInjector(t)
	ctx := context.Background()

	storeSecret(t, mgr, "key_a", ScopeGlobal, "KEY_A", "value_a")
	storeSecret(t, mgr, "key_b", ScopeGlobal, "KEY_B", "value_b")

	cmd := exec.Command("echo")
	req := InjectionRequest{
		Secrets: map[string]SecretRef{
			"KEY_A": {Name: "key_a", Scope: ScopeGlobal},
			"KEY_B": {Name: "key_b", Scope: ScopeGlobal},
		},
		Requester: primaryCtx(),
	}

	if err := inj.InjectIntoCmd(ctx, cmd, req); err != nil {
		t.Fatalf("InjectIntoCmd: %v", err)
	}

	envMap := parseEnvSlice(cmd.Env)
	if envMap["KEY_A"] != "value_a" {
		t.Errorf("KEY_A: want value_a, got %q", envMap["KEY_A"])
	}
	if envMap["KEY_B"] != "value_b" {
		t.Errorf("KEY_B: want value_b, got %q", envMap["KEY_B"])
	}
}

func TestInjector_InjectIntoCmd_MissingSecret_ReturnsError(t *testing.T) {
	inj, _ := newTestInjector(t)
	ctx := context.Background()

	cmd := exec.Command("echo")
	req := InjectionRequest{
		Secrets: map[string]SecretRef{
			"MISSING_VAR": {Name: "does_not_exist", Scope: ScopeGlobal},
		},
		Requester: primaryCtx(),
	}

	err := inj.InjectIntoCmd(ctx, cmd, req)
	if err == nil {
		t.Error("expected error for missing secret, got nil")
	}
}

func TestInjector_InjectIntoCmd_PreservesExistingEnv(t *testing.T) {
	inj, mgr := newTestInjector(t)
	ctx := context.Background()

	storeSecret(t, mgr, "new_key", ScopeGlobal, "NEW_KEY", "new_value")

	cmd := exec.Command("echo")
	cmd.Env = []string{"EXISTING=already_here"}

	req := InjectionRequest{
		Secrets: map[string]SecretRef{
			"NEW_KEY": {Name: "new_key", Scope: ScopeGlobal},
		},
		Requester: primaryCtx(),
	}

	if err := inj.InjectIntoCmd(ctx, cmd, req); err != nil {
		t.Fatalf("InjectIntoCmd: %v", err)
	}

	envMap := parseEnvSlice(cmd.Env)
	if envMap["EXISTING"] != "already_here" {
		t.Errorf("existing env var lost: EXISTING=%q", envMap["EXISTING"])
	}
	if envMap["NEW_KEY"] != "new_value" {
		t.Errorf("new env var missing: NEW_KEY=%q", envMap["NEW_KEY"])
	}
}

// ---------------------------------------------------------------------------
// FilteredPrefixes — sanitizeEnv
// ---------------------------------------------------------------------------

func TestSanitizeEnv_FiltersBlackcatSecrets(t *testing.T) {
	base := []string{
		"NORMAL_VAR=keep",
		"BLACKCAT_SECRET_HIDDEN=secret_value",
		"ANOTHER_VAR=also_keep",
		"BLACKCAT_SECRET_TOKEN=another_secret",
	}

	cleaned := sanitizeEnv(base, nil)

	cleanMap := parseEnvSlice(cleaned)
	if cleanMap["NORMAL_VAR"] != "keep" {
		t.Errorf("NORMAL_VAR should be kept, got %q", cleanMap["NORMAL_VAR"])
	}
	if cleanMap["ANOTHER_VAR"] != "also_keep" {
		t.Errorf("ANOTHER_VAR should be kept, got %q", cleanMap["ANOTHER_VAR"])
	}
	if _, ok := cleanMap["BLACKCAT_SECRET_HIDDEN"]; ok {
		t.Error("BLACKCAT_SECRET_HIDDEN should be filtered out")
	}
	if _, ok := cleanMap["BLACKCAT_SECRET_TOKEN"]; ok {
		t.Error("BLACKCAT_SECRET_TOKEN should be filtered out")
	}
}

func TestSanitizeEnv_CustomFilteredPrefixes(t *testing.T) {
	base := []string{
		"NORMAL=keep",
		"SENSITIVE_DATA=strip_this",
		"ALSO_SENSITIVE=strip_this_too",
	}

	cleaned := sanitizeEnv(base, []string{"SENSITIVE_", "ALSO_"})
	cleanMap := parseEnvSlice(cleaned)

	if cleanMap["NORMAL"] != "keep" {
		t.Errorf("NORMAL should be kept")
	}
	if _, ok := cleanMap["SENSITIVE_DATA"]; ok {
		t.Error("SENSITIVE_DATA should be filtered")
	}
	if _, ok := cleanMap["ALSO_SENSITIVE"]; ok {
		t.Error("ALSO_SENSITIVE should be filtered")
	}
}

func TestSanitizeEnv_NilBase_ReturnsNil(t *testing.T) {
	result := sanitizeEnv(nil, nil)
	if result != nil {
		t.Errorf("nil base should return nil, got %v", result)
	}
}

func TestSanitizeEnv_EmptyBase(t *testing.T) {
	result := sanitizeEnv([]string{}, nil)
	if len(result) != 0 {
		t.Errorf("empty base should return empty, got %v", result)
	}
}

// ---------------------------------------------------------------------------
// AutoInjectForScope
// ---------------------------------------------------------------------------

func TestInjector_AutoInjectForScope_OnlyEnvVarSecrets(t *testing.T) {
	inj, mgr := newTestInjector(t)
	ctx := context.Background()

	// Secret with EnvVar configured — should be injected.
	storeSecret(t, mgr, "auto_key", ScopeGlobal, "AUTO_KEY", "auto_value")

	// Secret without EnvVar — should be skipped.
	now := time.Now().UTC().Truncate(time.Second)
	noEnvMeta := SecretMetadata{
		Name:      "no_env_key",
		Type:      TypeCustom,
		Scope:     ScopeGlobal,
		EnvVar:    "", // empty — no auto-inject
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := mgr.Set(ctx, noEnvMeta, []byte("hidden")); err != nil {
		t.Fatalf("Set no_env_key: %v", err)
	}

	env, err := inj.AutoInjectForScope(ctx, ScopeGlobal, primaryCtx())
	if err != nil {
		t.Fatalf("AutoInjectForScope: %v", err)
	}

	envMap := parseEnvSlice(env)
	if envMap["AUTO_KEY"] != "auto_value" {
		t.Errorf("AUTO_KEY: want auto_value, got %q", envMap["AUTO_KEY"])
	}
	// no_env_key has no EnvVar so it must not appear under any name.
	for _, e := range env {
		if strings.Contains(e, "hidden") {
			t.Errorf("secret without EnvVar should not be injected: found %q", e)
		}
	}
}

func TestInjector_AutoInjectForScope_EmptyScope(t *testing.T) {
	inj, _ := newTestInjector(t)
	ctx := context.Background()

	env, err := inj.AutoInjectForScope(ctx, ScopeGlobal, primaryCtx())
	if err != nil {
		t.Fatalf("AutoInjectForScope on empty scope: %v", err)
	}
	if len(env) != 0 {
		t.Errorf("expected 0 env vars for empty scope, got %d", len(env))
	}
}

func TestInjector_AutoInjectForScope_AccessDenied_Skipped(t *testing.T) {
	inj, mgr := newTestInjector(t)
	ctx := context.Background()

	// Store a secret restricted to a specific tool that the requester does not use.
	now := time.Now().UTC().Truncate(time.Second)
	restrictedMeta := SecretMetadata{
		Name:         "restricted_key",
		Type:         TypeAPIKey,
		Scope:        ScopeGlobal,
		EnvVar:       "RESTRICTED_KEY",
		AllowedTools: []string{"only_this_tool"},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := mgr.Set(ctx, restrictedMeta, []byte("restricted_value")); err != nil {
		t.Fatalf("Set restricted_key: %v", err)
	}

	// Requester uses "other_tool" which is not in AllowedTools — access denied.
	requester := AccessContext{
		AgentID:  "primary",
		ToolName: "other_tool",
	}

	env, err := inj.AutoInjectForScope(ctx, ScopeGlobal, requester)
	if err != nil {
		t.Fatalf("AutoInjectForScope: %v", err)
	}
	// Denied secret should be silently skipped, not an error.
	envMap := parseEnvSlice(env)
	if _, ok := envMap["RESTRICTED_KEY"]; ok {
		t.Error("access-denied secret should not appear in auto-inject output")
	}
}

// ---------------------------------------------------------------------------
// BuildEnvSlice
// ---------------------------------------------------------------------------

func TestInjector_BuildEnvSlice_Basic(t *testing.T) {
	inj, mgr := newTestInjector(t)
	ctx := context.Background()

	storeSecret(t, mgr, "build_key", ScopeGlobal, "BUILD_KEY", "build_value")

	req := InjectionRequest{
		Secrets: map[string]SecretRef{
			"BUILD_KEY": {Name: "build_key", Scope: ScopeGlobal},
		},
		Requester: primaryCtx(),
	}

	env, err := inj.BuildEnvSlice(ctx, req)
	if err != nil {
		t.Fatalf("BuildEnvSlice: %v", err)
	}

	envMap := parseEnvSlice(env)
	if envMap["BUILD_KEY"] != "build_value" {
		t.Errorf("BUILD_KEY: want build_value, got %q", envMap["BUILD_KEY"])
	}
}

func TestInjector_BuildEnvSlice_MissingSecret_ReturnsError(t *testing.T) {
	inj, _ := newTestInjector(t)
	ctx := context.Background()

	req := InjectionRequest{
		Secrets: map[string]SecretRef{
			"MISSING": {Name: "missing_secret", Scope: ScopeGlobal},
		},
		Requester: primaryCtx(),
	}

	_, err := inj.BuildEnvSlice(ctx, req)
	if err == nil {
		t.Error("expected error for missing secret in BuildEnvSlice")
	}
}

// ---------------------------------------------------------------------------
// Access control integration
// ---------------------------------------------------------------------------

func TestInjector_SubAgentBlockedByDefault(t *testing.T) {
	inj, mgr := newTestInjector(t)
	ctx := context.Background()

	storeSecret(t, mgr, "primary_only", ScopeGlobal, "PRIMARY_ONLY", "secret_val")

	// Sub-agent does not have access unless explicitly listed in AllowedAgents.
	subAgentCtx := AccessContext{
		AgentID: "sub-agent:worker1",
	}

	req := InjectionRequest{
		Secrets: map[string]SecretRef{
			"PRIMARY_ONLY": {Name: "primary_only", Scope: ScopeGlobal},
		},
		Requester: subAgentCtx,
	}

	err := inj.InjectIntoCmd(ctx, exec.Command("echo"), req)
	if err == nil {
		t.Error("expected access denied for sub-agent, got nil")
	}
}

func TestInjector_AllowedAgentCanAccess(t *testing.T) {
	inj, mgr := newTestInjector(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	meta := SecretMetadata{
		Name:          "worker_key",
		Type:          TypeAPIKey,
		Scope:         ScopeGlobal,
		EnvVar:        "WORKER_KEY",
		AllowedAgents: []string{"sub-agent:worker1"},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := mgr.Set(ctx, meta, []byte("worker_value")); err != nil {
		t.Fatalf("Set worker_key: %v", err)
	}

	workerCtx := AccessContext{
		AgentID: "sub-agent:worker1",
	}

	cmd := exec.Command("echo")
	req := InjectionRequest{
		Secrets: map[string]SecretRef{
			"WORKER_KEY": {Name: "worker_key", Scope: ScopeGlobal},
		},
		Requester: workerCtx,
	}

	if err := inj.InjectIntoCmd(ctx, cmd, req); err != nil {
		t.Fatalf("InjectIntoCmd for allowed sub-agent: %v", err)
	}

	envMap := parseEnvSlice(cmd.Env)
	if envMap["WORKER_KEY"] != "worker_value" {
		t.Errorf("WORKER_KEY: want worker_value, got %q", envMap["WORKER_KEY"])
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// parseEnvSlice converts a []string of "KEY=VALUE" into a map.
func parseEnvSlice(env []string) map[string]string {
	m := make(map[string]string, len(env))
	for _, entry := range env {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) == 2 {
			m[parts[0]] = parts[1]
		}
	}
	return m
}

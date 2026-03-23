package secrets

import (
	"context"
	"errors"
	"testing"
)

// ---------------------------------------------------------------------------
// DefaultAccessPolicy.CheckAccess
// ---------------------------------------------------------------------------

func TestDefaultAccessPolicy_PrimaryAgentAllowed(t *testing.T) {
	policy := NewDefaultAccessPolicy()
	meta := SecretMetadata{
		Name:  "my-secret",
		Scope: ScopeGlobal,
	}
	req := AccessContext{AgentID: "primary"}
	err := policy.CheckAccess(context.Background(), meta, req)
	if err != nil {
		t.Errorf("primary agent should be allowed, got %v", err)
	}
}

func TestDefaultAccessPolicy_SubAgentBlockedByDefault(t *testing.T) {
	policy := NewDefaultAccessPolicy()
	meta := SecretMetadata{
		Name:          "sub-blocked",
		Scope:         ScopeGlobal,
		AllowedAgents: nil, // empty = primary only
	}
	req := AccessContext{AgentID: "sub-agent:worker-1"}
	err := policy.CheckAccess(context.Background(), meta, req)
	if !errors.Is(err, ErrAccessDenied) {
		t.Errorf("sub-agent should be denied when AllowedAgents is empty, got %v", err)
	}
}

func TestDefaultAccessPolicy_SubAgentExplicitlyAllowed(t *testing.T) {
	policy := NewDefaultAccessPolicy()
	meta := SecretMetadata{
		Name:          "sub-allowed",
		Scope:         ScopeGlobal,
		AllowedAgents: []string{"sub-agent:worker-1", "sub-agent:worker-2"},
	}
	tests := []struct {
		agentID string
		allowed bool
	}{
		{"primary", true},
		{"sub-agent:worker-1", true},
		{"sub-agent:worker-2", true},
		{"sub-agent:worker-3", false},
		{"sub-agent:unknown", false},
	}
	for _, tt := range tests {
		t.Run(tt.agentID, func(t *testing.T) {
			req := AccessContext{AgentID: tt.agentID}
			err := policy.CheckAccess(context.Background(), meta, req)
			if tt.allowed && err != nil {
				t.Errorf("expected access allowed for %q, got %v", tt.agentID, err)
			}
			if !tt.allowed && !errors.Is(err, ErrAccessDenied) {
				t.Errorf("expected ErrAccessDenied for %q, got %v", tt.agentID, err)
			}
		})
	}
}

func TestDefaultAccessPolicy_ToolAllowlist(t *testing.T) {
	policy := NewDefaultAccessPolicy()
	meta := SecretMetadata{
		Name:         "tool-guarded",
		Scope:        ScopeGlobal,
		AllowedTools: []string{"shell", "http"},
	}
	tests := []struct {
		tool    string
		allowed bool
	}{
		{"shell", true},
		{"http", true},
		{"database", false},
		{"file", false},
	}
	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			req := AccessContext{AgentID: "primary", ToolName: tt.tool}
			err := policy.CheckAccess(context.Background(), meta, req)
			if tt.allowed && err != nil {
				t.Errorf("tool %q should be allowed, got %v", tt.tool, err)
			}
			if !tt.allowed && !errors.Is(err, ErrAccessDenied) {
				t.Errorf("tool %q should be denied, got %v", tt.tool, err)
			}
		})
	}
}

func TestDefaultAccessPolicy_EmptyToolName_SkipsToolCheck(t *testing.T) {
	// If the requester does not specify a tool name, the allowlist is not enforced.
	policy := NewDefaultAccessPolicy()
	meta := SecretMetadata{
		Name:         "tool-guarded",
		Scope:        ScopeGlobal,
		AllowedTools: []string{"shell"},
	}
	req := AccessContext{AgentID: "primary", ToolName: ""}
	err := policy.CheckAccess(context.Background(), meta, req)
	if err != nil {
		t.Errorf("empty ToolName should skip tool check, got %v", err)
	}
}

func TestDefaultAccessPolicy_ProjectScopeEnforcement(t *testing.T) {
	policy := NewDefaultAccessPolicy()
	meta := SecretMetadata{
		Name:        "project-secret",
		Scope:       ScopeProject,
		ProjectPath: "/home/user/projectA",
	}
	tests := []struct {
		projectPath string
		allowed     bool
	}{
		{"/home/user/projectA", true},
		{"/home/user/projectB", false},
		{"", false},
		{"/home/user/projectA/subdir", false},
	}
	for _, tt := range tests {
		t.Run(tt.projectPath, func(t *testing.T) {
			req := AccessContext{AgentID: "primary", ProjectPath: tt.projectPath}
			err := policy.CheckAccess(context.Background(), meta, req)
			if tt.allowed && err != nil {
				t.Errorf("project %q should be allowed, got %v", tt.projectPath, err)
			}
			if !tt.allowed && !errors.Is(err, ErrAccessDenied) {
				t.Errorf("project %q should be denied, got %v", tt.projectPath, err)
			}
		})
	}
}

func TestDefaultAccessPolicy_ProjectScopeNoPath_NotEnforced(t *testing.T) {
	// If meta.ProjectPath is empty, scope enforcement does not apply.
	policy := NewDefaultAccessPolicy()
	meta := SecretMetadata{
		Name:        "project-secret-no-path",
		Scope:       ScopeProject,
		ProjectPath: "",
	}
	req := AccessContext{AgentID: "primary", ProjectPath: "/any/path"}
	err := policy.CheckAccess(context.Background(), meta, req)
	if err != nil {
		t.Errorf("no-path project secret should be accessible from any project, got %v", err)
	}
}

func TestDefaultAccessPolicy_GlobalScope_NoProjectEnforcement(t *testing.T) {
	policy := NewDefaultAccessPolicy()
	meta := SecretMetadata{
		Name:        "global-secret",
		Scope:       ScopeGlobal,
		ProjectPath: "/some/project", // has a path but scope is global
	}
	req := AccessContext{AgentID: "primary", ProjectPath: "/different/project"}
	err := policy.CheckAccess(context.Background(), meta, req)
	if err != nil {
		t.Errorf("global scope should not enforce project path, got %v", err)
	}
}

func TestDefaultAccessPolicy_CombinedRules(t *testing.T) {
	policy := NewDefaultAccessPolicy()
	meta := SecretMetadata{
		Name:          "combo-secret",
		Scope:         ScopeProject,
		ProjectPath:   "/project/path",
		AllowedTools:  []string{"shell"},
		AllowedAgents: []string{"sub-agent:trusted"},
	}
	tests := []struct {
		name         string
		req          AccessContext
		expectDenied bool
	}{
		{
			"correct project + correct tool + allowed sub-agent",
			AccessContext{AgentID: "sub-agent:trusted", ToolName: "shell", ProjectPath: "/project/path"},
			false,
		},
		{
			"wrong project",
			AccessContext{AgentID: "primary", ToolName: "shell", ProjectPath: "/wrong/path"},
			true,
		},
		{
			"wrong tool",
			AccessContext{AgentID: "primary", ToolName: "database", ProjectPath: "/project/path"},
			true,
		},
		{
			"unlisted sub-agent",
			AccessContext{AgentID: "sub-agent:rogue", ToolName: "shell", ProjectPath: "/project/path"},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := policy.CheckAccess(context.Background(), meta, tt.req)
			if tt.expectDenied && !errors.Is(err, ErrAccessDenied) {
				t.Errorf("expected ErrAccessDenied, got %v", err)
			}
			if !tt.expectDenied && err != nil {
				t.Errorf("expected allowed, got %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// SecureManager.Get
// ---------------------------------------------------------------------------

// acTestManager builds a Manager backed by mgrTestBackend and mgrMetaStore (from manager_test.go).
func acTestManager(t *testing.T) (*Manager, *mgrMetaStore) {
	t.Helper()
	be := newMgrBackend()
	ms := newMgrMetaStore()
	al := &mgrAuditLog{}
	mgr, err := NewManager(ManagerOpts{
		Backends:      []Backend{be},
		MetadataStore: ms,
		AuditLog:      al,
	})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	return mgr, ms
}

func TestSecureManager_Get_AllowedAccess(t *testing.T) {
	mgr, _ := acTestManager(t)
	mgrSetSecret(t, mgr, "allowed-secret", ScopeGlobal, []byte("the-value"))

	sm := NewSecureManager(mgr, NewDefaultAccessPolicy())
	req := AccessContext{AgentID: "primary"}
	got, err := sm.Get(context.Background(), "allowed-secret", ScopeGlobal, req)
	if err != nil {
		t.Fatalf("SecureManager.Get: %v", err)
	}
	if string(got) != "the-value" {
		t.Errorf("unexpected value: %q", got)
	}
}

func TestSecureManager_Get_DeniedSubAgent(t *testing.T) {
	mgr, _ := acTestManager(t)
	mgrSetSecret(t, mgr, "restricted-secret", ScopeGlobal, []byte("value"))

	sm := NewSecureManager(mgr, NewDefaultAccessPolicy())
	req := AccessContext{AgentID: "sub-agent:worker"}
	_, err := sm.Get(context.Background(), "restricted-secret", ScopeGlobal, req)
	if !errors.Is(err, ErrAccessDenied) {
		t.Errorf("expected ErrAccessDenied for sub-agent, got %v", err)
	}
}

func TestSecureManager_Get_DeniedWrongProject(t *testing.T) {
	mgr, ms := acTestManager(t)

	_ = ms.SaveMeta(context.Background(), SecretMetadata{
		Name:        "project-only",
		Scope:       ScopeProject,
		ProjectPath: "/project/alpha",
	})
	_ = mgr.primary.Set(
		context.Background(),
		storageKey("project-only", ScopeProject),
		[]byte("value"),
	)

	sm := NewSecureManager(mgr, NewDefaultAccessPolicy())
	req := AccessContext{AgentID: "primary", ProjectPath: "/project/beta"}
	_, err := sm.Get(context.Background(), "project-only", ScopeProject, req)
	if !errors.Is(err, ErrAccessDenied) {
		t.Errorf("expected ErrAccessDenied for wrong project, got %v", err)
	}
}

func TestSecureManager_Get_NotFound(t *testing.T) {
	mgr, _ := acTestManager(t)
	sm := NewSecureManager(mgr, NewDefaultAccessPolicy())
	req := AccessContext{AgentID: "primary"}
	_, err := sm.Get(context.Background(), "no-such-secret", ScopeGlobal, req)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestSecureManager_Get_AllowedSubAgent(t *testing.T) {
	mgr, ms := acTestManager(t)

	_ = ms.SaveMeta(context.Background(), SecretMetadata{
		Name:          "sub-allowed-secret",
		Scope:         ScopeGlobal,
		AllowedAgents: []string{"sub-agent:trusted"},
	})
	_ = mgr.primary.Set(
		context.Background(),
		storageKey("sub-allowed-secret", ScopeGlobal),
		[]byte("trusted-value"),
	)

	sm := NewSecureManager(mgr, NewDefaultAccessPolicy())
	req := AccessContext{AgentID: "sub-agent:trusted"}
	got, err := sm.Get(context.Background(), "sub-allowed-secret", ScopeGlobal, req)
	if err != nil {
		t.Fatalf("expected success for explicitly allowed sub-agent, got %v", err)
	}
	if string(got) != "trusted-value" {
		t.Errorf("unexpected value: %q", got)
	}
}

func TestSecureManager_Get_ToolNotAllowed(t *testing.T) {
	mgr, ms := acTestManager(t)

	_ = ms.SaveMeta(context.Background(), SecretMetadata{
		Name:         "tool-restricted",
		Scope:        ScopeGlobal,
		AllowedTools: []string{"http"},
	})
	_ = mgr.primary.Set(
		context.Background(),
		storageKey("tool-restricted", ScopeGlobal),
		[]byte("value"),
	)

	sm := NewSecureManager(mgr, NewDefaultAccessPolicy())
	req := AccessContext{AgentID: "primary", ToolName: "shell"}
	_, err := sm.Get(context.Background(), "tool-restricted", ScopeGlobal, req)
	if !errors.Is(err, ErrAccessDenied) {
		t.Errorf("expected ErrAccessDenied for disallowed tool, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// contains helper (internal function)
// ---------------------------------------------------------------------------

func TestContains(t *testing.T) {
	tests := []struct {
		slice []string
		item  string
		want  bool
	}{
		{[]string{"a", "b", "c"}, "b", true},
		{[]string{"a", "b", "c"}, "d", false},
		{[]string{}, "a", false},
		{nil, "a", false},
	}
	for _, tt := range tests {
		got := contains(tt.slice, tt.item)
		if got != tt.want {
			t.Errorf("contains(%v, %q) = %v, want %v", tt.slice, tt.item, got, tt.want)
		}
	}
}

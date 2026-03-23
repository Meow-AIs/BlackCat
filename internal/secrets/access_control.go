package secrets

import (
	"context"
	"fmt"
)

// AccessContext describes who is requesting a secret and why.
type AccessContext struct {
	// AgentID is "primary" for the main agent, or "sub-agent:<id>" for sub-agents.
	AgentID string

	// ToolName is the name of the tool requesting the secret (e.g. "shell", "http").
	ToolName string

	// ProjectPath is the current working project root.
	ProjectPath string

	// Reason is a human-readable description of why the secret is needed.
	Reason string
}

// AccessPolicy decides whether a given AccessContext may access a secret.
type AccessPolicy interface {
	// CheckAccess returns nil if access is allowed, ErrAccessDenied otherwise.
	CheckAccess(ctx context.Context, meta SecretMetadata, requester AccessContext) error
}

// DefaultAccessPolicy implements the standard access control rules:
//
//  1. Project-scoped secrets are only accessible from matching project paths.
//  2. If AllowedTools is non-empty, only listed tools may access the secret.
//  3. If AllowedAgents is non-empty, only listed agent IDs may access the secret.
//     An empty AllowedAgents list means only the primary agent (no sub-agents).
//  4. Sub-agents never get access to secrets unless explicitly listed in AllowedAgents.
type DefaultAccessPolicy struct{}

// NewDefaultAccessPolicy creates the default access policy.
func NewDefaultAccessPolicy() *DefaultAccessPolicy {
	return &DefaultAccessPolicy{}
}

func (p *DefaultAccessPolicy) CheckAccess(_ context.Context, meta SecretMetadata, req AccessContext) error {
	// Rule 1: Project scope enforcement.
	if meta.Scope == ScopeProject && meta.ProjectPath != "" {
		if req.ProjectPath != meta.ProjectPath {
			return fmt.Errorf("%w: secret %q is scoped to project %q, current project is %q",
				ErrAccessDenied, meta.Name, meta.ProjectPath, req.ProjectPath)
		}
	}

	// Rule 2: Tool allowlist.
	if len(meta.AllowedTools) > 0 && req.ToolName != "" {
		if !contains(meta.AllowedTools, req.ToolName) {
			return fmt.Errorf("%w: tool %q is not in the allowed list for secret %q",
				ErrAccessDenied, req.ToolName, meta.Name)
		}
	}

	// Rule 3: Agent allowlist — sub-agents are blocked by default.
	if req.AgentID != "primary" {
		if len(meta.AllowedAgents) == 0 {
			return fmt.Errorf("%w: sub-agent %q cannot access secret %q (not in allowed agents)",
				ErrAccessDenied, req.AgentID, meta.Name)
		}
		if !contains(meta.AllowedAgents, req.AgentID) {
			return fmt.Errorf("%w: agent %q is not in the allowed list for secret %q",
				ErrAccessDenied, req.AgentID, meta.Name)
		}
	}

	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// SecureManager wraps Manager with access control policy enforcement.
// This is the interface that tool execution and sub-agents use.
type SecureManager struct {
	manager *Manager
	policy  AccessPolicy
}

// NewSecureManager creates a SecureManager with the given policy.
func NewSecureManager(manager *Manager, policy AccessPolicy) *SecureManager {
	return &SecureManager{
		manager: manager,
		policy:  policy,
	}
}

// Get retrieves a secret after checking access control.
func (sm *SecureManager) Get(ctx context.Context, name string, scope Scope, requester AccessContext) ([]byte, error) {
	meta, err := sm.manager.meta.GetMeta(ctx, name, scope)
	if err != nil {
		return nil, err
	}

	if err := sm.policy.CheckAccess(ctx, meta, requester); err != nil {
		sm.manager.logAudit(ctx, name, scope, "read", requester.AgentID, false, err.Error())
		return nil, err
	}

	val, err := sm.manager.Get(ctx, name, scope)
	if err != nil {
		return nil, err
	}

	sm.manager.logAudit(ctx, name, scope, "read", requester.AgentID, true, requester.Reason)
	return val, nil
}

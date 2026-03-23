package secrets

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Injector prepares environment variables for subprocess execution.
// Secrets are injected as env vars and never passed as command-line arguments
// (which would be visible in /proc and ps output).
type Injector struct {
	manager *SecureManager
}

// NewInjector creates a secret injector.
func NewInjector(manager *SecureManager) *Injector {
	return &Injector{manager: manager}
}

// InjectionRequest specifies which secrets to inject and how.
type InjectionRequest struct {
	// Secrets lists the secrets to inject, keyed by the target env var name.
	// Example: {"OPENAI_API_KEY": SecretRef{Name: "openai_api_key", Scope: ScopeGlobal}}
	Secrets map[string]SecretRef

	// Requester is the access context for authorization.
	Requester AccessContext

	// InheritEnv controls whether the parent process environment is inherited.
	// Default: true. Set to false for maximum isolation.
	InheritEnv bool

	// FilteredPrefixes lists env var prefixes to strip from the inherited environment.
	// Default: strips BLACKCAT_SECRET_* to prevent secret leakage.
	FilteredPrefixes []string
}

// InjectIntoCmd populates a *exec.Cmd with the requested secrets as environment
// variables. The cmd's Env field is set; any previous Env entries are preserved.
//
// This is the ONLY approved way to pass secrets to subprocesses. Never use
// command-line arguments or stdin for secrets.
func (inj *Injector) InjectIntoCmd(ctx context.Context, cmd *exec.Cmd, req InjectionRequest) error {
	env := cmd.Env
	if env == nil && req.InheritEnv {
		env = sanitizeEnv(nil, req.FilteredPrefixes)
	}

	for envVar, ref := range req.Secrets {
		val, err := inj.manager.Get(ctx, ref.Name, ref.Scope, req.Requester)
		if err != nil {
			return fmt.Errorf("inject secret %q as %s: %w", ref.Name, envVar, err)
		}
		env = append(env, envVar+"="+string(val))
		SecureWipe(val)
	}

	cmd.Env = env
	return nil
}

// BuildEnvSlice resolves secrets and returns a slice of "KEY=VALUE" strings
// suitable for os/exec.Cmd.Env. This is useful when you need the env slice
// but are not using *exec.Cmd directly (e.g., for MCP server spawning).
func (inj *Injector) BuildEnvSlice(ctx context.Context, req InjectionRequest) ([]string, error) {
	var env []string

	for envVar, ref := range req.Secrets {
		val, err := inj.manager.Get(ctx, ref.Name, ref.Scope, req.Requester)
		if err != nil {
			return nil, fmt.Errorf("resolve secret %q as %s: %w", ref.Name, envVar, err)
		}
		env = append(env, envVar+"="+string(val))
		SecureWipe(val)
	}

	return env, nil
}

// AutoInjectForScope resolves all secrets that have an EnvVar configured and
// match the given scope. Returns env slice with all auto-injectable secrets.
func (inj *Injector) AutoInjectForScope(ctx context.Context, scope Scope, requester AccessContext) ([]string, error) {
	metas, err := inj.manager.manager.List(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("list secrets for auto-injection: %w", err)
	}

	var env []string
	for _, meta := range metas {
		if meta.EnvVar == "" {
			continue
		}

		val, err := inj.manager.Get(ctx, meta.Name, meta.Scope, requester)
		if err != nil {
			// Access denied or expired — skip silently, do not fail the entire injection.
			continue
		}
		env = append(env, meta.EnvVar+"="+string(val))
		SecureWipe(val)
	}

	return env, nil
}

// sanitizeEnv filters dangerous env vars from the inherited environment.
func sanitizeEnv(base []string, filteredPrefixes []string) []string {
	if base == nil {
		// Import current process environment, minus filtered prefixes.
		// We don't call os.Environ() here to avoid importing it;
		// the caller should pass os.Environ() as base if needed.
		return nil
	}

	defaultFiltered := []string{envPrefix} // BLACKCAT_SECRET_*
	allFiltered := append(defaultFiltered, filteredPrefixes...)

	var clean []string
	for _, entry := range base {
		key := strings.SplitN(entry, "=", 2)[0]
		filtered := false
		for _, prefix := range allFiltered {
			if strings.HasPrefix(key, prefix) {
				filtered = true
				break
			}
		}
		if !filtered {
			clean = append(clean, entry)
		}
	}
	return clean
}

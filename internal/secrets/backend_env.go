package secrets

import (
	"context"
	"os"
	"strings"
)

// EnvBackend reads secrets from environment variables. This is the last-resort
// backend, used when neither OS keychain nor encrypted file is available.
//
// It is read-only: Set and Delete are no-ops because writing secrets to
// environment variables would be visible to child processes and potentially
// logged by the OS.
//
// Key mapping: a secret named "openai_api_key" with scope "global" maps to
// the environment variable "BLACKCAT_SECRET_OPENAI_API_KEY".
const envPrefix = "BLACKCAT_SECRET_"

// EnvBackend provides read-only access to secrets stored as environment variables.
type EnvBackend struct{}

// NewEnvBackend creates an environment variable backend.
func NewEnvBackend() *EnvBackend {
	return &EnvBackend{}
}

func (e *EnvBackend) Name() string {
	return "env"
}

// Available always returns true — environment variables are always accessible.
func (e *EnvBackend) Available() bool {
	return true
}

func (e *EnvBackend) Get(_ context.Context, key string) ([]byte, error) {
	envKey := envVarName(key)
	val, ok := os.LookupEnv(envKey)
	if !ok {
		return nil, ErrNotFound
	}
	return []byte(val), nil
}

// Set is a no-op. Environment variables should not be written by the agent.
func (e *EnvBackend) Set(_ context.Context, _ string, _ []byte) error {
	return nil
}

// Delete is a no-op. Environment variables should not be modified by the agent.
func (e *EnvBackend) Delete(_ context.Context, _ string) error {
	return nil
}

// List returns all environment variable keys that have the BLACKCAT_SECRET_ prefix.
func (e *EnvBackend) List(_ context.Context) ([]string, error) {
	var keys []string
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 && strings.HasPrefix(parts[0], envPrefix) {
			name := strings.TrimPrefix(parts[0], envPrefix)
			keys = append(keys, strings.ToLower(name))
		}
	}
	return keys, nil
}

// envVarName converts a secret name to its environment variable form.
func envVarName(key string) string {
	normalized := strings.ToUpper(strings.ReplaceAll(key, "-", "_"))
	return envPrefix + normalized
}

package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadFromFile reads a YAML config file, expands env vars, and merges with defaults.
func LoadFromFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	expanded := expandEnvVars(string(data))

	cfg := Default()
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}

// MergeProjectConfig loads a project-level config and merges it over the global config.
// If the project config file doesn't exist, the global config is returned unchanged.
func MergeProjectConfig(global Config, projectPath string) (Config, error) {
	data, err := os.ReadFile(projectPath)
	if err != nil {
		if os.IsNotExist(err) {
			return global, nil
		}
		return Config{}, fmt.Errorf("read project config: %w", err)
	}

	expanded := expandEnvVars(string(data))

	merged := global
	if err := yaml.Unmarshal([]byte(expanded), &merged); err != nil {
		return Config{}, fmt.Errorf("parse project config: %w", err)
	}

	return merged, nil
}

// expandEnvVars replaces $VAR_NAME and ${VAR_NAME} with environment variable values.
func expandEnvVars(s string) string {
	return os.Expand(s, func(key string) string {
		// Strip surrounding quotes if present from YAML values like "$VAR"
		key = strings.Trim(key, "\"'")
		return os.Getenv(key)
	})
}

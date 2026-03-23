package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ServerConfig describes an MCP server's connection parameters.
type ServerConfig struct {
	Name      string            `json:"name"`
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	Transport string            `json:"transport"`
	URL       string            `json:"url,omitempty"`
}

// LoadServerConfigs reads a JSON file containing an array of ServerConfig entries.
func LoadServerConfigs(configPath string) ([]ServerConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read server config: %w", err)
	}

	var configs []ServerConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, fmt.Errorf("parse server config %q: %w", configPath, err)
	}

	return configs, nil
}

// DiscoverServers scans a directory for JSON config files and loads all
// ServerConfig entries found. Non-JSON files are ignored. Returns an error
// only if the directory itself cannot be read.
func DiscoverServers(configDir string) ([]ServerConfig, error) {
	entries, err := os.ReadDir(configDir)
	if err != nil {
		return nil, fmt.Errorf("read config directory %q: %w", configDir, err)
	}

	var allConfigs []ServerConfig
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			continue
		}

		path := filepath.Join(configDir, entry.Name())
		configs, err := LoadServerConfigs(path)
		if err != nil {
			// Skip malformed config files rather than aborting discovery
			continue
		}
		allConfigs = append(allConfigs, configs...)
	}

	return allConfigs, nil
}

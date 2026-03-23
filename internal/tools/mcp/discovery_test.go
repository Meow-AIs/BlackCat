package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadServerConfigsValid(t *testing.T) {
	dir := t.TempDir()
	configs := []ServerConfig{
		{
			Name:      "test-server",
			Command:   "node",
			Args:      []string{"server.js"},
			Transport: "stdio",
		},
		{
			Name:      "sse-server",
			Transport: "sse",
			URL:       "http://localhost:3000",
		},
	}
	data, _ := json.MarshalIndent(configs, "", "  ")
	fpath := filepath.Join(dir, "servers.json")
	os.WriteFile(fpath, data, 0644)

	loaded, err := LoadServerConfigs(fpath)
	if err != nil {
		t.Fatalf("LoadServerConfigs failed: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 configs, got %d", len(loaded))
	}
	if loaded[0].Name != "test-server" {
		t.Errorf("expected name 'test-server', got %q", loaded[0].Name)
	}
	if loaded[0].Transport != "stdio" {
		t.Errorf("expected transport 'stdio', got %q", loaded[0].Transport)
	}
	if loaded[1].URL != "http://localhost:3000" {
		t.Errorf("expected URL 'http://localhost:3000', got %q", loaded[1].URL)
	}
}

func TestLoadServerConfigsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "bad.json")
	os.WriteFile(fpath, []byte("not json"), 0644)

	_, err := LoadServerConfigs(fpath)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadServerConfigsNonexistentFile(t *testing.T) {
	_, err := LoadServerConfigs("/nonexistent/servers.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadServerConfigsWithEnv(t *testing.T) {
	dir := t.TempDir()
	configs := []ServerConfig{
		{
			Name:      "env-server",
			Command:   "python",
			Args:      []string{"-m", "mcp_server"},
			Transport: "stdio",
			Env: map[string]string{
				"MODEL": "gpt-4",
			},
		},
	}
	data, _ := json.MarshalIndent(configs, "", "  ")
	fpath := filepath.Join(dir, "servers.json")
	os.WriteFile(fpath, data, 0644)

	loaded, err := LoadServerConfigs(fpath)
	if err != nil {
		t.Fatalf("LoadServerConfigs failed: %v", err)
	}
	if loaded[0].Env["MODEL"] != "gpt-4" {
		t.Errorf("expected env MODEL='gpt-4', got %q", loaded[0].Env["MODEL"])
	}
}

func TestDiscoverServers(t *testing.T) {
	dir := t.TempDir()

	// Create two config files
	config1 := []ServerConfig{
		{Name: "server-a", Command: "cmd-a", Transport: "stdio"},
	}
	config2 := []ServerConfig{
		{Name: "server-b", Command: "cmd-b", Transport: "stdio"},
		{Name: "server-c", Transport: "sse", URL: "http://localhost:4000"},
	}

	data1, _ := json.MarshalIndent(config1, "", "  ")
	data2, _ := json.MarshalIndent(config2, "", "  ")

	os.WriteFile(filepath.Join(dir, "servers1.json"), data1, 0644)
	os.WriteFile(filepath.Join(dir, "servers2.json"), data2, 0644)
	// Non-JSON file should be ignored
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("ignore me"), 0644)

	configs, err := DiscoverServers(dir)
	if err != nil {
		t.Fatalf("DiscoverServers failed: %v", err)
	}
	if len(configs) != 3 {
		t.Fatalf("expected 3 configs, got %d", len(configs))
	}

	names := map[string]bool{}
	for _, c := range configs {
		names[c.Name] = true
	}
	for _, expected := range []string{"server-a", "server-b", "server-c"} {
		if !names[expected] {
			t.Errorf("expected config %q not found", expected)
		}
	}
}

func TestDiscoverServersEmptyDir(t *testing.T) {
	dir := t.TempDir()
	configs, err := DiscoverServers(dir)
	if err != nil {
		t.Fatalf("DiscoverServers failed: %v", err)
	}
	if len(configs) != 0 {
		t.Errorf("expected 0 configs, got %d", len(configs))
	}
}

func TestDiscoverServersNonexistentDir(t *testing.T) {
	_, err := DiscoverServers("/nonexistent/config/dir")
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

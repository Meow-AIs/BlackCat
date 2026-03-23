package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCustomCommands(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "commands.json")

	content := `[
		{"name": "scan", "description": "Run security scan", "skill_id": "devsecops/secret-scanner", "action": "run_skill"},
		{"name": "diagram", "description": "Generate diagram", "action": "inject_prompt", "prompt": "Generate a C4 diagram"}
	]`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cmds, err := LoadCustomCommands(path)
	if err != nil {
		t.Fatalf("LoadCustomCommands error: %v", err)
	}
	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(cmds))
	}
	if cmds[0].Name != "scan" {
		t.Errorf("expected name 'scan', got %q", cmds[0].Name)
	}
	if cmds[0].SkillID != "devsecops/secret-scanner" {
		t.Errorf("expected skill_id 'devsecops/secret-scanner', got %q", cmds[0].SkillID)
	}
	if cmds[1].Action != "inject_prompt" {
		t.Errorf("expected action 'inject_prompt', got %q", cmds[1].Action)
	}
	if cmds[1].Prompt != "Generate a C4 diagram" {
		t.Errorf("expected prompt text, got %q", cmds[1].Prompt)
	}
}

func TestLoadCustomCommandsFileNotFound(t *testing.T) {
	cmds, err := LoadCustomCommands("/nonexistent/path/commands.json")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if len(cmds) != 0 {
		t.Errorf("expected empty slice for missing file, got %d", len(cmds))
	}
}

func TestLoadCustomCommandsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "commands.json")
	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadCustomCommands(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestSaveCustomCommands(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "commands.json")

	cmds := []CustomCommand{
		{Name: "scan", Description: "Run scan", Action: "run_skill", SkillID: "scanner"},
		{Name: "review", Description: "Code review", Action: "inject_prompt", Prompt: "Review code"},
	}

	if err := SaveCustomCommands(path, cmds); err != nil {
		t.Fatalf("SaveCustomCommands error: %v", err)
	}

	loaded, err := LoadCustomCommands(path)
	if err != nil {
		t.Fatalf("LoadCustomCommands error: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 commands after round-trip, got %d", len(loaded))
	}
	if loaded[0].Name != "scan" {
		t.Errorf("expected 'scan', got %q", loaded[0].Name)
	}
	if loaded[1].Prompt != "Review code" {
		t.Errorf("expected prompt 'Review code', got %q", loaded[1].Prompt)
	}
}

func TestToCommandDef(t *testing.T) {
	tests := []struct {
		cmd      CustomCommand
		wantCat  string
		checkLLM bool
	}{
		{
			cmd:      CustomCommand{Name: "scan", Description: "Run scan", Action: "run_skill", SkillID: "scanner"},
			wantCat:  "custom",
			checkLLM: false,
		},
		{
			cmd:      CustomCommand{Name: "review", Description: "Code review", Action: "inject_prompt", Prompt: "Review all code"},
			wantCat:  "custom",
			checkLLM: true,
		},
		{
			cmd:      CustomCommand{Name: "deploy", Description: "Deploy app", Action: "run_plugin", PluginID: "deployer"},
			wantCat:  "custom",
			checkLLM: false,
		},
	}

	for _, tt := range tests {
		def := tt.cmd.ToCommandDef()
		if def.Name != tt.cmd.Name {
			t.Errorf("expected name %q, got %q", tt.cmd.Name, def.Name)
		}
		if def.Description != tt.cmd.Description {
			t.Errorf("expected description %q, got %q", tt.cmd.Description, def.Description)
		}
		if def.Category != tt.wantCat {
			t.Errorf("expected category %q, got %q", tt.wantCat, def.Category)
		}
		if def.Handler == nil {
			t.Error("Handler should not be nil")
			continue
		}
		result := def.Handler(nil)
		if result.Output == "" {
			t.Error("Handler should return non-empty output")
		}
		if tt.checkLLM && !result.InjectToLLM {
			t.Error("inject_prompt action should set InjectToLLM=true")
		}
	}
}

func TestCustomCommandValidation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "commands.json")

	// Empty name should still load (validation at registration time)
	content := `[{"name": "", "description": "No name", "action": "run_skill"}]`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cmds, err := LoadCustomCommands(path)
	if err != nil {
		t.Fatalf("LoadCustomCommands error: %v", err)
	}
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
}

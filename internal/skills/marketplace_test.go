package skills

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestMarketplace_Install(t *testing.T) {
	server, _ := setupMockRegistry(t)
	mgr := newMemSkillManager()
	client := NewRegistryClient(
		[]RegistrySource{{Name: "test", BaseURL: server.URL, Type: "http"}},
		t.TempDir(),
	)
	installDir := t.TempDir()
	mp := NewMarketplace(client, mgr, installDir)

	err := mp.Install(context.Background(), "devsecops/secret-scanner", "1.2.0")
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	if !mp.IsInstalled("devsecops/secret-scanner") {
		t.Error("expected skill to be installed")
	}

	// Verify skill was stored in manager
	skills, _ := mgr.List(context.Background())
	found := false
	for _, s := range skills {
		if s.Name == "devsecops/secret-scanner" {
			found = true
			if s.Source != "marketplace" {
				t.Errorf("expected source marketplace, got %s", s.Source)
			}
			if s.Version != "1.2.0" {
				t.Errorf("expected version 1.2.0, got %s", s.Version)
			}
		}
	}
	if !found {
		t.Error("installed skill not found in manager")
	}
}

func TestMarketplace_Install_NotFound(t *testing.T) {
	server, _ := setupMockRegistry(t)
	mgr := newMemSkillManager()
	client := NewRegistryClient(
		[]RegistrySource{{Name: "test", BaseURL: server.URL, Type: "http"}},
		t.TempDir(),
	)
	mp := NewMarketplace(client, mgr, t.TempDir())

	err := mp.Install(context.Background(), "nonexistent/skill", "1.0.0")
	if err == nil {
		t.Error("expected error for nonexistent skill")
	}
}

func TestMarketplace_Uninstall(t *testing.T) {
	server, _ := setupMockRegistry(t)
	mgr := newMemSkillManager()
	client := NewRegistryClient(
		[]RegistrySource{{Name: "test", BaseURL: server.URL, Type: "http"}},
		t.TempDir(),
	)
	mp := NewMarketplace(client, mgr, t.TempDir())

	mp.Install(context.Background(), "devsecops/secret-scanner", "1.2.0")
	err := mp.Uninstall(context.Background(), "devsecops/secret-scanner")
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}
	if mp.IsInstalled("devsecops/secret-scanner") {
		t.Error("expected skill to be uninstalled")
	}
}

func TestMarketplace_Uninstall_NotInstalled(t *testing.T) {
	server, _ := setupMockRegistry(t)
	mgr := newMemSkillManager()
	client := NewRegistryClient(
		[]RegistrySource{{Name: "test", BaseURL: server.URL, Type: "http"}},
		t.TempDir(),
	)
	mp := NewMarketplace(client, mgr, t.TempDir())

	err := mp.Uninstall(context.Background(), "nonexistent/skill")
	if err == nil {
		t.Error("expected error for uninstalling non-installed skill")
	}
}

func TestMarketplace_ListInstalled(t *testing.T) {
	server, _ := setupMockRegistry(t)
	mgr := newMemSkillManager()
	client := NewRegistryClient(
		[]RegistrySource{{Name: "test", BaseURL: server.URL, Type: "http"}},
		t.TempDir(),
	)
	mp := NewMarketplace(client, mgr, t.TempDir())

	mp.Install(context.Background(), "devsecops/secret-scanner", "1.2.0")
	mp.Install(context.Background(), "devops/docker-deploy", "2.0.0")

	installed := mp.ListInstalled()
	if len(installed) != 2 {
		t.Errorf("expected 2 installed, got %d", len(installed))
	}
}

func TestMarketplace_Search(t *testing.T) {
	server, _ := setupMockRegistry(t)
	mgr := newMemSkillManager()
	client := NewRegistryClient(
		[]RegistrySource{{Name: "test", BaseURL: server.URL, Type: "http"}},
		t.TempDir(),
	)
	mp := NewMarketplace(client, mgr, t.TempDir())

	entries, err := mp.Search(context.Background(), "docker")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected at least one search result")
	}
}

func TestMarketplace_CheckUpdates(t *testing.T) {
	server, entries := setupMockRegistry(t)
	mgr := newMemSkillManager()
	client := NewRegistryClient(
		[]RegistrySource{{Name: "test", BaseURL: server.URL, Type: "http"}},
		t.TempDir(),
	)
	mp := NewMarketplace(client, mgr, t.TempDir())

	// Install an older version by faking the installed record
	mp.Install(context.Background(), "devsecops/secret-scanner", "1.2.0")
	// Manually set installed version to something older
	if inst, ok := mp.installed["devsecops/secret-scanner"]; ok {
		inst.Package.Metadata.Version = "1.0.0"
		mp.installed["devsecops/secret-scanner"] = inst
	}

	updates, err := mp.CheckUpdates(context.Background())
	if err != nil {
		t.Fatalf("CheckUpdates failed: %v", err)
	}

	if len(updates) == 0 {
		t.Error("expected at least one available update")
	}
	if len(updates) > 0 {
		if updates[0].CurrentVersion != "1.0.0" {
			t.Errorf("expected current 1.0.0, got %s", updates[0].CurrentVersion)
		}
		if updates[0].LatestVersion != entries[0].Package.Metadata.Version {
			t.Errorf("expected latest %s, got %s", entries[0].Package.Metadata.Version, updates[0].LatestVersion)
		}
	}
}

func TestMarketplace_SaveLoadState(t *testing.T) {
	server, _ := setupMockRegistry(t)
	mgr := newMemSkillManager()
	installDir := t.TempDir()
	client := NewRegistryClient(
		[]RegistrySource{{Name: "test", BaseURL: server.URL, Type: "http"}},
		t.TempDir(),
	)
	mp := NewMarketplace(client, mgr, installDir)

	mp.Install(context.Background(), "devsecops/secret-scanner", "1.2.0")

	if err := mp.SaveState(); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Verify state file exists
	statePath := filepath.Join(installDir, "marketplace_state.json")
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Error("expected state file to exist")
	}

	// Load into a new marketplace
	mp2 := NewMarketplace(client, mgr, installDir)
	if err := mp2.LoadState(); err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	if !mp2.IsInstalled("devsecops/secret-scanner") {
		t.Error("expected skill to be installed after loading state")
	}
}

func TestMarketplace_Update(t *testing.T) {
	server, _ := setupMockRegistry(t)
	mgr := newMemSkillManager()
	client := NewRegistryClient(
		[]RegistrySource{{Name: "test", BaseURL: server.URL, Type: "http"}},
		t.TempDir(),
	)
	mp := NewMarketplace(client, mgr, t.TempDir())

	mp.Install(context.Background(), "devsecops/secret-scanner", "1.2.0")

	// Set to older version
	if inst, ok := mp.installed["devsecops/secret-scanner"]; ok {
		inst.Package.Metadata.Version = "1.0.0"
		mp.installed["devsecops/secret-scanner"] = inst
	}

	updated, err := mp.Update(context.Background(), "devsecops/secret-scanner")
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if !updated {
		t.Error("expected update to return true")
	}

	inst := mp.installed["devsecops/secret-scanner"]
	if inst.Package.Metadata.Version != "1.2.0" {
		t.Errorf("expected updated to 1.2.0, got %s", inst.Package.Metadata.Version)
	}
}

func TestMarketplace_UpdateAll(t *testing.T) {
	server, _ := setupMockRegistry(t)
	mgr := newMemSkillManager()
	client := NewRegistryClient(
		[]RegistrySource{{Name: "test", BaseURL: server.URL, Type: "http"}},
		t.TempDir(),
	)
	mp := NewMarketplace(client, mgr, t.TempDir())

	mp.Install(context.Background(), "devsecops/secret-scanner", "1.2.0")
	mp.Install(context.Background(), "devops/docker-deploy", "2.0.0")

	// Set both to older versions
	for name, inst := range mp.installed {
		inst.Package.Metadata.Version = "0.1.0"
		mp.installed[name] = inst
	}

	updated, err := mp.UpdateAll(context.Background())
	if err != nil {
		t.Fatalf("UpdateAll failed: %v", err)
	}
	if len(updated) != 2 {
		t.Errorf("expected 2 updates, got %d", len(updated))
	}
}

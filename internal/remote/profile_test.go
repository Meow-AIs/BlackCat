package remote

import (
	"testing"
)

// --- ProfileStore Tests ---

func TestNewMemoryProfileStore(t *testing.T) {
	store := NewMemoryProfileStore()
	if store == nil {
		t.Fatal("expected non-nil store")
	}
	profiles := store.List()
	if len(profiles) != 0 {
		t.Fatalf("expected empty list, got %d", len(profiles))
	}
}

func TestMemoryProfileStore_SaveAndGet(t *testing.T) {
	store := NewMemoryProfileStore()
	profile := ConnectionProfile{
		Name:        "dev-server",
		Type:        ConnSSH,
		Host:        "10.0.0.1",
		Port:        22,
		User:        "deploy",
		Environment: "dev",
		Permissions: DefaultPermsForEnv("dev"),
	}

	err := store.Save(profile)
	if err != nil {
		t.Fatalf("unexpected error saving profile: %v", err)
	}

	got, err := store.Get("dev-server")
	if err != nil {
		t.Fatalf("unexpected error getting profile: %v", err)
	}
	if got.Name != "dev-server" {
		t.Errorf("expected name dev-server, got %s", got.Name)
	}
	if got.Host != "10.0.0.1" {
		t.Errorf("expected host 10.0.0.1, got %s", got.Host)
	}
	if got.Type != ConnSSH {
		t.Errorf("expected type ssh, got %s", got.Type)
	}
}

func TestMemoryProfileStore_GetNotFound(t *testing.T) {
	store := NewMemoryProfileStore()
	_, err := store.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent profile")
	}
}

func TestMemoryProfileStore_List(t *testing.T) {
	store := NewMemoryProfileStore()
	profiles := []ConnectionProfile{
		{Name: "a", Type: ConnSSH, Host: "h1", Environment: "dev", Permissions: DefaultPermsForEnv("dev")},
		{Name: "b", Type: ConnKubectl, KubeContext: "ctx", Environment: "staging", Permissions: DefaultPermsForEnv("staging")},
	}
	for _, p := range profiles {
		if err := store.Save(p); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	list := store.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(list))
	}
}

func TestMemoryProfileStore_SaveOverwrite(t *testing.T) {
	store := NewMemoryProfileStore()
	p1 := ConnectionProfile{Name: "srv", Type: ConnSSH, Host: "old", Environment: "dev", Permissions: DefaultPermsForEnv("dev")}
	p2 := ConnectionProfile{Name: "srv", Type: ConnSSH, Host: "new", Environment: "dev", Permissions: DefaultPermsForEnv("dev")}

	_ = store.Save(p1)
	_ = store.Save(p2)

	got, _ := store.Get("srv")
	if got.Host != "new" {
		t.Errorf("expected overwritten host 'new', got %s", got.Host)
	}
	if len(store.List()) != 1 {
		t.Errorf("expected 1 profile after overwrite, got %d", len(store.List()))
	}
}

func TestMemoryProfileStore_Delete(t *testing.T) {
	store := NewMemoryProfileStore()
	p := ConnectionProfile{Name: "del-me", Type: ConnSSH, Host: "h", Environment: "dev", Permissions: DefaultPermsForEnv("dev")}
	_ = store.Save(p)

	err := store.Delete("del-me")
	if err != nil {
		t.Fatalf("unexpected error deleting: %v", err)
	}
	_, err = store.Get("del-me")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestMemoryProfileStore_DeleteNotFound(t *testing.T) {
	store := NewMemoryProfileStore()
	err := store.Delete("ghost")
	if err == nil {
		t.Fatal("expected error deleting nonexistent profile")
	}
}

// --- Immutability Tests ---

func TestMemoryProfileStore_GetReturnsImmutableCopy(t *testing.T) {
	store := NewMemoryProfileStore()
	p := ConnectionProfile{Name: "imm", Type: ConnSSH, Host: "original", Environment: "dev", Permissions: DefaultPermsForEnv("dev")}
	_ = store.Save(p)

	got, _ := store.Get("imm")
	got.Host = "mutated"

	got2, _ := store.Get("imm")
	if got2.Host != "original" {
		t.Error("store was mutated through returned value")
	}
}

func TestMemoryProfileStore_ListReturnsImmutableCopies(t *testing.T) {
	store := NewMemoryProfileStore()
	p := ConnectionProfile{Name: "imm2", Type: ConnSSH, Host: "original", Environment: "dev", Permissions: DefaultPermsForEnv("dev")}
	_ = store.Save(p)

	list := store.List()
	list[0].Host = "mutated"

	got, _ := store.Get("imm2")
	if got.Host != "original" {
		t.Error("store was mutated through list value")
	}
}

// --- ValidateProfile Tests ---

func TestValidateProfile_Valid_SSH(t *testing.T) {
	p := ConnectionProfile{
		Name:        "valid-ssh",
		Type:        ConnSSH,
		Host:        "example.com",
		Environment: "dev",
		Permissions: DefaultPermsForEnv("dev"),
	}
	if err := ValidateProfile(p); err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}
}

func TestValidateProfile_Valid_Kubectl(t *testing.T) {
	p := ConnectionProfile{
		Name:        "valid-k8s",
		Type:        ConnKubectl,
		KubeContext: "my-cluster",
		Environment: "staging",
		Permissions: DefaultPermsForEnv("staging"),
	}
	if err := ValidateProfile(p); err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}
}

func TestValidateProfile_MissingName(t *testing.T) {
	p := ConnectionProfile{Type: ConnSSH, Host: "h", Environment: "dev"}
	if err := ValidateProfile(p); err == nil {
		t.Error("expected error for missing name")
	}
}

func TestValidateProfile_MissingType(t *testing.T) {
	p := ConnectionProfile{Name: "n", Host: "h", Environment: "dev"}
	if err := ValidateProfile(p); err == nil {
		t.Error("expected error for missing type")
	}
}

func TestValidateProfile_InvalidType(t *testing.T) {
	p := ConnectionProfile{Name: "n", Type: "ftp", Host: "h", Environment: "dev"}
	if err := ValidateProfile(p); err == nil {
		t.Error("expected error for invalid type")
	}
}

func TestValidateProfile_SSH_MissingHost(t *testing.T) {
	p := ConnectionProfile{Name: "n", Type: ConnSSH, Environment: "dev"}
	if err := ValidateProfile(p); err == nil {
		t.Error("expected error for SSH without host")
	}
}

func TestValidateProfile_Kubectl_MissingContext(t *testing.T) {
	p := ConnectionProfile{Name: "n", Type: ConnKubectl, Environment: "staging"}
	if err := ValidateProfile(p); err == nil {
		t.Error("expected error for kubectl without context")
	}
}

func TestValidateProfile_MissingEnvironment(t *testing.T) {
	p := ConnectionProfile{Name: "n", Type: ConnSSH, Host: "h"}
	if err := ValidateProfile(p); err == nil {
		t.Error("expected error for missing environment")
	}
}

func TestValidateProfile_InvalidEnvironment(t *testing.T) {
	p := ConnectionProfile{Name: "n", Type: ConnSSH, Host: "h", Environment: "yolo"}
	if err := ValidateProfile(p); err == nil {
		t.Error("expected error for invalid environment")
	}
}

// --- DefaultPermsForEnv Tests ---

func TestDefaultPermsForEnv_Dev(t *testing.T) {
	perms := DefaultPermsForEnv("dev")
	if !perms.AllowExec {
		t.Error("dev: expected AllowExec=true")
	}
	if !perms.AllowTransfer {
		t.Error("dev: expected AllowTransfer=true")
	}
	if perms.RequireApproval {
		t.Error("dev: expected RequireApproval=false")
	}
}

func TestDefaultPermsForEnv_Staging(t *testing.T) {
	perms := DefaultPermsForEnv("staging")
	if !perms.AllowExec {
		t.Error("staging: expected AllowExec=true")
	}
	if !perms.AllowTransfer {
		t.Error("staging: expected AllowTransfer=true")
	}
	if !perms.RequireApproval {
		t.Error("staging: expected RequireApproval=true")
	}
}

func TestDefaultPermsForEnv_Prod(t *testing.T) {
	perms := DefaultPermsForEnv("prod")
	if !perms.AllowExec {
		t.Error("prod: expected AllowExec=true")
	}
	if perms.AllowTransfer {
		t.Error("prod: expected AllowTransfer=false")
	}
	if !perms.RequireApproval {
		t.Error("prod: expected RequireApproval=true")
	}
	if len(perms.DeniedCmds) == 0 {
		t.Error("prod: expected non-empty DeniedCmds")
	}
	expected := []string{"rm -rf", "DROP", "TRUNCATE", "shutdown"}
	for _, cmd := range expected {
		found := false
		for _, d := range perms.DeniedCmds {
			if d == cmd {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("prod: expected DeniedCmds to contain %q", cmd)
		}
	}
}

func TestDefaultPermsForEnv_Unknown(t *testing.T) {
	perms := DefaultPermsForEnv("unknown")
	// Unknown environments should default to most restrictive (prod-like)
	if !perms.RequireApproval {
		t.Error("unknown env: expected RequireApproval=true")
	}
	if perms.AllowTransfer {
		t.Error("unknown env: expected AllowTransfer=false")
	}
}

func TestDefaultPermsForEnv_MaxConcurrent(t *testing.T) {
	for _, env := range []string{"dev", "staging", "prod"} {
		perms := DefaultPermsForEnv(env)
		if perms.MaxConcurrent != 5 {
			t.Errorf("%s: expected MaxConcurrent=5, got %d", env, perms.MaxConcurrent)
		}
	}
}

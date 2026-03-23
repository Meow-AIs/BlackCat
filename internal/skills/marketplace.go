package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// InstalledSkill tracks a marketplace skill that has been installed locally.
type InstalledSkill struct {
	Package     SkillPackage `json:"package"`
	InstalledAt int64        `json:"installed_at"`
	UpdatedAt   int64        `json:"updated_at"`
	Source      string       `json:"source"`
	Checksum    string       `json:"checksum"`
	AutoUpdate  bool         `json:"auto_update"`
}

// UpdateAvailable describes a pending update for an installed skill.
type UpdateAvailable struct {
	Name           string `json:"name"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
}

// Marketplace orchestrates install, update, and removal of registry skills.
type Marketplace struct {
	registry   *RegistryClient
	manager    Manager
	installed  map[string]InstalledSkill
	installDir string
}

// NewMarketplace creates a marketplace backed by the given registry and manager.
func NewMarketplace(registry *RegistryClient, manager Manager, installDir string) *Marketplace {
	return &Marketplace{
		registry:   registry,
		manager:    manager,
		installed:  make(map[string]InstalledSkill),
		installDir: installDir,
	}
}

// Search queries the registry for skills matching the query string.
func (m *Marketplace) Search(ctx context.Context, query string) ([]RegistryEntry, error) {
	result, err := m.registry.Search(ctx, query, nil, 1)
	if err != nil {
		return nil, fmt.Errorf("marketplace search: %w", err)
	}
	return result.Entries, nil
}

// Install fetches a skill from the registry and stores it locally.
func (m *Marketplace) Install(ctx context.Context, name string, version string) error {
	pkg, err := m.registry.GetPackage(ctx, name, version)
	if err != nil {
		return fmt.Errorf("fetch package: %w", err)
	}

	checksum, err := m.registry.FetchChecksum(ctx, name, version)
	if err != nil {
		// Non-fatal: checksum verification is best-effort for now
		checksum = ""
	}

	skill := pkg.ToSkill()
	skill.Checksum = checksum

	if err := m.manager.Store(ctx, skill); err != nil {
		return fmt.Errorf("store skill: %w", err)
	}

	now := time.Now().Unix()
	m.installed[name] = InstalledSkill{
		Package:     *pkg,
		InstalledAt: now,
		UpdatedAt:   now,
		Source:      "registry",
		Checksum:    checksum,
	}

	return nil
}

// Uninstall removes an installed skill.
func (m *Marketplace) Uninstall(ctx context.Context, name string) error {
	inst, ok := m.installed[name]
	if !ok {
		return fmt.Errorf("skill %q is not installed", name)
	}

	skillID := inst.Package.Metadata.Name + "@" + inst.Package.Metadata.Version
	if err := m.manager.Delete(ctx, skillID); err != nil {
		return fmt.Errorf("delete skill: %w", err)
	}

	delete(m.installed, name)
	return nil
}

// Update checks for a newer version and installs it if available.
// Returns true if an update was applied.
func (m *Marketplace) Update(ctx context.Context, name string) (bool, error) {
	inst, ok := m.installed[name]
	if !ok {
		return false, fmt.Errorf("skill %q is not installed", name)
	}

	versions, err := m.registry.ListVersions(ctx, name)
	if err != nil {
		return false, fmt.Errorf("list versions: %w", err)
	}
	if len(versions) == 0 {
		return false, nil
	}

	latest := latestVersion(versions)
	if CompareVersions(latest, inst.Package.Metadata.Version) <= 0 {
		return false, nil
	}

	// Remove old, install new
	oldID := inst.Package.Metadata.Name + "@" + inst.Package.Metadata.Version
	m.manager.Delete(ctx, oldID)

	if err := m.Install(ctx, name, latest); err != nil {
		return false, fmt.Errorf("install update: %w", err)
	}

	return true, nil
}

// UpdateAll updates all installed skills. Returns the names of updated skills.
func (m *Marketplace) UpdateAll(ctx context.Context) ([]string, error) {
	var updated []string
	for name := range m.installed {
		ok, err := m.Update(ctx, name)
		if err != nil {
			continue
		}
		if ok {
			updated = append(updated, name)
		}
	}
	return updated, nil
}

// ListInstalled returns all installed marketplace skills.
func (m *Marketplace) ListInstalled() []InstalledSkill {
	result := make([]InstalledSkill, 0, len(m.installed))
	for _, inst := range m.installed {
		result = append(result, inst)
	}
	return result
}

// IsInstalled returns true if the named skill is installed.
func (m *Marketplace) IsInstalled(name string) bool {
	_, ok := m.installed[name]
	return ok
}

// CheckUpdates checks all installed skills for available updates.
func (m *Marketplace) CheckUpdates(ctx context.Context) ([]UpdateAvailable, error) {
	var updates []UpdateAvailable

	for name, inst := range m.installed {
		versions, err := m.registry.ListVersions(ctx, name)
		if err != nil {
			continue
		}
		if len(versions) == 0 {
			continue
		}

		latest := latestVersion(versions)
		if CompareVersions(latest, inst.Package.Metadata.Version) > 0 {
			updates = append(updates, UpdateAvailable{
				Name:           name,
				CurrentVersion: inst.Package.Metadata.Version,
				LatestVersion:  latest,
			})
		}
	}

	return updates, nil
}

const stateFileName = "marketplace_state.json"

// SaveState persists the installed skills map to disk.
func (m *Marketplace) SaveState() error {
	data, err := json.MarshalIndent(m.installed, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	statePath := filepath.Join(m.installDir, stateFileName)
	if err := os.WriteFile(statePath, data, 0644); err != nil {
		return fmt.Errorf("write state: %w", err)
	}
	return nil
}

// LoadState restores the installed skills map from disk.
func (m *Marketplace) LoadState() error {
	statePath := filepath.Join(m.installDir, stateFileName)
	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no state to load
		}
		return fmt.Errorf("read state: %w", err)
	}

	installed := make(map[string]InstalledSkill)
	if err := json.Unmarshal(data, &installed); err != nil {
		return fmt.Errorf("parse state: %w", err)
	}
	m.installed = installed
	return nil
}

// latestVersion finds the highest semver from a list of version strings.
func latestVersion(versions []string) string {
	if len(versions) == 0 {
		return ""
	}
	best := versions[0]
	for _, v := range versions[1:] {
		if CompareVersions(v, best) > 0 {
			best = v
		}
	}
	return best
}

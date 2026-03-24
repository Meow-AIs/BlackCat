// Package updater provides self-update functionality for BlackCat.
// It checks GitHub Releases for newer versions and downloads the
// appropriate binary for the current OS/architecture.
package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// Version represents a semantic version.
type Version struct {
	Major int
	Minor int
	Patch int
}

// ParseVersion parses a semver string like "0.2.0" or "v0.2.0".
func ParseVersion(s string) (Version, error) {
	s = strings.TrimPrefix(s, "v")
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("invalid version: %q", s)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, fmt.Errorf("invalid major: %w", err)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return Version{}, fmt.Errorf("invalid minor: %w", err)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return Version{}, fmt.Errorf("invalid patch: %w", err)
	}
	return Version{Major: major, Minor: minor, Patch: patch}, nil
}

// String returns the version as "major.minor.patch".
func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// IsNewer reports whether v is newer than other.
func (v Version) IsNewer(other Version) bool {
	if v.Major != other.Major {
		return v.Major > other.Major
	}
	if v.Minor != other.Minor {
		return v.Minor > other.Minor
	}
	return v.Patch > other.Patch
}

// GitHubRelease is the response from the GitHub releases API.
type GitHubRelease struct {
	TagName string         `json:"tag_name"`
	Assets  []ReleaseAsset `json:"assets"`
}

// ReleaseAsset is a downloadable file attached to a release.
type ReleaseAsset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
	Size        int64  `json:"size"`
}

// UpdateInfo contains the result of an update check.
type UpdateInfo struct {
	Available      bool
	CurrentVersion string
	LatestVersion  string
	DownloadURL    string
	AssetName      string
}

// Updater checks for and applies updates from GitHub Releases.
type Updater struct {
	repo           string // "Meow-AIs/BlackCat"
	currentVersion string
	httpClient     *http.Client
	apiURL         string // override for testing
}

// NewUpdater creates an updater for the given repo and current version.
func NewUpdater(repo, currentVersion string) *Updater {
	return &Updater{
		repo:           repo,
		currentVersion: currentVersion,
		httpClient:     &http.Client{},
	}
}

// CheckForUpdate queries GitHub for the latest release and compares versions.
func (u *Updater) CheckForUpdate() (UpdateInfo, error) {
	url := u.apiURL
	if url == "" {
		url = fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", u.repo)
	}

	resp, err := u.httpClient.Get(url)
	if err != nil {
		return UpdateInfo{}, fmt.Errorf("fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return UpdateInfo{}, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return UpdateInfo{}, fmt.Errorf("parse release: %w", err)
	}

	current, err := ParseVersion(u.currentVersion)
	if err != nil {
		return UpdateInfo{}, fmt.Errorf("parse current version: %w", err)
	}

	latest, err := ParseVersion(release.TagName)
	if err != nil {
		return UpdateInfo{}, fmt.Errorf("parse latest version: %w", err)
	}

	info := UpdateInfo{
		CurrentVersion: current.String(),
		LatestVersion:  latest.String(),
	}

	if latest.IsNewer(current) {
		info.Available = true
		if asset, found := FindAsset(release.Assets, runtime.GOOS, runtime.GOARCH); found {
			info.DownloadURL = asset.DownloadURL
			info.AssetName = asset.Name
		}
	}

	return info, nil
}

// DownloadUpdate downloads the binary from the given URL.
func (u *Updater) DownloadUpdate(url string) ([]byte, error) {
	resp, err := u.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download returned %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read download: %w", err)
	}
	return data, nil
}

// ReplaceBinary replaces the current executable with new data.
func ReplaceBinary(newBinary []byte) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find executable: %w", err)
	}

	// Write to temp file first
	tmpPath := exePath + ".new"
	if err := os.WriteFile(tmpPath, newBinary, 0o755); err != nil {
		return fmt.Errorf("write new binary: %w", err)
	}

	// Backup old binary
	bakPath := exePath + ".bak"
	if err := os.Rename(exePath, bakPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("backup old binary: %w", err)
	}

	// Move new binary into place
	if err := os.Rename(tmpPath, exePath); err != nil {
		// Restore backup
		os.Rename(bakPath, exePath)
		return fmt.Errorf("install new binary: %w", err)
	}

	// Clean up backup
	os.Remove(bakPath)
	return nil
}

// FindAsset finds the release asset matching the given OS and architecture.
func FindAsset(assets []ReleaseAsset, goos, goarch string) (ReleaseAsset, bool) {
	target := fmt.Sprintf("blackcat-%s-%s", goos, goarch)
	for _, a := range assets {
		if strings.HasPrefix(a.Name, target) {
			return a, true
		}
	}
	return ReleaseAsset{}, false
}

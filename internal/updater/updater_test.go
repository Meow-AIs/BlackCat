package updater

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input               string
		major, minor, patch int
	}{
		{"0.1.0", 0, 1, 0},
		{"0.2.0", 0, 2, 0},
		{"1.0.0", 1, 0, 0},
		{"v0.1.4", 0, 1, 4},
		{"v2.10.3", 2, 10, 3},
	}
	for _, tt := range tests {
		v, err := ParseVersion(tt.input)
		if err != nil {
			t.Errorf("ParseVersion(%q) error: %v", tt.input, err)
			continue
		}
		if v.Major != tt.major || v.Minor != tt.minor || v.Patch != tt.patch {
			t.Errorf("ParseVersion(%q) = %d.%d.%d, want %d.%d.%d",
				tt.input, v.Major, v.Minor, v.Patch, tt.major, tt.minor, tt.patch)
		}
	}
}

func TestParseVersion_Invalid(t *testing.T) {
	invalids := []string{"", "abc", "1.2", "1.2.3.4"}
	for _, s := range invalids {
		if _, err := ParseVersion(s); err == nil {
			t.Errorf("ParseVersion(%q) should error", s)
		}
	}
}

func TestVersion_IsNewer(t *testing.T) {
	tests := []struct {
		current, latest string
		newer           bool
	}{
		{"0.1.0", "0.2.0", true},
		{"0.2.0", "0.2.0", false},
		{"0.2.0", "0.1.0", false},
		{"1.0.0", "0.9.9", false},
		{"0.1.4", "0.2.0", true},
		{"0.9.9", "1.0.0", true},
	}
	for _, tt := range tests {
		c, _ := ParseVersion(tt.current)
		l, _ := ParseVersion(tt.latest)
		if got := l.IsNewer(c); got != tt.newer {
			t.Errorf("%s.IsNewer(%s) = %v, want %v", tt.latest, tt.current, got, tt.newer)
		}
	}
}

func TestCheckForUpdate(t *testing.T) {
	release := GitHubRelease{
		TagName: "v0.3.0",
		Assets: []ReleaseAsset{
			{Name: "blackcat-linux-amd64.tar.gz", DownloadURL: "https://example.com/linux.tar.gz"},
			{Name: "blackcat-windows-amd64.zip", DownloadURL: "https://example.com/windows.zip"},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(release)
	}))
	defer srv.Close()

	u := NewUpdater("Meow-AIs/BlackCat", "0.2.0")
	u.apiURL = srv.URL

	info, err := u.CheckForUpdate()
	if err != nil {
		t.Fatalf("CheckForUpdate error: %v", err)
	}
	if !info.Available {
		t.Fatal("expected update available")
	}
	if info.LatestVersion != "0.3.0" {
		t.Errorf("LatestVersion = %q, want %q", info.LatestVersion, "0.3.0")
	}
	if info.CurrentVersion != "0.2.0" {
		t.Errorf("CurrentVersion = %q, want %q", info.CurrentVersion, "0.2.0")
	}
}

func TestCheckForUpdate_AlreadyLatest(t *testing.T) {
	release := GitHubRelease{TagName: "v0.2.0"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(release)
	}))
	defer srv.Close()

	u := NewUpdater("Meow-AIs/BlackCat", "0.2.0")
	u.apiURL = srv.URL

	info, err := u.CheckForUpdate()
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if info.Available {
		t.Fatal("should not be available when already on latest")
	}
}

func TestCheckForUpdate_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	u := NewUpdater("Meow-AIs/BlackCat", "0.2.0")
	u.apiURL = srv.URL

	_, err := u.CheckForUpdate()
	if err == nil {
		t.Fatal("expected error on 500")
	}
}

func TestFindAsset(t *testing.T) {
	assets := []ReleaseAsset{
		{Name: "blackcat-linux-amd64.tar.gz", DownloadURL: "https://example.com/linux-amd64.tar.gz"},
		{Name: "blackcat-linux-arm64.tar.gz", DownloadURL: "https://example.com/linux-arm64.tar.gz"},
		{Name: "blackcat-windows-amd64.zip", DownloadURL: "https://example.com/windows-amd64.zip"},
		{Name: "blackcat-darwin-arm64.tar.gz", DownloadURL: "https://example.com/darwin-arm64.tar.gz"},
		{Name: "checksums.txt", DownloadURL: "https://example.com/checksums.txt"},
	}

	tests := []struct {
		goos, goarch string
		wantName     string
		wantFound    bool
	}{
		{"linux", "amd64", "blackcat-linux-amd64.tar.gz", true},
		{"linux", "arm64", "blackcat-linux-arm64.tar.gz", true},
		{"windows", "amd64", "blackcat-windows-amd64.zip", true},
		{"darwin", "arm64", "blackcat-darwin-arm64.tar.gz", true},
		{"freebsd", "amd64", "", false},
	}

	for _, tt := range tests {
		asset, found := FindAsset(assets, tt.goos, tt.goarch)
		if found != tt.wantFound {
			t.Errorf("FindAsset(%s/%s) found=%v, want %v", tt.goos, tt.goarch, found, tt.wantFound)
		}
		if found && asset.Name != tt.wantName {
			t.Errorf("FindAsset(%s/%s) = %q, want %q", tt.goos, tt.goarch, asset.Name, tt.wantName)
		}
	}
}

func TestVersion_String(t *testing.T) {
	v := Version{Major: 1, Minor: 2, Patch: 3}
	if v.String() != "1.2.3" {
		t.Errorf("String() = %q, want %q", v.String(), "1.2.3")
	}
}

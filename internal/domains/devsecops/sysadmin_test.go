package devsecops

import (
	"strings"
	"testing"
)

func TestLoadLinuxHardeningRules(t *testing.T) {
	rules := LoadLinuxHardeningRules()
	if len(rules) < 20 {
		t.Errorf("expected at least 20 Linux hardening rules, got %d", len(rules))
	}

	for _, r := range rules {
		if r.ID == "" {
			t.Error("rule has empty ID")
		}
		if r.Category == "" {
			t.Error("rule has empty Category")
		}
		if r.OS != "linux" && r.OS != "both" {
			t.Errorf("Linux rule %s has OS %q, expected 'linux' or 'both'", r.ID, r.OS)
		}
		if r.Description == "" {
			t.Errorf("rule %s has empty Description", r.ID)
		}
		if r.Check == "" {
			t.Errorf("rule %s has empty Check", r.ID)
		}
		if r.Remediation == "" {
			t.Errorf("rule %s has empty Remediation", r.ID)
		}
		if r.Severity == "" {
			t.Errorf("rule %s has empty Severity", r.ID)
		}
	}
}

func TestLoadWindowsHardeningRules(t *testing.T) {
	rules := LoadWindowsHardeningRules()
	if len(rules) < 15 {
		t.Errorf("expected at least 15 Windows hardening rules, got %d", len(rules))
	}

	for _, r := range rules {
		if r.ID == "" {
			t.Error("rule has empty ID")
		}
		if r.OS != "windows" && r.OS != "both" {
			t.Errorf("Windows rule %s has OS %q, expected 'windows' or 'both'", r.ID, r.OS)
		}
		if r.Description == "" {
			t.Errorf("rule %s has empty Description", r.ID)
		}
	}
}

func TestLoadLinuxHardeningRules_Categories(t *testing.T) {
	rules := LoadLinuxHardeningRules()
	categories := map[string]bool{}
	for _, r := range rules {
		categories[r.Category] = true
	}

	required := []string{"ssh", "network", "filesystem", "access", "audit", "services", "kernel"}
	for _, cat := range required {
		if !categories[cat] {
			t.Errorf("missing required category %q in Linux rules", cat)
		}
	}
}

func TestGenerateSSHConfig(t *testing.T) {
	cfg := SSHConfig{
		PermitRootLogin:     "no",
		PasswordAuth:        "no",
		MaxAuthTries:        4,
		Protocol:            2,
		AllowUsers:          []string{"admin", "deploy"},
		ClientAliveInterval: 300,
		X11Forwarding:       "no",
	}

	result := GenerateSSHConfig(cfg)

	checks := []string{
		"PermitRootLogin no",
		"PasswordAuthentication no",
		"MaxAuthTries 4",
		"Protocol 2",
		"AllowUsers admin deploy",
		"ClientAliveInterval 300",
		"X11Forwarding no",
	}

	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("SSH config missing %q", check)
		}
	}
}

func TestGenerateSSHConfig_EmptyAllowUsers(t *testing.T) {
	cfg := SSHConfig{
		PermitRootLogin: "no",
		PasswordAuth:    "no",
		MaxAuthTries:    3,
		Protocol:        2,
		X11Forwarding:   "no",
	}

	result := GenerateSSHConfig(cfg)
	if strings.Contains(result, "AllowUsers") {
		t.Error("SSH config should not include AllowUsers when empty")
	}
}

func TestGenerateFirewallRules(t *testing.T) {
	result := GenerateFirewallRules("default-deny")
	if !strings.Contains(result, "iptables") {
		t.Error("firewall rules should contain iptables commands")
	}
	if !strings.Contains(result, "DROP") {
		t.Error("default-deny policy should contain DROP")
	}
	if !strings.Contains(result, "ESTABLISHED") {
		t.Error("should allow established connections")
	}
}

func TestGenerateFirewallRules_UnknownPolicy(t *testing.T) {
	result := GenerateFirewallRules("unknown-policy")
	// Should still return something reasonable
	if result == "" {
		t.Error("should return rules even for unknown policy")
	}
}

func TestGenerateAuditdRules(t *testing.T) {
	result := GenerateAuditdRules()
	if result == "" {
		t.Fatal("auditd rules should not be empty")
	}

	checks := []string{
		"/etc/passwd",
		"/etc/shadow",
		"auditctl",
	}
	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("auditd rules missing reference to %q", check)
		}
	}
}

func TestHardeningRuleCISRef(t *testing.T) {
	rules := LoadLinuxHardeningRules()
	hasCISRef := false
	for _, r := range rules {
		if r.CISRef != "" {
			hasCISRef = true
			break
		}
	}
	if !hasCISRef {
		t.Error("expected at least one rule with CIS reference")
	}
}

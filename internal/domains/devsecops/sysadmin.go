package devsecops

import (
	"fmt"
	"strings"
)

// HardeningRule represents a system hardening check based on CIS Benchmarks.
type HardeningRule struct {
	ID          string
	Category    string // "ssh", "network", "filesystem", "access", "audit", "services", "kernel"
	OS          string // "linux", "windows", "both"
	Description string
	Check       string // command to check
	Remediation string // command to fix
	CISRef      string // CIS Benchmark reference
	Severity    string
}

// SSHConfig defines SSH server configuration parameters.
type SSHConfig struct {
	PermitRootLogin     string // "no"
	PasswordAuth        string // "no"
	MaxAuthTries        int    // 4
	Protocol            int    // 2
	AllowUsers          []string
	ClientAliveInterval int // 300
	X11Forwarding       string // "no"
}

// LoadLinuxHardeningRules returns CIS-based hardening rules for Linux systems.
func LoadLinuxHardeningRules() []HardeningRule {
	return []HardeningRule{
		// SSH rules
		{ID: "LIN-SSH-001", Category: "ssh", OS: "linux", Severity: "critical",
			Description: "Disable SSH root login",
			Check:       "grep -i '^PermitRootLogin' /etc/ssh/sshd_config",
			Remediation: "sed -i 's/^PermitRootLogin.*/PermitRootLogin no/' /etc/ssh/sshd_config",
			CISRef:      "5.2.10"},
		{ID: "LIN-SSH-002", Category: "ssh", OS: "linux", Severity: "high",
			Description: "Disable SSH password authentication",
			Check:       "grep -i '^PasswordAuthentication' /etc/ssh/sshd_config",
			Remediation: "sed -i 's/^PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config",
			CISRef:      "5.2.12"},
		{ID: "LIN-SSH-003", Category: "ssh", OS: "linux", Severity: "medium",
			Description: "Set SSH MaxAuthTries to 4 or less",
			Check:       "grep -i '^MaxAuthTries' /etc/ssh/sshd_config",
			Remediation: "sed -i 's/^MaxAuthTries.*/MaxAuthTries 4/' /etc/ssh/sshd_config",
			CISRef:      "5.2.7"},

		// Network rules
		{ID: "LIN-NET-001", Category: "network", OS: "linux", Severity: "high",
			Description: "Disable IP forwarding",
			Check:       "sysctl net.ipv4.ip_forward",
			Remediation: "sysctl -w net.ipv4.ip_forward=0",
			CISRef:      "3.1.1"},
		{ID: "LIN-NET-002", Category: "network", OS: "linux", Severity: "high",
			Description: "Disable ICMP redirects",
			Check:       "sysctl net.ipv4.conf.all.accept_redirects",
			Remediation: "sysctl -w net.ipv4.conf.all.accept_redirects=0",
			CISRef:      "3.2.2"},
		{ID: "LIN-NET-003", Category: "network", OS: "linux", Severity: "medium",
			Description: "Enable TCP SYN cookies",
			Check:       "sysctl net.ipv4.tcp_syncookies",
			Remediation: "sysctl -w net.ipv4.tcp_syncookies=1",
			CISRef:      "3.2.8"},

		// Filesystem rules
		{ID: "LIN-FS-001", Category: "filesystem", OS: "linux", Severity: "high",
			Description: "Ensure /tmp is a separate partition with noexec",
			Check:       "mount | grep ' /tmp '",
			Remediation: "mount -o remount,noexec,nosuid,nodev /tmp",
			CISRef:      "1.1.2"},
		{ID: "LIN-FS-002", Category: "filesystem", OS: "linux", Severity: "medium",
			Description: "Ensure sticky bit on world-writable directories",
			Check:       "find / -type d -perm -0002 ! -perm -1000 2>/dev/null",
			Remediation: "chmod +t <directory>",
			CISRef:      "1.1.21"},
		{ID: "LIN-FS-003", Category: "filesystem", OS: "linux", Severity: "high",
			Description: "Disable automounting",
			Check:       "systemctl is-enabled autofs 2>/dev/null",
			Remediation: "systemctl disable autofs",
			CISRef:      "1.1.22"},

		// Access control rules
		{ID: "LIN-ACC-001", Category: "access", OS: "linux", Severity: "critical",
			Description: "Ensure password expiration is 365 days or less",
			Check:       "grep PASS_MAX_DAYS /etc/login.defs",
			Remediation: "sed -i 's/^PASS_MAX_DAYS.*/PASS_MAX_DAYS 365/' /etc/login.defs",
			CISRef:      "5.4.1.1"},
		{ID: "LIN-ACC-002", Category: "access", OS: "linux", Severity: "high",
			Description: "Ensure minimum password length is 14 characters",
			Check:       "grep pam_pwquality /etc/pam.d/common-password",
			Remediation: "echo 'minlen = 14' >> /etc/security/pwquality.conf",
			CISRef:      "5.4.1"},
		{ID: "LIN-ACC-003", Category: "access", OS: "linux", Severity: "critical",
			Description: "Ensure no accounts have empty passwords",
			Check:       "awk -F: '($2 == \"\") {print $1}' /etc/shadow",
			Remediation: "passwd -l <username>",
			CISRef:      "6.2.1"},

		// Audit rules
		{ID: "LIN-AUD-001", Category: "audit", OS: "linux", Severity: "high",
			Description: "Ensure auditd is installed and enabled",
			Check:       "systemctl is-enabled auditd",
			Remediation: "apt-get install auditd && systemctl enable auditd",
			CISRef:      "4.1.1.1"},
		{ID: "LIN-AUD-002", Category: "audit", OS: "linux", Severity: "high",
			Description: "Ensure audit log storage size is configured",
			Check:       "grep max_log_file /etc/audit/auditd.conf",
			Remediation: "echo 'max_log_file = 128' >> /etc/audit/auditd.conf",
			CISRef:      "4.1.1.2"},
		{ID: "LIN-AUD-003", Category: "audit", OS: "linux", Severity: "medium",
			Description: "Ensure login and logout events are collected",
			Check:       "auditctl -l | grep logins",
			Remediation: "echo '-w /var/log/lastlog -p wa -k logins' >> /etc/audit/rules.d/audit.rules",
			CISRef:      "4.1.8"},

		// Services rules
		{ID: "LIN-SVC-001", Category: "services", OS: "linux", Severity: "high",
			Description: "Ensure unnecessary services are disabled",
			Check:       "systemctl list-unit-files --type=service --state=enabled",
			Remediation: "systemctl disable <service>",
			CISRef:      "2.2"},
		{ID: "LIN-SVC-002", Category: "services", OS: "linux", Severity: "critical",
			Description: "Ensure NFS is not enabled unless required",
			Check:       "systemctl is-enabled nfs-server 2>/dev/null",
			Remediation: "systemctl disable nfs-server",
			CISRef:      "2.2.7"},
		{ID: "LIN-SVC-003", Category: "services", OS: "linux", Severity: "medium",
			Description: "Ensure time synchronization is configured (NTP/chrony)",
			Check:       "systemctl is-enabled chronyd || systemctl is-enabled ntpd",
			Remediation: "apt-get install chrony && systemctl enable chronyd",
			CISRef:      "2.2.1.1"},

		// Kernel rules
		{ID: "LIN-KRN-001", Category: "kernel", OS: "linux", Severity: "high",
			Description: "Ensure ASLR is enabled",
			Check:       "sysctl kernel.randomize_va_space",
			Remediation: "sysctl -w kernel.randomize_va_space=2",
			CISRef:      "1.5.2"},
		{ID: "LIN-KRN-002", Category: "kernel", OS: "linux", Severity: "medium",
			Description: "Ensure core dumps are restricted",
			Check:       "grep 'hard core' /etc/security/limits.conf",
			Remediation: "echo '* hard core 0' >> /etc/security/limits.conf",
			CISRef:      "1.5.1"},
	}
}

// LoadWindowsHardeningRules returns CIS-based hardening rules for Windows systems.
func LoadWindowsHardeningRules() []HardeningRule {
	return []HardeningRule{
		{ID: "WIN-ACC-001", Category: "access", OS: "windows", Severity: "critical",
			Description: "Ensure account lockout threshold is 5 or fewer attempts",
			Check:       "net accounts | findstr /i lockout",
			Remediation: "net accounts /lockoutthreshold:5",
			CISRef:      "1.2.1"},
		{ID: "WIN-ACC-002", Category: "access", OS: "windows", Severity: "high",
			Description: "Ensure minimum password length is 14 characters",
			Check:       "net accounts | findstr /i 'Minimum password length'",
			Remediation: "net accounts /minpwlen:14",
			CISRef:      "1.1.4"},
		{ID: "WIN-ACC-003", Category: "access", OS: "windows", Severity: "high",
			Description: "Ensure password complexity is enabled",
			Check:       "secedit /export /cfg C:\\temp\\secpol.cfg && findstr PasswordComplexity C:\\temp\\secpol.cfg",
			Remediation: "Set PasswordComplexity = 1 in Local Security Policy",
			CISRef:      "1.1.5"},
		{ID: "WIN-ACC-004", Category: "access", OS: "windows", Severity: "medium",
			Description: "Ensure password history is 24 or more",
			Check:       "net accounts | findstr /i 'password history'",
			Remediation: "net accounts /uniquepw:24",
			CISRef:      "1.1.1"},

		{ID: "WIN-AUD-001", Category: "audit", OS: "windows", Severity: "high",
			Description: "Ensure audit policy for logon events is enabled",
			Check:       "auditpol /get /category:\"Logon/Logoff\"",
			Remediation: "auditpol /set /subcategory:\"Logon\" /success:enable /failure:enable",
			CISRef:      "17.5.1"},
		{ID: "WIN-AUD-002", Category: "audit", OS: "windows", Severity: "high",
			Description: "Ensure audit policy for account management is enabled",
			Check:       "auditpol /get /category:\"Account Management\"",
			Remediation: "auditpol /set /subcategory:\"User Account Management\" /success:enable /failure:enable",
			CISRef:      "17.2.1"},
		{ID: "WIN-AUD-003", Category: "audit", OS: "windows", Severity: "medium",
			Description: "Ensure Windows Event Log service is running",
			Check:       "sc query EventLog",
			Remediation: "sc config EventLog start= auto && net start EventLog",
			CISRef:      "18.9.25"},

		{ID: "WIN-NET-001", Category: "network", OS: "windows", Severity: "high",
			Description: "Ensure Windows Firewall is enabled for all profiles",
			Check:       "netsh advfirewall show allprofiles | findstr State",
			Remediation: "netsh advfirewall set allprofiles state on",
			CISRef:      "9.1.1"},
		{ID: "WIN-NET-002", Category: "network", OS: "windows", Severity: "medium",
			Description: "Ensure SMBv1 is disabled",
			Check:       "Get-WindowsFeature FS-SMB1 | Select InstallState",
			Remediation: "Disable-WindowsOptionalFeature -Online -FeatureName SMB1Protocol",
			CISRef:      "18.3.3"},
		{ID: "WIN-NET-003", Category: "network", OS: "windows", Severity: "high",
			Description: "Ensure Remote Desktop requires NLA",
			Check:       "reg query \"HKLM\\SYSTEM\\CurrentControlSet\\Control\\Terminal Server\\WinStations\\RDP-Tcp\" /v UserAuthentication",
			Remediation: "reg add \"HKLM\\SYSTEM\\CurrentControlSet\\Control\\Terminal Server\\WinStations\\RDP-Tcp\" /v UserAuthentication /t REG_DWORD /d 1 /f",
			CISRef:      "18.9.58.3.9.2"},

		{ID: "WIN-SVC-001", Category: "services", OS: "windows", Severity: "high",
			Description: "Ensure unnecessary services are disabled (Telnet, FTP)",
			Check:       "sc query TlntSvr && sc query ftpsvc",
			Remediation: "sc config TlntSvr start= disabled && sc config ftpsvc start= disabled",
			CISRef:      "2.3"},
		{ID: "WIN-SVC-002", Category: "services", OS: "windows", Severity: "medium",
			Description: "Ensure Windows Update service is running",
			Check:       "sc query wuauserv",
			Remediation: "sc config wuauserv start= auto && net start wuauserv",
			CISRef:      "18.9.100"},

		{ID: "WIN-FS-001", Category: "filesystem", OS: "windows", Severity: "high",
			Description: "Ensure NTFS permissions on system directories are restricted",
			Check:       "icacls C:\\Windows\\System32",
			Remediation: "Reset permissions with icacls to restrict access",
			CISRef:      "5.1"},
		{ID: "WIN-FS-002", Category: "filesystem", OS: "windows", Severity: "critical",
			Description: "Ensure BitLocker drive encryption is enabled",
			Check:       "manage-bde -status",
			Remediation: "manage-bde -on C:",
			CISRef:      "18.8.12.1"},

		{ID: "WIN-KRN-001", Category: "kernel", OS: "windows", Severity: "high",
			Description: "Ensure DEP (Data Execution Prevention) is enabled",
			Check:       "wmic OS get DataExecutionPrevention_SupportPolicy",
			Remediation: "bcdedit /set nx AlwaysOn",
			CISRef:      "18.3.1"},
	}
}

// GenerateSSHConfig generates an sshd_config file from the given configuration.
func GenerateSSHConfig(cfg SSHConfig) string {
	var b strings.Builder
	b.WriteString("# SSH Server Configuration - Security Hardened\n")
	b.WriteString("# Generated by BlackCat DevSecOps\n\n")

	b.WriteString(fmt.Sprintf("Protocol %d\n", cfg.Protocol))
	b.WriteString(fmt.Sprintf("PermitRootLogin %s\n", cfg.PermitRootLogin))
	b.WriteString(fmt.Sprintf("PasswordAuthentication %s\n", cfg.PasswordAuth))
	b.WriteString(fmt.Sprintf("MaxAuthTries %d\n", cfg.MaxAuthTries))
	b.WriteString(fmt.Sprintf("ClientAliveInterval %d\n", cfg.ClientAliveInterval))
	b.WriteString(fmt.Sprintf("X11Forwarding %s\n", cfg.X11Forwarding))

	if len(cfg.AllowUsers) > 0 {
		b.WriteString(fmt.Sprintf("AllowUsers %s\n", strings.Join(cfg.AllowUsers, " ")))
	}

	b.WriteString("\n# Additional hardening\n")
	b.WriteString("PermitEmptyPasswords no\n")
	b.WriteString("UsePAM yes\n")
	b.WriteString("PrintMotd no\n")
	b.WriteString("AcceptEnv LANG LC_*\n")
	b.WriteString("Subsystem sftp /usr/lib/openssh/sftp-server\n")

	return b.String()
}

// GenerateFirewallRules generates iptables rules for the given policy.
func GenerateFirewallRules(policy string) string {
	var b strings.Builder
	b.WriteString("#!/bin/bash\n")
	b.WriteString("# Firewall Rules - Generated by BlackCat DevSecOps\n\n")

	b.WriteString("# Flush existing rules\n")
	b.WriteString("iptables -F\n")
	b.WriteString("iptables -X\n\n")

	switch policy {
	case "default-deny":
		b.WriteString("# Default policy: DROP all traffic\n")
		b.WriteString("iptables -P INPUT DROP\n")
		b.WriteString("iptables -P FORWARD DROP\n")
		b.WriteString("iptables -P OUTPUT ACCEPT\n\n")

		b.WriteString("# Allow loopback\n")
		b.WriteString("iptables -A INPUT -i lo -j ACCEPT\n")
		b.WriteString("iptables -A OUTPUT -o lo -j ACCEPT\n\n")

		b.WriteString("# Allow ESTABLISHED and RELATED connections\n")
		b.WriteString("iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT\n\n")

		b.WriteString("# Allow SSH (port 22) from trusted networks only\n")
		b.WriteString("iptables -A INPUT -p tcp --dport 22 -m state --state NEW -j ACCEPT\n\n")

		b.WriteString("# Allow HTTP/HTTPS\n")
		b.WriteString("iptables -A INPUT -p tcp --dport 80 -m state --state NEW -j ACCEPT\n")
		b.WriteString("iptables -A INPUT -p tcp --dport 443 -m state --state NEW -j ACCEPT\n\n")

		b.WriteString("# Allow ICMP (ping)\n")
		b.WriteString("iptables -A INPUT -p icmp --icmp-type echo-request -j ACCEPT\n\n")

		b.WriteString("# Log dropped packets\n")
		b.WriteString("iptables -A INPUT -j LOG --log-prefix \"DROPPED: \" --log-level 4\n")
	default:
		b.WriteString("# Default policy: DROP (unknown policy requested, applying safe defaults)\n")
		b.WriteString("iptables -P INPUT DROP\n")
		b.WriteString("iptables -P FORWARD DROP\n")
		b.WriteString("iptables -P OUTPUT ACCEPT\n\n")
		b.WriteString("# Allow ESTABLISHED and RELATED connections\n")
		b.WriteString("iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT\n\n")
		b.WriteString("# Allow loopback\n")
		b.WriteString("iptables -A INPUT -i lo -j ACCEPT\n")
	}

	return b.String()
}

// GenerateAuditdRules generates key Linux audit rules for security monitoring.
func GenerateAuditdRules() string {
	var b strings.Builder
	b.WriteString("# Audit Rules - Generated by BlackCat DevSecOps\n")
	b.WriteString("# Load with: auditctl -R /etc/audit/rules.d/blackcat.rules\n\n")

	b.WriteString("# Delete all existing rules\n")
	b.WriteString("-D\n\n")

	b.WriteString("# Set buffer size\n")
	b.WriteString("-b 8192\n\n")

	b.WriteString("# Monitor identity files\n")
	b.WriteString("-w /etc/passwd -p wa -k identity\n")
	b.WriteString("-w /etc/shadow -p wa -k identity\n")
	b.WriteString("-w /etc/group -p wa -k identity\n")
	b.WriteString("-w /etc/gshadow -p wa -k identity\n\n")

	b.WriteString("# Monitor SSH configuration\n")
	b.WriteString("-w /etc/ssh/sshd_config -p wa -k sshd_config\n\n")

	b.WriteString("# Monitor sudoers\n")
	b.WriteString("-w /etc/sudoers -p wa -k sudoers\n")
	b.WriteString("-w /etc/sudoers.d/ -p wa -k sudoers\n\n")

	b.WriteString("# Monitor login/logout\n")
	b.WriteString("-w /var/log/lastlog -p wa -k logins\n")
	b.WriteString("-w /var/log/faillog -p wa -k logins\n")
	b.WriteString("-w /var/log/wtmp -p wa -k logins\n")
	b.WriteString("-w /var/log/btmp -p wa -k logins\n\n")

	b.WriteString("# Monitor cron configuration\n")
	b.WriteString("-w /etc/crontab -p wa -k cron\n")
	b.WriteString("-w /etc/cron.d/ -p wa -k cron\n\n")

	b.WriteString("# Monitor kernel modules\n")
	b.WriteString("-w /sbin/insmod -p x -k modules\n")
	b.WriteString("-w /sbin/rmmod -p x -k modules\n")
	b.WriteString("-w /sbin/modprobe -p x -k modules\n\n")

	b.WriteString("# Monitor auditctl itself\n")
	b.WriteString("-w /usr/sbin/auditctl -p x -k audit_tools\n")
	b.WriteString("-w /usr/sbin/auditd -p x -k audit_tools\n\n")

	b.WriteString("# Make the configuration immutable (requires reboot to change)\n")
	b.WriteString("-e 2\n")

	return b.String()
}

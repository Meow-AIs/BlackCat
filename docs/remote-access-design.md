# Remote Server Access System — Design Document

**Project**: BlackCat by MeowAI
**Date**: 2026-03-21
**Scope**: SSH, kubectl, VPN/network, connection management, and security architecture for DevSecOps workflows

---

## 1. Overview and Design Goals

BlackCat needs to execute commands on remote servers as a first-class capability, not as a thin wrapper over `ssh` subprocess calls. The design must satisfy:

- Credentials never enter LLM context (prompts, tool results, memory snapshots)
- Every remote command goes through the same permission gate as local commands, with stricter defaults
- All remote operations are audit-logged to SQLite alongside the agent session
- Connection state is managed independently of the agent session lifecycle
- The system plugs into the existing `tools.Tool` interface and `security.Checker` without breaking them
- Binary size impact is minimal — pure-Go where possible, no heavy SDKs

---

## 2. Package Layout

The remote access system lives under `internal/remote/`. It is a peer of `internal/security/` and `internal/tools/`, not nested inside either.

```
internal/remote/
├── remote.go              # Top-level types: Profile, Target, ActionType, Decision
├── ssh/
│   ├── client.go          # SSHClient — connects, executes, transfers
│   ├── config_parser.go   # Parse ~/.ssh/config for Host stanzas
│   ├── key_store.go       # KeyStore — resolves identity files, agent socket
│   ├── jump.go            # JumpChain — ProxyJump / multi-hop tunnel
│   ├── sftp.go            # SFTP/SCP file transfer
│   └── session_mux.go     # ControlMaster-style session multiplexer
├── kubectl/
│   ├── client.go          # KubectlClient — exec, logs, port-forward
│   ├── kubeconfig.go      # Multi-context kubeconfig loader
│   └── rbac.go            # Pre-flight RBAC capability check
├── network/
│   ├── probe.go           # Reachability probes (TCP, ICMP)
│   └── proxy.go           # SOCKS5 / HTTP CONNECT dial wrapper
├── pool/
│   ├── pool.go            # ConnectionPool — lifecycle, health checks, reuse
│   └── health.go          # Health checker goroutine
├── audit/
│   └── logger.go          # AuditLogger — writes to SQLite remote_audit table
├── sanitize/
│   └── sanitizer.go       # OutputSanitizer — strip secrets/IPs before LLM
├── permission/
│   └── remote_checker.go  # RemoteChecker — wraps security.Checker, adds host/env layer
└── tools/
    ├── ssh_exec.go        # tools.Tool: remote_exec
    ├── ssh_transfer.go    # tools.Tool: remote_transfer
    ├── kubectl_exec.go    # tools.Tool: kube_exec
    ├── kubectl_logs.go    # tools.Tool: kube_logs
    └── kubectl_portfwd.go # tools.Tool: kube_port_forward
```

---

## 3. Core Types (`internal/remote/remote.go`)

```go
package remote

import "time"

// Environment classifies a target by risk level.
// It drives the default permission level for that target.
type Environment string

const (
    EnvDev     Environment = "dev"
    EnvStaging Environment = "staging"
    EnvProd    Environment = "prod"
    EnvCI      Environment = "ci"
)

// Profile is a named remote access configuration stored in
// ~/.blackcat/config.yaml under remote.profiles.
// It is immutable after construction — updates create new values.
type Profile struct {
    Name          string            `yaml:"name"`
    Host          string            `yaml:"host"`           // SSH hostname or kube context
    Port          int               `yaml:"port"`           // 0 = use default (22 / 6443)
    User          string            `yaml:"user"`           // SSH user
    IdentityFile  string            `yaml:"identity_file"`  // path to private key; empty = use agent
    JumpHosts     []string          `yaml:"jump_hosts"`     // ProxyJump chain (ordered)
    Environment   Environment       `yaml:"environment"`
    Tags          []string          `yaml:"tags,omitempty"`
    Timeout       time.Duration     `yaml:"timeout"`        // per-command timeout
    CommandAllow  []string          `yaml:"command_allow"`  // glob allowlist; empty = use env default
    CommandDeny   []string          `yaml:"command_deny"`   // glob denylist; always checked first
    RequireConfirm bool             `yaml:"require_confirm"`// override: always ask, even for safe cmds
    AccessWindow   *AccessWindow    `yaml:"access_window,omitempty"`
    Labels        map[string]string `yaml:"labels,omitempty"`
}

// AccessWindow restricts when remote commands may be executed.
type AccessWindow struct {
    WeekdaysOnly bool   `yaml:"weekdays_only"`
    StartHour    int    `yaml:"start_hour"` // 0-23 UTC
    EndHour      int    `yaml:"end_hour"`   // 0-23 UTC
}

// Target is the resolved, runtime representation of where a command executes.
// It is derived from a Profile plus any session-level overrides.
type Target struct {
    ProfileName string
    Addr        string // host:port
    User        string
    Env         Environment
    Kind        TargetKind
}

// TargetKind distinguishes SSH hosts from Kubernetes contexts.
type TargetKind string

const (
    TargetSSH   TargetKind = "ssh"
    TargetKube  TargetKind = "kube"
)

// RemoteAction is what the agent wants to do on a remote target.
// It extends security.Action with target information.
type RemoteAction struct {
    Target  Target
    Command string
    Args    []string
    // EnvVars intentionally omitted — pass via command string with explicit values
}

// RemoteResult is the sanitized output returned to the agent.
// Raw output is written to audit log; this struct goes to the LLM.
type RemoteResult struct {
    Output      string
    ExitCode    int
    TimedOut    bool
    Sanitized   bool   // true if OutputSanitizer modified the output
    AuditID     string // reference to the audit log entry
}
```

---

## 4. SSH Client (`internal/remote/ssh/client.go`)

### Package recommendation

Use `golang.org/x/crypto/ssh` directly. Do not use a higher-level wrapper library. The reasons:

- The `x/crypto/ssh` package is maintained by the Go team, stable, and well-audited
- Direct use gives precise control over `HostKeyCallback`, auth methods, and keepalive
- Libraries like `github.com/melbahja/goph` add convenience but obscure the auth chain, making credential isolation harder to verify

For SFTP: `github.com/pkg/sftp` (pure Go, builds on top of `x/crypto/ssh` sessions).

### Interface

```go
package ssh

import (
    "context"
    "io"
    "time"

    gossh "golang.org/x/crypto/ssh"
)

// Client manages a single SSH connection to one host.
// All methods are safe for concurrent use from multiple goroutines.
type Client interface {
    // Execute runs a non-interactive command and returns its output.
    // The command string is NOT passed through a shell; it is sent as-is
    // via SSH exec channel. Callers must shell-quote arguments if needed.
    Execute(ctx context.Context, command string) (ExecResult, error)

    // Stream runs a command and writes stdout/stderr to the provided writers
    // in real time. Used for long-running commands (log tailing, builds).
    Stream(ctx context.Context, command string, stdout, stderr io.Writer) (int, error)

    // Upload copies a local file to a remote path via SFTP.
    Upload(ctx context.Context, localPath, remotePath string) error

    // Download copies a remote file to a local path via SFTP.
    Download(ctx context.Context, remotePath, localPath string) error

    // Close terminates the connection and releases all resources.
    Close() error

    // Ping sends an SSH keepalive and returns the round-trip latency.
    Ping(ctx context.Context) (time.Duration, error)

    // Target returns the resolved target this client is connected to.
    Target() Target
}

// ExecResult is the output of a non-streaming Execute call.
type ExecResult struct {
    Stdout   string
    Stderr   string
    ExitCode int
    TimedOut bool
}

// Dialer creates new SSH clients.
// Separating Dialer from Client enables connection pooling and testing.
type Dialer interface {
    Dial(ctx context.Context, profile Profile, keys KeyStore) (Client, error)
}
```

### Key design decisions in the implementation

**Host key verification**

Never use `gossh.InsecureIgnoreHostKey()`. Implement a `KnownHostsCallback` that:
1. Reads `~/.ssh/known_hosts`
2. Falls back to `~/.blackcat/known_hosts` (BlackCat's own store)
3. On first connection to an unknown host: prompt the user via the TUI permission dialog (same flow as `LevelAsk`), then persist the fingerprint if accepted

```go
// KnownHostsCallback constructs a HostKeyCallback that enforces known_hosts.
// unknownHostFn is called when the host is not in any known_hosts file.
// It must return true to accept and persist the key, false to reject.
func KnownHostsCallback(
    knownHostsFiles []string,
    unknownHostFn func(hostname, fingerprint string) bool,
) (gossh.HostKeyCallback, error)
```

**Authentication methods — credential isolation**

The `x/crypto/ssh` auth method is constructed at dial time from the `Profile` and `KeyStore`. The resolved `gossh.Signer` or `gossh.AuthMethod` value is passed directly to the `gossh.ClientConfig`. It never becomes a string, never enters a `map[string]any`, and is therefore structurally incapable of leaking into LLM messages or memory snapshots.

```go
// KeyStore resolves SSH authentication for a profile.
// It never exposes key material as strings.
type KeyStore interface {
    // AuthMethods returns the ordered list of SSH auth methods for a profile.
    // Tries: SSH agent socket, identity file from profile, default identity files.
    AuthMethods(profile Profile) ([]gossh.AuthMethod, error)
}

// defaultKeyStore is the concrete implementation.
type defaultKeyStore struct {
    agentSocketPath string // $SSH_AUTH_SOCK
    passphraseFn    func(keyPath string) ([]byte, error)
}
```

The `passphraseFn` callback prompts via the TUI — it never stores the passphrase in any struct field or passes it to the LLM.

**Jump host (ProxyJump) chain**

```go
// JumpChain builds a net.Conn that tunnels through one or more jump hosts.
// Each hop creates an SSH channel to the next, using the same KeyStore.
// The final net.Conn is handed to the target host's gossh.NewClientConn.
func BuildJumpChain(
    ctx context.Context,
    jumps []Profile,
    target Profile,
    keys KeyStore,
) (net.Conn, error)
```

This mirrors exactly what `ssh -J` does. The jump host connections are tracked in the pool as `TargetKind = TargetSSH` entries and reused.

**SSH config parsing (`config_parser.go`)**

```go
// SSHConfigParser reads ~/.ssh/config and extracts Host stanzas.
// It does NOT execute ProxyCommand strings — those are a shell injection
// risk and are incompatible with the credential isolation model.
// ProxyJump is supported; ProxyCommand is surfaced as a warning.
type SSHConfigParser interface {
    // LookupHost returns the effective parameters for a given hostname,
    // applying wildcard and pattern matching per ssh_config(5).
    LookupHost(hostname string) (HostEntry, error)
}

// HostEntry is the resolved SSH config for one host.
type HostEntry struct {
    Hostname     string
    Port         int
    User         string
    IdentityFile []string
    ProxyJump    []string
    // ProxyCommand is not executed; stored for user warning only
    ProxyCommand string
}
```

Do not use a third-party SSH config parser. Implement a minimal one — the format is simple and a dependency here would expand the attack surface for a security-sensitive code path. The full `ssh_config(5)` grammar is large; implement only the keys BlackCat actually uses.

**Session multiplexing**

Rather than implementing ControlMaster-style Unix socket multiplexing (which is complex and platform-specific), the pool (Section 7) achieves the same goal: one `*gossh.Client` per target is reused across multiple concurrent commands by opening new `gossh.Session` objects on the existing client connection. This is the correct Go idiom.

```go
// openSession opens a new SSH session on an existing client connection.
// Multiple sessions may be open concurrently on the same client.
func openSession(ctx context.Context, c *gossh.Client) (*gossh.Session, error)
```

---

## 5. Kubectl Client (`internal/remote/kubectl/client.go`)

### Package recommendation

Use `k8s.io/client-go` for all Kubernetes API operations. Specifically:

- `k8s.io/client-go/tools/clientcmd` — kubeconfig loading
- `k8s.io/client-go/kubernetes` — typed client for RBAC checks
- `k8s.io/client-go/rest` — raw REST for exec/port-forward
- `k8s.io/client-go/tools/remotecommand` — `kubectl exec` equivalent

Note: `client-go` adds meaningful binary size. Evaluate whether the Kubernetes capability is optional (compiled in via build tag) or always present. Recommended approach: make it a build tag so the default binary stays small.

```
go build -tags kubernetes ./...
```

### Interface

```go
package kubectl

import (
    "context"
    "io"
)

// Client wraps a Kubernetes client for a single context.
type Client interface {
    // Exec runs a command inside a pod container.
    // stdin may be nil for non-interactive commands.
    Exec(ctx context.Context, req ExecRequest) (ExecResult, error)

    // Logs streams logs from a pod container.
    // The stream is closed when ctx is cancelled or the container exits.
    Logs(ctx context.Context, req LogsRequest, out io.Writer) error

    // PortForward forwards localPort -> pod:remotePort.
    // The forward is active until ctx is cancelled.
    PortForward(ctx context.Context, req PortForwardRequest) error

    // CanI checks whether the current service account has a specific verb
    // on a resource, using SubjectAccessReview. Used for pre-flight checks.
    CanI(ctx context.Context, verb, resource, namespace string) (bool, error)

    // Context returns the kubeconfig context name this client is bound to.
    Context() string

    // Close releases the client resources.
    Close() error
}

// ExecRequest specifies a kubectl exec operation.
type ExecRequest struct {
    Namespace   string
    Pod         string
    Container   string   // empty = first container
    Command     []string // NOT a shell string; each element is a separate arg
    Stdin       io.Reader
    TTY         bool
}

// ExecResult is the output of a non-streaming Exec call.
type ExecResult struct {
    Stdout   string
    Stderr   string
    ExitCode int
}

// LogsRequest specifies a kubectl logs operation.
type LogsRequest struct {
    Namespace  string
    Pod        string
    Container  string
    Previous   bool
    Follow     bool
    TailLines  int64
    SinceTime  string // RFC3339
}

// PortForwardRequest specifies a port-forward operation.
type PortForwardRequest struct {
    Namespace   string
    Pod         string
    LocalPort   int
    RemotePort  int
}
```

### Kubeconfig management

```go
// KubeconfigLoader loads and manages multiple kubeconfig files.
type KubeconfigLoader interface {
    // LoadContext returns a rest.Config for the named context.
    LoadContext(contextName string) (*rest.Config, error)

    // ListContexts returns all available context names from all kubeconfig files.
    ListContexts() ([]ContextInfo, error)

    // CurrentContext returns the current context from the active kubeconfig.
    CurrentContext() (string, error)
}

// ContextInfo describes a kubeconfig context.
type ContextInfo struct {
    Name      string
    Cluster   string
    Namespace string
    User      string
}
```

The loader merges `$KUBECONFIG` (colon-separated paths), `~/.kube/config`, and any paths specified in `~/.blackcat/config.yaml` under `remote.kubeconfigs`. It follows the same merge semantics as `kubectl`.

### RBAC pre-flight check

Before executing any `kubectl exec`, call `CanI` and return a structured error if the check fails, rather than letting the API call fail with a cryptic 403. This prevents the agent from wasting an LLM turn on an operation that will always fail.

```go
// PreflightExec validates that the current service account can exec into pods
// in the given namespace before attempting the operation.
func PreflightExec(ctx context.Context, c Client, namespace string) error {
    ok, err := c.CanI(ctx, "create", "pods/exec", namespace)
    if err != nil {
        return fmt.Errorf("rbac preflight check failed: %w", err)
    }
    if !ok {
        return fmt.Errorf("service account lacks pods/exec permission in namespace %q", namespace)
    }
    return nil
}
```

---

## 6. Network Probing and Proxy Support (`internal/remote/network/`)

### Reachability probe

Before opening any SSH or kubectl connection, run a cheap connectivity probe so that connection failures produce actionable errors rather than vague timeouts.

```go
// Probe checks whether a target address is reachable.
type Probe interface {
    // TCPReachable returns nil if a TCP connection to addr can be established
    // within timeout. addr is "host:port".
    TCPReachable(ctx context.Context, addr string, timeout time.Duration) error

    // Latency returns the round-trip time to addr.
    Latency(ctx context.Context, addr string) (time.Duration, error)
}
```

The probe result is surfaced to the agent as a structured error:

```
remote: target "prod-web-01:22" is not reachable (timeout after 5s)
hint: check VPN connection or bastion host availability
```

### Proxy dialer

```go
// ProxyDialer creates net.Conn through a SOCKS5 or HTTP CONNECT proxy.
type ProxyDialer interface {
    // DialContext creates a connection to addr routed through the proxy.
    DialContext(ctx context.Context, network, addr string) (net.Conn, error)
}

// NewSOCKS5Dialer creates a SOCKS5 proxy dialer.
// username and password may be empty for unauthenticated proxies.
func NewSOCKS5Dialer(proxyAddr, username, password string) ProxyDialer

// NewHTTPConnectDialer creates an HTTP CONNECT proxy dialer.
func NewHTTPConnectDialer(proxyAddr string, headers http.Header) ProxyDialer
```

The proxy dialer is injected into the SSH `Dial` function via `gossh.ClientConfig.Dial` field, keeping the proxy concern separate from the SSH client logic.

VPN state detection is intentionally not automated. Attempting to start/stop VPN connections (WireGuard, OpenVPN) from inside an AI agent is high-risk. Instead:

1. The probe reports that the target is unreachable
2. The agent tells the user to connect to VPN manually
3. The user confirms, and the agent retries

This is the correct behavior for a DevSecOps context.

---

## 7. Connection Pool (`internal/remote/pool/pool.go`)

### Design

The pool holds live connections keyed by profile name. It is a shared resource across all agent sessions running under one `blackcat serve` instance.

```go
// Pool manages a set of live remote connections.
// It is safe for concurrent use.
type Pool interface {
    // Acquire returns a live client for the given profile, creating one if needed.
    // If the pool already has a healthy connection for this profile, it is reused.
    Acquire(ctx context.Context, profile Profile) (PooledClient, error)

    // Release returns a client to the pool. The client remains open for reuse.
    Release(client PooledClient)

    // Remove closes and removes the connection for the given profile.
    Remove(profileName string)

    // Stats returns current pool statistics.
    Stats() PoolStats
}

// PooledClient is a Client that has been acquired from the pool.
// Release must be called when the caller is finished.
type PooledClient interface {
    Client
    // ProfileName is the profile this connection belongs to.
    ProfileName() string
}

// PoolStats is a snapshot of pool state.
type PoolStats struct {
    ActiveConnections  int
    IdleConnections    int
    TotalAcquired      int64
    TotalCreated       int64
    TotalHealthFailed  int64
}
```

### Health checking

The health checker goroutine runs in the background and pings each idle connection every 30 seconds. If a ping fails, the connection is removed from the pool and will be re-created on the next `Acquire`.

```go
// HealthChecker runs periodic keepalive probes on pool connections.
type HealthChecker struct {
    pool     Pool
    interval time.Duration
    timeout  time.Duration
}

func (h *HealthChecker) Run(ctx context.Context)
```

### Per-environment connection limits

| Environment | Max connections | Idle timeout |
|-------------|----------------|--------------|
| dev         | 10             | 10 minutes   |
| staging     | 5              | 5 minutes    |
| prod        | 3              | 2 minutes    |
| ci          | 20             | 1 minute     |

Prod has a lower limit to reduce the blast radius of a compromised agent session.

---

## 8. Permission Model (`internal/remote/permission/remote_checker.go`)

The remote permission checker wraps the existing `security.Checker` and adds a layer that reasons about the target host and environment. This preserves the existing local permission model without modifying it.

```go
// RemoteChecker evaluates permission for a RemoteAction.
// It applies rules in this priority order:
//   1. Profile.CommandDeny (always block)
//   2. AccessWindow check (block if outside window)
//   3. Profile.RequireConfirm (force ask regardless of other rules)
//   4. DestructiveCommand detection (force ask for dangerous commands)
//   5. Profile.CommandAllow (allow if matched)
//   6. Environment defaults (see table below)
//   7. Underlying security.Checker for the command string
type RemoteChecker interface {
    // CheckRemote evaluates whether a remote action is permitted.
    CheckRemote(action RemoteAction) security.Decision

    // AddProfileRule adds a rule for a specific profile name.
    AddProfileRule(profileName string, rule ProfileRule)
}

// ProfileRule is a permission rule scoped to a named profile.
type ProfileRule struct {
    Patterns []string         `yaml:"patterns"`
    Level    security.Level   `yaml:"level"`
}
```

### Environment defaults

| Environment | Default level | Destructive commands | Unknown commands |
|-------------|--------------|----------------------|------------------|
| dev         | auto_approve | ask                  | ask              |
| staging     | ask          | ask                  | ask              |
| prod        | ask          | deny                 | deny             |
| ci          | auto_approve | ask                  | ask              |

"Destructive commands" are detected by a pattern set maintained in the checker:

```go
// defaultDestructivePatterns are commands that require explicit confirmation
// on any environment, regardless of CommandAllow rules.
// These are checked BEFORE allow rules.
var defaultDestructivePatterns = []string{
    "rm -rf*",
    "drop table*",
    "drop database*",
    "truncate*",
    "DELETE FROM*",
    "kubectl delete*",
    "terraform destroy*",
    "systemctl stop*",
    "kill -9*",
    "pkill*",
    "shutdown*",
    "reboot*",
    "> /dev/*",       // device overwrite
    "dd if=*",
    "mkfs*",
}
```

### Sub-agent restrictions

Sub-agents inherit a restricted copy of the parent's `RemoteChecker` with all prod profiles removed from the available profile set. A sub-agent cannot access production unless the parent explicitly grants it via an allow rule with the sub-agent's ID in the rule metadata. This is enforced by the pool — `Acquire` checks the caller's agent ID against the profile's allowed callers list.

### Config representation

The `Profile` already contains `CommandAllow`, `CommandDeny`, and `RequireConfirm`. These map directly to the checker's rule evaluation. In `~/.blackcat/config.yaml`:

```yaml
remote:
  profiles:
    - name: prod-web-01
      host: web-01.prod.example.com
      user: deploy
      identity_file: ~/.ssh/prod_deploy_ed25519
      environment: prod
      require_confirm: true
      command_deny:
        - "rm -rf*"
        - "shutdown*"
        - "reboot*"
      access_window:
        weekdays_only: true
        start_hour: 9
        end_hour: 18

    - name: dev-box
      host: dev.internal.example.com
      user: ubuntu
      environment: dev
      command_allow:
        - "make*"
        - "go *"
        - "docker *"
        - "kubectl *"
```

---

## 9. Output Sanitizer (`internal/remote/sanitize/sanitizer.go`)

All command output passes through the sanitizer before being returned to the agent as a `tools.Result`. The raw output is written to the audit log first.

```go
// Sanitizer removes sensitive data from command output before it reaches
// the LLM context, memory snapshots, or channel messages.
type Sanitizer interface {
    // Sanitize returns a cleaned copy of the input and a flag indicating
    // whether any substitutions were made.
    Sanitize(input string) (output string, modified bool)
}

// Rule describes one sanitization substitution.
type Rule struct {
    Name        string         // human-readable label for audit
    Pattern     *regexp.Regexp
    Replacement string
}

// defaultRules are applied to all remote output.
var defaultRules = []Rule{
    {
        Name:        "private_ip",
        Pattern:     regexp.MustCompile(`\b(10\.\d{1,3}\.\d{1,3}\.\d{1,3}|172\.(1[6-9]|2\d|3[01])\.\d{1,3}\.\d{1,3}|192\.168\.\d{1,3}\.\d{1,3})\b`),
        Replacement: "[PRIVATE_IP]",
    },
    {
        Name:        "aws_access_key",
        Pattern:     regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`),
        Replacement: "[AWS_KEY]",
    },
    {
        Name:        "aws_secret_key",
        Pattern:     regexp.MustCompile(`(?i)aws_secret_access_key\s*=\s*[A-Za-z0-9+/]{40}`),
        Replacement: "[AWS_SECRET]",
    },
    {
        Name:        "generic_token",
        Pattern:     regexp.MustCompile(`(?i)(token|password|secret|key)\s*[=:]\s*\S+`),
        Replacement: "[REDACTED]",
    },
    {
        Name:        "bearer_header",
        Pattern:     regexp.MustCompile(`(?i)Authorization:\s*Bearer\s+\S+`),
        Replacement: "Authorization: Bearer [REDACTED]",
    },
    {
        Name:        "ssh_private_key",
        Pattern:     regexp.MustCompile(`-----BEGIN[A-Z ]+ PRIVATE KEY-----[\s\S]*?-----END[A-Z ]+ PRIVATE KEY-----`),
        Replacement: "[SSH_PRIVATE_KEY]",
    },
    {
        Name:        "kubeconfig_token",
        Pattern:     regexp.MustCompile(`(?i)token:\s+[A-Za-z0-9._-]{20,}`),
        Replacement: "token: [REDACTED]",
    },
}
```

The sanitizer is applied at the boundary where `RemoteResult` is converted to `tools.Result`. The conversion function:

```go
// ToToolResult converts a RemoteResult to a tools.Result for the agent.
// It applies sanitization and appends a brief audit reference.
func ToToolResult(r RemoteResult) tools.Result {
    output := r.Output
    note := ""
    if r.Sanitized {
        note = "\n[Note: some sensitive values were redacted from this output]"
    }
    return tools.Result{
        Output:   output + note,
        ExitCode: r.ExitCode,
    }
}
```

---

## 10. Audit Logger (`internal/remote/audit/logger.go`)

All remote operations are written to a `remote_audit` table in the same SQLite database as the memory system (`~/.blackcat/memory.db`).

### Schema

```sql
CREATE TABLE IF NOT EXISTS remote_audit (
    id          TEXT PRIMARY KEY,          -- UUID
    session_id  TEXT NOT NULL,
    profile     TEXT NOT NULL,
    target_addr TEXT NOT NULL,
    command     TEXT NOT NULL,             -- raw, un-sanitized command
    exit_code   INTEGER,
    duration_ms INTEGER,
    sanitized   INTEGER NOT NULL DEFAULT 0,
    decision    TEXT NOT NULL,             -- allow | ask_accepted | ask_rejected | deny
    user_id     TEXT,
    created_at  INTEGER NOT NULL           -- Unix timestamp
);

CREATE INDEX IF NOT EXISTS idx_remote_audit_session ON remote_audit(session_id);
CREATE INDEX IF NOT EXISTS idx_remote_audit_profile ON remote_audit(profile);
CREATE INDEX IF NOT EXISTS idx_remote_audit_created ON remote_audit(created_at);
```

The `command` column stores the raw command before sanitization. The audit log is local to the machine running BlackCat and is not sent to the LLM. Access to the audit log requires local filesystem access.

### Interface

```go
// AuditLogger writes remote operation records to persistent storage.
type AuditLogger interface {
    // Log records a remote operation. It is non-blocking; the write
    // happens asynchronously on a background goroutine with a write queue.
    Log(entry AuditEntry)

    // Query returns audit entries matching the filter.
    Query(ctx context.Context, filter AuditFilter) ([]AuditEntry, error)
}

// AuditEntry is one record of a remote operation.
type AuditEntry struct {
    ID         string
    SessionID  string
    Profile    string
    TargetAddr string
    Command    string        // raw, NOT sanitized
    ExitCode   int
    Duration   time.Duration
    Sanitized  bool
    Decision   string        // permission decision
    UserID     string
    CreatedAt  time.Time
}

// AuditFilter specifies query constraints.
type AuditFilter struct {
    Profile    string
    SessionID  string
    Since      time.Time
    Until      time.Time
    Limit      int
}
```

---

## 11. Tool Implementations (`internal/remote/tools/`)

Each tool implements `tools.Tool` and is registered in the main registry under the `"remote"` category. The agent sees them as regular tools.

### `remote_exec` (`ssh_exec.go`)

```go
func (t *RemoteExecTool) Info() tools.Definition {
    return tools.Definition{
        Name:        "remote_exec",
        Description: "Execute a command on a remote server via SSH. Requires an active profile.",
        Category:    "remote",
        Parameters: []tools.Parameter{
            {Name: "profile",  Type: "string", Description: "Remote profile name from config", Required: true},
            {Name: "command",  Type: "string", Description: "Command to execute on the remote host", Required: true},
            {Name: "timeout",  Type: "string", Description: "Optional timeout, e.g. '30s'. Default: profile timeout.", Required: false},
        },
    }
}
```

The Execute method:
1. Resolves the profile by name from config
2. Calls `RemoteChecker.CheckRemote` — returns error if denied, prompts if ask
3. Runs reachability probe
4. Acquires connection from pool
5. Runs `Client.Execute`
6. Writes raw result to audit log
7. Sanitizes output
8. Returns `tools.Result`

### `remote_transfer` (`ssh_transfer.go`)

```go
Parameters: []tools.Parameter{
    {Name: "profile",      Type: "string", Description: "Remote profile name", Required: true},
    {Name: "direction",    Type: "string", Description: "'upload' or 'download'", Required: true, Enum: []string{"upload", "download"}},
    {Name: "local_path",   Type: "string", Description: "Local file path", Required: true},
    {Name: "remote_path",  Type: "string", Description: "Remote file path", Required: true},
}
```

File transfers are subject to the same permission gate. Uploads to production always require confirmation regardless of `CommandAllow` rules — file transfer is treated as a write operation and mapped to `security.ActionWriteFile` in the underlying checker.

### `kube_exec` (`kubectl_exec.go`)

```go
Parameters: []tools.Parameter{
    {Name: "context",    Type: "string", Description: "Kubeconfig context name", Required: true},
    {Name: "namespace",  Type: "string", Description: "Kubernetes namespace", Required: true},
    {Name: "pod",        Type: "string", Description: "Pod name", Required: true},
    {Name: "container",  Type: "string", Description: "Container name (optional, defaults to first container)", Required: false},
    {Name: "command",    Type: "string", Description: "Command to execute inside the pod", Required: true},
}
```

The Execute method calls `PreflightExec` before attempting the operation, returning a structured error if RBAC denies it.

### `kube_logs` (`kubectl_logs.go`)

```go
Parameters: []tools.Parameter{
    {Name: "context",     Type: "string", Description: "Kubeconfig context name", Required: true},
    {Name: "namespace",   Type: "string", Description: "Kubernetes namespace", Required: true},
    {Name: "pod",         Type: "string", Description: "Pod name", Required: true},
    {Name: "container",   Type: "string", Description: "Container name (optional)", Required: false},
    {Name: "tail_lines",  Type: "integer", Description: "Number of lines from the end (default 100)", Required: false},
    {Name: "since",       Type: "string", Description: "Only return logs since this time (RFC3339)", Required: false},
}
```

Log output is capped at `MaxOutputBytes` (default 512KB for remote tools, configurable) before being returned. The cap prevents LLM context flooding from verbose logs.

### `kube_port_forward` (`kubectl_portfwd.go`)

Port forwarding is a special case — it is a long-running background operation. The tool:
1. Starts the port forward in a goroutine managed by the pool
2. Returns immediately with the allocated local port
3. The forward is tracked by session ID and automatically cleaned up when the session ends

```go
// Result example:
// "Port forward active: localhost:8080 -> pod/my-pod:8080 (session: sess-1234)"
```

---

## 12. Configuration Extension

Add the following to `internal/config/config.go`:

```go
// Config root — add this field
type Config struct {
    // ... existing fields ...
    Remote RemoteConfig `yaml:"remote" json:"remote"`
}

// RemoteConfig holds all remote access configuration.
type RemoteConfig struct {
    Enabled     bool            `yaml:"enabled"      json:"enabled"`
    Profiles    []RemoteProfile `yaml:"profiles"     json:"profiles"`
    Kubeconfigs []string        `yaml:"kubeconfigs"  json:"kubeconfigs"`  // extra kubeconfig paths
    Pool        PoolConfig      `yaml:"pool"         json:"pool"`
    Audit       AuditConfig     `yaml:"audit"        json:"audit"`
    Sanitize    SanitizeConfig  `yaml:"sanitize"     json:"sanitize"`
}

// RemoteProfile mirrors remote.Profile for YAML serialization.
// Identical struct; re-declared here to avoid import cycle.
type RemoteProfile struct {
    Name           string            `yaml:"name"`
    Host           string            `yaml:"host"`
    Port           int               `yaml:"port,omitempty"`
    User           string            `yaml:"user"`
    IdentityFile   string            `yaml:"identity_file,omitempty"`
    JumpHosts      []string          `yaml:"jump_hosts,omitempty"`
    Environment    string            `yaml:"environment"`
    Timeout        string            `yaml:"timeout,omitempty"`       // parsed to time.Duration
    CommandAllow   []string          `yaml:"command_allow,omitempty"`
    CommandDeny    []string          `yaml:"command_deny,omitempty"`
    RequireConfirm bool              `yaml:"require_confirm,omitempty"`
    AccessWindow   *AccessWindowCfg  `yaml:"access_window,omitempty"`
    Labels         map[string]string `yaml:"labels,omitempty"`
}

// AccessWindowCfg is the YAML-serializable form of remote.AccessWindow.
type AccessWindowCfg struct {
    WeekdaysOnly bool `yaml:"weekdays_only"`
    StartHour    int  `yaml:"start_hour"`
    EndHour      int  `yaml:"end_hour"`
}

// PoolConfig controls connection pool behavior.
type PoolConfig struct {
    MaxPerProfile  int    `yaml:"max_per_profile"`   // default 3
    IdleTimeout    string `yaml:"idle_timeout"`      // default "5m"
    HealthInterval string `yaml:"health_interval"`   // default "30s"
}

// AuditConfig controls audit logging.
type AuditConfig struct {
    Enabled    bool   `yaml:"enabled"`     // default true
    RetainDays int    `yaml:"retain_days"` // default 90
}

// SanitizeConfig controls output sanitization.
type SanitizeConfig struct {
    Enabled         bool     `yaml:"enabled"`           // default true
    CustomPatterns  []string `yaml:"custom_patterns"`   // additional regexes
    SanitizePrivateIPs bool  `yaml:"sanitize_private_ips"` // default true
}
```

---

## 13. Security Architecture Summary

### Threat model and mitigations

| Threat | Mitigation |
|--------|-----------|
| LLM prompt injection causes destructive remote commands | `RemoteChecker` evaluates commands before execution; destructive patterns always require confirmation regardless of allow rules |
| SSH credentials leak into LLM context | `KeyStore` returns `gossh.AuthMethod` values (opaque interfaces), never strings; credentials are resolved at dial time and do not flow through `map[string]any` tool arguments |
| Command output contains secrets | `OutputSanitizer` redacts tokens, keys, IPs before output reaches `tools.Result` |
| Agent exploited to pivot through SSH jump hosts | Jump host chains are declared statically in profile config; the agent cannot construct arbitrary ProxyJump chains |
| Sub-agent escalates to production access | `Pool.Acquire` enforces agent-ID based access; sub-agents receive a checker with prod profiles excluded |
| Compromised session replays audit-logged commands | Audit log is append-only; old entries require local DB access to read, not remote API access |
| Time-of-check/time-of-use race on permissions | Permission check and command execution are coupled inside the tool's `Execute` method; there is no gap where the decision could be stale |
| Uncontrolled connection creation exhausts resources | `Pool` enforces per-profile and per-environment connection limits |
| Remote commands exceed expected duration | `Profile.Timeout` is enforced via `context.WithTimeout` before the SSH exec channel is opened |
| Known-hosts spoofing | `KnownHostsCallback` enforces strict host key verification; new hosts prompt the user |
| SFTP used to exfiltrate files | File transfers require explicit `remote_transfer` tool call, which goes through `RemoteChecker`; download from prod is LevelAsk by default |

### What is intentionally NOT implemented

- ProxyCommand execution (shell injection risk)
- Automatic VPN connection management (high-risk automation)
- Password-based SSH authentication (prefer keys + agent)
- Agent forwarding via `ssh -A` (lateral movement risk; use explicit jump host chains instead)
- Writing kubeconfig files or modifying cluster RBAC from within the agent
- Dynamic creation of new profiles at runtime (profiles are static config)

---

## 14. Dependency Summary

| Dependency | Purpose | Binary size impact |
|-----------|---------|-------------------|
| `golang.org/x/crypto/ssh` | SSH client | ~500KB |
| `github.com/pkg/sftp` | SFTP/SCP over SSH | ~200KB |
| `k8s.io/client-go` (optional, build tag) | Kubernetes exec, logs, port-forward | ~8-12MB |
| No new deps for network probe | Pure Go TCP dial | 0 |
| No new deps for SOCKS5 | Implement directly using `golang.org/x/net/proxy` | ~50KB |

Without the Kubernetes build tag, the remote access system adds approximately 750KB to the binary, staying well within the 18MB target. With it, the binary grows by ~10MB — still under 18MB total from the current ~2.4MB base.

The recommended approach is to ship two variants:
- `blackcat` — SSH + network probe only (~3.2MB)
- `blackcat-k8s` — full including kubectl (~13MB)

---

## 15. Integration with Existing Code

### New action types

Add to `internal/security/security.go`:

```go
const (
    // Existing types ...
    ActionRemoteExec     ActionType = "remote_exec"
    ActionRemoteTransfer ActionType = "remote_transfer"
    ActionKubeExec       ActionType = "kube_exec"
    ActionKubeLogs       ActionType = "kube_logs"
)
```

### Tool action type mapping

Add to `internal/agent/core.go` in the `toolActionType` switch:

```go
case "remote_exec":
    return security.ActionRemoteExec
case "remote_transfer":
    return security.ActionRemoteTransfer
case "kube_exec":
    return security.ActionKubeExec
case "kube_logs":
    return security.ActionKubeLogs
```

### Default permission rules

Add to `internal/security/permission.go` in `addDefaults`:

```go
// Remote actions default to ask — no remote command runs silently
pc.rules = append(pc.rules, PermissionRule{
    Action: ActionRemoteExec,
    Level:  LevelAsk,
})
pc.rules = append(pc.rules, PermissionRule{
    Action: ActionRemoteTransfer,
    Level:  LevelAsk,
})
pc.rules = append(pc.rules, PermissionRule{
    Action: ActionKubeExec,
    Level:  LevelAsk,
})
```

### Registration

In the main initialization path (`cmd/blackcat/main.go` or the serve command):

```go
if cfg.Remote.Enabled {
    pool := pool.NewPool(cfg.Remote.Pool)
    audit := audit.NewLogger(db)
    sanitizer := sanitize.NewSanitizer(cfg.Remote.Sanitize)
    remoteChecker := permission.NewRemoteChecker(checker, cfg.Remote.Profiles)

    registry.Register(remotetool.NewRemoteExecTool(pool, remoteChecker, audit, sanitizer))
    registry.Register(remotetool.NewRemoteTransferTool(pool, remoteChecker, audit, sanitizer))

    if kubeclient != nil {
        registry.Register(remotetool.NewKubeExecTool(kubeclient, remoteChecker, audit, sanitizer))
        registry.Register(remotetool.NewKubeLogsTool(kubeclient, remoteChecker, audit))
        registry.Register(remotetool.NewKubePortForwardTool(kubeclient, remoteChecker))
    }
}
```

---

## 16. Rate Limiting

Remote operations are rate-limited per profile using a token bucket:

```go
// RateLimiter controls the frequency of remote operations per profile.
type RateLimiter interface {
    // Allow returns true if the operation is within the rate limit.
    // It is non-blocking; callers should respect the returned bool.
    Allow(profileName string) bool

    // Wait blocks until the operation is within the rate limit or ctx is cancelled.
    Wait(ctx context.Context, profileName string) error
}
```

Default limits by environment:

| Environment | Max ops/minute | Burst |
|-------------|---------------|-------|
| dev         | 60            | 10    |
| staging     | 30            | 5     |
| prod        | 10            | 2     |
| ci          | 120           | 20    |

These limits prevent an agent loop running amok from spamming a production server. The prod limit of 10 ops/minute is intentionally conservative — a legitimate DevSecOps task rarely needs more than that.

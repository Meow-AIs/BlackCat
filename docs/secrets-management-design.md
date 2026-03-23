# BlackCat Secret Management System — Design Document

## Overview

This document describes the secret management subsystem for BlackCat, providing secure storage, access control, injection, and lifecycle management for all credential types used by the AI agent.

**Design Principles:**
- Secrets never exist in plaintext config files
- Secrets never leak into LLM context (prompts, tool outputs, memory vectors, logs)
- Defense in depth: OS keychain > encrypted file > env vars
- Least privilege: sub-agents and tools only get secrets they need
- Auditability: every secret access is logged without logging the value

---

## 1. Storage Architecture

### Backend Priority Chain

```
┌─────────────────┐    ┌──────────────────┐    ┌──────────────────┐
│  OS Keychain     │───>│  Encrypted File  │───>│  Environment Var │
│  (preferred)     │    │  (fallback)      │    │  (last resort)   │
└─────────────────┘    └──────────────────┘    └──────────────────┘
  macOS: Keychain        ~/.blackcat/            BLACKCAT_SECRET_*
  Win: Cred Manager      secrets.enc             (read-only)
  Linux: libsecret       XChaCha20-Poly1305
```

**Writes** always go to the first available backend. **Reads** cascade through all backends until a match is found. This allows migration: a secret can exist in env vars initially and be promoted to the keychain when the user runs `blackcat secret set`.

### Metadata vs Values

Secret **values** are stored in backends. Secret **metadata** (name, type, scope, expiry, access rules) is stored in SQLite alongside the existing memory.db. This separation means:

- Listing secrets never touches the backend
- Access control is evaluated from metadata before touching the backend
- Audit logs are in the same database as metadata

### File Layout

```
~/.blackcat/
├── config.yaml          # references secrets by name, never by value
├── memory.db            # SQLite: memory + secret_metadata + secret_audit_log tables
├── secrets.enc          # encrypted file backend (fallback)
└── .blackcat.yaml       # project-level config (in project root)
```

---

## 2. Encryption

### Key Derivation: Argon2id

```
Passphrase ──> Argon2id(time=3, mem=64MB, threads=4) ──> 256-bit key
```

- **Algorithm**: Argon2id (winner of Password Hashing Competition)
- **Parameters**: 3 iterations, 64 MB memory, 4 threads, 32-byte output
- **Salt**: 16 random bytes, prepended to ciphertext
- **Why Argon2id**: Resistant to both GPU and side-channel attacks. The id variant combines Argon2i (side-channel resistant) and Argon2d (GPU resistant).

### Symmetric Encryption: XChaCha20-Poly1305

```
Key + Nonce ──> XChaCha20-Poly1305 ──> Ciphertext + Auth Tag
```

- **Algorithm**: XChaCha20-Poly1305 (AEAD)
- **Nonce**: 24 bytes (randomly generated per encryption)
- **Why XChaCha20**: 192-bit nonce eliminates nonce reuse risk with random generation. No AES-NI dependency (works on ARM, old x86). Constant-time. Available in Go's `golang.org/x/crypto`.

### Ciphertext Format

```
[ salt: 16 bytes ][ nonce: 24 bytes ][ ciphertext + poly1305 tag ]
```

### Master Key Management

| Environment | Master Key Source |
|---|---|
| Interactive TUI | Prompted from user on first use, cached in memory for session |
| `blackcat serve` (daemon) | `BLACKCAT_MASTER_PASSWORD` env var at startup |
| CI/CD | `BLACKCAT_MASTER_PASSWORD` env var |
| OS keychain available | No master password needed (OS handles encryption) |

### Memory Protection

- `SecureWipe()` zeros byte slices after use (best-effort; Go GC may copy)
- Master key held only in `EncryptedFileBackend` struct, wiped on `Lock()`
- Secret values returned to callers with documentation to call `SecureWipe(val)` when done

---

## 3. Secret Scoping

### Global Secrets (`~/.blackcat/`)

Available to all projects. Typical contents:
- LLM provider API keys
- Cloud provider credentials
- Git tokens
- SSH keys

### Project Secrets (`.blackcat.yaml` directory)

Available only when BlackCat is running from that project. Typical contents:
- Database connection strings
- Project-specific API keys
- Staging/production environment secrets

### Storage Key Format

```
<scope>/<name>
  global/openai_api_key
  project/my_db_password
```

### Config Reference Syntax

Instead of putting secrets in config.yaml:

```yaml
# WRONG (current state — secrets in plaintext)
providers:
  anthropic:
    api_key: "sk-ant-..."

# CORRECT (new system — reference by name)
providers:
  anthropic:
    api_key: "${secret:anthropic_api_key}"
```

The config loader resolves `${secret:NAME}` references at load time through the Manager.

---

## 4. Access Control

### Rules (evaluated in order)

1. **Project scope enforcement**: Project-scoped secrets are only accessible when the current working directory matches the secret's project path.

2. **Tool allowlist**: If `allowed_tools` is non-empty on a secret's metadata, only those tools can request the secret. Example: only the `shell` and `http` tools can access `github_token`.

3. **Sub-agent isolation**: Sub-agents are **denied by default**. A secret must explicitly list a sub-agent ID in `allowed_agents` for that sub-agent to access it. This prevents a compromised or confused sub-agent from exfiltrating credentials.

4. **Primary agent**: The primary agent can access any secret that passes rules 1-2.

### Access Context

Every secret request carries an `AccessContext`:

```go
type AccessContext struct {
    AgentID     string  // "primary" or "sub-agent:<id>"
    ToolName    string  // "shell", "http", "mcp:server-name"
    ProjectPath string  // current project root
    Reason      string  // human-readable justification
}
```

---

## 5. Injection Patterns

### Environment Variable Injection (Primary Method)

Secrets are injected as environment variables into subprocess execution. This is the only approved method because:
- Env vars are per-process (not visible in `ps` output on modern OS)
- No shell expansion issues (unlike command-line args)
- Standard approach understood by all tools and libraries

```
┌──────────┐    ┌──────────────┐    ┌──────────────┐    ┌────────────┐
│ Sandbox  │───>│ Injector     │───>│ SecureManager │───>│ Backend    │
│ Execute()│    │ InjectIntoCmd│    │ Get()         │    │ Get()      │
└──────────┘    └──────────────┘    └──────────────┘    └────────────┘
                     │                                        │
                     │ cmd.Env = [..., "OPENAI_API_KEY=sk-.."]│
                     └────────────────────────────────────────┘
```

### Auto-Injection

Secrets with an `env_var` field are automatically injected into every subprocess. The Injector's `AutoInjectForScope()` method resolves all eligible secrets for the current scope.

### Filtered Inheritance

When a subprocess inherits the parent environment, the Injector strips:
- `BLACKCAT_SECRET_*` (prevents double-injection)
- `BLACKCAT_MASTER_PASSWORD` (prevents master key leakage to children)

### MCP Server Injection

MCP server configs reference secrets by name instead of embedding values:

```yaml
mcp:
  servers:
    - name: github
      command: npx
      args: ["@modelcontextprotocol/server-github"]
      env:
        GITHUB_TOKEN: "${secret:github_token}"
```

---

## 6. Output Sanitization

### Sanitizer Architecture

```
Tool Output ──> Sanitizer.Redact() ──> Clean Output ──> Agent / Memory / LLM
```

The `Sanitizer` maintains an in-memory set of all known secret values and replaces occurrences with `[REDACTED:secret_name]`.

### Integration Points

| Component | How Sanitization is Applied |
|---|---|
| Sandbox.Execute() | Post-hook: sanitize stdout+stderr before returning Result |
| Memory.Store() | Pre-hook: sanitize content before embedding and storage |
| LLM.Chat/Stream() | Pre-hook: sanitize messages before sending to provider |
| Logger | Pre-hook: sanitize log messages before writing |
| TUI | Post-hook: sanitize displayed text |

### Pattern-Based Redaction

Beyond value-matching, `SanitizeForLLM()` applies heuristic patterns:
- `Bearer <token>` in HTTP headers
- `Authorization: <value>` headers
- `api_key=<value>` in URLs and configs
- `password=<value>` in connection strings

### Registration Lifecycle

1. On startup: `preloadSanitizer()` loads all existing secrets
2. On `Set()`: new value registered with sanitizer
3. On `Rotate()`: old value unregistered, new value registered
4. On `Delete()`: value unregistered
5. On shutdown: `sanitizer.Clear()`

---

## 7. Import / Export

### Supported Import Sources

| Source | Command | What it imports |
|---|---|---|
| .env file | `blackcat secret import dotenv .env` | All KEY=VALUE pairs |
| AWS credentials | `blackcat secret import aws` | Access keys from ~/.aws/credentials |
| GitHub token | `blackcat secret import github` | Token from `gh auth token` |
| Kubernetes | `blackcat secret import kubeconfig` | Certs and tokens from ~/.kube/config |
| JSON backup | `blackcat secret import json backup.enc` | Previously exported backup |
| 1Password CLI | `blackcat secret import 1password` | Via `op` CLI |

### Type Inference

The importer infers `SecretType` from environment variable names:
- `OPENAI_API_KEY` -> `TypeAPIKey`
- `AWS_SECRET_ACCESS_KEY` -> `TypeCloudCred`
- `DATABASE_URL` -> `TypeDBCred`
- `GITHUB_TOKEN` -> `TypeGitToken`

### Encrypted Export

```bash
blackcat secret export --scope global --output backup.enc
# Prompts for export passphrase (independent of master passphrase)
```

Format: Argon2id salt + XChaCha20-Poly1305 encrypted JSON. The JSON contains secret names, values, types, env vars, and tags.

---

## 8. Rotation

### Expiry Tracking

Every secret can have:
- `expires_at`: hard expiry date (secret becomes unusable after this)
- `rotation_days`: recommended rotation interval

### Rotation Workflow

```
blackcat secret check-expiry --within 30
# Lists secrets expiring in the next 30 days

blackcat secret rotate openai_api_key
# Prompts for new value, updates backend + metadata + sanitizer
```

### Automated Checks

When `blackcat serve` is running, the scheduler checks for expiring secrets daily and sends notifications through configured channels (Telegram, Discord, etc.).

### Rotation Lifecycle

1. Check `ExpiryStatus` via `Manager.CheckExpiry()`
2. User provides new value
3. `Manager.Rotate()` updates value + pushes expiry forward by `rotation_days`
4. Sanitizer: old value unregistered, new value registered
5. Audit log records the rotation event

---

## 9. Audit Trail

### What is Logged

| Field | Example |
|---|---|
| Timestamp | 2026-03-21T14:30:00Z |
| Secret Name | openai_api_key |
| Scope | global |
| Action | read, write, delete, rotate, inject |
| Actor | agent, sub-agent:research-1, tool:shell, user, scheduler |
| Reason | "Injecting for subprocess execution" |
| Success | true/false |
| Error | "access denied: sub-agent not in allowed list" |

### What is NOT Logged

- The secret value (never)
- The secret fingerprint changes (only on write/rotate actions)

### Query Interface

```bash
blackcat secret audit openai_api_key --limit 50
blackcat secret audit --actor "sub-agent:*" --since "2026-03-01"
```

---

## 10. Go Implementation

### Package Structure

```
internal/secrets/
├── types.go                  # SecretMetadata, SecretRef, AuditEntry, ExpiryStatus
├── store.go                  # Store, Backend, MetadataStore, AuditLog interfaces
├── crypto.go                 # Argon2id + XChaCha20-Poly1305
├── manager.go                # Manager (high-level coordination)
├── access_control.go         # AccessPolicy, SecureManager
├── backend_keychain.go       # OS keychain via go-keyring
├── backend_encrypted_file.go # Encrypted JSON file
├── backend_env.go            # Environment variable (read-only)
├── injection.go              # Injector for subprocess env vars
├── sanitizer.go              # Output redaction
├── importer.go               # Import from .env, AWS, etc.
├── metadata_sqlite.go        # SQLite metadata + audit log
└── integration.go            # Setup() entry point
```

### Go Dependencies

| Package | Purpose | Why This One |
|---|---|---|
| `golang.org/x/crypto/argon2` | Key derivation | Standard Go extended lib, Argon2id support |
| `golang.org/x/crypto/chacha20poly1305` | AEAD encryption | Standard Go extended lib, XChaCha20 support |
| `github.com/zalando/go-keyring` | OS keychain | Cross-platform (macOS/Win/Linux), well-maintained, small |
| `github.com/mattn/go-sqlite3` | Metadata storage | Already a dependency in BlackCat |

No new C dependencies. `go-keyring` is pure Go on macOS/Windows; on Linux it uses D-Bus to talk to libsecret (no CGo).

### Integration with Existing Code

**Config loader** (`internal/config/loader.go`):
```go
// Add secret reference resolution:
// "${secret:name}" -> Manager.Get(ctx, name, ScopeGlobal)
```

**Sandbox** (`internal/security/sandbox.go`):
```go
// After Execute(), pass output through Sanitizer.Redact()
// Before Execute(), call Injector.InjectIntoCmd() for secret env vars
```

**Memory** (`internal/memory/`):
```go
// Before storing any text, pass through Sanitizer.Redact()
```

**LLM providers** (`internal/llm/`):
```go
// Use Manager.Get() for API keys instead of reading from config
// Before sending messages, pass through SanitizeForLLM()
```

**Sub-agent pool** (`internal/agent/`):
```go
// Each sub-agent gets a SecureManager with its own AccessContext
// Sub-agents default to no secret access unless explicitly granted
```

### CLI Commands

```
blackcat secret set <name> [--type api_key] [--env-var OPENAI_API_KEY] [--scope global]
blackcat secret get <name> [--scope global]          # only for debugging, warns user
blackcat secret list [--scope global|project] [--type api_key]
blackcat secret delete <name> [--scope global]
blackcat secret rotate <name>
blackcat secret check-expiry [--within 30]
blackcat secret audit [name] [--actor ...] [--limit 50]
blackcat secret import dotenv <path>
blackcat secret import aws [--profile default]
blackcat secret export [--scope global] --output <path>
blackcat secret info                                  # shows backend, counts, health
```

---

## Security Properties Summary

| Property | How It Is Achieved |
|---|---|
| No plaintext in config | `${secret:name}` references resolved at runtime |
| No secrets in LLM context | Sanitizer applied to all messages before sending |
| No secrets in memory DB | Sanitizer applied before vector storage |
| No secrets in logs | Sanitizer applied to all log output |
| No secrets in tool output | Sanitizer applied to sandbox results |
| No secrets in ps/proc | Env var injection, not CLI args |
| Sub-agent isolation | Default-deny access policy for sub-agents |
| At-rest encryption | OS keychain or Argon2id+XChaCha20-Poly1305 |
| Rotation tracking | Expiry metadata + scheduler checks |
| Full audit trail | SQLite audit log with who/when/what (never the value) |
| Memory wiping | SecureWipe() on all transient secret copies |

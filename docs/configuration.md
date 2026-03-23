# BlackCat Configuration

> Complete configuration reference. For architecture context, see [Architecture](./architecture.md). For provider setup, see [Providers](./providers.md).

## Configuration Files

BlackCat uses two configuration layers, merged at startup:

| File | Scope | Location |
|------|-------|----------|
| Global config | All projects | `~/.blackcat/config.yaml` |
| Project config | Current project | `.blackcat.yaml` (project root) |

Project-level config overrides global config on a per-key basis.

## Full Configuration Reference

> **Important**: API keys and tokens should be stored in the encrypted secret store using `/config set`, not in this file. The `api_key` and `token` fields below are shown empty; use `/config set <provider>_api_key <value>` to store them securely. See [Secret Management](#secret-management).

```yaml
# ~/.blackcat/config.yaml — Global Configuration

# Default model (provider/model format)
model: "anthropic/claude-sonnet-4-6"

# ─── LLM Providers ───────────────────────────────────────────────────────────
# API keys: use `/config set anthropic_api_key <value>` (encrypted store)
# or set ANTHROPIC_API_KEY env var. Do NOT put keys here in plaintext.
providers:
  anthropic:
    api_key: ""                    # use secret store or ANTHROPIC_API_KEY env var
    models:                        # optional: restrict to specific models
      - claude-sonnet-4-6
      - claude-haiku-4-5

  openai:
    api_key: ""                    # use secret store or OPENAI_API_KEY env var
    models:
      - gpt-4o
      - gpt-4o-mini

  openrouter:
    api_key: ""                    # use secret store or OPENROUTER_API_KEY env var
    models: []                     # empty = all available models

  ollama:
    base_url: "http://localhost:11434"
    models:
      - llama3.2
      - codellama
      - nomic-embed-text

# ─── Model Router ────────────────────────────────────────────────────────────
# The router maps task types to provider tiers.
# Default: main = primary model, auxiliary = cheaper model, local = Ollama
#
# Task routing (built-in defaults):
#   Main tier:      reasoning, code_gen, vision
#   Auxiliary tier:  summarize, classify, extract_facts, memory_search, danger_assess, compression
#   Local tier:      embed (falls back to auxiliary if no local provider)

# ─── Memory ──────────────────────────────────────────────────────────────────
memory:
  enabled: true
  max_vectors: 10000               # maximum vectors before eviction
  embedding: "local"               # "local" (bundled ONNX) | "openai" | "ollama"
  retention_days: 30               # days before decay starts
  db_path: ""                      # default: ~/.blackcat/memory.db

# ─── Agent Behavior ──────────────────────────────────────────────────────────
agent:
  max_sub_agents: 3                # max concurrent sub-agents
  sub_agent_model: "anthropic/claude-haiku-4-5"  # cheaper model for sub-agents
  sub_agent_timeout: "300s"        # timeout per sub-agent

# ─── Permissions ─────────────────────────────────────────────────────────────
# Four levels: allow (silent), auto_approve (pattern match), ask (confirm), deny (block)
permissions:
  allow:
    - action: read_file
    - action: list_directory
    - action: search_code
    - action: shell
      patterns: ["git status*", "git log*", "git diff*", "git branch*", "git show*"]

  auto_approve:
    - action: shell
      patterns: ["go test*", "go build*", "make*"]
      excludes: ["*.env", "*.key"]
    - action: write_file
      patterns: ["*.go", "*.md", "*.yaml"]
      excludes: ["*.env", "*.secret*"]

  ask:
    - action: shell
    - action: write_file
    - action: web

  deny:
    - action: shell
      patterns: ["rm -rf /*", "mkfs*", "rm -rf .*"]
    - action: write_file
      patterns: ["*.env", "*.key", "*.pem", "*.secret*", "credentials*"]

# ─── MCP Servers ─────────────────────────────────────────────────────────────
mcp:
  servers:
    - name: "filesystem"
      command: "npx"
      args: ["-y", "@anthropic/mcp-server-filesystem", "/home/user/projects"]

    - name: "postgres"
      command: "npx"
      args: ["-y", "@anthropic/mcp-server-postgres"]
      env:
        DATABASE_URL: "postgresql://localhost:5432/mydb"

# ─── Channels ────────────────────────────────────────────────────────────────
# Channel tokens: use `/config set telegram_bot_token <value>` (encrypted store)
# or set environment variables. Do NOT put tokens here in plaintext.
channels:
  telegram:
    enabled: false
    token: ""                      # use secret store or TELEGRAM_BOT_TOKEN env var
    allowed_users: []              # Telegram user IDs (integers)
    mode: "private"                # "private" | "group"

  discord:
    enabled: false
    token: ""                      # use secret store or DISCORD_BOT_TOKEN env var
    allowed_guilds: []             # Guild (server) IDs
    allowed_channels: []           # Channel IDs

  slack:
    enabled: false
    app_token: ""                  # use secret store or SLACK_APP_TOKEN env var
    bot_token: ""                  # use secret store or SLACK_BOT_TOKEN env var
    allowed_channels: []           # Channel IDs

  whatsapp:
    enabled: false
    session_path: ""               # Path to Baileys session directory
    allowed_numbers: []            # Phone numbers in E.164 format

# ─── Signal & Email ─────────────────────────────────────────────────────────
# Signal and Email channel adapters exist in internal/channels/signal/ and
# internal/channels/email/ but do not yet have config struct entries.
# They must be configured programmatically. Config struct integration is planned.

# ─── Scheduler ───────────────────────────────────────────────────────────────
scheduler:
  enabled: false
  schedules:
    - name: "daily-security-scan"
      cron: "0 8 * * *"           # every day at 08:00
      task: "Run a security scan on the project"
      channel: "telegram"          # optional: send results to channel

    - name: "weekly-deps-check"
      cron: "0 9 * * 1"           # every Monday at 09:00
      task: "Check for outdated dependencies and known vulnerabilities"
```

## Provider-Specific Configuration

> **Security note**: Store API keys using the encrypted secret store (`/config set`), not in plaintext config files. The config file examples below show the `api_key` field for reference, but the recommended approach is to use `/config set <provider>_api_key <value>` or set the corresponding environment variable. See [Secret Management](#secret-management) for details.

### Anthropic

**Recommended** (encrypted store):
```
/config set anthropic_api_key sk-ant-api03-...
```

Or via environment variable:
```bash
export ANTHROPIC_API_KEY="sk-ant-..."
```

Config file (not recommended for API keys):
```yaml
providers:
  anthropic:
    api_key: ""   # leave empty when using secret store or env var
```

### OpenAI

```
/config set openai_api_key sk-...
```

```bash
export OPENAI_API_KEY="sk-..."
```

### OpenRouter

```
/config set openrouter_api_key sk-or-...
```

```bash
export OPENROUTER_API_KEY="sk-or-..."
```

### Ollama (Local)

No API key needed. Just ensure Ollama is running:

```yaml
providers:
  ollama:
    base_url: "http://localhost:11434"
    models:
      - llama3.2
      - nomic-embed-text
```

### Groq, ZAI, Kimi, xAI

These providers follow the same pattern as OpenAI. Set their respective API keys via environment variables:

```bash
export GROQ_API_KEY="gsk_..."
export ZAI_API_KEY="..."
export KIMI_API_KEY="..."
export XAI_API_KEY="xai-..."
```

## Environment Variables

Sensitive values can be provided via environment variables, the encrypted secret store (`/config set`), or both. The preferred approach is the encrypted store; environment variables serve as a fallback for CI/CD and containerized environments.

| Variable | Purpose |
|----------|---------|
| `ANTHROPIC_API_KEY` | Anthropic API key |
| `OPENAI_API_KEY` | OpenAI API key |
| `OPENROUTER_API_KEY` | OpenRouter API key |
| `GROQ_API_KEY` | Groq API key |
| `ZAI_API_KEY` | ZAI API key |
| `KIMI_API_KEY` | Kimi (Moonshot) API key |
| `XAI_API_KEY` | xAI (Grok) API key |
| `TELEGRAM_BOT_TOKEN` | Telegram bot token |
| `DISCORD_BOT_TOKEN` | Discord bot token |
| `SLACK_APP_TOKEN` | Slack app-level token |
| `SLACK_BOT_TOKEN` | Slack bot token |
| `BLACKCAT_DB_PATH` | Override memory database path |
| `BLACKCAT_MASTER_PASSWORD` | Master passphrase for the encrypted file backend (headless only) |
| `BLACKCAT_SECRET_*` | Read-only secret fallback (e.g., `BLACKCAT_SECRET_OPENAI_API_KEY`) |

## Project-Level Configuration

Create `.blackcat.yaml` in your project root to override global settings:

```yaml
# .blackcat.yaml — Project Configuration

# Override default model for this project
model: "openai/gpt-4o"

# Project-specific permissions
permissions:
  auto_approve:
    - action: shell
      patterns: ["npm test*", "npm run build*"]

# Project-specific MCP servers
mcp:
  servers:
    - name: "project-db"
      command: "npx"
      args: ["-y", "@anthropic/mcp-server-postgres"]
      env:
        DATABASE_URL: "postgresql://localhost:5432/project_db"
```

## Storage Paths

| Path | Purpose |
|------|---------|
| `~/.blackcat/config.yaml` | Global configuration |
| `~/.blackcat/memory.db` | SQLite database (memory + vectors + schedules) |
| `~/.blackcat/plugins/` | Installed plugin binaries and manifests |
| `~/.blackcat/skills/` | Installed skill packages |
| `~/.blackcat/secrets.enc` | Encrypted secret values (XChaCha20-Poly1305, used when OS keychain is unavailable) |
| `.blackcat.yaml` | Project-level config override |

## Default Values

When no configuration is provided, BlackCat uses these defaults:

```go
Config{
    Model: "anthropic/claude-sonnet-4-6",
    Memory: MemoryConfig{
        Enabled:       true,
        MaxVectors:    10000,
        Embedding:     "local",
        RetentionDays: 30,
    },
    Agent: AgentConfig{
        MaxSubAgents:    3,
        SubAgentModel:   "anthropic/claude-haiku-4-5",
        SubAgentTimeout: "300s",
    },
    Scheduler: SchedulerConfig{
        Enabled: false,
    },
    Providers: ProvidersConfig{
        Ollama: OllamaEntry{
            BaseURL: "http://localhost:11434",
        },
    },
}
```

## Secret Management

BlackCat includes an encrypted secret management system (`internal/secrets/`) that keeps API keys, tokens, and credentials out of plaintext config files and prevents them from being sent to external LLM providers.

### Setting Secrets Securely

Use `/config set` to store API keys in the encrypted secret store, **not** in config files:

```
/config set anthropic_api_key sk-ant-api03-...
/config set openai_api_key sk-...
/config set telegram_bot_token 123456:ABCdef...
```

This stores the value in the secure backend (OS Keychain or encrypted file), **never** in `config.yaml`.

### Backend Priority Chain

Secrets are stored and retrieved in this preference order:

| Priority | Backend | Platform | Security |
|----------|---------|----------|----------|
| 1 | OS Keychain | macOS Keychain, Windows Credential Manager, Linux Secret Service (libsecret) | Hardware-backed, OS-managed encryption |
| 2 | Encrypted File | All platforms (`~/.blackcat/secrets.enc`) | Argon2id key derivation + XChaCha20-Poly1305 |
| 3 | Environment Variables | All platforms (read-only, `BLACKCAT_SECRET_*` prefix) | Plaintext in memory (last resort) |

If the OS keychain is available, it is always preferred. The encrypted file backend is used on headless servers and containers where no keyring daemon is available. Environment variables are a read-only fallback.

### Config File References

The `config.yaml` file should use `${VAR}` placeholder references or leave `api_key` fields empty. The examples in this document that show literal API key values (e.g., `api_key: "sk-ant-..."`) are for illustration only. In practice, use:

```yaml
providers:
  anthropic:
    api_key: ""   # set via: /config set anthropic_api_key <value>
```

Or use environment variables:

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
```

### Master Password (Headless Environments)

When running on headless servers without an OS keychain, the encrypted file backend requires a master password. Set it via environment variable:

```bash
export BLACKCAT_MASTER_PASSWORD="your-strong-passphrase"
```

This passphrase is used with Argon2id (3 iterations, 64 MB memory, 4 threads) to derive a 256-bit XChaCha20-Poly1305 encryption key. The encrypted file is stored at `~/.blackcat/secrets.enc`.

### Secret Scoping

Secrets can be scoped globally or per-project:

| Scope | Availability | Use Case |
|-------|-------------|----------|
| `global` | All projects | LLM API keys, channel tokens |
| `project` | Single project only | Database credentials, project-specific tokens |

### Access Control

Secrets support fine-grained access restrictions:

- **Tool-level**: Only specified tools may read a secret (`allowed_tools`)
- **Agent-level**: Sub-agents are blocked by default; must be explicitly listed in `allowed_agents`
- **Expiry**: Secrets can have an expiry date and rotation policy
- **Audit logging**: Every read, write, delete, and rotate is logged with timestamp, actor, and success status

### Importing Secrets

BlackCat can import secrets from common formats:

```
# Import from a .env file
blackcat import dotenv /path/to/.env

# Import from AWS credentials file
blackcat import aws-credentials ~/.aws/credentials
```

The original files are not modified. Imported secrets are stored in the encrypted backend and tracked with an `imported_from` metadata field.

### Secret Leak Prevention

All tool output, log messages, memory entries, and LLM messages pass through a multi-layer sanitization pipeline before secrets can reach an external provider. See [Security: Output Sanitization](./security.md#output-sanitization) for details.

## Validating Configuration

Use the `/doctor` slash command to verify your configuration is correct:

```
/doctor
```

This checks:
- LLM provider connectivity
- Memory database health
- Plugin status
- Config file syntax
- Secret store accessibility

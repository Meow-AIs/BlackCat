# Slash Commands Reference

> All built-in slash commands for BlackCat. For configuration, see [Configuration](./configuration.md). For skills and plugins, see [Skills](./skills.md) and [Plugins](./plugins.md).

## Overview

BlackCat provides 30+ built-in slash commands organized by category. Type `/help` to see all available commands, or `/help <command>` for detailed usage.

Commands are registered in `internal/commands/builtin.go` and processed through a middleware chain (`internal/commands/middleware.go`).

## General Commands

| Command | Aliases | Description | Usage |
|---------|---------|-------------|-------|
| `/help` | `/h` | Show all commands or help for a specific command | `/help [command]` |
| `/clear` | | Clear conversation history | `/clear` |
| `/compact` | | Trigger context compression to free tokens | `/compact` |
| `/cost` | | Show session cost and token usage | `/cost` |
| `/model` | | Switch or show current model | `/model [provider/name]` |
| `/think` | | Toggle extended thinking mode | `/think` |
| `/fast` | | Toggle fast mode (skip reasoning) | `/fast` |
| `/status` | | Show system status (model, domain, memory, plugins) | `/status` |
| `/version` | | Show BlackCat version | `/version` |
| `/exit` | `/quit`, `/q` | Exit the session | `/exit` |

### Examples

```
/help                           # list all commands
/help memory                    # help for memory commands
/model anthropic/claude-opus-4-5  # switch to Opus
/model                          # show current model
/cost                           # show session spending
/status                         # show system overview
```

## Memory Commands

| Command | Description | Usage |
|---------|-------------|-------|
| `/memory search` | Search memory with hybrid retrieval | `/memory search <query>` |
| `/memory stats` | Show memory statistics (entries per tier, vector count) | `/memory stats` |
| `/memory forget` | Delete a memory entry by ID | `/memory forget <id>` |
| `/memory list` | List recent memories, optionally filtered by tier | `/memory list [tier]` |
| `/memory export` | Export all memories as JSON | `/memory export` |

Alias: `/mem` works in place of `/memory`.

### Examples

```
/memory search "kubernetes deployment"
/memory stats
/memory list episodic
/memory list semantic
/memory forget ep-2024-01-15-001
/mem search "database migration"
```

See [Memory System](./memory-system.md) for details on how search works.

## Skills Commands

| Command | Description | Usage |
|---------|-------------|-------|
| `/skills search` | Search the skill marketplace | `/skills search <query>` |
| `/skills install` | Install a skill by name or path | `/skills install <name>` |
| `/skills uninstall` | Remove an installed skill | `/skills uninstall <name>` |
| `/skills list` | List all installed skills | `/skills list` |
| `/skills update` | Update all skills to latest versions | `/skills update` |

### Examples

```
/skills search "secret scanner"
/skills install devsecops/secret-scanner
/skills list
/skills uninstall devsecops/secret-scanner
/skills update
```

See [Skills](./skills.md) for the full skill development and marketplace guide.

## Configuration Commands

| Command | Description | Usage |
|---------|-------------|-------|
| `/config show` | Show current configuration | `/config show` |
| `/config set` | Set a configuration value or store a secret | `/config set <key> <value>` |
| `/config reset` | Reset configuration to defaults | `/config reset` |

### Examples

```
/config show
/config set model openai/gpt-4o
/config reset
```

### Storing Secrets

When the key name matches a known secret pattern (contains `api_key`, `token`, `password`, `secret`, or `credential`), the value is automatically stored in the **encrypted secret store** (OS Keychain or XChaCha20-Poly1305 encrypted file) instead of the config file:

```
/config set anthropic_api_key sk-ant-api03-...
/config set openai_api_key sk-...
/config set telegram_bot_token 123456:ABCdef...
/config set openrouter_api_key sk-or-...
```

Stored secrets are:
- Encrypted at rest (OS Keychain or XChaCha20-Poly1305)
- Automatically registered with the output sanitizer for redaction
- Resolved at runtime for LLM API calls and tool execution
- Never written to `config.yaml` or sent to LLM providers

See [Configuration: Secret Management](./configuration.md#secret-management) for full details.

## Domain Commands

| Command | Description | Usage |
|---------|-------------|-------|
| `/domain` | Show current domain | `/domain` |
| `/domain set` | Switch to a specific domain | `/domain set <name>` |
| `/domain detect` | Auto-detect domain from project files | `/domain detect` |

Available domains: `general`, `devsecops`, `architect`.

### Examples

```
/domain                         # show current: "general"
/domain set devsecops           # switch to DevSecOps
/domain set architect           # switch to Architect
/domain detect                  # auto-detect from project
```

See [DevSecOps](./devsecops.md) and [Architect](./architect.md) for domain-specific features.

## Plugin Commands

| Command | Description | Usage |
|---------|-------------|-------|
| `/plugin list` | List all plugins with status | `/plugin list` |
| `/plugin install` | Install a plugin from path or URL | `/plugin install <name>` |
| `/plugin start` | Start an installed plugin | `/plugin start <name>` |
| `/plugin stop` | Stop a running plugin | `/plugin stop <name>` |

### Examples

```
/plugin list
/plugin install acme/my-provider
/plugin start acme/my-provider
/plugin stop acme/my-provider
```

See [Plugins](./plugins.md) for the plugin development guide.

## Hook Commands

| Command | Description | Usage |
|---------|-------------|-------|
| `/hooks list` | List all active hooks | `/hooks list` |
| `/hooks enable` | Enable a hook by ID | `/hooks enable <id>` |
| `/hooks disable` | Disable a hook by ID | `/hooks disable <id>` |

### Examples

```
/hooks list
/hooks enable auto-format
/hooks disable auto-format
```

## Git Commands

| Command | Description | Usage |
|---------|-------------|-------|
| `/diff` | Show git diff of current changes | `/diff` |
| `/commit` | Stage all changes and commit with message | `/commit <message>` |
| `/undo` | Undo last file change | `/undo` |

### Examples

```
/diff
/commit "feat: add user authentication"
/undo
```

## Debug Commands

| Command | Description | Usage |
|---------|-------------|-------|
| `/doctor` | Run system health check | `/doctor` |
| `/tokens` | Show token usage breakdown | `/tokens` |
| `/context` | Show system prompt layer sizes | `/context` |
| `/debug` | Toggle debug mode | `/debug [on\|off]` |

### Examples

```
/doctor
# Health check:
#   LLM: ok
#   Memory: ok
#   Plugins: ok
#   Config: ok

/tokens
# Token usage:
#   System prompt: 1,234
#   Conversation: 5,678
#   Total: 6,912

/context
# Context layers:
#   Base: 200 tokens
#   Domain: 450 tokens
#   Skills: 100 tokens
#   Memory: 800 tokens
#   Total: 1,550 tokens

/debug on
/debug off
```

## Custom Commands from Skills

Skills can register custom slash commands. When a skill is installed, its trigger pattern may be exposed as a command. See [Skills](./skills.md) for details.

## Custom Commands from Plugins

Plugins of type `domain` or `hook` can register additional commands through the plugin bridge. See [Plugins](./plugins.md) for details.

## Command Middleware

All commands pass through a middleware chain (`internal/commands/middleware.go`) that can:

- Log command execution
- Check permissions
- Transform arguments
- Add pre/post-processing

The middleware chain runs before the command handler and can short-circuit execution if needed.

## Command Registry

Commands are managed by a central registry (`internal/commands/registry.go`) that supports:

- Registration with name, aliases, description, usage, and category
- Sub-command trees (e.g., `/memory search`, `/memory stats`)
- Help text generation
- Command lookup by name or alias
- Custom command registration (from skills/plugins)

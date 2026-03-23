# Plugin Development Guide

> How to extend BlackCat with plugins. For architecture context, see [Architecture](./architecture.md). For security, see [Security](./security.md).

## Overview

BlackCat plugins are external processes that communicate via JSON-RPC over stdin/stdout. A plugin can extend BlackCat with new LLM providers, messaging channels, domain knowledge, security scanners, or event hooks.

### Plugin Types

| Type | Purpose | Bridge Adapter | Internal Interface |
|------|---------|---------------|-------------------|
| `provider` | LLM provider | `ProviderBridge` | `llm.Provider` |
| `channel` | Messaging platform | `ChannelBridge` | `channels.Adapter` |
| `domain` | Domain specialization | `DomainBridge` | Domain system prompt + tools |
| `scanner` | Security scanner | (via domain bridge) | `devsecops.Scanner` |
| `hook` | Event hook | (via hook engine) | `hooks.HookHandler` |

## JSON-RPC Protocol

Plugins communicate with BlackCat using JSON-RPC over stdin/stdout pipes. Each message is a single JSON line terminated by `\n`.

### Request Format

```json
{
  "id": "req-1",
  "method": "chat",
  "params": {
    "model": "my-model",
    "messages": [{"role": "user", "content": "Hello"}]
  }
}
```

### Response Format

```json
{
  "id": "req-1",
  "result": {
    "content": "Hello! How can I help?",
    "model": "my-model",
    "usage": {"prompt_tokens": 10, "completion_tokens": 15, "total_tokens": 25}
  }
}
```

### Error Response

```json
{
  "id": "req-1",
  "error": "model not found: invalid-model"
}
```

### Required Methods

Every plugin must implement:

| Method | Purpose |
|--------|---------|
| `ping` | Health check. Must return `"pong"`. |

### Provider Plugin Methods

| Method | Params | Returns |
|--------|--------|---------|
| `chat` | `model`, `messages`, `max_tokens?`, `temperature?`, `tools?` | `ChatResponse` |
| `models` | (none) | `[]ModelInfo` |

### Channel Plugin Methods

| Method | Params | Returns |
|--------|--------|---------|
| `start` | (none) | (none) |
| `stop` | (none) | (none) |
| `send` | `channel_id`, `text`, `reply_to_id?`, `format?` | (none) |

### Domain Plugin Methods

| Method | Params | Returns |
|--------|--------|---------|
| `system_prompt` | (none) | `string` (system prompt text) |
| `detect` | `path` (project path) | `{"confidence": 0.0-1.0}` |
| `tools` | (none) | `[]string` (tool names) |

## Plugin Manifest

Every plugin requires a `manifest.json`:

```json
{
  "name": "acme/my-provider",
  "version": "1.0.0",
  "type": "provider",
  "description": "ACME Corp LLM provider for BlackCat",
  "author": "ACME Corp",
  "license": "MIT",
  "command": "./acme-provider",
  "args": ["--port", "0"],
  "protocol": "jsonrpc",
  "port": 0,
  "capabilities": ["chat", "stream", "models"],
  "config": {
    "api_key": {
      "type": "secret",
      "description": "ACME API key",
      "required": true
    },
    "base_url": {
      "type": "string",
      "description": "API base URL",
      "default": "https://api.acme.com/v1",
      "required": false
    }
  },
  "min_version": "0.1.0"
}
```

### Manifest Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Unique ID in `author/name` format |
| `version` | string | Yes | Semantic version (major.minor.patch) |
| `type` | string | Yes | One of: `provider`, `channel`, `domain`, `scanner`, `hook` |
| `description` | string | Yes | Short description |
| `author` | string | Yes | Author name or organization |
| `license` | string | Yes | SPDX license identifier |
| `command` | string | Yes | Binary to execute |
| `args` | []string | No | Command-line arguments |
| `protocol` | string | Yes | `"jsonrpc"` (only supported protocol currently) |
| `port` | int | No | Network port (0 = auto-assign, not used for stdio) |
| `capabilities` | []string | Yes | What the plugin provides |
| `config` | object | No | User-configurable fields |
| `min_version` | string | No | Minimum BlackCat version required |

### Config Field Types

| Type | Description |
|------|-------------|
| `string` | Plain text value |
| `int` | Integer value |
| `bool` | Boolean value |
| `secret` | Sensitive value (stored in secret store, not config file) |

## Creating a Provider Plugin

### Step 1: Create the Binary

Write a program that reads JSON-RPC requests from stdin and writes responses to stdout.

**Example in Go:**

```go
package main

import (
    "bufio"
    "encoding/json"
    "fmt"
    "os"
)

type Request struct {
    ID     string         `json:"id"`
    Method string         `json:"method"`
    Params map[string]any `json:"params,omitempty"`
}

type Response struct {
    ID     string `json:"id"`
    Result any    `json:"result,omitempty"`
    Error  string `json:"error,omitempty"`
}

func main() {
    scanner := bufio.NewScanner(os.Stdin)
    for scanner.Scan() {
        var req Request
        if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
            continue
        }

        var resp Response
        resp.ID = req.ID

        switch req.Method {
        case "ping":
            resp.Result = "pong"

        case "chat":
            // Implement your LLM call here
            resp.Result = map[string]any{
                "content":       "Hello from my plugin!",
                "model":         "my-model",
                "finish_reason": "stop",
                "usage": map[string]int{
                    "prompt_tokens":     10,
                    "completion_tokens": 5,
                    "total_tokens":      15,
                },
            }

        case "models":
            resp.Result = []map[string]any{
                {
                    "id":                "my-model",
                    "name":              "My Model",
                    "max_tokens":        4096,
                    "input_cost_per_1m":  1.0,
                    "output_cost_per_1m": 2.0,
                },
            }

        default:
            resp.Error = fmt.Sprintf("unknown method: %s", req.Method)
        }

        data, _ := json.Marshal(resp)
        fmt.Println(string(data))
    }
}
```

**Example in Python:**

```python
import json
import sys

def handle_request(req):
    method = req.get("method")

    if method == "ping":
        return "pong"
    elif method == "chat":
        return {
            "content": "Hello from Python plugin!",
            "model": "py-model",
            "finish_reason": "stop",
            "usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}
        }
    elif method == "models":
        return [{"id": "py-model", "name": "Python Model", "max_tokens": 4096}]
    else:
        raise ValueError(f"unknown method: {method}")

for line in sys.stdin:
    req = json.loads(line)
    try:
        result = handle_request(req)
        resp = {"id": req["id"], "result": result}
    except Exception as e:
        resp = {"id": req["id"], "error": str(e)}
    print(json.dumps(resp), flush=True)
```

### Step 2: Create the Manifest

Create `manifest.json` alongside your binary:

```json
{
  "name": "you/my-provider",
  "version": "1.0.0",
  "type": "provider",
  "description": "My custom LLM provider",
  "author": "Your Name",
  "license": "MIT",
  "command": "./my-provider",
  "protocol": "jsonrpc",
  "capabilities": ["chat", "models"]
}
```

### Step 3: Install

```
/plugin install /path/to/my-provider
```

Or copy to `~/.blackcat/plugins/you/my-provider/`.

### Step 4: Use

```
/plugin start you/my-provider
/model you/my-provider/my-model
```

## Creating a Channel Plugin

A channel plugin implements `start`, `stop`, `send`, and streams incoming messages.

```json
{
  "name": "acme/matrix-channel",
  "version": "1.0.0",
  "type": "channel",
  "description": "Matrix messaging for BlackCat",
  "author": "ACME",
  "license": "MIT",
  "command": "./matrix-channel",
  "protocol": "jsonrpc",
  "capabilities": ["start", "stop", "send", "receive"],
  "config": {
    "homeserver": {
      "type": "string",
      "description": "Matrix homeserver URL",
      "required": true
    },
    "access_token": {
      "type": "secret",
      "description": "Matrix access token",
      "required": true
    }
  }
}
```

The channel bridge (`ChannelBridge`) translates `channels.Adapter` calls into JSON-RPC:

| Adapter Method | JSON-RPC Method |
|---------------|-----------------|
| `Start(ctx)` | `start` |
| `Stop(ctx)` | `stop` |
| `Send(ctx, msg)` | `send` with `channel_id`, `text`, `reply_to_id`, `format` |
| `Receive()` | Returns a Go channel fed by plugin notifications |
| `Platform()` | Returns `"plugin:<manifest.name>"` |

## Bridge Adapters

Bridge adapters connect plugin processes to internal BlackCat interfaces:

### ProviderBridge

Wraps a plugin as `llm.Provider`:
- `Chat()` calls `"chat"` method
- `Stream()` currently wraps `Chat()` with a single-chunk channel (full streaming requires protocol extension)
- `Models()` calls `"models"` method
- `Name()` returns the manifest name

### ChannelBridge

Wraps a plugin as `channels.Adapter`:
- `Start()` / `Stop()` manage plugin lifecycle
- `Send()` delivers messages via `"send"` method
- `Receive()` returns an incoming message channel
- `Platform()` returns `"plugin:<name>"`

### DomainBridge

Wraps a plugin as domain knowledge:
- `SystemPrompt()` retrieves the domain-specific system prompt
- `Detect(path)` asks the plugin how well it matches a project
- `Tools()` lists domain-specific tool names

## Plugin Registry

The plugin registry manages all installed plugins:

```
/plugin list                    # list all plugins with status
/plugin install <path|url>      # install a plugin
/plugin start <name>            # start a plugin
/plugin stop <name>             # stop a plugin
```

### Plugin Lifecycle States

| State | Description |
|-------|-------------|
| `installed` | Manifest loaded, binary present |
| `active` | Process running, responding to requests |
| `stopped` | Explicitly stopped |
| `error` | Process crashed or failed health check |

### Health Checks

BlackCat periodically pings active plugins. If a plugin fails to respond to `ping` within the timeout, it transitions to the `error` state.

## Plugin Security

### Process Isolation

Plugins run as separate processes communicating over stdin/stdout JSON-RPC. They:

- **Do not inherit sensitive environment variables.** BlackCat's sandbox environment filter strips variables containing SECRET, KEY, TOKEN, PASSWORD, AUTH, API_, CREDENTIAL, and PRIVATE from the inherited environment before spawning plugin processes.
- **Do not receive secret values in JSON-RPC messages.** Secret values from the encrypted store are never included in JSON-RPC request payloads. If a plugin needs an API key (e.g., a custom LLM provider), the key is configured as a `"secret"` config field type, stored in the encrypted secret store, and injected as an environment variable only for that plugin's process.
- **Have their output sanitized.** All plugin responses pass through the output sanitization pipeline before reaching the LLM context, redacting any secrets that may appear in the response.

### Secret Config Fields

Plugin manifests can declare config fields with `"type": "secret"`:

```json
{
  "config": {
    "api_key": {
      "type": "secret",
      "description": "Provider API key",
      "required": true
    }
  }
}
```

Secret-typed config values are stored in the encrypted secret store (not in the plugin manifest or config file) and are only injected as environment variables when the plugin process starts.

## Best Practices

1. **Always implement `ping`** -- BlackCat uses it for health checks
2. **Use line-delimited JSON** -- each request and response must be a single line
3. **Flush stdout after every response** -- buffered output will cause timeouts
4. **Handle unknown methods gracefully** -- return an error, do not crash
5. **Log to stderr** -- stdout is reserved for JSON-RPC communication
6. **Include a `min_version`** if you use features from a specific BlackCat version
7. **Use `secret` type** for config fields that contain API keys or tokens — they will be stored in the encrypted secret store, not in plaintext config

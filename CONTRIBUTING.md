# Contributing to BlackCat

Thank you for your interest in contributing to BlackCat. This guide covers everything you need to get started.

---

## Table of Contents

- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Workflow](#workflow)
- [Code Style](#code-style)
- [Testing](#testing)
- [Commit Messages](#commit-messages)
- [Pull Request Process](#pull-request-process)
- [Project Structure](#project-structure)

---

## Getting Started

1. **Fork** the repository on GitHub.
2. **Clone** your fork locally:
   ```bash
   git clone https://github.com/<your-username>/blackcat.git
   cd blackcat
   ```
3. **Add the upstream remote:**
   ```bash
   git remote add upstream https://github.com/Meow-AIs/BlackCat.git
   ```
4. **Create a feature branch:**
   ```bash
   git checkout -b feat/my-feature main
   ```

---

## Development Setup

### Prerequisites

- **Go 1.25+** — [install](https://go.dev/dl/)
- **CGo enabled** — required for SQLite, sqlite-vec, and ONNX runtime
- **Make** — for build targets
- **zig** (optional) — for cross-compilation (`make build-all`)

### Build and Verify

```bash
make build          # Build the binary
make test           # Run unit tests
make lint           # Run linter + vet + fmt check
```

---

## Workflow

We follow a test-driven development (TDD) approach:

1. **Write a failing test** (RED) — define the expected behavior before writing implementation code.
2. **Implement the minimum code** (GREEN) — make the test pass with the simplest correct solution.
3. **Refactor** (IMPROVE) — clean up while keeping all tests green.
4. **Verify coverage** — aim for 80%+ on new code.

```bash
# Run tests with coverage
go test ./internal/your-package/... -cover -v

# Generate a coverage report
go test ./... -coverprofile=cover.out
go tool cover -html=cover.out
```

---

## Code Style

### Immutability

Always create new objects instead of mutating existing ones. This prevents hidden side effects and simplifies debugging.

```go
// Correct: return a new struct with the updated field
func WithTimeout(cfg Config, timeout time.Duration) Config {
    return Config{
        Provider: cfg.Provider,
        Model:    cfg.Model,
        Timeout:  timeout,
    }
}

// Incorrect: mutating the original
func SetTimeout(cfg *Config, timeout time.Duration) {
    cfg.Timeout = timeout
}
```

### File Size Limits

| Guideline | Limit |
|-----------|-------|
| Typical file | 200-400 lines |
| Maximum file | 800 lines |
| Function length | < 50 lines |
| Nesting depth | < 4 levels |

If a file grows beyond 400 lines, consider extracting a focused sub-package or utility.

### Naming

- Use clear, descriptive names. Avoid abbreviations except for well-known ones (`ctx`, `err`, `cfg`).
- Exported functions and types must have doc comments.
- Package names should be short and lowercase with no underscores.

### Error Handling

- Handle errors explicitly at every level.
- Wrap errors with context using `fmt.Errorf("operation: %w", err)`.
- Never silently discard errors.
- Provide user-friendly messages in CLI-facing code and detailed context in internal code.

### Input Validation

- Validate all inputs at system boundaries (CLI args, config files, API responses, tool output).
- Fail fast with clear error messages.

---

## Testing

### Requirements

- All new features must include unit tests.
- All bug fixes must include a regression test.
- Integration tests go in files named `*_integration_test.go` with the `//go:build integration` tag.
- Target 80%+ coverage on new code.

### Running Tests

```bash
make test                # Unit tests
make test-integration    # Integration tests (requires external services)
make test-e2e            # End-to-end tests
make bench               # Benchmarks

# Specific package
go test ./internal/memory/... -run TestHybridRetrieval -v
```

### Test Organization

- Place test files alongside the code they test (`foo.go` / `foo_test.go`).
- Use table-driven tests for multiple cases.
- Mock external dependencies (LLM APIs, network calls) — never make real API calls in unit tests.

---

## Commit Messages

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>: <description>

<optional body>
```

### Types

| Type | When to Use |
|------|-------------|
| `feat` | New feature |
| `fix` | Bug fix |
| `refactor` | Code change that neither fixes a bug nor adds a feature |
| `test` | Adding or updating tests |
| `docs` | Documentation only |
| `chore` | Build process, dependencies, tooling |
| `perf` | Performance improvement |
| `ci` | CI/CD configuration |

### Examples

```
feat: add Kimi LLM provider with streaming support

fix: prevent memory leak in vector deduplication

refactor: extract permission gate into dedicated package

test: add integration tests for hybrid retrieval pipeline
```

Keep the subject line under 70 characters. Use the body for additional context when needed.

---

## Pull Request Process

1. **Sync with upstream** before starting work:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Ensure all checks pass** locally:
   ```bash
   make lint
   make test
   ```

3. **Push your branch** and open a PR:
   ```bash
   git push -u origin feat/my-feature
   ```

4. **PR description** should include:
   - A summary of what changed and why
   - How to test it
   - Any breaking changes or migration steps

5. **Review process:**
   - At least one maintainer approval required
   - All CI checks must pass
   - Address review feedback with new commits (do not force-push during review)

6. **After merge:** delete your feature branch.

---

## Project Structure

```
blackcat/
├── cmd/blackcat/         # CLI entry point
├── internal/
│   ├── agent/            # Agent core: reasoning, planning, execution, sub-agents
│   ├── channels/         # Channel adapters: Telegram, Discord, Slack, WhatsApp, Email, Signal
│   ├── config/           # Configuration loading and validation
│   ├── domains/          # Domain modules: DevSecOps, Architect
│   ├── eval/             # Evaluation harness and test suites
│   ├── llm/              # LLM providers, router, caching, cost tracking
│   ├── memory/           # Vector store, hybrid retrieval, embedding, decay
│   ├── remote/           # SSH/kubectl remote access
│   ├── security/         # Permission gate, sandbox, injection scanning
│   └── tools/            # Tool registry, executor, MCP, custom tools
├── pkg/                  # Public library packages
├── embed/                # Embedded assets (ONNX model)
├── configs/              # Default configuration templates
├── docs/                 # Documentation
├── scripts/              # Build and release scripts
├── Makefile              # Build targets
└── go.mod                # Go module definition
```

### Key Interfaces

When adding a new component, implement the relevant interface:

- **LLM Provider** — `internal/llm/provider.go` — methods: `Chat`, `Stream`, `Embed`, `Models`
- **Channel Adapter** — `internal/channels/adapter.go` — unified messaging interface
- **Tool** — `internal/tools/tool.go` — register via `internal/tools/registry.go`
- **Domain Module** — `internal/domains/domain.go` — inject prompts, tools, knowledge

---

## Questions?

Open a [GitHub Discussion](https://github.com/Meow-AIs/BlackCat/discussions) or file an issue. We are happy to help.

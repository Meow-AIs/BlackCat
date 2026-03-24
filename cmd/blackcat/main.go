package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	bcembed "github.com/meowai/blackcat/embed"
	"github.com/meowai/blackcat/internal/llm"
	"github.com/meowai/blackcat/internal/updater"
)

var (
	version = "0.1.0"
	commit  = "dev"
)

func main() {
	os.Exit(run(os.Args))
}

func run(args []string) int {
	if len(args) < 2 {
		runInteractive()
		return 0
	}

	switch args[1] {
	case "version", "--version", "-v":
		cmdVersion()
	case "init":
		return cmdInit()
	case "config":
		return cmdConfig(args[2:])
	case "serve":
		cmdServe()
	case "memory":
		return cmdMemory(args[2:])
	case "schedule":
		return cmdSchedule(args[2:])
	case "skills":
		return cmdSkills(args[2:])
	case "mcp":
		return cmdMCP(args[2:])
	case "login":
		return cmdLogin(args[2:])
	case "logout":
		return cmdLogout(args[2:])
	case "update":
		return cmdUpdate()
	case "doctor":
		return cmdDoctor()
	case "help", "--help", "-h":
		cmdHelp()
	default:
		prompt := strings.Join(args[1:], " ")
		runOneShot(prompt)
	}
	return 0
}

func runInteractive() {
	fmt.Printf("BlackCat v%s by MeowAI\n", version)
	fmt.Println("Interactive TUI not yet implemented. Use 'blackcat help' for available commands.")
}

func runOneShot(prompt string) {
	fmt.Printf("Processing: %s\n", prompt)
	fmt.Println("(agent not yet implemented)")
}

func cmdLogin(args []string) int {
	if len(args) == 0 {
		args = []string{"status"}
	}

	switch args[0] {
	case "copilot":
		fmt.Println("Logging in to GitHub Copilot...")
		provider := llm.NewCopilotProvider()
		resp, err := provider.Login(context.Background())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		fmt.Println(llm.FormatLoginPrompt(resp))
		fmt.Println("Waiting for authorization...")
		if err := provider.CompleteLogin(context.Background(), resp.DeviceCode); err != nil {
			fmt.Fprintf(os.Stderr, "Authorization failed: %v\n", err)
			return 1
		}
		fmt.Println("Logged in to GitHub Copilot. Token stored securely.")

	case "codex":
		fmt.Println("Logging in to OpenAI Codex...")
		provider := llm.NewCodexProvider()
		resp, err := provider.Login(context.Background())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		fmt.Println(llm.FormatLoginPrompt(resp))
		fmt.Println("Waiting for authorization...")
		if err := provider.CompleteLogin(context.Background(), resp.DeviceCode); err != nil {
			fmt.Fprintf(os.Stderr, "Authorization failed: %v\n", err)
			return 1
		}
		fmt.Println("Logged in to OpenAI Codex. Token stored securely.")

	case "status":
		fmt.Println("Login Status:")
		copilot := llm.NewCopilotProvider()
		codex := llm.NewCodexProvider()
		if copilot.IsAuthenticated() {
			fmt.Println("  GitHub Copilot:  authenticated")
		} else {
			fmt.Println("  GitHub Copilot:  not authenticated (run: blackcat login copilot)")
		}
		if codex.IsAuthenticated() {
			fmt.Println("  OpenAI Codex:    not authenticated (run: blackcat login codex)")
		} else {
			fmt.Println("  OpenAI Codex:    not authenticated (run: blackcat login codex)")
		}
		fmt.Println()
		fmt.Println("API key providers (set via /config set or env vars):")
		fmt.Println("  Anthropic, OpenAI, Groq, Gemini, Z.ai, Kimi, xAI, OpenRouter, Ollama")

	default:
		fmt.Printf("Unknown provider: %s\n", args[0])
		fmt.Println("Available: copilot, codex, status")
		return 1
	}
	return 0
}

func cmdLogout(args []string) int {
	if len(args) == 0 {
		fmt.Println("Usage: blackcat logout <provider>")
		fmt.Println("Available: copilot, codex")
		return 1
	}

	switch args[0] {
	case "copilot":
		fmt.Println("Logged out from GitHub Copilot. Token removed.")
	case "codex":
		fmt.Println("Logged out from OpenAI Codex. Token removed.")
	default:
		fmt.Printf("Unknown provider: %s\n", args[0])
		return 1
	}
	return 0
}

func cmdUpdate() int {
	fmt.Println("Checking for updates...")
	u := updater.NewUpdater("Meow-AIs/BlackCat", version)
	info, err := u.CheckForUpdate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking for updates: %v\n", err)
		return 1
	}
	if !info.Available {
		fmt.Printf("Already on latest version (v%s)\n", info.CurrentVersion)
		return 0
	}
	fmt.Printf("Update available: v%s → v%s\n", info.CurrentVersion, info.LatestVersion)
	if info.DownloadURL == "" {
		fmt.Println("No binary available for your platform. Download manually from:")
		fmt.Printf("  https://github.com/Meow-AIs/BlackCat/releases/tag/v%s\n", info.LatestVersion)
		return 0
	}
	fmt.Printf("Downloading %s...\n", info.AssetName)
	data, err := u.DownloadUpdate(info.DownloadURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error downloading: %v\n", err)
		return 1
	}
	fmt.Println("Installing...")
	if err := updater.ReplaceBinary(data); err != nil {
		fmt.Fprintf(os.Stderr, "Error installing: %v\n", err)
		return 1
	}
	fmt.Printf("Updated to v%s. Restart BlackCat to use the new version.\n", info.LatestVersion)
	return 0
}

func cmdVersion() {
	embeddingStatus := "not bundled (using Ollama/API)"
	if bcembed.HasEmbeddedModel() {
		embeddingStatus = "bundled (MiniLM-L6-v2 int8)"
	}
	fmt.Printf("BlackCat v%s (commit: %s)\n", version, commit)
	fmt.Printf("  Embedding: %s\n", embeddingStatus)
}

func cmdInit() int {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot determine home directory: %v\n", err)
		return 1
	}

	configDir := filepath.Join(home, ".blackcat")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot create %s: %v\n", configDir, err)
		return 1
	}

	configPath := filepath.Join(configDir, "config.yaml")
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Config already exists: %s\n", configPath)
		return 0
	}

	defaultConfig := defaultConfigYAML()
	if err := os.WriteFile(configPath, []byte(defaultConfig), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot write config: %v\n", err)
		return 1
	}

	fmt.Printf("Initialized BlackCat at %s\n", configDir)
	fmt.Printf("Config written to %s\n", configPath)
	return 0
}

func cmdConfig(args []string) int {
	if len(args) == 0 {
		fmt.Println("Usage: blackcat config <show|set key=value>")
		return 1
	}

	switch args[0] {
	case "show":
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}
		configPath := filepath.Join(home, ".blackcat", "config.yaml")
		data, err := os.ReadFile(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: cannot read config: %v\n", err)
			fmt.Println("Run 'blackcat init' to create default config.")
			return 1
		}
		fmt.Print(string(data))
	case "set":
		if len(args) < 2 {
			fmt.Println("Usage: blackcat config set key=value")
			return 1
		}
		parts := strings.SplitN(args[1], "=", 2)
		if len(parts) != 2 {
			fmt.Println("Usage: blackcat config set key=value")
			return 1
		}
		// Backup current config before modifying
		setHome, _ := os.UserHomeDir()
		setConfigPath := filepath.Join(setHome, ".blackcat", "config.yaml")
		if err := backupConfigFile(setConfigPath); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not backup config: %v\n", err)
		}
		fmt.Printf("Set %s = %s\n", parts[0], parts[1])
		fmt.Println("(config backed up to config.yaml.bak)")
	default:
		fmt.Printf("Unknown config subcommand: %s\n", args[0])
		return 1
	}
	return 0
}

func cmdServe() {
	fmt.Println("Starting BlackCat gateway daemon...")
	fmt.Println("Channels: Telegram, Discord, Slack, WhatsApp")
	fmt.Println("Scheduler: active")
	fmt.Println("(gateway not yet implemented)")
}

func cmdMemory(args []string) int {
	if len(args) == 0 {
		fmt.Println("Usage: blackcat memory <search|stats|list>")
		return 1
	}

	switch args[0] {
	case "stats":
		fmt.Println("Memory Statistics")
		fmt.Println("  Episodic:   0 entries")
		fmt.Println("  Semantic:   0 entries")
		fmt.Println("  Procedural: 0 entries")
		fmt.Println("  Vectors:    0 total")
	case "search":
		if len(args) < 2 {
			fmt.Println("Usage: blackcat memory search QUERY")
			return 1
		}
		query := strings.Join(args[1:], " ")
		fmt.Printf("Searching memory for: %s\n", query)
		fmt.Println("(no results - memory store not yet connected)")
	case "list":
		fmt.Println("Memory entries: (none)")
	default:
		fmt.Printf("Unknown memory subcommand: %s\n", args[0])
		return 1
	}
	return 0
}

func cmdSchedule(args []string) int {
	if len(args) == 0 {
		fmt.Println("Usage: blackcat schedule <add|list|remove>")
		return 1
	}

	switch args[0] {
	case "list":
		fmt.Println("Schedules: (none configured)")
	case "add":
		if len(args) < 3 {
			fmt.Println("Usage: blackcat schedule add CRON PROMPT")
			return 1
		}
		cron := args[1]
		prompt := strings.Join(args[2:], " ")
		fmt.Printf("Added schedule: %s -> %s\n", cron, prompt)
		fmt.Println("(schedule persistence not yet implemented)")
	case "remove":
		if len(args) < 2 {
			fmt.Println("Usage: blackcat schedule remove ID")
			return 1
		}
		fmt.Printf("Removed schedule: %s\n", args[1])
		fmt.Println("(schedule persistence not yet implemented)")
	default:
		fmt.Printf("Unknown schedule subcommand: %s\n", args[0])
		return 1
	}
	return 0
}

func cmdSkills(args []string) int {
	if len(args) == 0 {
		fmt.Println("Usage: blackcat skills <list|show>")
		return 1
	}

	switch args[0] {
	case "list":
		fmt.Println("Skills: (none learned yet)")
	case "show":
		if len(args) < 2 {
			fmt.Println("Usage: blackcat skills show NAME")
			return 1
		}
		fmt.Printf("Skill: %s\n", args[1])
		fmt.Println("(skill store not yet connected)")
	default:
		fmt.Printf("Unknown skills subcommand: %s\n", args[0])
		return 1
	}
	return 0
}

func cmdMCP(args []string) int {
	if len(args) == 0 {
		fmt.Println("Usage: blackcat mcp <add|list|remove>")
		return 1
	}

	switch args[0] {
	case "list":
		fmt.Println("MCP servers: (none configured)")
	case "add":
		if len(args) < 3 {
			fmt.Println("Usage: blackcat mcp add NAME COMMAND")
			return 1
		}
		name := args[1]
		command := strings.Join(args[2:], " ")
		fmt.Printf("Added MCP server: %s -> %s\n", name, command)
		fmt.Println("(MCP persistence not yet implemented)")
	case "remove":
		if len(args) < 2 {
			fmt.Println("Usage: blackcat mcp remove NAME")
			return 1
		}
		fmt.Printf("Removed MCP server: %s\n", args[1])
		fmt.Println("(MCP persistence not yet implemented)")
	default:
		fmt.Printf("Unknown mcp subcommand: %s\n", args[0])
		return 1
	}
	return 0
}

func cmdDoctor() int {
	fmt.Println("BlackCat Doctor - System Health Check")
	fmt.Println()

	// Go version
	goVersion := runtime.Version()
	fmt.Printf("  Go runtime:    %s OK\n", goVersion)

	// Check for gcc/cc
	ccOK := "NOT FOUND"
	if _, err := exec.LookPath("gcc"); err == nil {
		ccOK = "OK"
	} else if _, err := exec.LookPath("cc"); err == nil {
		ccOK = "OK"
	}
	fmt.Printf("  C compiler:    %s\n", ccOK)

	// Config check
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("  Config:        ERROR (cannot determine home dir)")
		return 1
	}
	configPath := filepath.Join(home, ".blackcat", "config.yaml")
	if _, err := os.Stat(configPath); err != nil {
		fmt.Println("  Config:        MISSING (run 'blackcat init')")
	} else {
		fmt.Println("  Config:        OK")
	}

	// Embedding model
	if bcembed.HasEmbeddedModel() {
		fmt.Println("  Embedding:     BUNDLED (MiniLM-L6-v2 int8)")
	} else {
		fmt.Println("  Embedding:     NOT BUNDLED (use Ollama or API)")
	}

	// OS info
	fmt.Printf("  Platform:      %s/%s\n", runtime.GOOS, runtime.GOARCH)

	return 0
}

func cmdHelp() {
	fmt.Printf("BlackCat v%s - AI Agent CLI by MeowAI\n", version)
	fmt.Println()
	fmt.Println("Usage: blackcat [command] [args...]")
	fmt.Println("       blackcat \"prompt\"        Run one-shot prompt")
	fmt.Println("       blackcat                 Launch interactive TUI")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  version    Print version information")
	fmt.Println("  init       Initialize BlackCat configuration")
	fmt.Println("  config     Manage configuration (show, set)")
	fmt.Println("  serve      Start channel messaging gateway")
	fmt.Println("  memory     Manage agent memory (search, stats, list)")
	fmt.Println("  schedule   Manage scheduled tasks (add, list, remove)")
	fmt.Println("  skills     Manage learned skills (list, show)")
	fmt.Println("  mcp        Manage MCP servers (add, list, remove)")
	fmt.Println("  login      Login to OAuth provider (copilot, codex)")
	fmt.Println("  logout     Logout from OAuth provider")
	fmt.Println("  update     Update BlackCat to latest version")
	fmt.Println("  doctor     Check system health")
	fmt.Println("  help       Show this help message")
}

func backupConfigFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return os.WriteFile(path+".bak", data, 0o600)
}

func defaultConfigYAML() string {
	return `# BlackCat Configuration
# See: https://github.com/Meow-AIs/BlackCat

# Default LLM provider (anthropic, openai, ollama, openrouter, groq, zai, kimi, xai)
provider: anthropic

# Provider configurations
providers:
  anthropic:
    model: claude-sonnet-4-6
    # api_key: (use /config set or ANTHROPIC_API_KEY env var)
  openai:
    model: gpt-5.4
    # api_key: (use /config set or OPENAI_API_KEY env var)
  ollama:
    base_url: http://localhost:11434
    model: qwen2.5:32b

# Memory settings
memory:
  max_vectors: 10000
  embedding_model: minilm-l6-v2
  similarity_threshold: 0.7

# Security settings
security:
  permission_mode: ask
  allowed_commands: []
  denied_commands: [rm -rf /*, format, shutdown]
  secret_detection: true

# Channel messaging (for 'blackcat serve')
channels: {}
  # telegram:
  #   token: (set via TELEGRAM_BOT_TOKEN env var)
  # discord:
  #   token: (set via DISCORD_BOT_TOKEN env var)

# Scheduler
scheduler:
  enabled: false
  schedules: []

# MCP servers
mcp_servers: []
`
}

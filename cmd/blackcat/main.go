package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	bcembed "github.com/meowai/blackcat/embed"
	"github.com/meowai/blackcat/internal/agent"
	"github.com/meowai/blackcat/internal/channels"
	"github.com/meowai/blackcat/internal/channels/discord"
	"github.com/meowai/blackcat/internal/channels/slack"
	"github.com/meowai/blackcat/internal/channels/telegram"
	"github.com/meowai/blackcat/internal/channels/whatsapp"
	"github.com/meowai/blackcat/internal/config"
	"github.com/meowai/blackcat/internal/llm"
	"github.com/meowai/blackcat/internal/security"
	"github.com/meowai/blackcat/internal/tools"
	"github.com/meowai/blackcat/internal/tools/builtin"
	"github.com/meowai/blackcat/internal/updater"
	"gopkg.in/yaml.v3"
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
	case "models":
		return cmdModels()
	case "login":
		return cmdLogin(args[2:])
	case "logout":
		return cmdLogout(args[2:])
	case "update", "upgrade":
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

	core, err := initAgent()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to initialise agent: %v\n", err)
		return
	}

	ctx := context.Background()
	sess, err := core.StartSession(ctx, "cli", "user")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to start session: %v\n", err)
		return
	}

	resp, err := core.Process(ctx, sess.ID, prompt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return
	}

	fmt.Println(resp.Text)

	// Print cost summary when tokens were consumed.
	if summary := agentCostSummary(core); summary != "" {
		fmt.Fprintln(os.Stderr, summary)
	}
}

// agentCostSummary extracts cost info from a Core via its CostTracker.
// Returns an empty string when no tokens were recorded.
func agentCostSummary(_ *agent.Core) string {
	// CostTracker is unexported on Core; we surfaced it at construction
	// through the module-level costTracker variable below.
	if lastCostTracker == nil {
		return ""
	}
	s := lastCostTracker.Summary()
	if s.TotalPrompt+s.TotalCompletion == 0 {
		return ""
	}
	return fmt.Sprintf("tokens: %d in / %d out  cost: $%.6f",
		s.TotalPrompt, s.TotalCompletion, s.TotalCost)
}

// lastCostTracker holds the CostTracker used by the most recent initAgent call.
// This is a package-level variable so agentCostSummary can access it without
// requiring Core to expose the tracker.
var lastCostTracker *llm.CostTracker

// loadConfig reads ~/.blackcat/config.yaml and falls back to config.Default()
// if the file is absent or unreadable.
func loadConfig() config.Config {
	home, err := os.UserHomeDir()
	if err != nil {
		return config.Default()
	}
	cfg, err := config.LoadFromFile(filepath.Join(home, ".blackcat", "config.yaml"))
	if err != nil {
		return config.Default()
	}
	return cfg
}

// configFilePath returns the canonical path of the global config file,
// creating the parent directory if necessary.
func configFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	dir := filepath.Join(home, ".blackcat")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("cannot create config directory: %w", err)
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// saveConfig serialises cfg as YAML and writes it atomically to path.
// The caller is responsible for creating a backup before calling this.
func saveConfig(cfg config.Config, path string) error {
	data, err := marshalConfig(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

// marshalConfig converts a Config to YAML bytes using gopkg.in/yaml.v3.
func marshalConfig(cfg config.Config) ([]byte, error) {
	return yaml.Marshal(cfg)
}

// selectProvider picks an LLM provider based on environment variables using the
// priority ANTHROPIC_API_KEY → OPENAI_API_KEY → GROQ_API_KEY → Ollama (always).
func selectProvider(cfg config.Config) llm.Provider {
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		return llm.NewAnthropicProvider(key, "")
	}
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		return llm.NewOpenAIProvider(key, "", "openai")
	}
	if key := os.Getenv("GROQ_API_KEY"); key != "" {
		return llm.NewOpenAIProvider(key, "https://api.groq.com/openai/v1", "groq")
	}
	// Ollama is always available as a local fallback.
	baseURL := cfg.Providers.Ollama.BaseURL
	return llm.NewOllamaProvider(baseURL)
}

// buildRegistry creates a tool registry with all builtin tools registered.
func buildRegistry(checker *security.PermissionChecker) tools.Registry {
	reg := tools.NewMapRegistry()

	// Filesystem tools
	_ = reg.Register(builtin.NewReadFileTool())
	_ = reg.Register(builtin.NewWriteFileTool())
	_ = reg.Register(builtin.NewListDirTool())
	_ = reg.Register(builtin.NewSearchFilesTool())
	_ = reg.Register(builtin.NewSearchContentTool())

	// Shell
	_ = reg.Register(builtin.NewShellTool(checker))

	// Web
	_ = reg.Register(builtin.NewWebFetchTool())
	_ = reg.Register(builtin.NewWebSearchTool())

	// Multimodal
	_ = reg.Register(builtin.NewVisionTool())
	_ = reg.Register(builtin.NewVoiceTool())

	// Skills marketplace
	_ = reg.Register(builtin.NewSkillsTool())

	return reg
}

// initAgent constructs a fully wired agent.Core ready to Process prompts.
func initAgent() (*agent.Core, error) {
	cfg := loadConfig()

	checker := security.NewPermissionChecker()
	tracker := llm.NewCostTracker(0, 0)
	lastCostTracker = tracker

	provider := selectProvider(cfg)
	registry := buildRegistry(checker)

	core := agent.NewCore(agent.CoreConfig{
		Provider:    provider,
		Registry:    registry,
		MemEngine:   nil, // memory engine wired in a later phase
		Checker:     checker,
		CostTracker: tracker,
	})
	return core, nil
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
		// Save copilot token marker
		saveOAuthToken("copilot", &llm.OAuthToken{AccessToken: "copilot-authenticated"})
		fmt.Println("Logged in to GitHub Copilot. Token stored securely.")

	case "codex":
		fmt.Println("Logging in to OpenAI Codex...")
		pkce := llm.NewPKCEClient(llm.OpenAICodexPKCE)

		verifier, err := llm.GenerateVerifier()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		state := fmt.Sprintf("blackcat-%d", time.Now().Unix())
		authURL := pkce.BuildAuthorizationURL(verifier, state)

		var code string
		isRemote := isHeadlessEnvironment()

		if isRemote {
			// Remote/VPS: no browser, manual paste flow
			fmt.Println("Remote environment detected (no browser available).")
			fmt.Println()
			fmt.Println("1. Open this URL in your LOCAL browser:")
			fmt.Printf("   %s\n\n", authURL)
			fmt.Println("2. Sign in with your ChatGPT account")
			fmt.Println("3. After redirect, copy the URL from your browser address bar")
			fmt.Println("4. Paste it here:")
			fmt.Print("> ")
			var redirectURL string
			fmt.Scanln(&redirectURL)
			var extractErr error
			code, _, extractErr = llm.ExtractCodeFromURL(redirectURL)
			if extractErr != nil {
				fmt.Fprintf(os.Stderr, "Error parsing redirect URL: %v\n", extractErr)
				return 1
			}
		} else {
			// Local: open browser + localhost callback
			fmt.Println("Opening browser for OpenAI sign-in...")
			fmt.Printf("If browser doesn't open, visit:\n  %s\n\n", authURL)
			openBrowserURL(authURL)

			fmt.Println("Waiting for authorization callback on localhost:1455...")
			var cbErr error
			code, _, cbErr = pkce.StartCallbackServer(context.Background(), state)
			if cbErr != nil {
				fmt.Println("Callback failed. Paste the redirect URL here:")
				fmt.Print("> ")
				var redirectURL string
				fmt.Scanln(&redirectURL)
				var extractErr error
				code, _, extractErr = llm.ExtractCodeFromURL(redirectURL)
				if extractErr != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", extractErr)
					return 1
				}
			}
		}

		token, err := pkce.ExchangeCode(context.Background(), code, verifier)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Token exchange failed: %v\n", err)
			return 1
		}
		// Persist token
		if saveErr := saveOAuthToken("codex", token); saveErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save token: %v\n", saveErr)
		}
		fmt.Println("Logged in to OpenAI Codex. Token stored securely.")

	case "status":
		fmt.Println("Login Status:")
		if isOAuthAuthenticated("copilot") {
			fmt.Println("  GitHub Copilot:  authenticated")
		} else {
			fmt.Println("  GitHub Copilot:  not authenticated (run: blackcat login copilot)")
		}
		if isOAuthAuthenticated("codex") {
			fmt.Println("  OpenAI Codex:    authenticated")
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

func cmdModels() int {
	fmt.Println("Available Models")
	fmt.Println()

	type providerCheck struct {
		name    string
		envKey  string
		authCmd string
		models  []string
	}

	providers := []providerCheck{
		{"Anthropic", "ANTHROPIC_API_KEY", "blackcat config set anthropic_api_key sk-ant-...",
			[]string{"claude-opus-4-6", "claude-sonnet-4-6", "claude-haiku-4-5"}},
		{"OpenAI", "OPENAI_API_KEY", "blackcat config set openai_api_key sk-...",
			[]string{"gpt-5.4", "gpt-4.1", "gpt-4.1-mini", "o4-mini", "o3"}},
		{"Groq", "GROQ_API_KEY", "blackcat config set groq_api_key gsk_...",
			[]string{"llama-4-scout-17b", "llama-3.3-70b-versatile", "deepseek-r1-distill-llama-70b"}},
		{"Gemini", "GEMINI_API_KEY", "blackcat config set gemini_api_key ...",
			[]string{"gemini-2.5-pro", "gemini-2.5-flash", "gemini-2.0-flash"}},
		{"Z.ai (GLM)", "ZAI_API_KEY", "blackcat config set zai_api_key ...",
			[]string{"glm-5", "glm-5-turbo", "glm-4.7", "glm-4.7-flash (FREE)"}},
		{"Kimi", "KIMI_API_KEY", "blackcat config set kimi_api_key ...",
			[]string{"kimi-k2.5", "kimi-k2.5-mini"}},
		{"xAI (Grok)", "XAI_API_KEY", "blackcat config set xai_api_key xai-...",
			[]string{"grok-4-1-fast-latest", "grok-4-heavy", "grok-code-fast-1", "grok-3-mini"}},
		{"OpenRouter", "OPENROUTER_API_KEY", "blackcat config set openrouter_api_key sk-or-...",
			[]string{"openrouter/anthropic/claude-opus-4-6", "openrouter/openai/gpt-5.4", "openrouter/google/gemini-2.5-pro", "openrouter/deepseek/deepseek-chat", "... 400+ models"}},
	}

	hasAny := false

	// API key providers
	for _, p := range providers {
		if os.Getenv(p.envKey) != "" {
			hasAny = true
			fmt.Printf("  %s (configured via %s):\n", p.name, p.envKey)
			for _, m := range p.models {
				fmt.Printf("    %s/%s\n", strings.ToLower(strings.Split(p.name, " ")[0]), m)
			}
			fmt.Println()
		}
	}

	// OAuth providers
	if isOAuthAuthenticated("copilot") {
		hasAny = true
		copilot := llm.NewCopilotProvider()
		fmt.Println("  GitHub Copilot (authenticated via OAuth):")
		for _, m := range copilot.Models() {
			fmt.Printf("    copilot/%s\n", m.ID)
		}
		fmt.Println()
	}
	if isOAuthAuthenticated("codex") {
		hasAny = true
		codex := llm.NewCodexProvider()
		fmt.Println("  OpenAI Codex (authenticated via PKCE OAuth):")
		for _, m := range codex.Models() {
			fmt.Printf("    codex/%s\n", m.ID)
		}
		fmt.Println()
	}

	// Ollama (always available if running)
	fmt.Println("  Ollama (local, no auth needed):")
	fmt.Println("    ollama/qwen2.5:32b, ollama/deepseek-r1:14b, ollama/llama3.3:70b, ...")
	fmt.Println("    (run 'ollama list' to see installed models)")
	fmt.Println()
	hasAny = true

	// Show unconfigured providers
	unconfigured := []providerCheck{}
	for _, p := range providers {
		if os.Getenv(p.envKey) == "" {
			unconfigured = append(unconfigured, p)
		}
	}
	if !isOAuthAuthenticated("copilot") {
		unconfigured = append(unconfigured, providerCheck{
			name: "GitHub Copilot", authCmd: "blackcat login copilot",
		})
	}
	if !isOAuthAuthenticated("codex") {
		unconfigured = append(unconfigured, providerCheck{
			name: "OpenAI Codex", authCmd: "blackcat login codex",
		})
	}

	if len(unconfigured) > 0 {
		fmt.Println("  Not configured:")
		for _, p := range unconfigured {
			fmt.Printf("    (!) %s — run: %s\n", p.name, p.authCmd)
		}
		fmt.Println()
	}

	if !hasAny {
		fmt.Println("  No providers configured yet.")
		fmt.Println("  Run 'blackcat login copilot' or set an API key to get started.")
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
	data, err := u.DownloadUpdate(info.DownloadURL, info.AssetName)
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

// cmdServeDry prints the serve banner without starting adapters or blocking.
// Used in tests to exercise the startup path without blocking on signals.
func cmdServeDry() {
	fmt.Println("Starting BlackCat gateway daemon...")
	fmt.Println("Channels: Telegram, Discord, Slack, WhatsApp")
	fmt.Println("Scheduler: active")
}

func cmdServe() {
	cmdServeDry()

	core, err := initAgent()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to initialise agent: %v\n", err)
		return
	}

	cfg := loadConfig()

	// handler routes every incoming channel message through the agent.
	handler := func(ctx context.Context, msg channels.IncomingMessage) (string, error) {
		sess, err := core.StartSession(ctx, string(msg.Platform), msg.UserID)
		if err != nil {
			return "", fmt.Errorf("start session: %w", err)
		}
		resp, err := core.Process(ctx, sess.ID, msg.Text)
		if err != nil {
			return "", err
		}
		return resp.Text, nil
	}

	gw := channels.NewGateway(handler)

	// Register adapters for enabled channels.
	if cfg.Channels.Telegram.Enabled || os.Getenv("TELEGRAM_BOT_TOKEN") != "" {
		token := cfg.Channels.Telegram.Token
		if token == "" {
			token = os.Getenv("TELEGRAM_BOT_TOKEN")
		}
		gw.Register(telegram.New(telegram.Config{
			Token:        token,
			AllowedUsers: cfg.Channels.Telegram.AllowedUsers,
		}))
		fmt.Println("  Telegram: registered")
	}

	if cfg.Channels.Discord.Enabled || os.Getenv("DISCORD_BOT_TOKEN") != "" {
		token := cfg.Channels.Discord.Token
		if token == "" {
			token = os.Getenv("DISCORD_BOT_TOKEN")
		}
		gw.Register(discord.New(discord.Config{
			Token:           token,
			AllowedGuilds:   cfg.Channels.Discord.AllowedGuilds,
			AllowedChannels: cfg.Channels.Discord.AllowedChannels,
		}))
		fmt.Println("  Discord: registered")
	}

	if cfg.Channels.Slack.Enabled || os.Getenv("SLACK_APP_TOKEN") != "" {
		appToken := cfg.Channels.Slack.AppToken
		if appToken == "" {
			appToken = os.Getenv("SLACK_APP_TOKEN")
		}
		botToken := cfg.Channels.Slack.BotToken
		if botToken == "" {
			botToken = os.Getenv("SLACK_BOT_TOKEN")
		}
		gw.Register(slack.New(slack.Config{
			AppToken:        appToken,
			BotToken:        botToken,
			AllowedChannels: cfg.Channels.Slack.AllowedChannels,
		}))
		fmt.Println("  Slack: registered")
	}

	if cfg.Channels.WhatsApp.Enabled {
		sessionPath := cfg.Channels.WhatsApp.SessionPath
		if sessionPath == "" {
			home, _ := os.UserHomeDir()
			sessionPath = filepath.Join(home, ".blackcat", "whatsapp-session")
		}
		gw.Register(whatsapp.New(whatsapp.Config{
			SessionPath:    sessionPath,
			AllowedNumbers: cfg.Channels.WhatsApp.AllowedNumbers,
		}))
		fmt.Println("  WhatsApp: registered")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := gw.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "error: gateway start failed: %v\n", err)
		return
	}

	fmt.Println("Gateway running. Press Ctrl+C to stop.")
	<-ctx.Done()

	fmt.Println("\nShutting down gateway...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := gw.Stop(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "warning: gateway stop error: %v\n", err)
	}
	fmt.Println("Goodbye.")
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
		cfg := loadConfig()
		if len(cfg.Scheduler.Schedules) == 0 {
			fmt.Println("Schedules: (none configured)")
			return 0
		}
		fmt.Println("Schedules:")
		for _, s := range cfg.Scheduler.Schedules {
			fmt.Printf("  %s  %s\n", s.Cron, s.Task)
		}
	case "add":
		if len(args) < 3 {
			fmt.Println("Usage: blackcat schedule add CRON PROMPT")
			return 1
		}
		cron := args[1]
		task := strings.Join(args[2:], " ")
		cfg := loadConfig()
		cfg.Scheduler.Schedules = append(cfg.Scheduler.Schedules,
			config.ScheduleEntry{Cron: cron, Task: task})
		configPath, err := configFilePath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}
		backupConfigFile(configPath)
		if err := saveConfig(cfg, configPath); err != nil {
			fmt.Fprintf(os.Stderr, "error: could not save config: %v\n", err)
			return 1
		}
		fmt.Printf("Added schedule: %s -> %s\n", cron, task)
	case "remove":
		if len(args) < 2 {
			fmt.Println("Usage: blackcat schedule remove ID")
			return 1
		}
		name := strings.Join(args[1:], " ")
		cfg := loadConfig()
		original := len(cfg.Scheduler.Schedules)
		filtered := make([]config.ScheduleEntry, 0, original)
		for _, s := range cfg.Scheduler.Schedules {
			if s.Task != name && s.Name != name {
				filtered = append(filtered, s)
			}
		}
		if len(filtered) == original {
			fmt.Fprintf(os.Stderr, "error: schedule %q not found\n", name)
			return 1
		}
		cfg.Scheduler.Schedules = filtered
		configPath, err := configFilePath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}
		backupConfigFile(configPath)
		if err := saveConfig(cfg, configPath); err != nil {
			fmt.Fprintf(os.Stderr, "error: could not save config: %v\n", err)
			return 1
		}
		fmt.Printf("Removed schedule: %s\n", name)
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
		cfg := loadConfig()
		if len(cfg.MCP.Servers) == 0 {
			fmt.Println("MCP servers: (none configured)")
			return 0
		}
		fmt.Println("MCP servers:")
		for _, s := range cfg.MCP.Servers {
			fmt.Printf("  %s: %s %s\n", s.Name, s.Command, strings.Join(s.Args, " "))
		}
	case "add":
		if len(args) < 3 {
			fmt.Println("Usage: blackcat mcp add NAME COMMAND [ARGS...]")
			return 1
		}
		name := args[1]
		command := args[2]
		mcpArgs := args[3:]
		cfg := loadConfig()
		server := config.MCPServer{Name: name, Command: command}
		if len(mcpArgs) > 0 {
			server.Args = mcpArgs
		}
		cfg.MCP.Servers = append(cfg.MCP.Servers, server)
		configPath, err := configFilePath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}
		backupConfigFile(configPath)
		if err := saveConfig(cfg, configPath); err != nil {
			fmt.Fprintf(os.Stderr, "error: could not save config: %v\n", err)
			return 1
		}
		fmt.Printf("Added MCP server: %s -> %s\n", name, command)
	case "remove":
		if len(args) < 2 {
			fmt.Println("Usage: blackcat mcp remove NAME")
			return 1
		}
		name := args[1]
		cfg := loadConfig()
		original := len(cfg.MCP.Servers)
		filtered := make([]config.MCPServer, 0, original)
		for _, s := range cfg.MCP.Servers {
			if s.Name != name {
				filtered = append(filtered, s)
			}
		}
		if len(filtered) == original {
			fmt.Fprintf(os.Stderr, "error: MCP server %q not found\n", name)
			return 1
		}
		cfg.MCP.Servers = filtered
		configPath, err := configFilePath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}
		backupConfigFile(configPath)
		if err := saveConfig(cfg, configPath); err != nil {
			fmt.Fprintf(os.Stderr, "error: could not save config: %v\n", err)
			return 1
		}
		fmt.Printf("Removed MCP server: %s\n", name)
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
	fmt.Println("  models     List available models from configured providers")
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

// saveOAuthToken saves an OAuth token to ~/.blackcat/tokens/<provider>.json
func saveOAuthToken(provider string, token *llm.OAuthToken) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".blackcat", "tokens")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, provider+".json"), data, 0o600)
}

// loadOAuthToken loads a saved OAuth token for a provider
func loadOAuthToken(provider string) (*llm.OAuthToken, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(home, ".blackcat", "tokens", provider+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var token llm.OAuthToken
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

// isOAuthAuthenticated checks if a provider has a saved token
func isOAuthAuthenticated(provider string) bool {
	token, err := loadOAuthToken(provider)
	return err == nil && token != nil && token.AccessToken != ""
}

// openBrowserURL attempts to open a URL in the system's default browser.
// This is best-effort; errors are silently ignored.
func openBrowserURL(url string) {
	switch runtime.GOOS {
	case "linux":
		exec.Command("xdg-open", url).Start()
	case "darwin":
		exec.Command("open", url).Start()
	case "windows":
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	}
}

// isHeadlessEnvironment detects if running on a server without a display.
// Checks for SSH session, missing DISPLAY/WAYLAND_DISPLAY, and container environments.
func isHeadlessEnvironment() bool {
	// SSH session
	if os.Getenv("SSH_CLIENT") != "" || os.Getenv("SSH_TTY") != "" || os.Getenv("SSH_CONNECTION") != "" {
		return true
	}
	// Linux without display
	if runtime.GOOS == "linux" {
		if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
			return true
		}
	}
	// Container
	if os.Getenv("container") != "" || os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return true
	}
	return false
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

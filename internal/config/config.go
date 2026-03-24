package config

import "github.com/meowai/blackcat/internal/security"

// Config is the root configuration for BlackCat.
type Config struct {
	Model       string           `yaml:"model"       json:"model"`
	Memory      MemoryConfig     `yaml:"memory"      json:"memory"`
	Permissions PermissionConfig `yaml:"permissions" json:"permissions"`
	MCP         MCPConfig        `yaml:"mcp"         json:"mcp"`
	Channels    ChannelsConfig   `yaml:"channels"    json:"channels"`
	Scheduler   SchedulerConfig  `yaml:"scheduler"   json:"scheduler"`
	Agent       AgentConfig      `yaml:"agent"       json:"agent"`
	Providers   ProvidersConfig  `yaml:"providers"   json:"providers"`
}

// MemoryConfig controls the memory engine.
type MemoryConfig struct {
	Enabled       bool   `yaml:"enabled"        json:"enabled"`
	MaxVectors    int    `yaml:"max_vectors"     json:"max_vectors"`
	Embedding     string `yaml:"embedding"       json:"embedding"` // "local" | "openai" | "ollama"
	RetentionDays int    `yaml:"retention_days"  json:"retention_days"`
	DBPath        string `yaml:"db_path"         json:"db_path"`
}

// PermissionConfig holds permission rules.
type PermissionConfig struct {
	Allow       []security.PermissionRule `yaml:"allow"        json:"allow"`
	AutoApprove []security.PermissionRule `yaml:"auto_approve" json:"auto_approve"`
	Ask         []security.PermissionRule `yaml:"ask"          json:"ask"`
	Deny        []security.PermissionRule `yaml:"deny"         json:"deny"`
}

// MCPConfig holds MCP server definitions.
type MCPConfig struct {
	Servers []MCPServer `yaml:"servers" json:"servers"`
}

// MCPServer defines an MCP server connection.
type MCPServer struct {
	Name    string            `yaml:"name"    json:"name"`
	Command string            `yaml:"command" json:"command"`
	Args    []string          `yaml:"args"    json:"args,omitempty"`
	Env     map[string]string `yaml:"env"     json:"env,omitempty"`
}

// ChannelsConfig holds messaging channel settings.
type ChannelsConfig struct {
	Telegram TelegramConfig `yaml:"telegram" json:"telegram"`
	Discord  DiscordConfig  `yaml:"discord"  json:"discord"`
	Slack    SlackConfig    `yaml:"slack"    json:"slack"`
	WhatsApp WhatsAppConfig `yaml:"whatsapp" json:"whatsapp"`
}

// TelegramConfig for Telegram bot.
type TelegramConfig struct {
	Enabled      bool    `yaml:"enabled"       json:"enabled"`
	Token        string  `yaml:"token"         json:"token"`
	AllowedUsers []int64 `yaml:"allowed_users" json:"allowed_users"`
	Mode         string  `yaml:"mode"          json:"mode"` // "private" | "group"
}

// DiscordConfig for Discord bot.
type DiscordConfig struct {
	Enabled         bool     `yaml:"enabled"          json:"enabled"`
	Token           string   `yaml:"token"            json:"token"`
	AllowedGuilds   []string `yaml:"allowed_guilds"   json:"allowed_guilds"`
	AllowedChannels []string `yaml:"allowed_channels" json:"allowed_channels"`
}

// SlackConfig for Slack app.
type SlackConfig struct {
	Enabled         bool     `yaml:"enabled"          json:"enabled"`
	AppToken        string   `yaml:"app_token"        json:"app_token"`
	BotToken        string   `yaml:"bot_token"        json:"bot_token"`
	AllowedChannels []string `yaml:"allowed_channels" json:"allowed_channels"`
}

// WhatsAppConfig for WhatsApp via Baileys.
type WhatsAppConfig struct {
	Enabled        bool     `yaml:"enabled"         json:"enabled"`
	SessionPath    string   `yaml:"session_path"    json:"session_path"`
	AllowedNumbers []string `yaml:"allowed_numbers" json:"allowed_numbers"`
}

// SchedulerConfig controls the built-in scheduler.
type SchedulerConfig struct {
	Enabled   bool            `yaml:"enabled"   json:"enabled"`
	Schedules []ScheduleEntry `yaml:"schedules" json:"schedules"`
}

// ScheduleEntry is a config-defined schedule.
type ScheduleEntry struct {
	Name    string `yaml:"name"    json:"name"`
	Cron    string `yaml:"cron"    json:"cron"`
	Task    string `yaml:"task"    json:"task"`
	Channel string `yaml:"channel" json:"channel"`
}

// AgentConfig controls agent behavior.
type AgentConfig struct {
	MaxSubAgents    int    `yaml:"max_sub_agents"    json:"max_sub_agents"`
	SubAgentModel   string `yaml:"sub_agent_model"   json:"sub_agent_model"`
	SubAgentTimeout string `yaml:"sub_agent_timeout" json:"sub_agent_timeout"`
}

// ProvidersConfig holds LLM provider API keys.
type ProvidersConfig struct {
	Anthropic  ProviderEntry `yaml:"anthropic"  json:"anthropic"`
	OpenAI     ProviderEntry `yaml:"openai"     json:"openai"`
	OpenRouter ProviderEntry `yaml:"openrouter" json:"openrouter"`
	Ollama     OllamaEntry   `yaml:"ollama"     json:"ollama"`
}

// ProviderEntry is a single provider's config.
type ProviderEntry struct {
	APIKey string   `yaml:"api_key" json:"api_key"`
	Models []string `yaml:"models"  json:"models,omitempty"`
}

// OllamaEntry is the Ollama-specific config.
type OllamaEntry struct {
	BaseURL string   `yaml:"base_url" json:"base_url"`
	Models  []string `yaml:"models"   json:"models,omitempty"`
}

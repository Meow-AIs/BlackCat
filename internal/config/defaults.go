package config

// Default returns a Config with sensible defaults for BlackCat.
func Default() Config {
	return Config{
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
}

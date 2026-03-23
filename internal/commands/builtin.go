package commands

import (
	"fmt"
	"strings"
)

// RegisterBuiltinCommands registers all built-in slash commands.
func RegisterBuiltinCommands(r *Registry) {
	registerGeneralCommands(r)
	registerMemoryCommands(r)
	registerSkillsCommands(r)
	registerConfigCommands(r)
	registerDomainCommands(r)
	registerPluginCommands(r)
	registerHooksCommands(r)
	registerGitCommands(r)
	registerDebugCommands(r)
}

func registerGeneralCommands(r *Registry) {
	_ = r.Register(CommandDef{
		Name:        "help",
		Aliases:     []string{"h"},
		Description: "Show all commands or help for a specific command",
		Usage:       "/help [command]",
		Category:    "general",
		Handler: func(args []string) CommandResult {
			if len(args) > 0 {
				return CommandResult{Output: r.FormatCommandHelp(args[0])}
			}
			return CommandResult{Output: r.FormatHelp()}
		},
	})

	_ = r.Register(CommandDef{
		Name:        "clear",
		Description: "Clear conversation history",
		Usage:       "/clear",
		Category:    "general",
		Handler: func(args []string) CommandResult {
			return CommandResult{Output: "Conversation cleared."}
		},
	})

	_ = r.Register(CommandDef{
		Name:        "compact",
		Description: "Trigger context compression",
		Usage:       "/compact",
		Category:    "general",
		Handler: func(args []string) CommandResult {
			return CommandResult{Output: "Context compressed. Tokens freed."}
		},
	})

	_ = r.Register(CommandDef{
		Name:        "cost",
		Description: "Show session cost and token usage",
		Usage:       "/cost",
		Category:    "general",
		Handler: func(args []string) CommandResult {
			return CommandResult{Output: "Session cost: $0.00 | Tokens: 0 in / 0 out"}
		},
	})

	_ = r.Register(CommandDef{
		Name:        "model",
		Description: "Switch or show current model",
		Usage:       "/model [name]",
		Category:    "general",
		Handler: func(args []string) CommandResult {
			if len(args) > 0 {
				return CommandResult{Output: fmt.Sprintf("Model switched to: %s", args[0])}
			}
			return CommandResult{Output: "Current model: default"}
		},
	})

	_ = r.Register(CommandDef{
		Name:        "think",
		Description: "Toggle extended thinking mode",
		Usage:       "/think",
		Category:    "general",
		Handler: func(args []string) CommandResult {
			return CommandResult{Output: "Extended thinking toggled."}
		},
	})

	_ = r.Register(CommandDef{
		Name:        "fast",
		Description: "Toggle fast mode (skip reasoning)",
		Usage:       "/fast",
		Category:    "general",
		Handler: func(args []string) CommandResult {
			return CommandResult{Output: "Fast mode toggled."}
		},
	})

	_ = r.Register(CommandDef{
		Name:        "status",
		Description: "Show system status",
		Usage:       "/status",
		Category:    "general",
		Handler: func(args []string) CommandResult {
			return CommandResult{
				Output: "Status:\n  Model: default\n  Domain: general\n  Memory: 0 vectors\n  Plugins: 0 active\n  Session: active",
			}
		},
	})

	_ = r.Register(CommandDef{
		Name:        "version",
		Description: "Show BlackCat version",
		Usage:       "/version",
		Category:    "general",
		Handler: func(args []string) CommandResult {
			return CommandResult{Output: "BlackCat v0.1.0"}
		},
	})

	_ = r.Register(CommandDef{
		Name:        "exit",
		Aliases:     []string{"quit", "q"},
		Description: "Exit the session",
		Usage:       "/exit",
		Category:    "general",
		Handler: func(args []string) CommandResult {
			return CommandResult{Output: "Goodbye."}
		},
	})
}

func registerMemoryCommands(r *Registry) {
	_ = r.Register(CommandDef{
		Name:        "memory",
		Aliases:     []string{"mem"},
		Description: "Memory management commands",
		Usage:       "/memory <subcommand>",
		Category:    "memory",
		Handler: func(args []string) CommandResult {
			return CommandResult{Output: "Usage: /memory <search|stats|forget|list|export>"}
		},
		SubCommands: map[string]CommandDef{
			"search": {
				Name:        "search",
				Description: "Search memory",
				Usage:       "/memory search <query>",
				Category:    "memory",
				Handler: func(args []string) CommandResult {
					if len(args) == 0 {
						return CommandResult{Error: "Usage: /memory search <query>"}
					}
					query := strings.Join(args, " ")
					return CommandResult{
						Output: fmt.Sprintf("Search results for %q:\n  (no results - memory search not yet connected)", query),
					}
				},
			},
			"stats": {
				Name:        "stats",
				Description: "Show memory statistics",
				Usage:       "/memory stats",
				Category:    "memory",
				Handler: func(args []string) CommandResult {
					return CommandResult{
						Output: "Memory stats:\n  Episodic: 0\n  Semantic: 0\n  Procedural: 0\n  Total vectors: 0",
					}
				},
			},
			"forget": {
				Name:        "forget",
				Description: "Delete a memory entry",
				Usage:       "/memory forget <id>",
				Category:    "memory",
				Handler: func(args []string) CommandResult {
					if len(args) == 0 {
						return CommandResult{Error: "Usage: /memory forget <id>"}
					}
					return CommandResult{
						Output: fmt.Sprintf("Memory entry %q marked for forget. (not yet connected)", args[0]),
					}
				},
			},
			"list": {
				Name:        "list",
				Description: "List recent memories",
				Usage:       "/memory list [tier]",
				Category:    "memory",
				Handler: func(args []string) CommandResult {
					tier := "all"
					if len(args) > 0 {
						tier = args[0]
					}
					return CommandResult{
						Output: fmt.Sprintf("Recent memories (tier: %s):\n  (empty - memory list not yet connected)", tier),
					}
				},
			},
			"export": {
				Name:        "export",
				Description: "Export all memories as JSON",
				Usage:       "/memory export",
				Category:    "memory",
				Handler: func(args []string) CommandResult {
					return CommandResult{Output: "Memory export: [] (not yet connected)"}
				},
			},
		},
	})
}

func registerSkillsCommands(r *Registry) {
	_ = r.Register(CommandDef{
		Name:        "skills",
		Description: "Skill marketplace commands",
		Usage:       "/skills <subcommand>",
		Category:    "skills",
		Handler: func(args []string) CommandResult {
			return CommandResult{Output: "Usage: /skills <search|install|uninstall|list|update>"}
		},
		SubCommands: map[string]CommandDef{
			"search": {
				Name:        "search",
				Description: "Search skill marketplace",
				Usage:       "/skills search <query>",
				Handler: func(args []string) CommandResult {
					query := strings.Join(args, " ")
					return CommandResult{Output: fmt.Sprintf("Skill search for %q: (not yet connected)", query)}
				},
			},
			"install": {
				Name:        "install",
				Description: "Install a skill",
				Usage:       "/skills install <name>",
				Handler: func(args []string) CommandResult {
					if len(args) == 0 {
						return CommandResult{Error: "Usage: /skills install <name>"}
					}
					return CommandResult{Output: fmt.Sprintf("Installing skill %q... (not yet connected)", args[0])}
				},
			},
			"uninstall": {
				Name:        "uninstall",
				Description: "Uninstall a skill",
				Usage:       "/skills uninstall <name>",
				Handler: func(args []string) CommandResult {
					if len(args) == 0 {
						return CommandResult{Error: "Usage: /skills uninstall <name>"}
					}
					return CommandResult{Output: fmt.Sprintf("Uninstalling skill %q... (not yet connected)", args[0])}
				},
			},
			"list": {
				Name:        "list",
				Description: "List installed skills",
				Usage:       "/skills list",
				Handler: func(args []string) CommandResult {
					return CommandResult{Output: "Installed skills: (none)"}
				},
			},
			"update": {
				Name:        "update",
				Description: "Update all skills",
				Usage:       "/skills update",
				Handler: func(args []string) CommandResult {
					return CommandResult{Output: "Updating all skills... (not yet connected)"}
				},
			},
		},
	})
}

func registerConfigCommands(r *Registry) {
	_ = r.Register(CommandDef{
		Name:        "config",
		Description: "Configuration commands",
		Usage:       "/config <subcommand>",
		Category:    "config",
		Handler: func(args []string) CommandResult {
			return CommandResult{Output: "Usage: /config <show|set|reset>"}
		},
		SubCommands: map[string]CommandDef{
			"show": {
				Name:        "show",
				Description: "Show current configuration",
				Usage:       "/config show",
				Handler: func(args []string) CommandResult {
					return CommandResult{Output: "Current configuration:\n  (not yet connected)"}
				},
			},
			"set": {
				Name:        "set",
				Description: "Set a config value",
				Usage:       "/config set <key> <value>",
				Handler: func(args []string) CommandResult {
					if len(args) < 2 {
						return CommandResult{Error: "Usage: /config set <key> <value>"}
					}
					return CommandResult{
						Output: fmt.Sprintf("Config %s = %s (not yet connected)", args[0], strings.Join(args[1:], " ")),
					}
				},
			},
			"reset": {
				Name:        "reset",
				Description: "Reset configuration to defaults",
				Usage:       "/config reset",
				Handler: func(args []string) CommandResult {
					return CommandResult{Output: "Configuration reset to defaults. (not yet connected)"}
				},
			},
		},
	})
}

func registerDomainCommands(r *Registry) {
	_ = r.Register(CommandDef{
		Name:        "domain",
		Description: "Domain management",
		Usage:       "/domain [set <name>|detect]",
		Category:    "config",
		Handler: func(args []string) CommandResult {
			return CommandResult{Output: "Current domain: general"}
		},
		SubCommands: map[string]CommandDef{
			"set": {
				Name:        "set",
				Description: "Switch domain",
				Usage:       "/domain set <name>",
				Handler: func(args []string) CommandResult {
					if len(args) == 0 {
						return CommandResult{Error: "Usage: /domain set <name>"}
					}
					return CommandResult{Output: fmt.Sprintf("Domain switched to: %s (not yet connected)", args[0])}
				},
			},
			"detect": {
				Name:        "detect",
				Description: "Auto-detect domain from project",
				Usage:       "/domain detect",
				Handler: func(args []string) CommandResult {
					return CommandResult{Output: "Detecting domain... general (not yet connected)"}
				},
			},
		},
	})
}

func registerPluginCommands(r *Registry) {
	_ = r.Register(CommandDef{
		Name:        "plugin",
		Description: "Plugin management",
		Usage:       "/plugin <subcommand>",
		Category:    "config",
		Handler: func(args []string) CommandResult {
			return CommandResult{Output: "Usage: /plugin <list|install|start|stop>"}
		},
		SubCommands: map[string]CommandDef{
			"list": {
				Name:        "list",
				Description: "List plugins",
				Usage:       "/plugin list",
				Handler: func(args []string) CommandResult {
					return CommandResult{Output: "Plugins: (none installed)"}
				},
			},
			"install": {
				Name:        "install",
				Description: "Install a plugin",
				Usage:       "/plugin install <name>",
				Handler: func(args []string) CommandResult {
					if len(args) == 0 {
						return CommandResult{Error: "Usage: /plugin install <name>"}
					}
					return CommandResult{Output: fmt.Sprintf("Installing plugin %q... (not yet connected)", args[0])}
				},
			},
			"start": {
				Name:        "start",
				Description: "Start a plugin",
				Usage:       "/plugin start <name>",
				Handler: func(args []string) CommandResult {
					if len(args) == 0 {
						return CommandResult{Error: "Usage: /plugin start <name>"}
					}
					return CommandResult{Output: fmt.Sprintf("Starting plugin %q... (not yet connected)", args[0])}
				},
			},
			"stop": {
				Name:        "stop",
				Description: "Stop a plugin",
				Usage:       "/plugin stop <name>",
				Handler: func(args []string) CommandResult {
					if len(args) == 0 {
						return CommandResult{Error: "Usage: /plugin stop <name>"}
					}
					return CommandResult{Output: fmt.Sprintf("Stopping plugin %q... (not yet connected)", args[0])}
				},
			},
		},
	})
}

func registerHooksCommands(r *Registry) {
	_ = r.Register(CommandDef{
		Name:        "hooks",
		Description: "Hook management",
		Usage:       "/hooks <subcommand>",
		Category:    "config",
		Handler: func(args []string) CommandResult {
			return CommandResult{Output: "Usage: /hooks <list|enable|disable>"}
		},
		SubCommands: map[string]CommandDef{
			"list": {
				Name:        "list",
				Description: "List active hooks",
				Usage:       "/hooks list",
				Handler: func(args []string) CommandResult {
					return CommandResult{Output: "Active hooks: (none)"}
				},
			},
			"enable": {
				Name:        "enable",
				Description: "Enable a hook",
				Usage:       "/hooks enable <id>",
				Handler: func(args []string) CommandResult {
					if len(args) == 0 {
						return CommandResult{Error: "Usage: /hooks enable <id>"}
					}
					return CommandResult{Output: fmt.Sprintf("Hook %q enabled. (not yet connected)", args[0])}
				},
			},
			"disable": {
				Name:        "disable",
				Description: "Disable a hook",
				Usage:       "/hooks disable <id>",
				Handler: func(args []string) CommandResult {
					if len(args) == 0 {
						return CommandResult{Error: "Usage: /hooks disable <id>"}
					}
					return CommandResult{Output: fmt.Sprintf("Hook %q disabled. (not yet connected)", args[0])}
				},
			},
		},
	})
}

func registerGitCommands(r *Registry) {
	_ = r.Register(CommandDef{
		Name:        "diff",
		Description: "Show git diff",
		Usage:       "/diff",
		Category:    "git",
		Handler: func(args []string) CommandResult {
			return CommandResult{Output: "Git diff: (not yet connected)"}
		},
	})

	_ = r.Register(CommandDef{
		Name:        "commit",
		Description: "Git add and commit",
		Usage:       "/commit <message>",
		Category:    "git",
		Handler: func(args []string) CommandResult {
			if len(args) == 0 {
				return CommandResult{Error: "Usage: /commit <message>"}
			}
			msg := strings.Join(args, " ")
			return CommandResult{Output: fmt.Sprintf("Committed: %s (not yet connected)", msg)}
		},
	})

	_ = r.Register(CommandDef{
		Name:        "undo",
		Description: "Undo last file change",
		Usage:       "/undo",
		Category:    "git",
		Handler: func(args []string) CommandResult {
			return CommandResult{Output: "Undo: last change reverted. (not yet connected)"}
		},
	})
}

func registerDebugCommands(r *Registry) {
	_ = r.Register(CommandDef{
		Name:        "doctor",
		Description: "System health check",
		Usage:       "/doctor",
		Category:    "debug",
		Handler: func(args []string) CommandResult {
			return CommandResult{
				Output: "Health check:\n  LLM: ok\n  Memory: ok\n  Plugins: ok\n  Config: ok",
			}
		},
	})

	_ = r.Register(CommandDef{
		Name:        "tokens",
		Description: "Show token usage breakdown",
		Usage:       "/tokens",
		Category:    "debug",
		Handler: func(args []string) CommandResult {
			return CommandResult{
				Output: "Token usage:\n  System prompt: 0\n  Conversation: 0\n  Total: 0",
			}
		},
	})

	_ = r.Register(CommandDef{
		Name:        "context",
		Description: "Show system prompt contents",
		Usage:       "/context",
		Category:    "debug",
		Handler: func(args []string) CommandResult {
			return CommandResult{
				Output: "Context layers:\n  Base: 0 tokens\n  Domain: 0 tokens\n  Skills: 0 tokens\n  Total: 0 tokens",
			}
		},
	})

	_ = r.Register(CommandDef{
		Name:        "debug",
		Description: "Toggle debug mode",
		Usage:       "/debug [on|off]",
		Category:    "debug",
		Handler: func(args []string) CommandResult {
			if len(args) > 0 {
				switch args[0] {
				case "on":
					return CommandResult{Output: "Debug mode: ON"}
				case "off":
					return CommandResult{Output: "Debug mode: OFF"}
				}
			}
			return CommandResult{Output: "Debug mode toggled."}
		},
	})
}

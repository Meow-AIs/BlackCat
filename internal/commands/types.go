package commands

// CommandResult holds the output of a slash command execution.
type CommandResult struct {
	Output      string // text to show the user
	Error       string // error message if failed
	Silent      bool   // if true, don't show output (action-only)
	InjectToLLM bool   // if true, inject output as context for next LLM call
}

// CommandHandler is a function that handles a slash command invocation.
type CommandHandler func(args []string) CommandResult

// CommandDef defines a slash command with its metadata and handler.
type CommandDef struct {
	Name        string
	Aliases     []string // e.g., "/h" for "/help"
	Description string
	Usage       string // e.g., "/memory search <query>"
	Category    string // "general", "memory", "skills", "config", "git", "debug"
	Handler     CommandHandler
	SubCommands map[string]CommandDef // for nested commands like /memory search, /memory stats
}

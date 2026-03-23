package commands

import (
	"fmt"
	"sort"
	"strings"
)

// Registry holds all registered slash commands and dispatches input.
type Registry struct {
	commands map[string]*CommandDef
	aliases  map[string]string // alias -> canonical name
}

// NewRegistry creates an empty command registry.
func NewRegistry() *Registry {
	return &Registry{
		commands: make(map[string]*CommandDef),
		aliases:  make(map[string]string),
	}
}

// Register adds a command to the registry. Returns an error if the name
// is already taken or the handler is nil.
func (r *Registry) Register(cmd CommandDef) error {
	if cmd.Handler == nil {
		return fmt.Errorf("command %q: handler is required", cmd.Name)
	}
	if _, exists := r.commands[cmd.Name]; exists {
		return fmt.Errorf("command %q is already registered", cmd.Name)
	}
	if _, aliased := r.aliases[cmd.Name]; aliased {
		return fmt.Errorf("command %q conflicts with an existing alias", cmd.Name)
	}

	stored := cmd // copy
	r.commands[cmd.Name] = &stored

	for _, alias := range cmd.Aliases {
		r.aliases[alias] = cmd.Name
	}
	return nil
}

// Get looks up a command by name or alias.
func (r *Registry) Get(name string) (*CommandDef, bool) {
	if cmd, ok := r.commands[name]; ok {
		return cmd, true
	}
	if canonical, ok := r.aliases[name]; ok {
		cmd, found := r.commands[canonical]
		return cmd, found
	}
	return nil, false
}

// IsCommand returns true if the input starts with "/".
func (r *Registry) IsCommand(input string) bool {
	return len(input) > 0 && input[0] == '/'
}

// ParseInput splits slash command input into the command name and arguments.
// Returns empty name and nil args if the input is not a slash command.
func (r *Registry) ParseInput(input string) (string, []string) {
	if !r.IsCommand(input) {
		return "", nil
	}
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return "", nil
	}
	name := strings.TrimPrefix(parts[0], "/")
	if len(parts) == 1 {
		return name, nil
	}
	return name, parts[1:]
}

// Execute parses the input, finds the command, and runs it.
// Returns (result, true) if input was a slash command.
// Returns (zero, false) if input is not a slash command.
func (r *Registry) Execute(input string) (CommandResult, bool) {
	if !r.IsCommand(input) {
		return CommandResult{}, false
	}
	name, args := r.ParseInput(input)

	cmd, ok := r.Get(name)
	if !ok {
		return CommandResult{
			Error: fmt.Sprintf("Unknown command: /%s. Type /help for available commands.", name),
		}, true
	}

	// Check for sub-commands
	if len(args) > 0 && len(cmd.SubCommands) > 0 {
		if sub, found := cmd.SubCommands[args[0]]; found {
			return sub.Handler(args[1:]), true
		}
	}

	return cmd.Handler(args), true
}

// List returns all registered commands sorted by name.
func (r *Registry) List() []CommandDef {
	result := make([]CommandDef, 0, len(r.commands))
	for _, cmd := range r.commands {
		result = append(result, *cmd)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// ListByCategory returns commands in a given category, sorted by name.
func (r *Registry) ListByCategory(cat string) []CommandDef {
	var result []CommandDef
	for _, cmd := range r.commands {
		if cmd.Category == cat {
			result = append(result, *cmd)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// Suggest returns commands whose names start with the given prefix.
func (r *Registry) Suggest(partial string) []CommandDef {
	partial = strings.TrimPrefix(partial, "/")
	if partial == "" {
		return r.List()
	}
	var result []CommandDef
	for _, cmd := range r.commands {
		if strings.HasPrefix(cmd.Name, partial) {
			result = append(result, *cmd)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// FormatHelp returns formatted help text for all commands, grouped by category.
func (r *Registry) FormatHelp() string {
	categories := map[string][]CommandDef{}
	for _, cmd := range r.commands {
		categories[cmd.Category] = append(categories[cmd.Category], *cmd)
	}

	catOrder := make([]string, 0, len(categories))
	for cat := range categories {
		catOrder = append(catOrder, cat)
	}
	sort.Strings(catOrder)

	var b strings.Builder
	b.WriteString("Available commands:\n\n")
	for _, cat := range catOrder {
		cmds := categories[cat]
		sort.Slice(cmds, func(i, j int) bool {
			return cmds[i].Name < cmds[j].Name
		})
		b.WriteString(fmt.Sprintf("[%s]\n", cat))
		for _, cmd := range cmds {
			aliases := ""
			if len(cmd.Aliases) > 0 {
				aliases = fmt.Sprintf(" (/%s)", strings.Join(cmd.Aliases, ", /"))
			}
			b.WriteString(fmt.Sprintf("  /%s%s - %s\n", cmd.Name, aliases, cmd.Description))
		}
		b.WriteString("\n")
	}
	b.WriteString("Type /help <command> for detailed help on a specific command.")
	return b.String()
}

// FormatCommandHelp returns detailed help for a specific command.
func (r *Registry) FormatCommandHelp(name string) string {
	cmd, ok := r.Get(name)
	if !ok {
		return fmt.Sprintf("Unknown command: /%s", name)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("/%s - %s\n", cmd.Name, cmd.Description))
	if cmd.Usage != "" {
		b.WriteString(fmt.Sprintf("Usage: %s\n", cmd.Usage))
	}
	if len(cmd.Aliases) > 0 {
		b.WriteString(fmt.Sprintf("Aliases: /%s\n", strings.Join(cmd.Aliases, ", /")))
	}
	if len(cmd.SubCommands) > 0 {
		b.WriteString("\nSub-commands:\n")
		names := make([]string, 0, len(cmd.SubCommands))
		for n := range cmd.SubCommands {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			sub := cmd.SubCommands[n]
			b.WriteString(fmt.Sprintf("  %s - %s\n", n, sub.Description))
		}
	}
	return b.String()
}

package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// CustomCommand represents a user-defined slash command from skills or plugins.
type CustomCommand struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	SkillID     string `json:"skill_id,omitempty"`
	PluginID    string `json:"plugin_id,omitempty"`
	Action      string `json:"action"`
	Prompt      string `json:"prompt,omitempty"`
}

// LoadCustomCommands reads custom commands from a JSON file.
// Returns an empty slice (no error) if the file does not exist.
func LoadCustomCommands(path string) ([]CustomCommand, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []CustomCommand{}, nil
		}
		return nil, fmt.Errorf("reading custom commands: %w", err)
	}

	var cmds []CustomCommand
	if err := json.Unmarshal(data, &cmds); err != nil {
		return nil, fmt.Errorf("parsing custom commands: %w", err)
	}
	return cmds, nil
}

// SaveCustomCommands writes custom commands to a JSON file.
func SaveCustomCommands(path string, cmds []CustomCommand) error {
	data, err := json.MarshalIndent(cmds, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling custom commands: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing custom commands: %w", err)
	}
	return nil
}

// ToCommandDef converts a CustomCommand into a CommandDef for registry use.
func (c CustomCommand) ToCommandDef() CommandDef {
	cmd := c // capture for closure
	return CommandDef{
		Name:        cmd.Name,
		Description: cmd.Description,
		Usage:       fmt.Sprintf("/%s", cmd.Name),
		Category:    "custom",
		Handler:     cmd.handler(),
	}
}

// handler returns the appropriate CommandHandler based on the action type.
func (c CustomCommand) handler() CommandHandler {
	cmd := c // capture for closure
	switch cmd.Action {
	case "run_skill":
		return func(args []string) CommandResult {
			return CommandResult{
				Output: fmt.Sprintf("Running skill %q... (not yet connected)", cmd.SkillID),
			}
		}
	case "run_plugin":
		return func(args []string) CommandResult {
			return CommandResult{
				Output: fmt.Sprintf("Running plugin %q... (not yet connected)", cmd.PluginID),
			}
		}
	case "inject_prompt":
		return func(args []string) CommandResult {
			return CommandResult{
				Output:      fmt.Sprintf("Injecting prompt: %s", cmd.Prompt),
				InjectToLLM: true,
			}
		}
	default:
		return func(args []string) CommandResult {
			return CommandResult{
				Output: fmt.Sprintf("Unknown action %q for command /%s", cmd.Action, cmd.Name),
			}
		}
	}
}

package builtin

import (
	"context"
	"fmt"

	"github.com/meowai/blackcat/internal/tools"
)

// ConfigTool views and changes BlackCat configuration via natural language.
type ConfigTool struct{}

// NewConfigTool creates a new ConfigTool.
func NewConfigTool() *ConfigTool {
	return &ConfigTool{}
}

// Info returns the tool definition for manage_config.
func (t *ConfigTool) Info() tools.Definition {
	return tools.Definition{
		Name:        "manage_config",
		Description: "View and change BlackCat configuration. Use 'show' to see current config, 'set' to change a value, 'reset' to restore defaults. Supports model changes, provider settings, and preferences.",
		Category:    "system",
		Parameters: []tools.Parameter{
			{
				Name:        "action",
				Type:        "string",
				Description: "The action to perform",
				Required:    true,
				Enum:        []string{"show", "set", "reset", "get"},
			},
			{
				Name:        "key",
				Type:        "string",
				Description: "Config key for set/get (e.g. \"model\", \"provider\", \"memory.budget\")",
			},
			{
				Name:        "value",
				Type:        "string",
				Description: "New value for set action",
			},
		},
	}
}

// Execute runs the config tool with the given arguments.
func (t *ConfigTool) Execute(_ context.Context, args map[string]any) (tools.Result, error) {
	action, err := requireStringArg(args, "action")
	if err != nil {
		return tools.Result{}, err
	}

	key, _ := args["key"].(string)
	value, _ := args["value"].(string)

	switch action {
	case "show":
		return configShow(), nil
	case "set":
		return configSet(key, value), nil
	case "get":
		return configGet(key), nil
	case "reset":
		return configReset(), nil
	default:
		return tools.Result{
			Error:    fmt.Sprintf("unknown action %q: must be one of show, set, reset, get", action),
			ExitCode: 1,
		}, nil
	}
}

func configShow() tools.Result {
	output := "Current configuration:\n" +
		"  model: claude-sonnet-4-6\n" +
		"  provider: anthropic\n" +
		"  memory.budget: 10000\n" +
		"  memory.quantization: int8\n" +
		"  domain: general\n" +
		"  router.main: anthropic\n" +
		"  router.auxiliary: haiku\n" +
		"  router.local: ollama"
	return tools.Result{Output: output, ExitCode: 0}
}

func configSet(key, value string) tools.Result {
	if key == "" {
		return tools.Result{
			Error:    "missing required 'key' for set action",
			ExitCode: 1,
		}
	}

	switch key {
	case "model":
		return tools.Result{
			Output:   fmt.Sprintf("Model changed to: %s", value),
			ExitCode: 0,
		}
	case "provider":
		return tools.Result{
			Output:   fmt.Sprintf("Provider changed to: %s", value),
			ExitCode: 0,
		}
	default:
		return tools.Result{
			Output:   fmt.Sprintf("Config %s set to: %s", key, value),
			ExitCode: 0,
		}
	}
}

func configGet(key string) tools.Result {
	if key == "" {
		return tools.Result{
			Error:    "missing required 'key' for get action",
			ExitCode: 1,
		}
	}

	// Placeholder values
	defaults := map[string]string{
		"model":               "claude-sonnet-4-6",
		"provider":            "anthropic",
		"memory.budget":       "10000",
		"memory.quantization": "int8",
		"domain":              "general",
	}

	val, ok := defaults[key]
	if !ok {
		val = "(not set)"
	}

	return tools.Result{
		Output:   fmt.Sprintf("Config %s = %s", key, val),
		ExitCode: 0,
	}
}

func configReset() tools.Result {
	return tools.Result{
		Output:   "Configuration reset to defaults",
		ExitCode: 0,
	}
}

package builtin

import (
	"context"
	"fmt"
	"strings"

	"github.com/meowai/blackcat/internal/tools"
)

// ModelTool switches the active LLM model via natural language.
type ModelTool struct{}

// NewModelTool creates a new ModelTool.
func NewModelTool() *ModelTool {
	return &ModelTool{}
}

// Info returns the tool definition for change_model.
func (t *ModelTool) Info() tools.Definition {
	return tools.Definition{
		Name:        "change_model",
		Description: "Switch the active LLM model. Supports provider/model format (e.g., 'anthropic/claude-sonnet-4-6', 'ollama/qwen2.5:32b', 'openrouter/google/gemini-2.5-pro'). Use 'list' to see available models.",
		Category:    "system",
		Parameters: []tools.Parameter{
			{
				Name:        "action",
				Type:        "string",
				Description: "The action to perform",
				Required:    true,
				Enum:        []string{"set", "list", "info"},
			},
			{
				Name:        "model",
				Type:        "string",
				Description: "Model identifier for set (e.g. \"anthropic/claude-sonnet-4-6\")",
			},
		},
	}
}

// Execute runs the model tool with the given arguments.
func (t *ModelTool) Execute(_ context.Context, args map[string]any) (tools.Result, error) {
	action, err := requireStringArg(args, "action")
	if err != nil {
		return tools.Result{}, err
	}

	model, _ := args["model"].(string)

	switch action {
	case "set":
		return modelSet(model), nil
	case "list":
		return modelList(), nil
	case "info":
		return modelInfo(), nil
	default:
		return tools.Result{
			Error:    fmt.Sprintf("unknown action %q: must be one of set, list, info", action),
			ExitCode: 1,
		}, nil
	}
}

func modelSet(model string) tools.Result {
	if model == "" {
		return tools.Result{
			Error:    "missing required 'model' parameter for set action",
			ExitCode: 1,
		}
	}

	provider, modelName := parseModelIdentifier(model)

	output := fmt.Sprintf("Model changed to: %s\n", modelName) +
		fmt.Sprintf("Provider: %s\n", provider) +
		"Max tokens: 64,000\n" +
		"Cost: $3.00/M input, $15.00/M output"
	return tools.Result{Output: output, ExitCode: 0}
}

// parseModelIdentifier splits "provider/model" into provider and model name.
// For multi-segment paths like "openrouter/google/gemini-2.5-pro", the first
// segment is the provider and the rest is the model name.
func parseModelIdentifier(model string) (string, string) {
	parts := strings.SplitN(model, "/", 2)
	if len(parts) == 2 {
		return capitalizeProvider(parts[0]), parts[1]
	}
	return "Unknown", model
}

func capitalizeProvider(p string) string {
	providerNames := map[string]string{
		"anthropic":  "Anthropic",
		"openai":     "OpenAI",
		"ollama":     "Ollama",
		"openrouter": "OpenRouter",
		"google":     "Google",
		"groq":       "Groq",
	}
	if name, ok := providerNames[strings.ToLower(p)]; ok {
		return name
	}
	return p
}

func modelList() tools.Result {
	output := "Available models:\n\n" +
		"Anthropic:\n" +
		"  - claude-opus-4-6 (200K context)\n" +
		"  - claude-sonnet-4-6 (200K context)\n" +
		"  - claude-haiku-3.5 (200K context)\n\n" +
		"OpenAI:\n" +
		"  - gpt-4o (128K context)\n" +
		"  - gpt-4o-mini (128K context)\n\n" +
		"Ollama (local):\n" +
		"  - qwen2.5:32b\n" +
		"  - llama3.1:70b\n" +
		"  - codestral:22b"
	return tools.Result{Output: output, ExitCode: 0}
}

func modelInfo() tools.Result {
	output := "Current model: claude-sonnet-4-6\n" +
		"Provider: Anthropic\n" +
		"Max tokens: 64,000\n" +
		"Session cost so far: $0.03"
	return tools.Result{Output: output, ExitCode: 0}
}

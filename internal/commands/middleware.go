package commands

import "fmt"

// InputMiddleware intercepts user input and routes slash commands
// before they reach the LLM.
type InputMiddleware struct {
	registry *Registry
}

// NewInputMiddleware creates a new middleware with the given command registry.
func NewInputMiddleware(registry *Registry) *InputMiddleware {
	return &InputMiddleware{registry: registry}
}

// Process checks if input is a slash command and executes it.
// Returns (result, true) if it was a command, (zero, false) if regular input.
func (m *InputMiddleware) Process(input string) (CommandResult, bool) {
	return m.registry.Execute(input)
}

// ShouldBypassLLM returns true if the input is a slash command
// that should NOT be sent to the LLM.
func (m *InputMiddleware) ShouldBypassLLM(input string) bool {
	return m.registry.IsCommand(input)
}

// Autocomplete returns command name suggestions for partial input.
// Each suggestion is prefixed with "/".
func (m *InputMiddleware) Autocomplete(partial string) []string {
	if !m.registry.IsCommand(partial) {
		return nil
	}
	trimmed := partial[1:] // strip leading "/"
	suggestions := m.registry.Suggest(trimmed)
	result := make([]string, 0, len(suggestions))
	for _, cmd := range suggestions {
		result = append(result, fmt.Sprintf("/%s", cmd.Name))
	}
	return result
}

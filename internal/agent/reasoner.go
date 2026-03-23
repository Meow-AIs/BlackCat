package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/meowai/blackcat/internal/llm"
	"github.com/meowai/blackcat/internal/tools"
)

// Reasoner builds LLM prompts with memory context and tool definitions.
type Reasoner struct {
	systemPrompt string
	maxTokens    int
}

// NewReasoner creates a reasoner with the given system prompt and token limit.
func NewReasoner(systemPrompt string, maxTokens int) *Reasoner {
	return &Reasoner{
		systemPrompt: systemPrompt,
		maxTokens:    maxTokens,
	}
}

// BuildMessages constructs the full message list for an LLM call.
// It prepends a system message containing the base prompt, memory snapshot,
// and tool descriptions, then appends the conversation history.
func (r *Reasoner) BuildMessages(history []llm.Message, memorySnapshot string, toolDefs []tools.Definition) []llm.Message {
	systemContent := r.buildSystemContent(memorySnapshot, toolDefs)

	// Create new slice — never mutate history
	messages := make([]llm.Message, 0, 1+len(history))
	messages = append(messages, llm.Message{
		Role:    llm.RoleSystem,
		Content: systemContent,
	})
	messages = append(messages, history...)

	return messages
}

// buildSystemContent assembles the system prompt with optional memory and tool sections.
func (r *Reasoner) buildSystemContent(memorySnapshot string, toolDefs []tools.Definition) string {
	var b strings.Builder
	b.WriteString(r.systemPrompt)

	if memorySnapshot != "" {
		b.WriteString("\n\n## Memory\n")
		b.WriteString(memorySnapshot)
	}

	if len(toolDefs) > 0 {
		b.WriteString("\n\n## Available Tools\n")
		for _, def := range toolDefs {
			b.WriteString(fmt.Sprintf("- **%s**: %s\n", def.Name, def.Description))
		}
	}

	return b.String()
}

// InjectContext returns a new message slice with additional context appended
// to the system message. The original slice is not modified.
func (r *Reasoner) InjectContext(messages []llm.Message, context string) []llm.Message {
	if context == "" {
		// Return a copy to maintain immutability contract
		result := make([]llm.Message, len(messages))
		copy(result, messages)
		return result
	}

	result := make([]llm.Message, len(messages))
	copy(result, messages)

	if len(result) > 0 && result[0].Role == llm.RoleSystem {
		// Create new message with appended context
		result[0] = llm.Message{
			Role:    llm.RoleSystem,
			Content: result[0].Content + "\n\n## Additional Context\n" + context,
		}
	}

	return result
}

// ExtractToolCalls parses tool call information from an LLM response into
// the agent's ToolCall format.
func (r *Reasoner) ExtractToolCalls(response llm.ChatResponse) []ToolCall {
	if len(response.ToolCalls) == 0 {
		return nil
	}

	calls := make([]ToolCall, len(response.ToolCalls))
	for i, tc := range response.ToolCalls {
		args := make(map[string]any)
		if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
			// Keep empty args map on parse failure
			args = make(map[string]any)
		}
		calls[i] = ToolCall{
			Name: tc.Name,
			Args: args,
		}
	}

	return calls
}

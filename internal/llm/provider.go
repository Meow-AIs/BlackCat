package llm

import "context"

// Role represents the role of a message in a conversation.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message represents a single message in a conversation.
type Message struct {
	Role       Role       `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ToolCall represents a tool invocation requested by the model.
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// ToolDefinition describes a tool available to the model.
type ToolDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"` // JSON Schema object
}

// ChatRequest is the input to a Chat or Stream call.
type ChatRequest struct {
	Model       string           `json:"model"`
	Messages    []Message        `json:"messages"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Temperature *float64         `json:"temperature,omitempty"`
	Stop        []string         `json:"stop,omitempty"`
}

// ChatResponse is the output of a Chat call.
type ChatResponse struct {
	Content      string     `json:"content"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	Model        string     `json:"model"`
	Usage        Usage      `json:"usage"`
	FinishReason string     `json:"finish_reason"`
}

// Usage tracks token consumption.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamChunk is a single piece of a streaming response.
type StreamChunk struct {
	Content      string     `json:"content,omitempty"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	Done         bool       `json:"done"`
	FinishReason string     `json:"finish_reason,omitempty"`
	Usage        *Usage     `json:"usage,omitempty"`
}

// ModelInfo describes a model available from a provider.
type ModelInfo struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	MaxTokens  int     `json:"max_tokens"`
	InputCost  float64 `json:"input_cost_per_1m"`  // USD per 1M tokens
	OutputCost float64 `json:"output_cost_per_1m"` // USD per 1M tokens
}

// Provider is the interface that all LLM backends must implement.
type Provider interface {
	// Chat sends a request and returns the full response.
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)

	// Stream sends a request and returns a channel of streaming chunks.
	Stream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error)

	// Models returns the list of models available from this provider.
	Models() []ModelInfo

	// Name returns the provider name (e.g., "anthropic", "openai").
	Name() string
}

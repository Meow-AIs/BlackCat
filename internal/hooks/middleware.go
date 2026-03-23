package hooks

import (
	"context"
	"errors"
	"fmt"
)

// ErrHookBlocked is returned when a hook blocks tool execution.
var ErrHookBlocked = errors.New("blocked by hook")

// ToolMiddleware wraps tool execution with before/after hooks.
type ToolMiddleware struct {
	engine *Engine
}

// NewToolMiddleware creates a new ToolMiddleware.
func NewToolMiddleware(engine *Engine) *ToolMiddleware {
	return &ToolMiddleware{engine: engine}
}

// WrapExecution fires before_tool, executes the tool, fires after_tool/tool_error.
func (m *ToolMiddleware) WrapExecution(
	ctx context.Context,
	toolName string,
	args map[string]any,
	execute func() (string, error),
) (string, error) {
	// Fire before_tool hooks.
	beforeData := map[string]any{
		"tool_name": toolName,
	}
	for k, v := range args {
		beforeData[k] = v
	}

	beforeResult := m.engine.Fire(EventBeforeTool, beforeData)
	if !beforeResult.Allow {
		msg := beforeResult.Message
		if msg == "" {
			msg = fmt.Sprintf("tool %q blocked by hook", toolName)
		}
		return "", fmt.Errorf("%w: %s", ErrHookBlocked, msg)
	}

	// Execute the tool.
	output, err := execute()
	if err != nil {
		// Fire tool_error hooks.
		errorData := map[string]any{
			"tool_name": toolName,
			"error":     err.Error(),
		}
		for k, v := range args {
			errorData[k] = v
		}
		m.engine.Fire(EventToolError, errorData)

		return "", err
	}

	// Fire after_tool hooks.
	afterData := map[string]any{
		"tool_name": toolName,
		"output":    output,
	}
	for k, v := range args {
		afterData[k] = v
	}
	if beforeResult.Modified != nil {
		for k, v := range beforeResult.Modified {
			afterData[k] = v
		}
	}

	afterResult := m.engine.Fire(EventAfterTool, afterData)

	// If after hooks modified the output, use the modified version.
	if modifiedOutput, ok := afterResult.Modified["output"].(string); ok {
		return modifiedOutput, nil
	}

	return output, nil
}

// ResponseMiddleware wraps LLM responses with hooks.
type ResponseMiddleware struct {
	engine *Engine
}

// NewResponseMiddleware creates a new ResponseMiddleware.
func NewResponseMiddleware(engine *Engine) *ResponseMiddleware {
	return &ResponseMiddleware{engine: engine}
}

// WrapResponse fires before_response hooks and returns potentially modified response.
func (m *ResponseMiddleware) WrapResponse(response string, sessionID string) string {
	data := map[string]any{
		"response":   response,
		"session_id": sessionID,
	}

	result := m.engine.Fire(EventBeforeResponse, data)

	if modifiedResponse, ok := result.Modified["response"].(string); ok {
		return modifiedResponse
	}

	return response
}

// MemoryMiddleware wraps memory operations with hooks.
type MemoryMiddleware struct {
	engine *Engine
}

// NewMemoryMiddleware creates a new MemoryMiddleware.
func NewMemoryMiddleware(engine *Engine) *MemoryMiddleware {
	return &MemoryMiddleware{engine: engine}
}

// ShouldStore checks if a memory entry should be stored by firing memory_store hooks.
func (m *MemoryMiddleware) ShouldStore(content string, tier string) bool {
	data := map[string]any{
		"content": content,
		"tier":    tier,
	}

	result := m.engine.Fire(EventMemoryStore, data)
	return result.Allow
}

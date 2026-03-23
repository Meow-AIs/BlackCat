package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// AnthropicProvider implements Provider for the Anthropic Messages API.
type AnthropicProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewAnthropicProvider creates an Anthropic Claude provider.
func NewAnthropicProvider(apiKey, baseURL string) *AnthropicProvider {
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	return &AnthropicProvider{apiKey: apiKey, baseURL: baseURL, client: &http.Client{}}
}

func (p *AnthropicProvider) Name() string { return "anthropic" }

func (p *AnthropicProvider) Models() []ModelInfo {
	return []ModelInfo{
		{ID: "claude-sonnet-4-6", Name: "Claude Sonnet 4.6", MaxTokens: 64000, InputCost: 3.0, OutputCost: 15.0},
		{ID: "claude-opus-4-6", Name: "Claude Opus 4.6", MaxTokens: 32000, InputCost: 15.0, OutputCost: 75.0},
		{ID: "claude-sonnet-4-5", Name: "Claude Sonnet 4.5", MaxTokens: 64000, InputCost: 3.0, OutputCost: 15.0},
		{ID: "claude-opus-4-5", Name: "Claude Opus 4.5", MaxTokens: 32000, InputCost: 15.0, OutputCost: 75.0},
		{ID: "claude-haiku-4-5", Name: "Claude Haiku 4.5", MaxTokens: 64000, InputCost: 0.8, OutputCost: 4.0},
	}
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	Messages  []anthropicMsg     `json:"messages"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Tools     []anthropicTool    `json:"tools,omitempty"`
}

type anthropicMsg struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // string or []contentBlock
}

type contentBlock struct {
	Type      string `json:"type"`
	Text      string `json:"text,omitempty"`
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Input     any    `json:"input,omitempty"`
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   string `json:"content,omitempty"`
}

type anthropicTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"input_schema"`
}

type anthropicResponse struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Role       string         `json:"role"`
	Content    []contentBlock `json:"content"`
	Model      string         `json:"model"`
	StopReason string         `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (p *AnthropicProvider) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	body := p.buildRequest(req)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("send request: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return ChatResponse{}, fmt.Errorf("API error (status %d): %s", httpResp.StatusCode, string(respBody))
	}

	var aResp anthropicResponse
	if err := json.Unmarshal(respBody, &aResp); err != nil {
		return ChatResponse{}, fmt.Errorf("parse response: %w", err)
	}

	return p.parseResponse(aResp), nil
}

func (p *AnthropicProvider) Stream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	return nil, fmt.Errorf("streaming not yet implemented")
}

func (p *AnthropicProvider) buildRequest(req ChatRequest) anthropicRequest {
	var system string
	var msgs []anthropicMsg

	for _, m := range req.Messages {
		if m.Role == RoleSystem {
			system = m.Content
			continue
		}
		if m.Role == RoleTool {
			msgs = append(msgs, anthropicMsg{
				Role: "user",
				Content: []contentBlock{{
					Type:      "tool_result",
					ToolUseID: m.ToolCallID,
					Content:   m.Content,
				}},
			})
			continue
		}
		msgs = append(msgs, anthropicMsg{Role: string(m.Role), Content: m.Content})
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	var tools []anthropicTool
	for _, td := range req.Tools {
		tools = append(tools, anthropicTool{
			Name:        td.Name,
			Description: td.Description,
			InputSchema: td.Parameters,
		})
	}

	return anthropicRequest{
		Model:     req.Model,
		Messages:  msgs,
		MaxTokens: maxTokens,
		System:    system,
		Tools:     tools,
	}
}

func (p *AnthropicProvider) parseResponse(resp anthropicResponse) ChatResponse {
	result := ChatResponse{
		Model:        resp.Model,
		FinishReason: resp.StopReason,
		Usage: Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			result.Content += block.Text
		case "tool_use":
			inputJSON, _ := json.Marshal(block.Input)
			result.ToolCalls = append(result.ToolCalls, ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: string(inputJSON),
			})
		}
	}

	return result
}

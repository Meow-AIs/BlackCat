package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// OpenAIProvider implements Provider for OpenAI-compatible APIs.
type OpenAIProvider struct {
	apiKey  string
	baseURL string
	name    string
	client  *http.Client
}

// NewOpenAIProvider creates a provider for any OpenAI-compatible API.
// If name is empty, defaults to "openai".
func NewOpenAIProvider(apiKey, baseURL, name string) *OpenAIProvider {
	if name == "" {
		name = "openai"
	}
	return &OpenAIProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		name:    name,
		client:  &http.Client{},
	}
}

func (p *OpenAIProvider) Name() string {
	return p.name
}

func (p *OpenAIProvider) Models() []ModelInfo {
	return []ModelInfo{
		{ID: "gpt-5.4", Name: "GPT-5.4", MaxTokens: 128000, InputCost: 5.0, OutputCost: 15.0},
		{ID: "gpt-4.1", Name: "GPT-4.1", MaxTokens: 1000000, InputCost: 2.0, OutputCost: 8.0},
		{ID: "gpt-4.1-mini", Name: "GPT-4.1 Mini", MaxTokens: 1000000, InputCost: 0.4, OutputCost: 1.6},
		{ID: "gpt-4.1-nano", Name: "GPT-4.1 Nano", MaxTokens: 1000000, InputCost: 0.1, OutputCost: 0.4},
		{ID: "o4-mini", Name: "o4 Mini", MaxTokens: 200000, InputCost: 1.1, OutputCost: 4.4},
		{ID: "o3", Name: "o3", MaxTokens: 200000, InputCost: 2.0, OutputCost: 8.0},
		{ID: "o3-mini", Name: "o3 Mini", MaxTokens: 200000, InputCost: 1.1, OutputCost: 4.4},
	}
}

// openaiRequest is the request body for the chat completions endpoint.
type openaiRequest struct {
	Model       string          `json:"model"`
	Messages    []openaiMessage `json:"messages"`
	Tools       []openaiTool    `json:"tools,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature *float64        `json:"temperature,omitempty"`
	Stop        []string        `json:"stop,omitempty"`
	Stream      bool            `json:"stream"`
}

type openaiMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	ToolCalls  []openaiToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openaiTool struct {
	Type     string         `json:"type"`
	Function openaiFunction `json:"function"`
}

type openaiFunction struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

type openaiToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// openaiResponse is the response body from chat completions.
type openaiResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role      string           `json:"role"`
			Content   string           `json:"content"`
			ToolCalls []openaiToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func (p *OpenAIProvider) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	body := p.buildRequest(req)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

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

	var oaiResp openaiResponse
	if err := json.Unmarshal(respBody, &oaiResp); err != nil {
		return ChatResponse{}, fmt.Errorf("parse response: %w", err)
	}

	return p.parseResponse(oaiResp), nil
}

func (p *OpenAIProvider) Stream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	// TODO: implement streaming in Phase 1.2
	return nil, fmt.Errorf("streaming not yet implemented")
}

func (p *OpenAIProvider) buildRequest(req ChatRequest) openaiRequest {
	msgs := make([]openaiMessage, len(req.Messages))
	for i, m := range req.Messages {
		msg := openaiMessage{
			Role:       string(m.Role),
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
		}
		for _, tc := range m.ToolCalls {
			msg.ToolCalls = append(msg.ToolCalls, openaiToolCall{
				ID:   tc.ID,
				Type: "function",
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      tc.Name,
					Arguments: tc.Arguments,
				},
			})
		}
		msgs[i] = msg
	}

	var tools []openaiTool
	for _, td := range req.Tools {
		tools = append(tools, openaiTool{
			Type: "function",
			Function: openaiFunction{
				Name:        td.Name,
				Description: td.Description,
				Parameters:  td.Parameters,
			},
		})
	}

	return openaiRequest{
		Model:       req.Model,
		Messages:    msgs,
		Tools:       tools,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stop:        req.Stop,
		Stream:      false,
	}
}

func (p *OpenAIProvider) parseResponse(resp openaiResponse) ChatResponse {
	result := ChatResponse{
		Model: resp.Model,
		Usage: Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}

	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		result.Content = choice.Message.Content
		result.FinishReason = choice.FinishReason

		for _, tc := range choice.Message.ToolCalls {
			result.ToolCalls = append(result.ToolCalls, ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			})
		}
	}

	return result
}

package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// OpenRouterProvider implements Provider for the OpenRouter API.
// It uses the OpenAI-compatible chat completions format with extra headers.
type OpenRouterProvider struct {
	apiKey  string
	baseURL string
	referer string
	title   string
	client  *http.Client
}

// OpenRouterOption configures an OpenRouterProvider.
type OpenRouterOption func(*OpenRouterProvider)

// WithReferer sets the HTTP-Referer header for OpenRouter rankings.
func WithReferer(referer string) OpenRouterOption {
	return func(p *OpenRouterProvider) { p.referer = referer }
}

// WithTitle sets the X-Title header for OpenRouter rankings.
func WithTitle(title string) OpenRouterOption {
	return func(p *OpenRouterProvider) { p.title = title }
}

// WithBaseURL overrides the default OpenRouter API base URL.
func WithBaseURL(baseURL string) OpenRouterOption {
	return func(p *OpenRouterProvider) { p.baseURL = baseURL }
}

// NewOpenRouterProvider creates an OpenRouter provider.
func NewOpenRouterProvider(apiKey string, opts ...OpenRouterOption) *OpenRouterProvider {
	p := &OpenRouterProvider{
		apiKey:  apiKey,
		baseURL: "https://openrouter.ai/api/v1",
		client:  &http.Client{},
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *OpenRouterProvider) Name() string { return "openrouter" }

func (p *OpenRouterProvider) Models() []ModelInfo {
	return []ModelInfo{
		// Anthropic
		{ID: "anthropic/claude-sonnet-4-6", Name: "Claude Sonnet 4.6", MaxTokens: 64000, InputCost: 3.0, OutputCost: 15.0},
		{ID: "anthropic/claude-opus-4-6", Name: "Claude Opus 4.6", MaxTokens: 32000, InputCost: 15.0, OutputCost: 75.0},
		{ID: "anthropic/claude-haiku-4-5", Name: "Claude Haiku 4.5", MaxTokens: 64000, InputCost: 0.8, OutputCost: 4.0},
		// OpenAI
		{ID: "openai/gpt-5.4", Name: "GPT-5.4", MaxTokens: 128000, InputCost: 5.0, OutputCost: 15.0},
		{ID: "openai/gpt-4.1", Name: "GPT-4.1", MaxTokens: 1000000, InputCost: 2.0, OutputCost: 8.0},
		{ID: "openai/gpt-4.1-mini", Name: "GPT-4.1 Mini", MaxTokens: 1000000, InputCost: 0.4, OutputCost: 1.6},
		{ID: "openai/o4-mini", Name: "o4 Mini", MaxTokens: 200000, InputCost: 1.1, OutputCost: 4.4},
		// Google
		{ID: "google/gemini-2.5-pro", Name: "Gemini 2.5 Pro", MaxTokens: 1000000, InputCost: 1.25, OutputCost: 10.0},
		{ID: "google/gemini-2.5-flash", Name: "Gemini 2.5 Flash", MaxTokens: 1000000, InputCost: 0.15, OutputCost: 0.60},
		// Meta
		{ID: "meta-llama/llama-4-maverick", Name: "Llama 4 Maverick", MaxTokens: 1000000, InputCost: 0.2, OutputCost: 0.6},
		{ID: "meta-llama/llama-4-scout", Name: "Llama 4 Scout", MaxTokens: 10000000, InputCost: 0.11, OutputCost: 0.34},
		// DeepSeek
		{ID: "deepseek/deepseek-chat", Name: "DeepSeek Chat", MaxTokens: 65536, InputCost: 0.14, OutputCost: 0.28},
		{ID: "deepseek/deepseek-reasoner", Name: "DeepSeek Reasoner", MaxTokens: 65536, InputCost: 0.55, OutputCost: 2.19},
		// xAI
		{ID: "x-ai/grok-4-1-fast", Name: "Grok 4.1 Fast", MaxTokens: 131072, InputCost: 3.0, OutputCost: 15.0},
		// Kimi
		{ID: "moonshotai/kimi-k2.5", Name: "Kimi K2.5", MaxTokens: 256000, InputCost: 1.0, OutputCost: 4.0},
		// GLM
		{ID: "zai/glm-5", Name: "GLM-5", MaxTokens: 200000, InputCost: 1.0, OutputCost: 3.2},
	}
}

func (p *OpenRouterProvider) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	body := p.buildRequest(req, false)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("create request: %w", err)
	}

	p.setHeaders(httpReq)

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

func (p *OpenRouterProvider) Stream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	body := p.buildRequest(req, true)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	p.setHeaders(httpReq)

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		defer httpResp.Body.Close()
		respBody, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", httpResp.StatusCode, string(respBody))
	}

	ch := make(chan StreamChunk)
	go func() {
		defer close(ch)
		defer httpResp.Body.Close()

		scanner := bufio.NewScanner(httpResp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var event openaiStreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			chunk := StreamChunk{}
			if len(event.Choices) > 0 {
				choice := event.Choices[0]
				chunk.Content = choice.Delta.Content
				if choice.FinishReason != "" {
					chunk.Done = true
					chunk.FinishReason = choice.FinishReason
				}
				for _, tc := range choice.Delta.ToolCalls {
					chunk.ToolCalls = append(chunk.ToolCalls, ToolCall{
						ID:        tc.ID,
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					})
				}
			}
			if event.Usage != nil {
				chunk.Usage = &Usage{
					PromptTokens:     event.Usage.PromptTokens,
					CompletionTokens: event.Usage.CompletionTokens,
					TotalTokens:      event.Usage.TotalTokens,
				}
			}

			select {
			case ch <- chunk:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch, nil
}

// openaiStreamEvent represents an SSE event from the OpenAI-compatible streaming API.
type openaiStreamEvent struct {
	ID      string `json:"id"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Content   string           `json:"content"`
			ToolCalls []openaiToolCall `json:"tool_calls,omitempty"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

func (p *OpenRouterProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	if p.referer != "" {
		req.Header.Set("HTTP-Referer", p.referer)
	}
	if p.title != "" {
		req.Header.Set("X-Title", p.title)
	}
}

func (p *OpenRouterProvider) buildRequest(req ChatRequest, stream bool) openaiRequest {
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
		Stream:      stream,
	}
}

func (p *OpenRouterProvider) parseResponse(resp openaiResponse) ChatResponse {
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

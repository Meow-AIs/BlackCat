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

// ZAIProvider implements Provider for the Z.ai (Zhipu AI / GLM) API.
// It uses the OpenAI-compatible chat completions format.
type ZAIProvider struct {
	apiKey     string
	baseURL    string
	codingURL  string
	httpClient *http.Client
	useCoding  bool
}

// ZAIOption configures a ZAIProvider.
type ZAIOption func(*ZAIProvider)

// WithZAICodingPlan configures the provider to use the coding endpoint.
func WithZAICodingPlan() ZAIOption {
	return func(p *ZAIProvider) {
		p.useCoding = true
	}
}

// WithZAIBaseURL overrides the default Z.ai API base URL.
func WithZAIBaseURL(url string) ZAIOption {
	return func(p *ZAIProvider) {
		p.baseURL = url
	}
}

// NewZAIProvider creates a Z.ai provider with the given API key and options.
func NewZAIProvider(apiKey string, opts ...ZAIOption) *ZAIProvider {
	p := &ZAIProvider{
		apiKey:     apiKey,
		baseURL:    "https://api.z.ai/api/paas/v4",
		codingURL:  "https://api.z.ai/api/coding/paas/v4",
		httpClient: &http.Client{},
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *ZAIProvider) Name() string { return "zai" }

func (p *ZAIProvider) Models() []ModelInfo {
	return []ModelInfo{
		{ID: "glm-5", Name: "GLM-5 (745B MoE)", MaxTokens: 200000, InputCost: 1.0, OutputCost: 3.2},
		{ID: "glm-5-turbo", Name: "GLM-5 Turbo (Agent-optimized)", MaxTokens: 200000, InputCost: 1.2, OutputCost: 4.0},
		{ID: "glm-4.7", Name: "GLM-4.7", MaxTokens: 128000, InputCost: 0.7, OutputCost: 2.2},
		{ID: "glm-4.7-flash", Name: "GLM-4.7 Flash", MaxTokens: 128000, InputCost: 0.0, OutputCost: 0.0},
		{ID: "glm-4.5", Name: "GLM-4.5", MaxTokens: 128000, InputCost: 0.7, OutputCost: 2.2},
	}
}

// endpointURL returns the appropriate base URL depending on coding plan mode.
func (p *ZAIProvider) endpointURL() string {
	if p.useCoding {
		return p.codingURL
	}
	return p.baseURL
}

func (p *ZAIProvider) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	body := buildOpenAIRequest(req, false)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.endpointURL()+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	httpResp, err := p.httpClient.Do(httpReq)
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

	return parseOpenAIResponse(oaiResp), nil
}

func (p *ZAIProvider) Stream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	body := buildOpenAIRequest(req, true)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.endpointURL()+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	httpResp, err := p.httpClient.Do(httpReq)
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
		streamOpenAISSE(ctx, httpResp.Body, ch)
	}()

	return ch, nil
}

// buildOpenAIRequest converts a ChatRequest to an openaiRequest.
// Shared by all OpenAI-compatible providers in this package.
func buildOpenAIRequest(req ChatRequest, stream bool) openaiRequest {
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

// parseOpenAIResponse converts an openaiResponse to a ChatResponse.
// Shared by all OpenAI-compatible providers in this package.
func parseOpenAIResponse(resp openaiResponse) ChatResponse {
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

// streamOpenAISSE reads SSE events from an OpenAI-compatible stream and sends chunks.
// Shared by all OpenAI-compatible providers in this package.
func streamOpenAISSE(ctx context.Context, body io.Reader, ch chan<- StreamChunk) {
	scanner := bufio.NewScanner(body)
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
}

package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// XAIProvider implements Provider for the xAI/Grok API.
// It uses the OpenAI-compatible chat completions format.
type XAIProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// XAIOption configures an XAIProvider.
type XAIOption func(*XAIProvider)

// WithXAIBaseURL overrides the default xAI API base URL.
func WithXAIBaseURL(url string) XAIOption {
	return func(p *XAIProvider) {
		p.baseURL = url
	}
}

// NewXAIProvider creates an xAI/Grok provider with the given API key and options.
func NewXAIProvider(apiKey string, opts ...XAIOption) *XAIProvider {
	p := &XAIProvider{
		apiKey:     apiKey,
		baseURL:    "https://api.x.ai/v1",
		httpClient: &http.Client{},
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *XAIProvider) Name() string { return "xai" }

func (p *XAIProvider) Models() []ModelInfo {
	return []ModelInfo{
		{ID: "grok-4-1-fast-latest", Name: "Grok 4.1 Fast", MaxTokens: 131072, InputCost: 3.0, OutputCost: 15.0},
		{ID: "grok-4", Name: "Grok 4", MaxTokens: 131072, InputCost: 3.0, OutputCost: 15.0},
		{ID: "grok-4-heavy", Name: "Grok 4 Heavy", MaxTokens: 131072, InputCost: 5.0, OutputCost: 25.0},
		{ID: "grok-code-fast-1", Name: "Grok Code Fast", MaxTokens: 131072, InputCost: 3.0, OutputCost: 15.0},
		{ID: "grok-3", Name: "Grok 3", MaxTokens: 131072, InputCost: 3.0, OutputCost: 15.0},
		{ID: "grok-3-mini", Name: "Grok 3 Mini", MaxTokens: 131072, InputCost: 0.3, OutputCost: 0.5},
	}
}

func (p *XAIProvider) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	body := buildOpenAIRequest(req, false)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
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

func (p *XAIProvider) Stream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	body := buildOpenAIRequest(req, true)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
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

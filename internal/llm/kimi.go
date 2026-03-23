package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// KimiProvider implements Provider for the Kimi/Moonshot API.
// It uses the OpenAI-compatible chat completions format.
type KimiProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// KimiOption configures a KimiProvider.
type KimiOption func(*KimiProvider)

// WithKimiBaseURL overrides the default Kimi API base URL.
func WithKimiBaseURL(url string) KimiOption {
	return func(p *KimiProvider) {
		p.baseURL = url
	}
}

// NewKimiProvider creates a Kimi/Moonshot provider with the given API key and options.
func NewKimiProvider(apiKey string, opts ...KimiOption) *KimiProvider {
	p := &KimiProvider{
		apiKey:     apiKey,
		baseURL:    "https://api.moonshot.ai/v1",
		httpClient: &http.Client{},
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *KimiProvider) Name() string { return "kimi" }

func (p *KimiProvider) Models() []ModelInfo {
	return []ModelInfo{
		{ID: "kimi-k2.5", Name: "Kimi K2.5", MaxTokens: 256000, InputCost: 0.0, OutputCost: 0.0},
		{ID: "kimi-k2.5-mini", Name: "Kimi K2.5 Mini", MaxTokens: 128000, InputCost: 0.0, OutputCost: 0.0},
	}
}

func (p *KimiProvider) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
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

func (p *KimiProvider) Stream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
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

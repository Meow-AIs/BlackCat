package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// GeminiProvider implements Provider for the Google Gemini API.
// It uses Gemini's OpenAI-compatible chat completions endpoint with
// x-goog-api-key authentication (AI Studio keys).
type GeminiProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// GeminiOption configures a GeminiProvider.
type GeminiOption func(*GeminiProvider)

// WithGeminiBaseURL overrides the default Gemini API base URL.
func WithGeminiBaseURL(url string) GeminiOption {
	return func(p *GeminiProvider) {
		p.baseURL = url
	}
}

// NewGeminiProvider creates a Google Gemini provider with the given API key and options.
// The default base URL points to the Gemini OpenAI-compatible endpoint.
func NewGeminiProvider(apiKey string, opts ...GeminiOption) *GeminiProvider {
	p := &GeminiProvider{
		apiKey:     apiKey,
		baseURL:    "https://generativelanguage.googleapis.com/v1beta/openai",
		httpClient: &http.Client{},
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Name returns the provider identifier.
func (p *GeminiProvider) Name() string { return "gemini" }

// Models returns the list of available Gemini models with pricing info.
func (p *GeminiProvider) Models() []ModelInfo {
	return []ModelInfo{
		{ID: "gemini-2.5-pro", Name: "Gemini 2.5 Pro", MaxTokens: 1000000, InputCost: 1.25, OutputCost: 10.0},
		{ID: "gemini-2.5-flash", Name: "Gemini 2.5 Flash", MaxTokens: 1000000, InputCost: 0.15, OutputCost: 0.60},
		{ID: "gemini-2.0-flash", Name: "Gemini 2.0 Flash", MaxTokens: 1000000, InputCost: 0.10, OutputCost: 0.40},
		{ID: "gemini-2.0-flash-lite", Name: "Gemini 2.0 Flash Lite", MaxTokens: 1000000, InputCost: 0.0, OutputCost: 0.0},
	}
}

// Chat sends a non-streaming chat completion request to Gemini.
// Uses the OpenAI-compatible format with x-goog-api-key authentication.
func (p *GeminiProvider) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
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
	httpReq.Header.Set("x-goog-api-key", p.apiKey)

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

// Stream sends a streaming chat completion request to Gemini.
// Uses SSE format compatible with OpenAI streaming protocol.
func (p *GeminiProvider) Stream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
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
	httpReq.Header.Set("x-goog-api-key", p.apiKey)

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

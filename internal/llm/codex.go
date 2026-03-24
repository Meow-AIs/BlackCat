package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// CodexProvider implements Provider for the OpenAI Codex API (ChatGPT subscription).
// It uses the OpenAI-compatible chat completions format with OAuth device flow auth.
type CodexProvider struct {
	oauth      *OAuthClient
	baseURL    string
	httpClient *http.Client
}

// CodexOption configures a CodexProvider.
type CodexOption func(*CodexProvider)

// WithCodexBaseURL overrides the default Codex API base URL.
func WithCodexBaseURL(url string) CodexOption {
	return func(p *CodexProvider) {
		p.baseURL = url
	}
}

// WithCodexToken sets a pre-existing token, skipping OAuth device flow.
func WithCodexToken(token string) CodexOption {
	return func(p *CodexProvider) {
		p.oauth.SetToken(&OAuthToken{
			AccessToken: token,
			TokenType:   "bearer",
			ExpiresAt:   time.Now().Add(1 * time.Hour).Unix(),
		})
	}
}

// NewCodexProvider creates an OpenAI Codex provider.
func NewCodexProvider(opts ...CodexOption) *CodexProvider {
	p := &CodexProvider{
		oauth:      NewOAuthClient(OpenAICodexOAuth),
		baseURL:    "https://api.openai.com/v1",
		httpClient: &http.Client{},
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Name returns the provider name.
func (p *CodexProvider) Name() string { return "codex" }

// Models returns the list of models available via OpenAI Codex.
func (p *CodexProvider) Models() []ModelInfo {
	return []ModelInfo{
		{ID: "gpt-5.4", Name: "GPT-5.4 (via Codex)", MaxTokens: 128000, InputCost: 0, OutputCost: 0},
		{ID: "gpt-4.1", Name: "GPT-4.1 (via Codex)", MaxTokens: 1000000, InputCost: 0, OutputCost: 0},
		{ID: "o4-mini", Name: "o4 Mini (via Codex)", MaxTokens: 200000, InputCost: 0, OutputCost: 0},
	}
}

// Chat sends a chat request to the OpenAI API and returns the full response.
func (p *CodexProvider) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	token, err := p.getAccessToken()
	if err != nil {
		return ChatResponse{}, err
	}

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
	httpReq.Header.Set("Authorization", "Bearer "+token)

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
		return ChatResponse{}, fmt.Errorf("API error (status %d): %s",
			httpResp.StatusCode, string(respBody))
	}

	var oaiResp openaiResponse
	if err := json.Unmarshal(respBody, &oaiResp); err != nil {
		return ChatResponse{}, fmt.Errorf("parse response: %w", err)
	}

	return parseOpenAIResponse(oaiResp), nil
}

// Stream sends a chat request and returns a channel of streaming chunks.
func (p *CodexProvider) Stream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	token, err := p.getAccessToken()
	if err != nil {
		return nil, err
	}

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
	httpReq.Header.Set("Authorization", "Bearer "+token)

	httpResp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		defer httpResp.Body.Close()
		respBody, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error (status %d): %s",
			httpResp.StatusCode, string(respBody))
	}

	ch := make(chan StreamChunk)
	go func() {
		defer close(ch)
		defer httpResp.Body.Close()
		streamOpenAISSE(ctx, httpResp.Body, ch)
	}()

	return ch, nil
}

// Login initiates the OAuth device flow and returns device code info for display.
func (p *CodexProvider) Login(ctx context.Context) (*DeviceCodeResponse, error) {
	return p.oauth.RequestDeviceCode(ctx)
}

// CompleteLogin polls for the token after the user has authorized the device.
func (p *CodexProvider) CompleteLogin(ctx context.Context, deviceCode string) error {
	_, err := p.oauth.PollForToken(ctx, deviceCode, 5)
	return err
}

// IsAuthenticated returns true if the provider has a valid OAuth token.
func (p *CodexProvider) IsAuthenticated() bool {
	return p.oauth.IsAuthenticated()
}

// getAccessToken returns the current access token or an error if not authenticated.
func (p *CodexProvider) getAccessToken() (string, error) {
	token, err := p.oauth.GetToken()
	if err != nil {
		return "", fmt.Errorf("codex: not authenticated — run login first")
	}
	return token.AccessToken, nil
}

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

// CopilotProvider implements Provider for the GitHub Copilot API.
// It uses the OpenAI-compatible chat completions format with OAuth device flow auth.
type CopilotProvider struct {
	oauth      *OAuthClient
	baseURL    string
	httpClient *http.Client
}

// CopilotOption configures a CopilotProvider.
type CopilotOption func(*CopilotProvider)

// WithCopilotBaseURL overrides the default Copilot API base URL.
func WithCopilotBaseURL(url string) CopilotOption {
	return func(p *CopilotProvider) {
		p.baseURL = url
	}
}

// WithCopilotToken sets a pre-existing token, skipping OAuth device flow.
func WithCopilotToken(token string) CopilotOption {
	return func(p *CopilotProvider) {
		p.oauth.SetToken(&OAuthToken{
			AccessToken: token,
			TokenType:   "bearer",
			ExpiresAt:   time.Now().Add(8 * time.Hour).Unix(),
		})
	}
}

// NewCopilotProvider creates a GitHub Copilot provider.
func NewCopilotProvider(opts ...CopilotOption) *CopilotProvider {
	p := &CopilotProvider{
		oauth:      NewOAuthClient(GitHubCopilotOAuth),
		baseURL:    "https://api.githubcopilot.com",
		httpClient: &http.Client{},
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Name returns the provider name.
func (p *CopilotProvider) Name() string { return "copilot" }

// Models returns the list of models available via GitHub Copilot.
func (p *CopilotProvider) Models() []ModelInfo {
	return []ModelInfo{
		{ID: "gpt-5.4", Name: "GPT-5.4 (via Copilot)", MaxTokens: 128000, InputCost: 0, OutputCost: 0},
		{ID: "gpt-4.1", Name: "GPT-4.1 (via Copilot)", MaxTokens: 1000000, InputCost: 0, OutputCost: 0},
		{ID: "claude-sonnet-4-6", Name: "Claude Sonnet 4.6 (via Copilot)", MaxTokens: 200000, InputCost: 0, OutputCost: 0},
		{ID: "claude-opus-4-6", Name: "Claude Opus 4.6 (via Copilot)", MaxTokens: 200000, InputCost: 0, OutputCost: 0},
		{ID: "o4-mini", Name: "o4 Mini (via Copilot)", MaxTokens: 200000, InputCost: 0, OutputCost: 0},
	}
}

// Chat sends a chat request to the Copilot API and returns the full response.
func (p *CopilotProvider) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
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
func (p *CopilotProvider) Stream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
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
func (p *CopilotProvider) Login(ctx context.Context) (*DeviceCodeResponse, error) {
	return p.oauth.RequestDeviceCode(ctx)
}

// CompleteLogin polls for the token after the user has authorized the device.
func (p *CopilotProvider) CompleteLogin(ctx context.Context, deviceCode string) error {
	_, err := p.oauth.PollForToken(ctx, deviceCode, 5)
	return err
}

// IsAuthenticated returns true if the provider has a valid OAuth token.
func (p *CopilotProvider) IsAuthenticated() bool {
	return p.oauth.IsAuthenticated()
}

// getAccessToken returns the current access token or an error if not authenticated.
func (p *CopilotProvider) getAccessToken() (string, error) {
	token, err := p.oauth.GetToken()
	if err != nil {
		return "", fmt.Errorf("copilot: not authenticated — run login first")
	}
	return token.AccessToken, nil
}

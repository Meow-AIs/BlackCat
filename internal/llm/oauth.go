package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// OAuthConfig holds the configuration for an OAuth device flow.
type OAuthConfig struct {
	ClientID      string
	DeviceCodeURL string // POST to get device code
	TokenURL      string // POST to poll for token
	Scopes        []string
	Audience      string // required by some providers (e.g. OpenAI)
}

// DeviceCodeResponse is returned when initiating the device flow.
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// OAuthToken represents an OAuth access token.
type OAuthToken struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
	ExpiresAt    int64  `json:"expires_at,omitempty"`
}

// oauthErrorResponse is the error shape returned by OAuth endpoints.
type oauthErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// OAuthClient manages OAuth device flow authentication.
type OAuthClient struct {
	config     OAuthConfig
	httpClient *http.Client
	token      *OAuthToken
	mu         sync.RWMutex
}

// GitHubCopilotOAuth is the predefined OAuth config for GitHub Copilot.
var GitHubCopilotOAuth = OAuthConfig{
	ClientID:      "Iv1.b507a08c87ecfe98",
	DeviceCodeURL: "https://github.com/login/device/code",
	TokenURL:      "https://github.com/login/oauth/access_token",
	Scopes:        []string{"read:user"},
}

// OpenAICodexOAuth is the predefined OAuth config for OpenAI Codex.
// NOTE: OpenAI does NOT support device code flow. They use PKCE browser flow only.
// This config is kept for reference. Use API key instead: blackcat config set openai_api_key
var OpenAICodexOAuth = OAuthConfig{
	ClientID:      "app_EMoamEEZ73f0CkXaXp7hrann",
	DeviceCodeURL: "", // OpenAI has no device code endpoint
	TokenURL:      "https://auth.openai.com/oauth/token",
	Scopes:        []string{"openid", "profile", "email", "offline_access"},
	Audience:      "https://api.openai.com/v1",
}

// NewOAuthClient creates a new OAuth client with the given configuration.
func NewOAuthClient(config OAuthConfig) *OAuthClient {
	return &OAuthClient{
		config:     config,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// RequestDeviceCode initiates the device flow by requesting a device code.
func (c *OAuthClient) RequestDeviceCode(ctx context.Context) (*DeviceCodeResponse, error) {
	if c.config.DeviceCodeURL == "" {
		return nil, fmt.Errorf("this provider does not support device code flow.\n" +
			"Use API key instead: blackcat config set openai_api_key sk-...")
	}

	// Build form-encoded request body (GitHub OAuth expects form, not JSON)
	form := url.Values{
		"client_id": {c.config.ClientID},
		"scope":     {strings.Join(c.config.Scopes, " ")},
	}
	if c.config.Audience != "" {
		form.Set("audience", c.config.Audience)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.config.DeviceCodeURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create device code request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send device code request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read device code response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Detect Cloudflare challenge (returns HTML instead of JSON)
		bodyStr := string(body)
		if strings.Contains(bodyStr, "cloudflare") || strings.Contains(bodyStr, "Just a moment") {
			return nil, fmt.Errorf("blocked by Cloudflare protection (status %d).\n"+
				"This provider requires browser-based authentication.\n"+
				"Try one of these alternatives:\n"+
				"  1. Use 'blackcat login copilot' instead (GitHub Copilot works from servers)\n"+
				"  2. Login from your local machine and copy the token\n"+
				"  3. Use an API key: blackcat config set openai_api_key sk-...",
				resp.StatusCode)
		}
		return nil, fmt.Errorf("device code request failed (status %d): %s",
			resp.StatusCode, bodyStr)
	}

	var dcResp DeviceCodeResponse
	if err := json.Unmarshal(body, &dcResp); err != nil {
		return nil, fmt.Errorf("parse device code response: %w", err)
	}

	return &dcResp, nil
}

// PollForToken polls the token endpoint until the user authorizes or an error occurs.
func (c *OAuthClient) PollForToken(ctx context.Context, deviceCode string, interval int) (*OAuthToken, error) {
	if interval <= 0 {
		interval = 1 // minimum 1 second for tests, real usage passes 5+
	}

	form := url.Values{
		"client_id":   {c.config.ClientID},
		"device_code": {deviceCode},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
	}

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		token, done, err := c.pollOnce(ctx, form)
		if err != nil {
			return nil, err
		}
		if done {
			c.mu.Lock()
			c.token = token
			c.mu.Unlock()
			return token, nil
		}

		// Wait before next poll
		timer := time.NewTimer(time.Duration(interval) * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

// pollOnce makes a single token poll request.
// Returns (token, true, nil) on success, (nil, false, nil) on pending, (nil, false, err) on failure.
func (c *OAuthClient) pollOnce(ctx context.Context, form url.Values) (*OAuthToken, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.config.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, false, fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("send token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, fmt.Errorf("read token response: %w", err)
	}

	// Try to parse as error first
	var errResp oauthErrorResponse
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != "" {
		switch errResp.Error {
		case "authorization_pending", "slow_down":
			return nil, false, nil
		default:
			return nil, false, fmt.Errorf("oauth error: %s: %s",
				errResp.Error, errResp.ErrorDescription)
		}
	}

	// Parse as token
	var token OAuthToken
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, false, fmt.Errorf("parse token response: %w", err)
	}

	if token.AccessToken == "" {
		return nil, false, nil
	}

	// Calculate expiry timestamp
	if token.ExpiresIn > 0 && token.ExpiresAt == 0 {
		token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second).Unix()
	}

	return &token, true, nil
}

// GetToken returns the current token. Returns an error if no token is set.
func (c *OAuthClient) GetToken() (*OAuthToken, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.token == nil {
		return nil, fmt.Errorf("no oauth token available")
	}
	return c.token, nil
}

// SetToken sets a pre-existing token (e.g., loaded from storage).
func (c *OAuthClient) SetToken(token *OAuthToken) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = token
}

// IsAuthenticated returns true if a valid, non-expired token is available.
func (c *OAuthClient) IsAuthenticated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.token == nil || c.token.AccessToken == "" {
		return false
	}
	if c.token.ExpiresAt > 0 && time.Now().Unix() >= c.token.ExpiresAt {
		return false
	}
	return true
}

// FormatLoginPrompt returns user-friendly login instructions.
func FormatLoginPrompt(resp *DeviceCodeResponse) string {
	return fmt.Sprintf(
		"To authenticate, visit: %s\nEnter code: %s",
		resp.VerificationURI, resp.UserCode,
	)
}

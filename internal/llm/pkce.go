package llm

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// PKCEConfig holds the configuration for a PKCE browser-based OAuth flow.
type PKCEConfig struct {
	ClientID     string
	AuthorizeURL string // e.g. https://auth.openai.com/oauth/authorize
	TokenURL     string // e.g. https://auth.openai.com/oauth/token
	RedirectPort int    // localhost port for callback (0 = auto-assign)
	Scopes       []string
	Audience     string
}

// PKCEClient manages PKCE browser-based OAuth authentication.
type PKCEClient struct {
	config     PKCEConfig
	httpClient *http.Client
}

// OpenAICodexPKCE is the predefined PKCE config for OpenAI Codex.
// OpenAI does NOT support device code flow; it uses PKCE browser OAuth only.
var OpenAICodexPKCE = PKCEConfig{
	ClientID:     "app_EMoamEEZ73f0CkXaXp7hrann",
	AuthorizeURL: "https://auth.openai.com/oauth/authorize",
	TokenURL:     "https://auth.openai.com/oauth/token",
	RedirectPort: 1455,
	Scopes:       []string{"openid", "profile", "email", "offline_access"},
	Audience:     "https://api.openai.com/v1",
}

// NewPKCEClient creates a new PKCE OAuth client with the given configuration.
func NewPKCEClient(config PKCEConfig) *PKCEClient {
	return &PKCEClient{
		config:     config,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// GenerateVerifier creates a random PKCE code verifier (32 bytes, base64url, no padding).
func GenerateVerifier() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate verifier: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// GenerateChallenge creates an S256 code challenge from a verifier.
func GenerateChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// redirectURI returns the localhost redirect URI for the given port.
func redirectURI(port int) string {
	return fmt.Sprintf("http://localhost:%d/auth/callback", port)
}

// effectivePort returns the configured port, defaulting to 1455 if zero.
func (c *PKCEClient) effectivePort() int {
	if c.config.RedirectPort == 0 {
		return 1455
	}
	return c.config.RedirectPort
}

// BuildAuthorizationURL constructs the full authorization URL with PKCE parameters.
func (c *PKCEClient) BuildAuthorizationURL(verifier string, state string) string {
	challenge := GenerateChallenge(verifier)
	port := c.effectivePort()

	params := url.Values{
		"client_id":             {c.config.ClientID},
		"response_type":         {"code"},
		"redirect_uri":          {redirectURI(port)},
		"scope":                 {strings.Join(c.config.Scopes, " ")},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
		"state":                 {state},
	}
	if c.config.Audience != "" {
		params.Set("audience", c.config.Audience)
	}

	return c.config.AuthorizeURL + "?" + params.Encode()
}

// ExchangeCode exchanges an authorization code for an access token.
func (c *PKCEClient) ExchangeCode(ctx context.Context, code string, verifier string) (*OAuthToken, error) {
	port := c.effectivePort()

	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI(port)},
		"client_id":     {c.config.ClientID},
		"code_verifier": {verifier},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.config.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp oauthErrorResponse
		if jsonErr := json.Unmarshal(body, &errResp); jsonErr == nil && errResp.Error != "" {
			return nil, fmt.Errorf("token exchange failed: %s: %s",
				errResp.Error, errResp.ErrorDescription)
		}
		return nil, fmt.Errorf("token exchange failed (status %d): %s",
			resp.StatusCode, string(body))
	}

	var token OAuthToken
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}

	if token.AccessToken == "" {
		return nil, fmt.Errorf("token response missing access_token")
	}

	if token.ExpiresIn > 0 && token.ExpiresAt == 0 {
		token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second).Unix()
	}

	return &token, nil
}

// callbackListener creates a TCP listener for the OAuth callback server.
// If RedirectPort is 0, an ephemeral port is assigned.
func (c *PKCEClient) callbackListener() (net.Listener, int, error) {
	addr := fmt.Sprintf("localhost:%d", c.config.RedirectPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, 0, fmt.Errorf("listen on %s: %w", addr, err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	return listener, port, nil
}

// StartCallbackServer starts a temporary HTTP server on localhost that waits
// for the OAuth callback. Returns the authorization code and the actual port.
// The portReady channel (if non-nil) receives the port once the server is listening.
func (c *PKCEClient) StartCallbackServer(ctx context.Context, expectedState string) (string, int, error) {
	listener, port, err := c.callbackListener()
	if err != nil {
		return "", 0, err
	}

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/auth/callback", func(w http.ResponseWriter, r *http.Request) {
		state := r.URL.Query().Get("state")
		if state != expectedState {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "<html><body><h2>State mismatch</h2>"+
				"<p>Expected state does not match. Please try again.</p></body></html>")
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "<html><body><h2>Missing code</h2>"+
				"<p>No authorization code received.</p></body></html>")
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<html><body><h2>Authorization successful!</h2>"+
			"<p>You can close this tab and return to BlackCat.</p></body></html>")

		codeCh <- code
	})

	server := &http.Server{Handler: mux}
	go func() {
		if sErr := server.Serve(listener); sErr != nil && sErr != http.ErrServerClosed {
			errCh <- fmt.Errorf("callback server: %w", sErr)
		}
	}()

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	select {
	case code := <-codeCh:
		return code, port, nil
	case sErr := <-errCh:
		return "", port, sErr
	case <-ctx.Done():
		return "", port, ctx.Err()
	}
}

// Login performs the full PKCE OAuth flow:
// 1. Generate verifier + state
// 2. Start callback server on localhost
// 3. Build auth URL
// 4. Wait for callback with authorization code
// 5. Exchange code for token
// Returns: token, authURL (for display/manual use), error.
func (c *PKCEClient) Login(ctx context.Context, openBrowser bool) (*OAuthToken, string, error) {
	verifier, err := GenerateVerifier()
	if err != nil {
		return nil, "", fmt.Errorf("pkce login: %w", err)
	}

	state := fmt.Sprintf("blackcat-%d", time.Now().UnixNano())

	// Create listener first so we know the actual port
	listener, port, err := c.callbackListener()
	if err != nil {
		return nil, "", fmt.Errorf("pkce login: %w", err)
	}

	// Build auth URL with the actual port
	authURL := c.buildAuthURLWithPort(verifier, state, port)

	// Start callback server using the pre-created listener
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/auth/callback", func(w http.ResponseWriter, r *http.Request) {
		cbState := r.URL.Query().Get("state")
		if cbState != state {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "<html><body><h2>State mismatch</h2></body></html>")
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "<html><body><h2>Missing code</h2></body></html>")
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<html><body><h2>Authorization successful!</h2>"+
			"<p>You can close this tab and return to BlackCat.</p></body></html>")
		codeCh <- code
	})

	server := &http.Server{Handler: mux}
	go func() {
		if sErr := server.Serve(listener); sErr != nil && sErr != http.ErrServerClosed {
			errCh <- fmt.Errorf("callback server: %w", sErr)
		}
	}()

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	// Wait for authorization code
	var code string
	select {
	case code = <-codeCh:
	case sErr := <-errCh:
		return nil, authURL, fmt.Errorf("callback server: %w", sErr)
	case <-ctx.Done():
		return nil, authURL, ctx.Err()
	}

	// Exchange code for token
	token, err := c.ExchangeCode(ctx, code, verifier)
	if err != nil {
		return nil, authURL, fmt.Errorf("token exchange: %w", err)
	}

	return token, authURL, nil
}

// buildAuthURLWithPort constructs the auth URL using a specific port.
func (c *PKCEClient) buildAuthURLWithPort(verifier, state string, port int) string {
	challenge := GenerateChallenge(verifier)

	params := url.Values{
		"client_id":             {c.config.ClientID},
		"response_type":         {"code"},
		"redirect_uri":          {redirectURI(port)},
		"scope":                 {strings.Join(c.config.Scopes, " ")},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
		"state":                 {state},
	}
	if c.config.Audience != "" {
		params.Set("audience", c.config.Audience)
	}

	return c.config.AuthorizeURL + "?" + params.Encode()
}

// ExtractCodeFromURL parses a redirect URL and extracts the authorization code and state.
// Used for remote/VPS flow where the user pastes the redirect URL manually.
func ExtractCodeFromURL(redirectURL string) (string, string, error) {
	if redirectURL == "" {
		return "", "", fmt.Errorf("empty redirect URL")
	}

	parsed, err := url.Parse(redirectURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid redirect URL: %w", err)
	}

	code := parsed.Query().Get("code")
	if code == "" {
		return "", "", fmt.Errorf("no authorization code found in URL")
	}

	state := parsed.Query().Get("state")
	return code, state, nil
}

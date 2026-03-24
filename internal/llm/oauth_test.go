package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewOAuthClient(t *testing.T) {
	client := NewOAuthClient(GitHubCopilotOAuth)
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.config.ClientID != GitHubCopilotOAuth.ClientID {
		t.Errorf("expected client ID %q, got %q", GitHubCopilotOAuth.ClientID, client.config.ClientID)
	}
	if client.httpClient == nil {
		t.Error("expected non-nil http client")
	}
	if client.IsAuthenticated() {
		t.Error("expected not authenticated initially")
	}
}

func TestOAuthClient_RequestDeviceCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("expected Accept application/json, got %s", r.Header.Get("Accept"))
		}

		err := r.ParseForm()
		if err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.FormValue("client_id") != "test-client-id" {
			t.Errorf("expected client_id 'test-client-id', got %q", r.FormValue("client_id"))
		}
		if r.FormValue("scope") != "copilot" {
			t.Errorf("expected scope 'copilot', got %q", r.FormValue("scope"))
		}

		resp := DeviceCodeResponse{
			DeviceCode:      "dev-code-123",
			UserCode:        "ABCD-1234",
			VerificationURI: "https://github.com/login/device",
			ExpiresIn:       900,
			Interval:        5,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := OAuthConfig{
		ClientID:      "test-client-id",
		DeviceCodeURL: server.URL,
		TokenURL:      server.URL + "/token",
		Scopes:        []string{"copilot"},
	}
	client := NewOAuthClient(config)

	resp, err := client.RequestDeviceCode(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.DeviceCode != "dev-code-123" {
		t.Errorf("expected device code 'dev-code-123', got %q", resp.DeviceCode)
	}
	if resp.UserCode != "ABCD-1234" {
		t.Errorf("expected user code 'ABCD-1234', got %q", resp.UserCode)
	}
	if resp.VerificationURI != "https://github.com/login/device" {
		t.Errorf("expected verification URI, got %q", resp.VerificationURI)
	}
	if resp.ExpiresIn != 900 {
		t.Errorf("expected expires_in 900, got %d", resp.ExpiresIn)
	}
	if resp.Interval != 5 {
		t.Errorf("expected interval 5, got %d", resp.Interval)
	}
}

func TestOAuthClient_RequestDeviceCode_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid_client"}`))
	}))
	defer server.Close()

	config := OAuthConfig{
		ClientID:      "bad-client",
		DeviceCodeURL: server.URL,
		TokenURL:      server.URL + "/token",
		Scopes:        []string{"copilot"},
	}
	client := NewOAuthClient(config)

	_, err := client.RequestDeviceCode(context.Background())
	if err == nil {
		t.Fatal("expected error for bad request")
	}
}

func TestOAuthClient_PollForToken_Success(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		if callCount < 3 {
			// First two polls return authorization_pending
			json.NewEncoder(w).Encode(map[string]string{
				"error": "authorization_pending",
			})
			return
		}

		// Third poll returns a token
		json.NewEncoder(w).Encode(OAuthToken{
			AccessToken: "gho_test_token_abc",
			TokenType:   "bearer",
			ExpiresIn:   28800,
			Scope:       "copilot",
		})
	}))
	defer server.Close()

	config := OAuthConfig{
		ClientID:      "test-client",
		DeviceCodeURL: server.URL + "/device",
		TokenURL:      server.URL,
		Scopes:        []string{"copilot"},
	}
	client := NewOAuthClient(config)

	// Use interval=0 so test doesn't actually wait
	token, err := client.PollForToken(context.Background(), "dev-code-123", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token.AccessToken != "gho_test_token_abc" {
		t.Errorf("expected access token 'gho_test_token_abc', got %q", token.AccessToken)
	}
	if token.TokenType != "bearer" {
		t.Errorf("expected token type 'bearer', got %q", token.TokenType)
	}
	if callCount != 3 {
		t.Errorf("expected 3 poll attempts, got %d", callCount)
	}
}

func TestOAuthClient_PollForToken_Denied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "access_denied",
			"error_description": "user denied access",
		})
	}))
	defer server.Close()

	config := OAuthConfig{
		ClientID:      "test-client",
		DeviceCodeURL: server.URL + "/device",
		TokenURL:      server.URL,
		Scopes:        []string{"copilot"},
	}
	client := NewOAuthClient(config)

	_, err := client.PollForToken(context.Background(), "dev-code-123", 0)
	if err == nil {
		t.Fatal("expected error for access denied")
	}
	if !strings.Contains(err.Error(), "access_denied") {
		t.Errorf("expected access_denied in error, got: %s", err.Error())
	}
}

func TestOAuthClient_PollForToken_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"error": "authorization_pending",
		})
	}))
	defer server.Close()

	config := OAuthConfig{
		ClientID:      "test-client",
		DeviceCodeURL: server.URL + "/device",
		TokenURL:      server.URL,
		Scopes:        []string{"copilot"},
	}
	client := NewOAuthClient(config)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := client.PollForToken(ctx, "dev-code-123", 0)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestOAuthClient_SetGetToken(t *testing.T) {
	client := NewOAuthClient(GitHubCopilotOAuth)

	token := &OAuthToken{
		AccessToken: "gho_existing_token",
		TokenType:   "bearer",
		ExpiresIn:   28800,
		ExpiresAt:   time.Now().Add(8 * time.Hour).Unix(),
	}
	client.SetToken(token)

	got, err := client.GetToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.AccessToken != "gho_existing_token" {
		t.Errorf("expected 'gho_existing_token', got %q", got.AccessToken)
	}
}

func TestOAuthClient_IsAuthenticated(t *testing.T) {
	client := NewOAuthClient(GitHubCopilotOAuth)

	if client.IsAuthenticated() {
		t.Error("expected not authenticated without token")
	}

	// Set an expired token
	client.SetToken(&OAuthToken{
		AccessToken: "expired-token",
		TokenType:   "bearer",
		ExpiresAt:   time.Now().Add(-1 * time.Hour).Unix(),
	})
	if client.IsAuthenticated() {
		t.Error("expected not authenticated with expired token")
	}

	// Set a valid token
	client.SetToken(&OAuthToken{
		AccessToken: "valid-token",
		TokenType:   "bearer",
		ExpiresAt:   time.Now().Add(1 * time.Hour).Unix(),
	})
	if !client.IsAuthenticated() {
		t.Error("expected authenticated with valid token")
	}
}

func TestOAuthClient_GetToken_NoToken(t *testing.T) {
	client := NewOAuthClient(GitHubCopilotOAuth)

	_, err := client.GetToken()
	if err == nil {
		t.Fatal("expected error when no token set")
	}
}

func TestFormatLoginPrompt(t *testing.T) {
	resp := &DeviceCodeResponse{
		UserCode:        "ABCD-1234",
		VerificationURI: "https://github.com/login/device",
	}

	prompt := FormatLoginPrompt(resp)
	if !strings.Contains(prompt, "ABCD-1234") {
		t.Error("expected user code in prompt")
	}
	if !strings.Contains(prompt, "https://github.com/login/device") {
		t.Error("expected verification URI in prompt")
	}
}

func TestOAuthConfigs(t *testing.T) {
	if GitHubCopilotOAuth.ClientID == "" {
		t.Error("expected non-empty Copilot client ID")
	}
	if GitHubCopilotOAuth.DeviceCodeURL == "" {
		t.Error("expected non-empty Copilot device code URL")
	}
	if GitHubCopilotOAuth.TokenURL == "" {
		t.Error("expected non-empty Copilot token URL")
	}

	if OpenAICodexOAuth.ClientID == "" {
		t.Error("expected non-empty Codex client ID")
	}
	if OpenAICodexOAuth.DeviceCodeURL != "" {
		t.Error("expected empty Codex device code URL (OpenAI has no device code flow)")
	}
	if OpenAICodexOAuth.TokenURL == "" {
		t.Error("expected non-empty Codex token URL")
	}
}

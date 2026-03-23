package builtin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebFetchToolInfo(t *testing.T) {
	tool := NewWebFetchTool()
	info := tool.Info()
	if info.Name != "web_fetch" {
		t.Errorf("expected name 'web_fetch', got %q", info.Name)
	}
	if info.Category != "web" {
		t.Errorf("expected category 'web', got %q", info.Category)
	}
}

func TestWebFetchToolGET(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("hello from server"))
	}))
	defer server.Close()

	tool := NewWebFetchTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"url": server.URL,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d; error: %s", result.ExitCode, result.Error)
	}
	if !strings.Contains(result.Output, "hello from server") {
		t.Errorf("expected output to contain 'hello from server', got %q", result.Output)
	}
}

func TestWebFetchToolPOST(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	tool := NewWebFetchTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"url":    server.URL,
		"method": "POST",
		"headers": map[string]any{
			"Content-Type": "application/json",
		},
		"body": `{"key":"value"}`,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d; error: %s", result.ExitCode, result.Error)
	}
	if !strings.Contains(result.Output, "ok") {
		t.Errorf("expected output to contain 'ok', got %q", result.Output)
	}
}

func TestWebFetchToolCustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("expected Authorization header 'Bearer test-token', got %q", auth)
		}
		w.Write([]byte("authorized"))
	}))
	defer server.Close()

	tool := NewWebFetchTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"url": server.URL,
		"headers": map[string]any{
			"Authorization": "Bearer test-token",
		},
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(result.Output, "authorized") {
		t.Errorf("expected 'authorized', got %q", result.Output)
	}
}

func TestWebFetchToolErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	tool := NewWebFetchTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"url": server.URL,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	// Should still return the body but with the status code
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code for 404 response")
	}
}

func TestWebFetchToolMissingURL(t *testing.T) {
	tool := NewWebFetchTool()
	_, err := tool.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Error("expected error for missing 'url' argument")
	}
}

func TestWebFetchToolInvalidURL(t *testing.T) {
	tool := NewWebFetchTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"url": "http://invalid.localhost.test:1/nope",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code for unreachable URL")
	}
}

func TestWebSearchToolInfo(t *testing.T) {
	tool := NewWebSearchTool()
	info := tool.Info()
	if info.Name != "web_search" {
		t.Errorf("expected name 'web_search', got %q", info.Name)
	}
}

func TestWebSearchToolPlaceholder(t *testing.T) {
	tool := NewWebSearchTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"query": "test query",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(result.Output, "not implemented") {
		t.Errorf("expected 'not implemented' placeholder, got %q", result.Output)
	}
}

func TestWebSearchToolMissingQuery(t *testing.T) {
	tool := NewWebSearchTool()
	_, err := tool.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Error("expected error for missing 'query' argument")
	}
}

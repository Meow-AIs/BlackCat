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
	if info.Category != "web" {
		t.Errorf("expected category 'web', got %q", info.Category)
	}
}

// TestWebSearchToolDuckDuckGo verifies that WebSearchTool calls the DuckDuckGo
// instant-answer API and returns formatted results from the JSON response.
func TestWebSearchToolDuckDuckGo(t *testing.T) {
	// Mock DuckDuckGo instant-answer API response
	mockResponse := map[string]any{
		"AbstractText":  "Go is a statically typed, compiled programming language designed at Google.",
		"AbstractURL":   "https://en.wikipedia.org/wiki/Go_(programming_language)",
		"AbstractTitle": "Go (programming language)",
		"RelatedTopics": []any{
			map[string]any{
				"Text":     "Go compiler - The Go compiler is a tool for compiling Go source code.",
				"FirstURL": "https://example.com/go-compiler",
			},
			map[string]any{
				"Text":     "Go modules - Module system for Go dependency management.",
				"FirstURL": "https://example.com/go-modules",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("format") != "json" {
			t.Errorf("expected format=json query param, got %q", r.URL.Query().Get("format"))
		}
		if r.URL.Query().Get("q") == "" {
			t.Error("expected non-empty q query param")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	tool := newWebSearchToolWithBase(server.URL)
	result, err := tool.Execute(context.Background(), map[string]any{
		"query": "golang programming",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d; error: %s", result.ExitCode, result.Error)
	}
	// Should not say "not implemented"
	if strings.Contains(result.Output, "not implemented") {
		t.Errorf("expected real results, got placeholder: %q", result.Output)
	}
	// Should contain the abstract text
	if !strings.Contains(result.Output, "statically typed") {
		t.Errorf("expected abstract text in output, got %q", result.Output)
	}
	// Should contain URL
	if !strings.Contains(result.Output, "wikipedia.org") {
		t.Errorf("expected abstract URL in output, got %q", result.Output)
	}
}

// TestWebSearchToolDuckDuckGoRelatedTopics verifies that related topics are included.
func TestWebSearchToolDuckDuckGoRelatedTopics(t *testing.T) {
	mockResponse := map[string]any{
		"AbstractText":  "",
		"AbstractURL":   "",
		"AbstractTitle": "",
		"RelatedTopics": []any{
			map[string]any{
				"Text":     "Topic one about something interesting.",
				"FirstURL": "https://example.com/topic-one",
			},
			map[string]any{
				"Text":     "Topic two about another thing.",
				"FirstURL": "https://example.com/topic-two",
			},
			map[string]any{
				"Text":     "Topic three about yet more things.",
				"FirstURL": "https://example.com/topic-three",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	tool := newWebSearchToolWithBase(server.URL)
	result, err := tool.Execute(context.Background(), map[string]any{
		"query": "test",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "Topic one") {
		t.Errorf("expected related topic in output, got %q", result.Output)
	}
	if !strings.Contains(result.Output, "example.com/topic-one") {
		t.Errorf("expected related topic URL in output, got %q", result.Output)
	}
}

// TestWebSearchToolEmptyResponse verifies graceful handling when no results are found.
func TestWebSearchToolEmptyResponse(t *testing.T) {
	mockResponse := map[string]any{
		"AbstractText":  "",
		"AbstractURL":   "",
		"AbstractTitle": "",
		"RelatedTopics": []any{},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	tool := newWebSearchToolWithBase(server.URL)
	result, err := tool.Execute(context.Background(), map[string]any{
		"query": "xyzzy-nonexistent-query-12345",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	// Should indicate no results found, not crash
	if result.Output == "" {
		t.Error("expected non-empty output even for empty results")
	}
}

// TestWebSearchToolAPIError verifies graceful handling of HTTP errors from the API.
func TestWebSearchToolAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	tool := newWebSearchToolWithBase(server.URL)
	result, err := tool.Execute(context.Background(), map[string]any{
		"query": "test query",
	})
	if err != nil {
		t.Fatalf("Execute should not return error for HTTP errors, got: %v", err)
	}
	// Should signal failure via ExitCode
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code for API error response")
	}
}

// TestWebSearchToolInvalidJSON verifies graceful handling of malformed JSON responses.
func TestWebSearchToolInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("this is not valid json {{{"))
	}))
	defer server.Close()

	tool := newWebSearchToolWithBase(server.URL)
	result, err := tool.Execute(context.Background(), map[string]any{
		"query": "test query",
	})
	if err != nil {
		t.Fatalf("Execute should not return error for bad JSON, got: %v", err)
	}
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code for invalid JSON response")
	}
}

// TestWebSearchToolMissingQuery verifies that missing query arg returns an error.
func TestWebSearchToolMissingQuery(t *testing.T) {
	tool := NewWebSearchTool()
	_, err := tool.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Error("expected error for missing 'query' argument")
	}
}

// TestWebSearchToolQueryEncoding verifies the query is properly URL-encoded.
func TestWebSearchToolQueryEncoding(t *testing.T) {
	receivedQuery := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.Query().Get("q")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"AbstractText":  "result",
			"AbstractURL":   "https://example.com",
			"RelatedTopics": []any{},
		})
	}))
	defer server.Close()

	tool := newWebSearchToolWithBase(server.URL)
	_, err := tool.Execute(context.Background(), map[string]any{
		"query": "hello world & special chars",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if receivedQuery != "hello world & special chars" {
		t.Errorf("expected query to be properly decoded by server, got %q", receivedQuery)
	}
}

// TestWebSearchToolLimitsResults verifies at most 5 related topics are returned.
func TestWebSearchToolLimitsResults(t *testing.T) {
	topics := make([]any, 10)
	for i := range topics {
		topics[i] = map[string]any{
			"Text":     "Topic entry",
			"FirstURL": "https://example.com/topic",
		}
	}
	mockResponse := map[string]any{
		"AbstractText":  "",
		"AbstractURL":   "",
		"RelatedTopics": topics,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	tool := newWebSearchToolWithBase(server.URL)
	result, err := tool.Execute(context.Background(), map[string]any{
		"query": "test",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	// Count occurrences of "example.com/topic" — should be at most 5
	count := strings.Count(result.Output, "example.com/topic")
	if count > 5 {
		t.Errorf("expected at most 5 results, got %d", count)
	}
}

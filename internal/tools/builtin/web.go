package builtin

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/meowai/blackcat/internal/tools"
)

const (
	defaultTimeout   = 30 * time.Second
	maxResponseBytes = 1024 * 1024 // 1MB
)

// --- WebFetchTool ---

// WebFetchTool makes HTTP requests and returns the response.
type WebFetchTool struct {
	client *http.Client
}

// NewWebFetchTool creates a new web fetch tool with a default 30s timeout.
func NewWebFetchTool() *WebFetchTool {
	return &WebFetchTool{
		client: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

func (t *WebFetchTool) Info() tools.Definition {
	return tools.Definition{
		Name:        "web_fetch",
		Description: "Make an HTTP request and return the response",
		Category:    "web",
		Parameters: []tools.Parameter{
			{Name: "url", Type: "string", Description: "URL to fetch", Required: true},
			{Name: "method", Type: "string", Description: "HTTP method (default: GET)"},
			{Name: "headers", Type: "object", Description: "Request headers as key-value pairs"},
			{Name: "body", Type: "string", Description: "Request body"},
		},
	}
}

func (t *WebFetchTool) Execute(ctx context.Context, args map[string]any) (tools.Result, error) {
	url, err := requireStringArg(args, "url")
	if err != nil {
		return tools.Result{}, err
	}

	method := "GET"
	if m, ok := args["method"].(string); ok && m != "" {
		method = strings.ToUpper(m)
	}

	var bodyReader io.Reader
	if body, ok := args["body"].(string); ok && body != "" {
		bodyReader = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return tools.Result{Error: err.Error(), ExitCode: 1}, nil
	}

	// Apply headers
	if headers, ok := args["headers"].(map[string]any); ok {
		for k, v := range headers {
			if s, ok := v.(string); ok {
				req.Header.Set(k, s)
			}
		}
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return tools.Result{Error: err.Error(), ExitCode: 1}, nil
	}
	defer resp.Body.Close()

	// Read response body with size limit
	limitedReader := io.LimitReader(resp.Body, maxResponseBytes)
	bodyBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		return tools.Result{Error: err.Error(), ExitCode: 1}, nil
	}

	output := fmt.Sprintf("HTTP %d %s\n\n%s", resp.StatusCode, resp.Status, string(bodyBytes))

	exitCode := 0
	if resp.StatusCode >= 400 {
		exitCode = 1
	}

	return tools.Result{Output: output, ExitCode: exitCode}, nil
}

// --- WebSearchTool ---

// WebSearchTool is a placeholder for web search functionality.
type WebSearchTool struct{}

// NewWebSearchTool creates a new web search tool.
func NewWebSearchTool() *WebSearchTool { return &WebSearchTool{} }

func (t *WebSearchTool) Info() tools.Definition {
	return tools.Definition{
		Name:        "web_search",
		Description: "Search the web (placeholder - requires API key)",
		Category:    "web",
		Parameters: []tools.Parameter{
			{Name: "query", Type: "string", Description: "Search query", Required: true},
			{Name: "engine", Type: "string", Description: "Search engine (default: google)", Enum: []string{"google", "bing", "duckduckgo"}},
		},
	}
}

func (t *WebSearchTool) Execute(_ context.Context, args map[string]any) (tools.Result, error) {
	_, err := requireStringArg(args, "query")
	if err != nil {
		return tools.Result{}, err
	}

	engine := "google"
	if e, ok := args["engine"].(string); ok && e != "" {
		engine = e
	}

	return tools.Result{
		Output:   fmt.Sprintf("web search not implemented (engine: %s). Configure an API key to enable search.", engine),
		ExitCode: 0,
	}, nil
}

package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/meowai/blackcat/internal/tools"
)

const (
	defaultTimeout   = 30 * time.Second
	maxResponseBytes = 1024 * 1024 // 1MB

	// duckduckgoAPIBase is the DuckDuckGo instant-answer API endpoint.
	// It can be overridden in tests via newWebSearchToolWithBase.
	duckduckgoAPIBase = "https://api.duckduckgo.com"

	// maxSearchResults is the maximum number of result entries to include.
	maxSearchResults = 5
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
	fetchURL, err := requireStringArg(args, "url")
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

	req, err := http.NewRequestWithContext(ctx, method, fetchURL, bodyReader)
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

// duckduckgoResponse is the JSON structure returned by the DuckDuckGo instant-answer API.
type duckduckgoResponse struct {
	AbstractText  string               `json:"AbstractText"`
	AbstractURL   string               `json:"AbstractURL"`
	AbstractTitle string               `json:"AbstractTitle"`
	RelatedTopics []duckduckgoTopicRaw `json:"RelatedTopics"`
}

// duckduckgoTopicRaw is a flat related-topic entry from the DuckDuckGo instant-answer API.
type duckduckgoTopicRaw struct {
	Text     string `json:"Text"`
	FirstURL string `json:"FirstURL"`
}

// WebSearchTool performs web searches via DuckDuckGo's instant-answer API.
type WebSearchTool struct {
	client  *http.Client
	apiBase string // overridable for testing
}

// NewWebSearchTool creates a new web search tool backed by DuckDuckGo.
func NewWebSearchTool() *WebSearchTool {
	return newWebSearchToolWithBase(duckduckgoAPIBase)
}

// newWebSearchToolWithBase creates a WebSearchTool that targets the given base URL.
// This is used in tests to point at a mock server.
func newWebSearchToolWithBase(apiBase string) *WebSearchTool {
	return &WebSearchTool{
		client: &http.Client{
			Timeout: defaultTimeout,
		},
		apiBase: apiBase,
	}
}

func (t *WebSearchTool) Info() tools.Definition {
	return tools.Definition{
		Name:        "web_search",
		Description: "Search the web using DuckDuckGo instant answers (no API key required)",
		Category:    "web",
		Parameters: []tools.Parameter{
			{Name: "query", Type: "string", Description: "Search query", Required: true},
			{Name: "engine", Type: "string", Description: "Search engine (default: duckduckgo)", Enum: []string{"duckduckgo"}},
		},
	}
}

func (t *WebSearchTool) Execute(ctx context.Context, args map[string]any) (tools.Result, error) {
	query, err := requireStringArg(args, "query")
	if err != nil {
		return tools.Result{}, err
	}

	return t.searchDuckDuckGo(ctx, query)
}

// searchDuckDuckGo queries the DuckDuckGo instant-answer API and returns formatted results.
func (t *WebSearchTool) searchDuckDuckGo(ctx context.Context, query string) (tools.Result, error) {
	apiURL := fmt.Sprintf("%s/?q=%s&format=json&no_html=1&skip_disambig=1",
		t.apiBase, url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return tools.Result{Error: err.Error(), ExitCode: 1}, nil
	}
	req.Header.Set("User-Agent", "BlackCat/1.0 (github.com/meowai/blackcat)")

	resp, err := t.client.Do(req)
	if err != nil {
		return tools.Result{Error: err.Error(), ExitCode: 1}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return tools.Result{
			Error:    fmt.Sprintf("DuckDuckGo API returned HTTP %d", resp.StatusCode),
			ExitCode: 1,
		}, nil
	}

	limitedReader := io.LimitReader(resp.Body, maxResponseBytes)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return tools.Result{Error: err.Error(), ExitCode: 1}, nil
	}

	var ddg duckduckgoResponse
	if err := json.Unmarshal(body, &ddg); err != nil {
		return tools.Result{
			Error:    fmt.Sprintf("failed to parse DuckDuckGo response: %v", err),
			ExitCode: 1,
		}, nil
	}

	return tools.Result{Output: formatDDGResults(query, ddg), ExitCode: 0}, nil
}

// formatDDGResults builds a human-readable string from a DuckDuckGo response.
func formatDDGResults(query string, ddg duckduckgoResponse) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "Search results for: %s\n\n", query)

	resultCount := 0

	// Include the abstract if present (counts as one result)
	if ddg.AbstractText != "" {
		resultCount++
		if ddg.AbstractTitle != "" {
			fmt.Fprintf(&sb, "%d. %s\n", resultCount, ddg.AbstractTitle)
		}
		fmt.Fprintf(&sb, "   %s\n", ddg.AbstractText)
		if ddg.AbstractURL != "" {
			fmt.Fprintf(&sb, "   URL: %s\n", ddg.AbstractURL)
		}
		sb.WriteString("\n")
	}

	// Include related topics up to maxSearchResults total
	for _, topic := range ddg.RelatedTopics {
		if resultCount >= maxSearchResults {
			break
		}
		if topic.Text == "" {
			continue
		}
		resultCount++
		fmt.Fprintf(&sb, "%d. %s\n", resultCount, topic.Text)
		if topic.FirstURL != "" {
			fmt.Fprintf(&sb, "   URL: %s\n", topic.FirstURL)
		}
		sb.WriteString("\n")
	}

	if resultCount == 0 {
		sb.WriteString("No results found.\n")
	}

	return sb.String()
}

package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/meowai/blackcat/pkg/jsonrpc"
)

// sensitiveEnvKeywords mirrors the same list used by the sandbox executor.
// They are duplicated here to keep the mcp package free of a direct import
// cycle with the security package.
var sensitiveEnvKeywords = []string{
	"SECRET", "KEY", "TOKEN", "PASSWORD", "PASSWD", "PASS",
	"CREDENTIAL", "AUTH", "API_", "PRIVATE", "SIGNING",
}

var safeEnvKeyPrefixes = []string{
	"PATH", "HOME", "USER", "LANG", "LC_", "TERM", "SHELL", "EDITOR",
	"TMPDIR", "TMP", "TEMP", "XDG_", "GOPATH", "GOROOT",
}

// filteredEnvForMCP returns a stripped environment suitable for passing to an
// MCP server subprocess — same logic as security.filterEnvironment.
func filteredEnvForMCP() []string {
	raw := os.Environ()
	out := make([]string, 0, len(raw))
	for _, entry := range raw {
		idx := strings.IndexByte(entry, '=')
		if idx < 0 {
			continue
		}
		upper := strings.ToUpper(entry[:idx])

		safe := false
		for _, pfx := range safeEnvKeyPrefixes {
			if strings.HasPrefix(upper, pfx) {
				safe = true
				break
			}
		}
		if safe {
			out = append(out, entry)
			continue
		}

		sensitive := false
		for _, kw := range sensitiveEnvKeywords {
			if strings.Contains(upper, kw) {
				sensitive = true
				break
			}
		}
		if !sensitive {
			out = append(out, entry)
		}
	}
	return out
}

// MCPTool is a tool discovered from an MCP server.
type MCPTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

// Client communicates with an MCP server over stdio.
type Client struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	scanner *bufio.Scanner
	mu      sync.Mutex
	reqID   atomic.Int64
}

// NewStdioClient spawns an MCP server process and connects via stdin/stdout.
func NewStdioClient(command string, args []string) (*Client, error) {
	cmd := exec.Command(command, args...)

	// Strip sensitive environment variables before launching the MCP server so
	// that credentials present in the parent process are not inherited by the
	// child.  We apply the same filtering logic used by the sandbox executor.
	cmd.Env = filteredEnvForMCP()

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start server: %w", err)
	}

	return &Client{
		cmd:     cmd,
		stdin:   stdin,
		scanner: bufio.NewScanner(stdout),
	}, nil
}

// NewClientFromIO creates a client from existing reader/writer (for testing).
func NewClientFromIO(r io.Reader, w io.WriteCloser) *Client {
	return &Client{
		stdin:   w,
		scanner: bufio.NewScanner(r),
	}
}

// Call sends a JSON-RPC request and returns the response.
func (c *Client) Call(ctx context.Context, method string, params any) (jsonrpc.Response, error) {
	id := c.reqID.Add(1)
	req, err := jsonrpc.NewRequest(id, method, params)
	if err != nil {
		return jsonrpc.Response{}, err
	}

	data, err := json.Marshal(req)
	if err != nil {
		return jsonrpc.Response{}, err
	}

	c.mu.Lock()
	_, err = fmt.Fprintf(c.stdin, "%s\n", data)
	c.mu.Unlock()
	if err != nil {
		return jsonrpc.Response{}, fmt.Errorf("write request: %w", err)
	}

	// Read response line
	if !c.scanner.Scan() {
		if err := c.scanner.Err(); err != nil {
			return jsonrpc.Response{}, fmt.Errorf("read response: %w", err)
		}
		return jsonrpc.Response{}, fmt.Errorf("server closed connection")
	}

	var resp jsonrpc.Response
	if err := json.Unmarshal(c.scanner.Bytes(), &resp); err != nil {
		return jsonrpc.Response{}, fmt.Errorf("parse response: %w", err)
	}

	return resp, nil
}

// Initialize sends the initialize handshake to the MCP server.
func (c *Client) Initialize(ctx context.Context) error {
	_, err := c.Call(ctx, "initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]string{
			"name":    "blackcat",
			"version": "0.1.0",
		},
	})
	return err
}

// ListTools retrieves available tools from the MCP server.
func (c *Client) ListTools(ctx context.Context) ([]MCPTool, error) {
	resp, err := c.Call(ctx, "tools/list", nil)
	if err != nil {
		return nil, err
	}

	type listResult struct {
		Tools []MCPTool `json:"tools"`
	}
	result, err := jsonrpc.ParseResult[listResult](resp)
	if err != nil {
		return nil, err
	}
	return result.Tools, nil
}

// CallTool invokes a tool on the MCP server.
func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (string, error) {
	resp, err := c.Call(ctx, "tools/call", map[string]any{
		"name":      name,
		"arguments": args,
	})
	if err != nil {
		return "", err
	}

	type callResult struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	result, err := jsonrpc.ParseResult[callResult](resp)
	if err != nil {
		return "", err
	}
	if len(result.Content) > 0 {
		return result.Content[0].Text, nil
	}
	return "", nil
}

// Close terminates the MCP server process.
func (c *Client) Close() error {
	c.stdin.Close()
	if c.cmd != nil {
		return c.cmd.Wait()
	}
	return nil
}

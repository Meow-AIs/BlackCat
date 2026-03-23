package plugin

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
)

// PluginRequest is sent from BlackCat to the plugin process via stdin.
type PluginRequest struct {
	ID     string         `json:"id"`
	Method string         `json:"method"`
	Params map[string]any `json:"params,omitempty"`
}

// PluginResponse is sent from the plugin process back via stdout.
type PluginResponse struct {
	ID     string `json:"id"`
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

// PluginClient communicates with a running plugin via stdin/stdout pipes.
type PluginClient struct {
	stdin   io.WriteCloser
	scanner *bufio.Scanner
	mu      sync.Mutex
	nextID  atomic.Int64

	// pending holds channels waiting for responses by request ID.
	pending   map[string]chan PluginResponse
	pendingMu sync.Mutex

	done chan struct{}
}

// NewPluginClient creates a client that writes requests to stdin and reads
// responses from stdout.
func NewPluginClient(stdin io.WriteCloser, stdout io.Reader) *PluginClient {
	c := &PluginClient{
		stdin:   stdin,
		scanner: bufio.NewScanner(stdout),
		pending: make(map[string]chan PluginResponse),
		done:    make(chan struct{}),
	}

	go c.readLoop()
	return c
}

// readLoop continuously reads JSON lines from stdout and dispatches to pending callers.
func (c *PluginClient) readLoop() {
	defer close(c.done)

	for c.scanner.Scan() {
		line := c.scanner.Bytes()

		var resp PluginResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			continue
		}

		c.pendingMu.Lock()
		ch, ok := c.pending[resp.ID]
		if ok {
			delete(c.pending, resp.ID)
		}
		c.pendingMu.Unlock()

		if ok {
			ch <- resp
		}
	}
}

// Call sends a request and blocks until the response arrives.
func (c *PluginClient) Call(method string, params map[string]any) (any, error) {
	return c.CallWithTimeout(context.Background(), method, params)
}

// CallWithTimeout sends a request and waits for the response or context cancellation.
func (c *PluginClient) CallWithTimeout(ctx context.Context, method string, params map[string]any) (any, error) {
	id := fmt.Sprintf("req-%d", c.nextID.Add(1))

	req := PluginRequest{
		ID:     id,
		Method: method,
		Params: params,
	}

	// Register pending channel before sending to avoid race
	ch := make(chan PluginResponse, 1)
	c.pendingMu.Lock()
	c.pending[id] = ch
	c.pendingMu.Unlock()

	// Send the request
	c.mu.Lock()
	data, err := json.Marshal(req)
	if err != nil {
		c.mu.Unlock()
		c.removePending(id)
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	data = append(data, '\n')
	_, err = c.stdin.Write(data)
	c.mu.Unlock()

	if err != nil {
		c.removePending(id)
		return nil, fmt.Errorf("write request: %w", err)
	}

	// Wait for response or context cancellation
	select {
	case resp := <-ch:
		if resp.Error != "" {
			return nil, fmt.Errorf("plugin error: %s", resp.Error)
		}
		return resp.Result, nil
	case <-ctx.Done():
		c.removePending(id)
		return nil, ctx.Err()
	case <-c.done:
		c.removePending(id)
		return nil, fmt.Errorf("plugin connection closed")
	}
}

// Ping sends a health-check ping and expects "pong" back.
func (c *PluginClient) Ping() error {
	result, err := c.Call("ping", nil)
	if err != nil {
		return err
	}
	if s, ok := result.(string); !ok || s != "pong" {
		return fmt.Errorf("unexpected ping response: %v", result)
	}
	return nil
}

// Close shuts down the client by closing the stdin pipe.
func (c *PluginClient) Close() error {
	return c.stdin.Close()
}

// removePending cleans up a pending request channel.
func (c *PluginClient) removePending(id string) {
	c.pendingMu.Lock()
	delete(c.pending, id)
	c.pendingMu.Unlock()
}

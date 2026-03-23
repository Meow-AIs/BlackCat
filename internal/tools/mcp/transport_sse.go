package mcp

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// SSETransport communicates with an MCP server over Server-Sent Events (SSE).
type SSETransport struct {
	baseURL    string
	httpClient *http.Client
	events     chan []byte
	done       chan struct{}
	closeOnce  sync.Once
	resp       *http.Response
}

// NewSSETransport creates a new SSE transport for the given base URL.
func NewSSETransport(baseURL string) *SSETransport {
	return &SSETransport{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 0, // no timeout for SSE streaming
		},
		events: make(chan []byte, 64),
		done:   make(chan struct{}),
	}
}

// Connect establishes the SSE event stream connection.
func (t *SSETransport) Connect() error {
	req, err := http.NewRequest(http.MethodGet, t.baseURL, nil)
	if err != nil {
		return fmt.Errorf("create SSE request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("connect to SSE: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return fmt.Errorf("SSE server returned status %d", resp.StatusCode)
	}

	t.resp = resp

	// Start reading events in background
	go t.readEvents(resp.Body)

	return nil
}

// readEvents reads SSE events from the response body and pushes them onto the events channel.
func (t *SSETransport) readEvents(body io.Reader) {
	defer close(t.events)

	scanner := bufio.NewScanner(body)
	var dataLines []string

	for scanner.Scan() {
		line := scanner.Text()

		select {
		case <-t.done:
			return
		default:
		}

		if strings.HasPrefix(line, "data: ") {
			dataLines = append(dataLines, strings.TrimPrefix(line, "data: "))
		} else if line == "" && len(dataLines) > 0 {
			// Empty line signals end of event
			data := strings.Join(dataLines, "\n")
			dataLines = nil

			select {
			case t.events <- []byte(data):
			case <-t.done:
				return
			}
		}
	}

	// Flush any remaining data
	if len(dataLines) > 0 {
		data := strings.Join(dataLines, "\n")
		select {
		case t.events <- []byte(data):
		case <-t.done:
		}
	}
}

// Send posts a JSON-RPC message to the server endpoint.
func (t *SSETransport) Send(msg []byte) error {
	req, err := http.NewRequest(http.MethodPost, t.baseURL, bytes.NewReader(msg))
	if err != nil {
		return fmt.Errorf("create POST request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Receive reads the next event from the SSE stream.
func (t *SSETransport) Receive() ([]byte, error) {
	select {
	case data, ok := <-t.events:
		if !ok {
			return nil, fmt.Errorf("SSE stream closed")
		}
		return data, nil
	case <-t.done:
		return nil, fmt.Errorf("transport closed")
	}
}

// Close terminates the SSE connection and cleans up resources.
func (t *SSETransport) Close() error {
	t.closeOnce.Do(func() {
		close(t.done)
		if t.resp != nil && t.resp.Body != nil {
			t.resp.Body.Close()
		}
	})
	return nil
}

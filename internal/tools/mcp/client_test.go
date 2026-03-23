package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/meowai/blackcat/pkg/jsonrpc"
)

// mockPipe creates an in-memory reader/writer pair that simulates MCP stdio.
type mockServer struct {
	responses []jsonrpc.Response
	idx       int
}

func (m *mockServer) nextResponse() []byte {
	if m.idx >= len(m.responses) {
		return nil
	}
	resp := m.responses[m.idx]
	m.idx++
	data, _ := json.Marshal(resp)
	return append(data, '\n')
}

func newMockClient(t *testing.T, responses []jsonrpc.Response) *Client {
	t.Helper()
	var buf bytes.Buffer
	for _, resp := range responses {
		data, _ := json.Marshal(resp)
		buf.Write(data)
		buf.WriteByte('\n')
	}

	r := io.NopCloser(&buf)
	pr, pw := io.Pipe()

	// Discard writes (we don't validate requests in these tests)
	go func() {
		b := make([]byte, 4096)
		for {
			_, err := pr.Read(b)
			if err != nil {
				return
			}
		}
	}()

	return NewClientFromIO(r.(io.Reader), pw)
}

func TestClientCall(t *testing.T) {
	resp := jsonrpc.Response{
		JSONRPC: "2.0",
		ID:      float64(1),
		Result:  json.RawMessage(`{"status": "ok"}`),
	}
	client := newMockClient(t, []jsonrpc.Response{resp})

	got, err := client.Call(context.Background(), "test/method", nil)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
	if got.Error != nil {
		t.Errorf("unexpected error: %v", got.Error)
	}
}

func TestClientListTools(t *testing.T) {
	result := json.RawMessage(`{"tools": [{"name": "read_file", "description": "Read a file"}]}`)
	client := newMockClient(t, []jsonrpc.Response{
		{JSONRPC: "2.0", ID: float64(1), Result: result},
	})

	tools, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	if tools[0].Name != "read_file" {
		t.Errorf("expected 'read_file', got %q", tools[0].Name)
	}
}

func TestClientCallTool(t *testing.T) {
	result := json.RawMessage(`{"content": [{"type": "text", "text": "file contents here"}]}`)
	client := newMockClient(t, []jsonrpc.Response{
		{JSONRPC: "2.0", ID: float64(1), Result: result},
	})

	output, err := client.CallTool(context.Background(), "read_file", map[string]any{"path": "main.go"})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if output != "file contents here" {
		t.Errorf("expected 'file contents here', got %q", output)
	}
}

func TestClientCallToolError(t *testing.T) {
	client := newMockClient(t, []jsonrpc.Response{
		{JSONRPC: "2.0", ID: float64(1), Error: &jsonrpc.Error{Code: -1, Message: "tool failed"}},
	})

	_, err := client.CallTool(context.Background(), "bad_tool", nil)
	if err == nil {
		t.Error("expected error for failed tool call")
	}
}

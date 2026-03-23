package plugin

import (
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"
)

func TestPluginRequestEncode(t *testing.T) {
	req := PluginRequest{
		ID:     "req-1",
		Method: "chat",
		Params: map[string]any{"model": "gpt-4", "messages": []any{}},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded PluginRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.ID != "req-1" {
		t.Errorf("expected id req-1, got %s", decoded.ID)
	}
	if decoded.Method != "chat" {
		t.Errorf("expected method chat, got %s", decoded.Method)
	}
}

func TestPluginResponseEncode(t *testing.T) {
	resp := PluginResponse{
		ID:     "req-1",
		Result: map[string]any{"content": "hello"},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded PluginResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.ID != "req-1" {
		t.Errorf("expected id req-1, got %s", decoded.ID)
	}
	if decoded.Error != "" {
		t.Errorf("expected no error, got %s", decoded.Error)
	}
}

func TestPluginResponseError(t *testing.T) {
	resp := PluginResponse{
		ID:    "req-2",
		Error: "something went wrong",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded PluginResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.Error != "something went wrong" {
		t.Errorf("expected error 'something went wrong', got %s", decoded.Error)
	}
}

// mockPlugin simulates a plugin process reading from stdin and writing to stdout.
func mockPlugin(input io.Reader, output io.Writer) {
	decoder := json.NewDecoder(input)
	encoder := json.NewEncoder(output)

	for {
		var req PluginRequest
		if err := decoder.Decode(&req); err != nil {
			return
		}

		var resp PluginResponse
		resp.ID = req.ID

		switch req.Method {
		case "ping":
			resp.Result = "pong"
		case "info":
			resp.Result = map[string]any{
				"name":    "test-plugin",
				"version": "1.0.0",
			}
		case "chat":
			resp.Result = map[string]any{
				"content": "Hello from plugin",
			}
		case "error_method":
			resp.Error = "intentional error"
		default:
			resp.Error = "unknown method: " + req.Method
		}

		encoder.Encode(resp)
	}
}

func TestPluginClientCall(t *testing.T) {
	// Create pipes to simulate stdin/stdout
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	// Start mock plugin in a goroutine
	go mockPlugin(stdinR, stdoutW)

	client := NewPluginClient(stdinW, stdoutR)
	defer client.Close()

	result, err := client.Call("ping", nil)
	if err != nil {
		t.Fatalf("call failed: %v", err)
	}
	if result != "pong" {
		t.Errorf("expected pong, got %v", result)
	}
}

func TestPluginClientCallInfo(t *testing.T) {
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	go mockPlugin(stdinR, stdoutW)

	client := NewPluginClient(stdinW, stdoutR)
	defer client.Close()

	result, err := client.Call("info", nil)
	if err != nil {
		t.Fatalf("call failed: %v", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["name"] != "test-plugin" {
		t.Errorf("expected test-plugin, got %v", m["name"])
	}
}

func TestPluginClientCallError(t *testing.T) {
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	go mockPlugin(stdinR, stdoutW)

	client := NewPluginClient(stdinW, stdoutR)
	defer client.Close()

	_, err := client.Call("error_method", nil)
	if err == nil {
		t.Fatal("expected error from error_method")
	}
}

func TestPluginClientPing(t *testing.T) {
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	go mockPlugin(stdinR, stdoutW)

	client := NewPluginClient(stdinW, stdoutR)
	defer client.Close()

	if err := client.Ping(); err != nil {
		t.Fatalf("ping failed: %v", err)
	}
}

func TestPluginClientCallWithTimeout(t *testing.T) {
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	go mockPlugin(stdinR, stdoutW)

	client := NewPluginClient(stdinW, stdoutR)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := client.CallWithTimeout(ctx, "chat", map[string]any{"model": "gpt-4"})
	if err != nil {
		t.Fatalf("call with timeout failed: %v", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["content"] != "Hello from plugin" {
		t.Errorf("expected 'Hello from plugin', got %v", m["content"])
	}
}

func TestPluginClientCallWithTimeoutExpired(t *testing.T) {
	stdinR, stdinW := io.Pipe()
	stdoutR, _ := io.Pipe() // no writer = will block forever

	// Drain stdinR so the write in CallWithTimeout doesn't block on the pipe
	go func() {
		buf := make([]byte, 4096)
		for {
			if _, err := stdinR.Read(buf); err != nil {
				return
			}
		}
	}()

	client := NewPluginClient(stdinW, stdoutR)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.CallWithTimeout(ctx, "ping", nil)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

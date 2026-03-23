package mcp

import (
	"strings"
	"testing"
)

func TestNewStdioTransport(t *testing.T) {
	tr := NewStdioTransport("echo", "hello")
	if tr == nil {
		t.Fatal("expected non-nil transport")
	}
	// Don't start it, just verify fields
	if tr.command != "echo" {
		t.Errorf("expected command 'echo', got %q", tr.command)
	}
	if len(tr.args) != 1 || tr.args[0] != "hello" {
		t.Errorf("expected args ['hello'], got %v", tr.args)
	}
}

func TestStdioTransportStartClose(t *testing.T) {
	// Use "cat" on Unix or "cmd /c type CON" on Windows — but cat is safer for tests.
	// We'll use a simple echo-based approach that writes and exits.
	tr := NewStdioTransport("echo", "test-line")
	err := tr.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Should be able to receive the echoed line
	data, err := tr.Receive()
	if err != nil {
		t.Fatalf("Receive failed: %v", err)
	}
	if !strings.Contains(string(data), "test-line") {
		t.Errorf("expected 'test-line', got %q", string(data))
	}

	err = tr.Close()
	if err != nil {
		// echo exits immediately, close may see exit status
		t.Logf("Close returned: %v (expected for echo)", err)
	}
}

func TestStdioTransportSendReceive(t *testing.T) {
	// Use "cat" which echoes stdin to stdout (Unix-like behavior)
	// On Windows in Git Bash, cat should be available
	tr := NewStdioTransport("cat")
	err := tr.Start()
	if err != nil {
		t.Skipf("cat not available: %v", err)
	}
	defer tr.Close()

	msg := []byte(`{"jsonrpc":"2.0","method":"test"}`)
	err = tr.Send(msg)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	data, err := tr.Receive()
	if err != nil {
		t.Fatalf("Receive failed: %v", err)
	}
	if string(data) != string(msg) {
		t.Errorf("expected %q, got %q", string(msg), string(data))
	}
}

func TestStdioTransportSendBeforeStart(t *testing.T) {
	tr := NewStdioTransport("cat")
	err := tr.Send([]byte("test"))
	if err == nil {
		t.Error("expected error when sending before Start")
	}
}

func TestStdioTransportReceiveBeforeStart(t *testing.T) {
	tr := NewStdioTransport("cat")
	_, err := tr.Receive()
	if err == nil {
		t.Error("expected error when receiving before Start")
	}
}

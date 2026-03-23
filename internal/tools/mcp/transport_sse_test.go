package mcp

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewSSETransport(t *testing.T) {
	tr := NewSSETransport("http://localhost:8080")
	if tr == nil {
		t.Fatal("expected non-nil transport")
	}
	if tr.baseURL != "http://localhost:8080" {
		t.Errorf("expected baseURL 'http://localhost:8080', got %q", tr.baseURL)
	}
}

func TestSSETransportSendPOST(t *testing.T) {
	var receivedBody string
	var receivedMethod string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		receivedBody = string(buf[:n])
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tr := NewSSETransport(server.URL)
	msg := []byte(`{"jsonrpc":"2.0","method":"test","id":1}`)
	err := tr.Send(msg)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if receivedMethod != "POST" {
		t.Errorf("expected POST, got %s", receivedMethod)
	}
	if !strings.Contains(receivedBody, "test") {
		t.Errorf("expected body to contain 'test', got %q", receivedBody)
	}
}

func TestSSETransportSendError(t *testing.T) {
	tr := NewSSETransport("http://invalid.localhost.test:1")
	err := tr.Send([]byte(`{}`))
	if err == nil {
		t.Error("expected error for unreachable server")
	}
}

func TestSSETransportReceive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/events" || r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "streaming not supported", http.StatusInternalServerError)
				return
			}
			fmt.Fprintf(w, "data: {\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{}}\n\n")
			flusher.Flush()
		}
	}))
	defer server.Close()

	tr := NewSSETransport(server.URL)
	err := tr.Connect()
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer tr.Close()

	// Give the SSE reader a moment to process
	time.Sleep(100 * time.Millisecond)

	data, err := tr.Receive()
	if err != nil {
		t.Fatalf("Receive failed: %v", err)
	}
	if !strings.Contains(string(data), "jsonrpc") {
		t.Errorf("expected JSON-RPC response, got %q", string(data))
	}
}

func TestSSETransportClose(t *testing.T) {
	tr := NewSSETransport("http://localhost:1")
	err := tr.Close()
	if err != nil {
		t.Errorf("Close on unconnected transport should not error, got: %v", err)
	}
}

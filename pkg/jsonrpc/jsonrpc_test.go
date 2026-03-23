package jsonrpc

import (
	"encoding/json"
	"testing"
)

func TestNewRequest(t *testing.T) {
	req, err := NewRequest(1, "tools/list", map[string]string{"foo": "bar"})
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}
	if req.JSONRPC != "2.0" {
		t.Errorf("expected '2.0', got %q", req.JSONRPC)
	}
	if req.Method != "tools/list" {
		t.Errorf("expected 'tools/list', got %q", req.Method)
	}
	if req.ID != 1 {
		t.Errorf("expected id 1, got %v", req.ID)
	}
}

func TestNewRequestNilParams(t *testing.T) {
	req, err := NewRequest("abc", "initialize", nil)
	if err != nil {
		t.Fatal(err)
	}
	if req.Params != nil {
		t.Error("expected nil params")
	}
}

func TestNewRequestMarshalUnmarshal(t *testing.T) {
	req, _ := NewRequest(1, "test", nil)
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed["jsonrpc"] != "2.0" {
		t.Error("missing jsonrpc field")
	}
	if parsed["method"] != "test" {
		t.Errorf("expected method 'test', got %v", parsed["method"])
	}
}

func TestParseResponse(t *testing.T) {
	data := []byte(`{"jsonrpc":"2.0","id":1,"result":{"name":"test_tool"}}`)
	resp, err := ParseResponse(data)
	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}
	if resp.JSONRPC != "2.0" {
		t.Errorf("expected '2.0', got %q", resp.JSONRPC)
	}
	if resp.IsError() {
		t.Error("expected no error")
	}
}

func TestParseResponseError(t *testing.T) {
	data := []byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"method not found"}}`)
	resp, err := ParseResponse(data)
	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}
	if !resp.IsError() {
		t.Error("expected error response")
	}
	if resp.Error.Code != CodeMethodNotFound {
		t.Errorf("expected code %d, got %d", CodeMethodNotFound, resp.Error.Code)
	}
	if resp.Error.Error() != "method not found" {
		t.Errorf("expected 'method not found', got %q", resp.Error.Error())
	}
}

func TestParseResponseInvalidJSON(t *testing.T) {
	_, err := ParseResponse([]byte(`{invalid`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestIsError(t *testing.T) {
	noErr := Response{JSONRPC: "2.0", ID: 1, Result: json.RawMessage(`{}`)}
	if noErr.IsError() {
		t.Error("expected IsError false for success response")
	}

	withErr := Response{JSONRPC: "2.0", ID: 1, Error: &Error{Code: -32600, Message: "bad"}}
	if !withErr.IsError() {
		t.Error("expected IsError true for error response")
	}
}

func TestParseResult(t *testing.T) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      1,
		Result:  json.RawMessage(`{"name": "test_tool"}`),
	}

	type toolInfo struct {
		Name string `json:"name"`
	}

	result, err := ParseResult[toolInfo](resp)
	if err != nil {
		t.Fatal(err)
	}
	if result.Name != "test_tool" {
		t.Errorf("expected 'test_tool', got %q", result.Name)
	}
}

func TestParseResultError(t *testing.T) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      1,
		Error:   &Error{Code: CodeMethodNotFound, Message: "method not found"},
	}

	type any2 struct{}
	_, err := ParseResult[any2](resp)
	if err == nil {
		t.Error("expected error")
	}
	if err.Error() != "method not found" {
		t.Errorf("expected 'method not found', got %q", err.Error())
	}
}

func TestErrorCodes(t *testing.T) {
	if CodeParseError != -32700 {
		t.Errorf("CodeParseError should be -32700, got %d", CodeParseError)
	}
	if CodeInvalidRequest != -32600 {
		t.Errorf("CodeInvalidRequest should be -32600, got %d", CodeInvalidRequest)
	}
	if CodeMethodNotFound != -32601 {
		t.Errorf("CodeMethodNotFound should be -32601, got %d", CodeMethodNotFound)
	}
	if CodeInvalidParams != -32602 {
		t.Errorf("CodeInvalidParams should be -32602, got %d", CodeInvalidParams)
	}
	if CodeInternalError != -32603 {
		t.Errorf("CodeInternalError should be -32603, got %d", CodeInternalError)
	}
}

func TestNewRequestInvalidParams(t *testing.T) {
	// Channels cannot be marshaled to JSON
	_, err := NewRequest(1, "test", make(chan int))
	if err == nil {
		t.Error("expected error for non-marshalable params")
	}
}

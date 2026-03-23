// Package jsonrpc provides minimal JSON-RPC 2.0 request/response types.
package jsonrpc

import "encoding/json"

// Request is a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// Error is a JSON-RPC 2.0 error object.
type Error struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *Error) Error() string { return e.Message }

// Standard JSON-RPC 2.0 error codes.
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
)

// NewRequest creates a JSON-RPC 2.0 request.
// Params is marshaled to JSON. Pass nil for no params.
func NewRequest(id any, method string, params any) (Request, error) {
	var raw json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return Request{}, err
		}
		raw = b
	}
	return Request{JSONRPC: "2.0", ID: id, Method: method, Params: raw}, nil
}

// ParseResponse deserializes a JSON-RPC 2.0 response from raw bytes.
func ParseResponse(data []byte) (Response, error) {
	var resp Response
	if err := json.Unmarshal(data, &resp); err != nil {
		return Response{}, err
	}
	return resp, nil
}

// IsError returns true if the response contains an error.
func (r Response) IsError() bool {
	return r.Error != nil
}

// ParseResult extracts a typed result from a Response.
// Returns the Error if the response is an error response.
func ParseResult[T any](resp Response) (T, error) {
	var zero T
	if resp.Error != nil {
		return zero, resp.Error
	}
	var result T
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return zero, err
	}
	return result, nil
}

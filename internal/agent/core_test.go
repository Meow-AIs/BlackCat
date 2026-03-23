package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/meowai/blackcat/internal/llm"
	"github.com/meowai/blackcat/internal/security"
	"github.com/meowai/blackcat/internal/tools"
)

// mockTool for testing
type mockTool struct {
	name   string
	output string
}

func (m *mockTool) Info() tools.Definition {
	return tools.Definition{Name: m.name, Category: "test", Description: m.name}
}

func (m *mockTool) Execute(_ context.Context, _ map[string]any) (tools.Result, error) {
	return tools.Result{Output: m.output, ExitCode: 0}, nil
}

func setupMockServer(t *testing.T, responses []string) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	callCount := &atomic.Int32{}
	idx := &atomic.Int32{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		i := int(idx.Add(1)) - 1
		respStr := responses[0]
		if i < len(responses) {
			respStr = responses[i]
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(respStr))
	}))

	return server, callCount
}

func TestCoreStartSession(t *testing.T) {
	core := NewCore(CoreConfig{})

	sess, err := core.StartSession(context.Background(), "proj1", "user1")
	if err != nil {
		t.Fatalf("StartSession failed: %v", err)
	}
	if sess.ID == "" {
		t.Error("expected non-empty session ID")
	}
	if sess.ProjectID != "proj1" {
		t.Errorf("expected projectID 'proj1', got %q", sess.ProjectID)
	}
	if sess.State != StateIdle {
		t.Errorf("expected state idle, got %q", sess.State)
	}
}

func TestCoreResumeSession(t *testing.T) {
	core := NewCore(CoreConfig{})
	sess, _ := core.StartSession(context.Background(), "proj1", "user1")

	resumed, err := core.ResumeSession(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("ResumeSession failed: %v", err)
	}
	if resumed.ID != sess.ID {
		t.Errorf("expected same session ID")
	}
}

func TestCoreResumeSessionNotFound(t *testing.T) {
	core := NewCore(CoreConfig{})
	_, err := core.ResumeSession(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}

func TestCoreProcessSimpleResponse(t *testing.T) {
	// Mock LLM returns simple text response (no tool calls)
	server, _ := setupMockServer(t, []string{`{
		"id": "msg_1", "type": "message", "role": "assistant",
		"content": [{"type": "text", "text": "Hello! I am BlackCat."}],
		"model": "claude-sonnet-4-6", "stop_reason": "end_turn",
		"usage": {"input_tokens": 10, "output_tokens": 5}
	}`})
	defer server.Close()

	provider := llm.NewAnthropicProvider("test-key", server.URL)
	core := NewCore(CoreConfig{
		Provider:    provider,
		Registry:    tools.NewMapRegistry(),
		CostTracker: llm.NewCostTracker(0, 0),
	})

	sess, _ := core.StartSession(context.Background(), "proj", "user")
	resp, err := core.Process(context.Background(), sess.ID, "Hello")
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}
	if resp.Text != "Hello! I am BlackCat." {
		t.Errorf("expected greeting, got %q", resp.Text)
	}
	if !resp.Done {
		t.Error("expected done=true")
	}
}

func TestCoreProcessWithToolCall(t *testing.T) {
	callIdx := &atomic.Int32{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		i := int(callIdx.Add(1))
		var resp string
		if i == 1 {
			// First call: LLM wants to use a tool
			resp = `{
				"id": "msg_1", "type": "message", "role": "assistant",
				"content": [
					{"type": "text", "text": "Let me check."},
					{"type": "tool_use", "id": "toolu_1", "name": "test_tool", "input": {"key": "val"}}
				],
				"model": "claude-sonnet-4-6", "stop_reason": "tool_use",
				"usage": {"input_tokens": 10, "output_tokens": 5}
			}`
		} else {
			// Second call: LLM responds after seeing tool result
			resp = `{
				"id": "msg_2", "type": "message", "role": "assistant",
				"content": [{"type": "text", "text": "The result is: test output"}],
				"model": "claude-sonnet-4-6", "stop_reason": "end_turn",
				"usage": {"input_tokens": 20, "output_tokens": 10}
			}`
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(resp))
	}))
	defer server.Close()

	provider := llm.NewAnthropicProvider("test-key", server.URL)
	registry := tools.NewMapRegistry()
	registry.Register(&mockTool{name: "test_tool", output: "test output"})

	core := NewCore(CoreConfig{
		Provider: provider,
		Registry: registry,
		Checker:  security.NewPermissionChecker(),
	})

	sess, _ := core.StartSession(context.Background(), "proj", "user")
	resp, err := core.Process(context.Background(), sess.ID, "use the tool")
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}
	if resp.Text != "The result is: test output" {
		t.Errorf("expected tool result response, got %q", resp.Text)
	}
	if len(resp.ToolCalls) != 1 {
		t.Errorf("expected 1 tool use recorded, got %d", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "test_tool" {
		t.Errorf("expected tool 'test_tool', got %q", resp.ToolCalls[0].Name)
	}
}

// --- P0 Fix 4: Error Message Sanitization Tests ---

func TestSanitizeForLLMStripsPostgresConnectionString(t *testing.T) {
	input := "failed to connect: postgres://admin:s3cr3tpass@db.example.com:5432/mydb"
	result := sanitizeForLLM(input)
	if strings.Contains(result, "s3cr3tpass") {
		t.Errorf("expected password stripped from postgres DSN, got %q", result)
	}
	// Should still contain useful context
	if !strings.Contains(result, "postgres://") {
		t.Errorf("expected sanitized output to keep scheme for context, got %q", result)
	}
}

func TestSanitizeForLLMStripsBearerToken(t *testing.T) {
	input := `HTTP 401: Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.payload.sig`
	result := sanitizeForLLM(input)
	if strings.Contains(result, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9") {
		t.Errorf("expected Bearer token stripped, got %q", result)
	}
}

func TestSanitizeForLLMStripsAPIKeyParam(t *testing.T) {
	input := "GET /v1/data?api_key=sk-abc123secretkey&format=json"
	result := sanitizeForLLM(input)
	if strings.Contains(result, "sk-abc123secretkey") {
		t.Errorf("expected api_key value stripped, got %q", result)
	}
	// The query parameter name can remain for context
	if !strings.Contains(result, "api_key") {
		t.Errorf("expected api_key param name preserved for context, got %q", result)
	}
}

func TestSanitizeForLLMPreservesNormalOutput(t *testing.T) {
	input := "file written successfully: 42 bytes to /tmp/output.txt"
	result := sanitizeForLLM(input)
	if result != input {
		t.Errorf("expected normal output unchanged, got %q", result)
	}
}

func TestSanitizeForLLMHandlesEmptyString(t *testing.T) {
	result := sanitizeForLLM("")
	if result != "" {
		t.Errorf("expected empty string unchanged, got %q", result)
	}
}

func TestSanitizeForLLMHandlesMultipleSecrets(t *testing.T) {
	input := "connect postgres://user:pass1@host/db then use Bearer token123abc for auth"
	result := sanitizeForLLM(input)
	if strings.Contains(result, "pass1") {
		t.Errorf("expected postgres password stripped in multi-secret input, got %q", result)
	}
	if strings.Contains(result, "token123abc") {
		t.Errorf("expected bearer token stripped in multi-secret input, got %q", result)
	}
}

func TestBuildJSONSchema(t *testing.T) {
	params := []tools.Parameter{
		{Name: "path", Type: "string", Description: "File path", Required: true},
		{Name: "encoding", Type: "string", Description: "Encoding", Enum: []string{"utf-8", "ascii"}},
	}

	schema := buildJSONSchema(params)
	data, _ := json.Marshal(schema)

	var parsed map[string]any
	json.Unmarshal(data, &parsed)

	if parsed["type"] != "object" {
		t.Error("expected type 'object'")
	}
	props := parsed["properties"].(map[string]any)
	if _, ok := props["path"]; !ok {
		t.Error("expected 'path' property")
	}
}

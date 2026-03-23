package agent

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// VerifyToolCall tests
// ---------------------------------------------------------------------------

func TestVerifyToolCall_ValidCall(t *testing.T) {
	v := NewVerifier(VerifyStandard)
	available := []string{"read_file", "write_file", "execute"}

	result := v.VerifyToolCall("read_file", map[string]any{"path": "/tmp/foo.go"}, available)

	if !result.Valid {
		t.Errorf("expected valid, got issues: %v", result.Issues)
	}
	if result.Confidence < 0.5 {
		t.Errorf("confidence = %f, want >= 0.5", result.Confidence)
	}
}

func TestVerifyToolCall_ToolNotInList(t *testing.T) {
	v := NewVerifier(VerifyBasic)
	available := []string{"read_file", "write_file"}

	result := v.VerifyToolCall("delete_everything", nil, available)

	if result.Valid {
		t.Error("expected invalid for unknown tool")
	}
	if len(result.Issues) == 0 {
		t.Error("expected at least one issue")
	}
}

func TestVerifyToolCall_EmptyArgs(t *testing.T) {
	v := NewVerifier(VerifyStandard)
	available := []string{"read_file"}

	result := v.VerifyToolCall("read_file", map[string]any{}, available)

	if result.Valid {
		t.Error("expected invalid for missing required args")
	}
}

func TestVerifyToolCall_DangerousCommand(t *testing.T) {
	v := NewVerifier(VerifyStandard)
	available := []string{"execute"}

	result := v.VerifyToolCall("execute", map[string]any{
		"command": "rm -rf /",
	}, available)

	if result.Valid {
		t.Error("expected invalid for dangerous command")
	}
	found := false
	for _, issue := range result.Issues {
		if strings.Contains(strings.ToLower(issue), "danger") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected danger-related issue, got: %v", result.Issues)
	}
}

func TestVerifyToolCall_EmptyStringArg(t *testing.T) {
	v := NewVerifier(VerifyStandard)
	available := []string{"read_file"}

	result := v.VerifyToolCall("read_file", map[string]any{"path": ""}, available)

	if result.Valid {
		t.Error("expected invalid for empty string arg")
	}
}

func TestVerifyToolCall_NoneLevel_SkipsAll(t *testing.T) {
	v := NewVerifier(VerifyNone)
	// Even with a nonexistent tool, none level should pass
	result := v.VerifyToolCall("nonexistent", nil, nil)

	if !result.Valid {
		t.Error("none level should always return valid")
	}
	if result.Confidence != 1.0 {
		t.Errorf("confidence = %f, want 1.0 for none level", result.Confidence)
	}
}

// ---------------------------------------------------------------------------
// VerifyToolResult tests
// ---------------------------------------------------------------------------

func TestVerifyToolResult_Success(t *testing.T) {
	v := NewVerifier(VerifyStandard)

	result := v.VerifyToolResult("read_file", map[string]any{"path": "/foo"}, "file contents here", 0)

	if !result.Valid {
		t.Errorf("expected valid, got issues: %v", result.Issues)
	}
}

func TestVerifyToolResult_EmptyOutputForRead(t *testing.T) {
	v := NewVerifier(VerifyStandard)

	result := v.VerifyToolResult("read_file", map[string]any{"path": "/foo"}, "", 0)

	if result.Valid {
		t.Error("expected invalid for empty read output")
	}
}

func TestVerifyToolResult_NonZeroExit(t *testing.T) {
	v := NewVerifier(VerifyStandard)

	result := v.VerifyToolResult("execute", map[string]any{"command": "ls"}, "error output", 1)

	if result.Valid {
		t.Error("expected invalid for non-zero exit code")
	}
}

func TestVerifyToolResult_ErrorInOutput(t *testing.T) {
	v := NewVerifier(VerifyStandard)

	result := v.VerifyToolResult("execute", map[string]any{"command": "go build"}, "Error: compilation failed\nline 42", 0)

	if result.Valid {
		t.Error("expected invalid when output contains error messages")
	}
}

func TestVerifyToolResult_NoneLevel(t *testing.T) {
	v := NewVerifier(VerifyNone)

	result := v.VerifyToolResult("execute", nil, "", 127)

	if !result.Valid {
		t.Error("none level should always return valid")
	}
}

func TestVerifyToolResult_TruncatedOutput(t *testing.T) {
	v := NewVerifier(VerifyStandard)

	result := v.VerifyToolResult("read_file", map[string]any{"path": "/big"}, "some text... [truncated]", 0)

	// Should be valid but flag truncation as an issue
	hasWarning := false
	for _, issue := range result.Issues {
		if strings.Contains(strings.ToLower(issue), "truncat") {
			hasWarning = true
		}
	}
	if !hasWarning {
		t.Error("expected truncation warning in issues")
	}
}

// ---------------------------------------------------------------------------
// VerifyResponse tests
// ---------------------------------------------------------------------------

func TestVerifyResponse_GroundedResponse(t *testing.T) {
	v := NewVerifier(VerifyStandard)
	toolOutputs := []string{"The function calculates the sum of two integers."}

	result := v.VerifyResponse(
		"This function calculates the sum of two integers and returns the result.",
		toolOutputs,
	)

	if !result.Valid {
		t.Errorf("expected valid for grounded response, got issues: %v", result.Issues)
	}
}

func TestVerifyResponse_UngroundedResponse(t *testing.T) {
	v := NewVerifier(VerifyStandard)
	toolOutputs := []string{"The function calculates the sum of two integers."}

	result := v.VerifyResponse(
		"This function performs complex machine learning inference on quantum data.",
		toolOutputs,
	)

	if result.Valid {
		t.Error("expected invalid for ungrounded response")
	}
}

func TestVerifyResponse_EmptyResponse(t *testing.T) {
	v := NewVerifier(VerifyStandard)

	result := v.VerifyResponse("", []string{"some output"})

	if result.Valid {
		t.Error("expected invalid for empty response")
	}
}

func TestVerifyResponse_VeryShortResponse(t *testing.T) {
	v := NewVerifier(VerifyStandard)

	result := v.VerifyResponse("ok", []string{"detailed output with lots of information"})

	if result.Valid {
		t.Error("expected invalid for suspiciously short response")
	}
}

func TestVerifyResponse_NoneLevel(t *testing.T) {
	v := NewVerifier(VerifyNone)

	result := v.VerifyResponse("", nil)

	if !result.Valid {
		t.Error("none level should always return valid")
	}
}

// ---------------------------------------------------------------------------
// ClassifyRisk tests
// ---------------------------------------------------------------------------

func TestClassifyRisk_WriteOperations(t *testing.T) {
	tools := []struct {
		name string
		args map[string]any
	}{
		{"write_file", map[string]any{"path": "/foo"}},
		{"execute", map[string]any{"command": "bash script.sh"}},
		{"git_commit", map[string]any{"message": "fix"}},
	}

	for _, tc := range tools {
		t.Run(tc.name, func(t *testing.T) {
			level := ClassifyRisk(tc.name, tc.args)
			if level != VerifyStrict {
				t.Errorf("ClassifyRisk(%q) = %q, want %q", tc.name, level, VerifyStrict)
			}
		})
	}
}

func TestClassifyRisk_ReadOperations(t *testing.T) {
	tools := []struct {
		name string
		args map[string]any
	}{
		{"read_file", map[string]any{"path": "/foo"}},
		{"search_files", map[string]any{"query": "test"}},
		{"search_content", map[string]any{"pattern": "func"}},
		{"list_dir", map[string]any{"path": "/"}},
	}

	for _, tc := range tools {
		t.Run(tc.name, func(t *testing.T) {
			level := ClassifyRisk(tc.name, tc.args)
			if level != VerifyBasic {
				t.Errorf("ClassifyRisk(%q) = %q, want %q", tc.name, level, VerifyBasic)
			}
		})
	}
}

func TestClassifyRisk_StandardOperations(t *testing.T) {
	tools := []struct {
		name string
		args map[string]any
	}{
		{"web_fetch", map[string]any{"url": "https://example.com"}},
		{"manage_skills", map[string]any{"action": "list"}},
		{"manage_plugins", map[string]any{"action": "install"}},
	}

	for _, tc := range tools {
		t.Run(tc.name, func(t *testing.T) {
			level := ClassifyRisk(tc.name, tc.args)
			if level != VerifyStandard {
				t.Errorf("ClassifyRisk(%q) = %q, want %q", tc.name, level, VerifyStandard)
			}
		})
	}
}

func TestClassifyRisk_UnknownTool(t *testing.T) {
	level := ClassifyRisk("unknown_tool", nil)
	if level != VerifyStandard {
		t.Errorf("ClassifyRisk(unknown) = %q, want %q", level, VerifyStandard)
	}
}

// ---------------------------------------------------------------------------
// SetLevel tests
// ---------------------------------------------------------------------------

func TestVerifier_SetLevel(t *testing.T) {
	v := NewVerifier(VerifyBasic)
	if v.level != VerifyBasic {
		t.Errorf("level = %q, want %q", v.level, VerifyBasic)
	}

	v.SetLevel(VerifyStrict)
	if v.level != VerifyStrict {
		t.Errorf("after SetLevel, level = %q, want %q", v.level, VerifyStrict)
	}
}

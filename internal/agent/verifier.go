package agent

import (
	"strings"
)

// VerificationResult holds the outcome of a verification check.
type VerificationResult struct {
	Valid      bool
	Confidence float64 // 0-1
	Issues     []string
	Suggestion string // what to fix
}

// VerificationLevel controls how thoroughly verification is performed.
type VerificationLevel string

const (
	// VerifyNone skips all verification checks.
	VerifyNone VerificationLevel = "none"
	// VerifyBasic performs syntax and format checks only.
	VerifyBasic VerificationLevel = "basic"
	// VerifyStandard checks that tool output makes sense.
	VerifyStandard VerificationLevel = "standard"
	// VerifyStrict performs full LLM-based review.
	VerifyStrict VerificationLevel = "strict"
)

// Verifier performs multi-level verification of tool calls and responses.
type Verifier struct {
	level VerificationLevel
}

// NewVerifier creates a Verifier at the given level.
func NewVerifier(level VerificationLevel) *Verifier {
	return &Verifier{level: level}
}

// SetLevel changes the verification level.
func (v *Verifier) SetLevel(level VerificationLevel) {
	v.level = level
}

// VerifyToolCall checks if a tool call is reasonable before execution.
func (v *Verifier) VerifyToolCall(toolName string, args map[string]any, availableTools []string) VerificationResult {
	if v.level == VerifyNone {
		return VerificationResult{Valid: true, Confidence: 1.0}
	}

	var issues []string

	// Check tool exists in available list.
	if !containsStr(availableTools, toolName) {
		issues = append(issues, "tool not found in available tools: "+toolName)
	}

	// Check required args for known tools.
	requiredArgs := requiredArgsForTool(toolName)
	for _, req := range requiredArgs {
		val, ok := args[req]
		if !ok {
			issues = append(issues, "missing required argument: "+req)
			continue
		}
		if s, isStr := val.(string); isStr && s == "" {
			issues = append(issues, "argument "+req+" is empty")
		}
	}

	// Check for empty args when tool is known to need them.
	if len(requiredArgs) > 0 && len(args) == 0 {
		issues = append(issues, "no arguments provided for tool that requires them")
	}

	// Check string args are non-empty.
	for key, val := range args {
		if s, isStr := val.(string); isStr && s == "" {
			// Only flag if not already flagged as a required arg.
			alreadyFlagged := false
			for _, issue := range issues {
				if strings.Contains(issue, key) {
					alreadyFlagged = true
					break
				}
			}
			if !alreadyFlagged {
				issues = append(issues, "argument "+key+" is empty string")
			}
		}
	}

	// Check for dangerous patterns in shell commands.
	if toolName == "execute" || toolName == "bash" {
		if cmd, ok := args["command"].(string); ok {
			if isDangerousCommand(cmd) {
				issues = append(issues, "dangerous command pattern detected")
			}
		}
	}

	confidence := 1.0
	if len(issues) > 0 {
		confidence = 0.0
	} else if v.level == VerifyBasic {
		confidence = 0.8
	}

	suggestion := ""
	if len(issues) > 0 {
		suggestion = "review and fix the issues before executing"
	}

	return VerificationResult{
		Valid:      len(issues) == 0,
		Confidence: confidence,
		Issues:     issues,
		Suggestion: suggestion,
	}
}

// VerifyToolResult checks if tool output is valid after execution.
func (v *Verifier) VerifyToolResult(toolName string, args map[string]any, output string, exitCode int) VerificationResult {
	if v.level == VerifyNone {
		return VerificationResult{Valid: true, Confidence: 1.0}
	}

	var issues []string

	// Check for empty output on read operations.
	if isReadOperation(toolName) && output == "" {
		issues = append(issues, "empty output for read operation")
	}

	// Check for non-zero exit code.
	if exitCode != 0 {
		issues = append(issues, "non-zero exit code indicates failure")
	}

	// Check for error messages in output.
	if containsErrorPattern(output) && exitCode == 0 {
		issues = append(issues, "output contains error messages despite zero exit code")
	}

	// Check for truncation indicators.
	if containsTruncation(output) {
		issues = append(issues, "output appears truncated")
	}

	confidence := 1.0
	if len(issues) > 0 {
		// Truncation alone doesn't invalidate the result.
		onlyTruncation := len(issues) == 1 && strings.Contains(issues[0], "truncat")
		if onlyTruncation {
			confidence = 0.7
		} else {
			confidence = 0.0
		}
	}

	valid := confidence > 0.5

	suggestion := ""
	if !valid {
		suggestion = "review tool output for errors"
	}

	return VerificationResult{
		Valid:      valid,
		Confidence: confidence,
		Issues:     issues,
		Suggestion: suggestion,
	}
}

// VerifyResponse checks if the LLM's final response is grounded in tool output.
func (v *Verifier) VerifyResponse(response string, toolOutputs []string) VerificationResult {
	if v.level == VerifyNone {
		return VerificationResult{Valid: true, Confidence: 1.0}
	}

	var issues []string

	// Check for empty response.
	if strings.TrimSpace(response) == "" {
		issues = append(issues, "response is empty")
		return VerificationResult{
			Valid:      false,
			Confidence: 0.0,
			Issues:     issues,
			Suggestion: "provide a substantive response",
		}
	}

	// Check for suspiciously short response.
	if len(response) < 10 && len(toolOutputs) > 0 {
		outputLen := 0
		for _, o := range toolOutputs {
			outputLen += len(o)
		}
		if outputLen > 50 {
			issues = append(issues, "response is suspiciously short given tool output")
		}
	}

	// Check grounding: response should reference content from tool outputs.
	if len(toolOutputs) > 0 {
		grounded := isGrounded(response, toolOutputs)
		if !grounded {
			issues = append(issues, "response does not appear grounded in tool output")
		}
	}

	confidence := 1.0
	if len(issues) > 0 {
		confidence = 0.2
	}

	suggestion := ""
	if len(issues) > 0 {
		suggestion = "ensure response references actual tool output"
	}

	return VerificationResult{
		Valid:      len(issues) == 0,
		Confidence: confidence,
		Issues:     issues,
		Suggestion: suggestion,
	}
}

// ClassifyRisk determines the appropriate verification level for a tool call.
func ClassifyRisk(toolName string, args map[string]any) VerificationLevel {
	switch toolName {
	case "write_file", "execute", "bash", "git_commit", "git_push",
		"delete_file", "rename_file", "move_file":
		return VerifyStrict

	case "read_file", "search_files", "search_content", "list_dir",
		"list_files", "glob", "grep":
		return VerifyBasic

	case "web_fetch", "manage_skills", "manage_plugins",
		"scan_secrets", "scan_dependencies":
		return VerifyStandard

	default:
		return VerifyStandard
	}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// dangerousPatterns are shell command fragments that indicate destructive ops.
var dangerousPatterns = []string{
	"rm -rf /",
	"rm -rf /*",
	"mkfs.",
	"dd if=",
	":(){:|:&};:",
	"> /dev/sda",
	"chmod -R 777 /",
	"wget|sh",
	"curl|sh",
	"format c:",
}

func isDangerousCommand(cmd string) bool {
	lower := strings.ToLower(cmd)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// requiredArgsForTool returns the expected required arguments for known tools.
func requiredArgsForTool(toolName string) []string {
	switch toolName {
	case "read_file", "write_file", "delete_file":
		return []string{"path"}
	case "execute", "bash":
		return []string{"command"}
	case "search_content", "search_files", "grep":
		return []string{"query"}
	case "web_fetch":
		return []string{"url"}
	default:
		return nil
	}
}

func isReadOperation(toolName string) bool {
	switch toolName {
	case "read_file", "search_content", "search_files",
		"list_dir", "list_files", "glob", "grep":
		return true
	}
	return false
}

func containsErrorPattern(output string) bool {
	lower := strings.ToLower(output)
	errorPrefixes := []string{
		"error:",
		"fatal:",
		"panic:",
		"exception:",
		"traceback (most recent call last):",
	}
	for _, prefix := range errorPrefixes {
		if strings.Contains(lower, prefix) {
			return true
		}
	}
	return false
}

func containsTruncation(output string) bool {
	lower := strings.ToLower(output)
	indicators := []string{
		"[truncated]",
		"... (truncated)",
		"output truncated",
	}
	for _, ind := range indicators {
		if strings.Contains(lower, ind) {
			return true
		}
	}
	return false
}

// isGrounded checks whether the response shares meaningful content with tool
// outputs by looking for overlapping multi-word phrases.
func isGrounded(response string, toolOutputs []string) bool {
	responseWords := strings.Fields(strings.ToLower(response))
	if len(responseWords) < 3 {
		return false
	}

	// Build bigrams from tool outputs.
	outputBigrams := make(map[string]bool)
	for _, output := range toolOutputs {
		words := strings.Fields(strings.ToLower(output))
		for i := 0; i < len(words)-1; i++ {
			bigram := words[i] + " " + words[i+1]
			outputBigrams[bigram] = true
		}
	}

	if len(outputBigrams) == 0 {
		return true // no output to ground against
	}

	// Check if response shares bigrams with output.
	matches := 0
	for i := 0; i < len(responseWords)-1; i++ {
		bigram := responseWords[i] + " " + responseWords[i+1]
		if outputBigrams[bigram] {
			matches++
		}
	}

	// Require at least one matching bigram.
	return matches > 0
}

func containsStr(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

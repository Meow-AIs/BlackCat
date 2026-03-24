package builtin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/meowai/blackcat/internal/tools"
)

// InteractiveShellTool allows the agent to work with interactive programs.
type InteractiveShellTool struct {
	manager *SessionManager
}

// NewInteractiveShellTool creates a new interactive shell tool.
func NewInteractiveShellTool(manager *SessionManager) *InteractiveShellTool {
	return &InteractiveShellTool{manager: manager}
}

// Info returns the tool definition.
func (t *InteractiveShellTool) Info() tools.Definition {
	return tools.Definition{
		Name:        "interactive_shell",
		Description: "Start and interact with interactive programs (ssh, python, mysql, node REPL, etc.). Use 'start' to begin a session, 'send' to send input, 'read' to get output, 'kill' to terminate.",
		Category:    "shell",
		Parameters: []tools.Parameter{
			{Name: "action", Type: "string", Description: "Action: start, send, read, list, kill", Required: true, Enum: []string{"start", "send", "read", "list", "kill"}},
			{Name: "session_id", Type: "string", Description: "Session ID (required for start/send/read/kill)", Required: false},
			{Name: "command", Type: "string", Description: "Command to run (for start action)", Required: false},
			{Name: "input", Type: "string", Description: "Input to send to the session (for send action)", Required: false},
			{Name: "timeout", Type: "integer", Description: "Timeout in seconds for read (default: 5)", Required: false},
		},
	}
}

// Execute runs the interactive shell tool with the given arguments.
func (t *InteractiveShellTool) Execute(_ context.Context, args map[string]any) (tools.Result, error) {
	action, err := requireStringArg(args, "action")
	if err != nil {
		return tools.Result{}, err
	}

	switch action {
	case "start":
		return t.handleStart(args)
	case "send":
		return t.handleSend(args)
	case "read":
		return t.handleRead(args)
	case "list":
		return t.handleList()
	case "kill":
		return t.handleKill(args)
	default:
		return tools.Result{
			Error:    fmt.Sprintf("unknown action: %s", action),
			ExitCode: -1,
		}, nil
	}
}

func (t *InteractiveShellTool) handleStart(args map[string]any) (tools.Result, error) {
	sessionID, _ := optionalStringArg(args, "session_id")
	if sessionID == "" {
		sessionID = fmt.Sprintf("session-%d", time.Now().UnixNano())
	}

	commandStr, _ := optionalStringArg(args, "command")
	if commandStr == "" {
		return tools.Result{
			Error:    "command is required for start action",
			ExitCode: -1,
		}, nil
	}

	// Parse command string into command + args
	parts := strings.Fields(commandStr)
	command := parts[0]
	var cmdArgs []string
	if len(parts) > 1 {
		cmdArgs = parts[1:]
	}

	sess, err := t.manager.Start(sessionID, command, cmdArgs, "")
	if err != nil {
		return tools.Result{
			Error:    err.Error(),
			ExitCode: -1,
		}, nil
	}

	// Brief pause to let initial output arrive
	time.Sleep(100 * time.Millisecond)

	output := fmt.Sprintf("Session %q started (command: %s)", sess.ID, commandStr)
	initialOutput := sess.stdout.String()
	if initialOutput != "" {
		output += "\n" + initialOutput
	}

	return tools.Result{Output: output}, nil
}

func (t *InteractiveShellTool) handleSend(args map[string]any) (tools.Result, error) {
	sessionID, _ := optionalStringArg(args, "session_id")
	if sessionID == "" {
		return tools.Result{
			Error:    "session_id is required for send action",
			ExitCode: -1,
		}, nil
	}

	input, _ := optionalStringArg(args, "input")
	if input == "" {
		return tools.Result{
			Error:    "input is required for send action",
			ExitCode: -1,
		}, nil
	}

	err := t.manager.SendInput(sessionID, input)
	if err != nil {
		return tools.Result{
			Error:    err.Error(),
			ExitCode: -1,
		}, nil
	}

	return tools.Result{Output: fmt.Sprintf("Sent input to session %q", sessionID)}, nil
}

func (t *InteractiveShellTool) handleRead(args map[string]any) (tools.Result, error) {
	sessionID, _ := optionalStringArg(args, "session_id")
	if sessionID == "" {
		return tools.Result{
			Error:    "session_id is required for read action",
			ExitCode: -1,
		}, nil
	}

	output, err := t.manager.ReadOutput(sessionID)
	if err != nil {
		return tools.Result{
			Error:    err.Error(),
			ExitCode: -1,
		}, nil
	}

	sess, ok := t.manager.GetSession(sessionID)
	if ok {
		output += fmt.Sprintf("\n[state: %s]", sess.State)
		if DetectPromptPattern(output) {
			output += "\n[prompt detected: program may be waiting for input]"
		}
	}

	return tools.Result{Output: output}, nil
}

func (t *InteractiveShellTool) handleList() (tools.Result, error) {
	sessions := t.manager.List()
	if len(sessions) == 0 {
		return tools.Result{Output: "No active sessions"}, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Sessions (%d):\n", len(sessions)))
	for i := range sessions {
		sb.WriteString(fmt.Sprintf("  %s: %s [%s] (started: %s)\n",
			sessions[i].ID, sessions[i].Command, sessions[i].State, sessions[i].Started.Format(time.RFC3339)))
	}

	return tools.Result{Output: sb.String()}, nil
}

func (t *InteractiveShellTool) handleKill(args map[string]any) (tools.Result, error) {
	sessionID, _ := optionalStringArg(args, "session_id")
	if sessionID == "" {
		return tools.Result{
			Error:    "session_id is required for kill action",
			ExitCode: -1,
		}, nil
	}

	err := t.manager.Kill(sessionID)
	if err != nil {
		return tools.Result{
			Error:    err.Error(),
			ExitCode: -1,
		}, nil
	}

	return tools.Result{Output: fmt.Sprintf("Session %q killed", sessionID)}, nil
}

// optionalStringArg extracts an optional string argument.
func optionalStringArg(args map[string]any, key string) (string, bool) {
	v, ok := args[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

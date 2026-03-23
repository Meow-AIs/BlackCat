package agent

import "context"

// SessionState tracks the current state of an agent session.
type SessionState string

const (
	StateIdle      SessionState = "idle"
	StateThinking  SessionState = "thinking"
	StateExecuting SessionState = "executing"
	StateDone      SessionState = "done"
)

// Session represents an active agent conversation.
type Session struct {
	ID        string       `json:"id"`
	ProjectID string       `json:"project_id"`
	UserID    string       `json:"user_id"`
	State     SessionState `json:"state"`
	CreatedAt int64        `json:"created_at"`
}

// Response is what the agent produces after processing input.
type Response struct {
	Text       string      `json:"text"`
	ToolCalls  []ToolUse   `json:"tool_calls,omitempty"`
	Done       bool        `json:"done"`
}

// ToolUse records a tool invocation during a session.
type ToolUse struct {
	Name   string         `json:"name"`
	Args   map[string]any `json:"args"`
	Result string         `json:"result"`
	Error  string         `json:"error,omitempty"`
}

// SubAgentTask describes work delegated to a sub-agent.
type SubAgentTask struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Model       string `json:"model,omitempty"` // override model for this sub-agent
	WorkDir     string `json:"work_dir"`
	Status      string `json:"status"` // pending, running, completed, failed
	Result      string `json:"result,omitempty"`
}

// Agent is the core orchestrator interface.
type Agent interface {
	// Process handles user input and returns a response.
	Process(ctx context.Context, sessionID string, input string) (Response, error)

	// StartSession creates a new session.
	StartSession(ctx context.Context, projectID string, userID string) (Session, error)

	// ResumeSession loads an existing session.
	ResumeSession(ctx context.Context, sessionID string) (Session, error)

	// SpawnSubAgent creates a sub-agent for parallel task execution.
	SpawnSubAgent(ctx context.Context, parentSessionID string, task SubAgentTask) (string, error)

	// ListSubAgents returns active sub-agents for a session.
	ListSubAgents(ctx context.Context, sessionID string) ([]SubAgentTask, error)
}

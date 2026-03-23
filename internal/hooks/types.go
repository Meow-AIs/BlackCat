package hooks

// HookEvent identifies the type of event that triggers hooks.
type HookEvent string

const (
	// Tool events.
	EventBeforeTool  HookEvent = "before_tool"
	EventAfterTool   HookEvent = "after_tool"
	EventToolError   HookEvent = "tool_error"

	// Session events.
	EventSessionStart HookEvent = "session_start"
	EventSessionEnd   HookEvent = "session_end"

	// Message events.
	EventBeforeResponse HookEvent = "before_response"
	EventAfterResponse  HookEvent = "after_response"

	// Memory events.
	EventMemoryStore  HookEvent = "memory_store"
	EventMemoryRecall HookEvent = "memory_recall"

	// Security events.
	EventPermissionAsk HookEvent = "permission_ask"
	EventSecurityAlert HookEvent = "security_alert"

	// Skill events.
	EventSkillInstall HookEvent = "skill_install"
	EventSkillExecute HookEvent = "skill_execute"

	// Plugin events.
	EventPluginStart HookEvent = "plugin_start"
	EventPluginStop  HookEvent = "plugin_stop"
)

// HookContext carries event data through the hook chain.
type HookContext struct {
	Event     HookEvent      `json:"event"`
	Timestamp int64          `json:"timestamp"`
	SessionID string         `json:"session_id,omitempty"`
	Data      map[string]any `json:"data"`
}

// HookResult is returned by hook handlers to control execution flow.
type HookResult struct {
	Allow    bool           `json:"allow"`
	Modified map[string]any `json:"modified,omitempty"`
	Message  string         `json:"message,omitempty"`
}

// HookPriority determines execution order (lower runs first).
type HookPriority int

const (
	PriorityFirst  HookPriority = 0
	PriorityNormal HookPriority = 50
	PriorityLast   HookPriority = 100
)

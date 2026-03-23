package security

// Level represents the permission level for an action.
type Level string

const (
	LevelAllow       Level = "allow"        // always allowed, no prompt
	LevelAutoApprove Level = "auto_approve" // allowed if pattern matches
	LevelAsk         Level = "ask"          // requires user confirmation
	LevelDeny        Level = "deny"         // always blocked
)

// Action describes something the agent wants to do.
type Action struct {
	Type    ActionType        `json:"type"`
	Command string            `json:"command,omitempty"` // for shell actions
	Path    string            `json:"path,omitempty"`    // for file actions
	Args    map[string]string `json:"args,omitempty"`
}

// ActionType categorizes an action for permission checking.
type ActionType string

const (
	ActionReadFile    ActionType = "read_file"
	ActionWriteFile   ActionType = "write_file"
	ActionShell       ActionType = "shell"
	ActionGit         ActionType = "git"
	ActionWeb         ActionType = "web"
	ActionListDir     ActionType = "list_directory"
	ActionSearchCode  ActionType = "search_code"
)

// Decision is the result of checking permissions for an action.
type Decision struct {
	Level   Level  `json:"level"`
	Reason  string `json:"reason,omitempty"`
	Allowed bool   `json:"allowed"`
}

// PermissionRule defines a single permission rule.
type PermissionRule struct {
	Action   ActionType `json:"action"`
	Patterns []string   `json:"patterns,omitempty"` // glob patterns
	Excludes []string   `json:"excludes,omitempty"`
	Level    Level      `json:"level"`
}

// Checker evaluates whether an action is permitted.
type Checker interface {
	// Check returns the permission decision for an action.
	Check(action Action) Decision

	// AddRule adds a permission rule.
	AddRule(rule PermissionRule)

	// Rules returns all configured rules.
	Rules() []PermissionRule
}

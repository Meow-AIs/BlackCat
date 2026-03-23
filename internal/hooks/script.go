package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/template"
)

// ScriptHook defines a hook via a simple rule-based config.
type ScriptHook struct {
	Name      string         `json:"name"`
	Event     HookEvent      `json:"event"`
	Priority  HookPriority   `json:"priority"`
	Enabled   bool           `json:"enabled"`
	Condition *HookCondition `json:"condition,omitempty"`
	Action    HookAction     `json:"action"`
}

// HookCondition is a simple expression evaluator.
type HookCondition struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

// HookAction defines what happens when the hook fires.
type HookAction struct {
	Type    string         `json:"type"`
	Message string         `json:"message,omitempty"`
	Modify  map[string]any `json:"modify,omitempty"`
	Command string         `json:"command,omitempty"`
}

// Evaluate checks if the condition matches the given data.
func (c *HookCondition) Evaluate(data map[string]any) bool {
	raw, ok := data[c.Field]
	if !ok {
		return false
	}

	fieldVal, ok := raw.(string)
	if !ok {
		fieldVal = fmt.Sprintf("%v", raw)
	}

	switch c.Operator {
	case "equals":
		return fieldVal == c.Value
	case "not_equals":
		return fieldVal != c.Value
	case "contains":
		return strings.Contains(fieldVal, c.Value)
	case "starts_with":
		return strings.HasPrefix(fieldVal, c.Value)
	case "matches":
		re, err := regexp.Compile(c.Value)
		if err != nil {
			return false
		}
		return re.MatchString(fieldVal)
	default:
		return false
	}
}

// LoadScriptHooks loads hook definitions from a JSON file.
func LoadScriptHooks(path string) ([]ScriptHook, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading hooks file: %w", err)
	}

	var hooks []ScriptHook
	if err := json.Unmarshal(data, &hooks); err != nil {
		return nil, fmt.Errorf("parsing hooks file: %w", err)
	}

	return hooks, nil
}

// SaveScriptHooks writes hook definitions to a JSON file.
func SaveScriptHooks(path string, hooks []ScriptHook) error {
	data, err := json.MarshalIndent(hooks, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling hooks: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing hooks file: %w", err)
	}

	return nil
}

// ToHandler converts a ScriptHook into a HookHandler function.
func (h *ScriptHook) ToHandler() HookHandler {
	return func(ctx HookContext) HookResult {
		// Check condition if present.
		if h.Condition != nil && !h.Condition.Evaluate(ctx.Data) {
			return HookResult{Allow: true}
		}

		message := expandTemplate(h.Action.Message, ctx.Data)

		switch h.Action.Type {
		case "block":
			return HookResult{
				Allow:   false,
				Message: message,
			}
		case "modify":
			return HookResult{
				Allow:    true,
				Modified: copyMap(h.Action.Modify),
				Message:  message,
			}
		case "allow", "log", "notify", "execute":
			return HookResult{
				Allow:   true,
				Message: message,
			}
		default:
			return HookResult{Allow: true, Message: message}
		}
	}
}

// expandTemplate performs simple {{.field}} substitution on the message.
func expandTemplate(msg string, data map[string]any) string {
	if !strings.Contains(msg, "{{") {
		return msg
	}

	tmpl, err := template.New("hook").Option("missingkey=default").Parse(msg)
	if err != nil {
		return msg
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return msg
	}

	return buf.String()
}

// copyMap creates a shallow copy of a map.
func copyMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

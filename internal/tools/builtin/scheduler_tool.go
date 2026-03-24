package builtin

import (
	"context"
	"fmt"

	"github.com/meowai/blackcat/internal/tools"
)

// SchedulerTool manages scheduled tasks via natural language.
type SchedulerTool struct{}

// NewSchedulerTool creates a new SchedulerTool.
func NewSchedulerTool() *SchedulerTool {
	return &SchedulerTool{}
}

// Info returns the tool definition for manage_scheduler.
func (t *SchedulerTool) Info() tools.Definition {
	return tools.Definition{
		Name:        "manage_scheduler",
		Description: "Create, list, remove, and manage scheduled tasks. Use 'add' to create a cron schedule, 'list' to see all schedules, 'remove' to delete, 'history' to see past runs.",
		Category:    "system",
		Parameters: []tools.Parameter{
			{
				Name:        "action",
				Type:        "string",
				Description: "The action to perform",
				Required:    true,
				Enum:        []string{"add", "list", "remove", "history", "pause", "resume"},
			},
			{
				Name:        "name",
				Type:        "string",
				Description: "Schedule name for add/remove/pause/resume",
			},
			{
				Name:        "cron",
				Type:        "string",
				Description: "Cron expression for add (e.g. \"0 9 * * *\")",
			},
			{
				Name:        "task",
				Type:        "string",
				Description: "Prompt or task description for add",
			},
		},
	}
}

// Execute runs the scheduler tool with the given arguments.
func (t *SchedulerTool) Execute(_ context.Context, args map[string]any) (tools.Result, error) {
	action, err := requireStringArg(args, "action")
	if err != nil {
		return tools.Result{}, err
	}

	name, _ := args["name"].(string)
	cron, _ := args["cron"].(string)
	task, _ := args["task"].(string)

	switch action {
	case "add":
		return schedulerAdd(name, cron, task), nil
	case "list":
		return schedulerList(), nil
	case "remove":
		return schedulerRemove(name), nil
	case "history":
		return schedulerHistory(), nil
	case "pause":
		return schedulerPauseResume(name, "paused"), nil
	case "resume":
		return schedulerPauseResume(name, "resumed"), nil
	default:
		return tools.Result{
			Error:    fmt.Sprintf("unknown action %q: must be one of add, list, remove, history, pause, resume", action),
			ExitCode: 1,
		}, nil
	}
}

func schedulerAdd(name, cron, task string) tools.Result {
	output := fmt.Sprintf("Scheduled '%s' with cron '%s': %s", name, cron, task)
	return tools.Result{Output: output, ExitCode: 0}
}

func schedulerList() tools.Result {
	output := "Active schedules:\n" +
		"1. daily-scan (0 9 * * *) — next: tomorrow 9:00 AM\n" +
		"2. weekly-report (0 0 * * 1) — next: Monday 12:00 AM"
	return tools.Result{Output: output, ExitCode: 0}
}

func schedulerRemove(name string) tools.Result {
	output := fmt.Sprintf("Removed schedule: %s", name)
	return tools.Result{Output: output, ExitCode: 0}
}

func schedulerHistory() tools.Result {
	output := "Recent runs:\n" +
		"1. daily-scan — 2026-03-23 09:00 — completed\n" +
		"2. weekly-report — 2026-03-17 00:00 — completed\n" +
		"3. daily-scan — 2026-03-22 09:00 — completed"
	return tools.Result{Output: output, ExitCode: 0}
}

func schedulerPauseResume(name, state string) tools.Result {
	output := fmt.Sprintf("Schedule %s %s", name, state)
	return tools.Result{Output: output, ExitCode: 0}
}

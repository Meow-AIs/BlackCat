package tui

import "strings"

// AppState represents the current mode of the application.
type AppState string

const (
	AppChat       AppState = "chat"
	AppPermission AppState = "permission"
	AppDiff       AppState = "diff"
)

// App is the main TUI application model combining all components.
type App struct {
	Chat       *ChatModel
	Input      *InputModel
	Toolbar    *ToolbarModel
	Permission *PermissionPrompt // nil when no prompt active
	DiffView   []DiffLine        // nil when no diff
	Width      int
	Height     int
	State      AppState
}

// NewApp creates a new App with the given terminal dimensions.
func NewApp(width, height int) *App {
	return &App{
		Chat:    NewChatModel(500),
		Input:   NewInputModel(4096),
		Toolbar: NewToolbarModel(),
		Width:   width,
		Height:  height,
		State:   AppChat,
	}
}

// Render produces the full TUI output based on the current state.
func (a *App) Render() string {
	var b strings.Builder

	// Toolbar always visible at top
	b.WriteString(a.Toolbar.Render(a.Width))
	b.WriteString("\n")

	switch a.State {
	case AppPermission:
		if a.Permission != nil {
			b.WriteString(a.Permission.Render(a.Width))
		}
	case AppDiff:
		b.WriteString(RenderDiff(a.DiffView, a.Width))
	default:
		// Chat state: chat area + input
		chatHeight := a.Height - 4 // toolbar + input + borders
		b.WriteString(a.Chat.Render(a.Width, chatHeight))
		b.WriteString("\n")
		b.WriteString(a.Input.Render(a.Width))
	}

	return b.String()
}

// Resize updates the terminal dimensions.
func (a *App) Resize(width, height int) {
	a.Width = width
	a.Height = height
}

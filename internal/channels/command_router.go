package channels

import "github.com/meowai/blackcat/internal/commands"

// CommandRouter intercepts slash commands from channel messages and routes
// them to the command registry before the LLM sees them.
type CommandRouter struct {
	middleware *commands.InputMiddleware
}

// NewCommandRouter creates a router that uses the given registry to handle
// slash commands arriving from any messaging channel.
func NewCommandRouter(registry *commands.Registry) *CommandRouter {
	return &CommandRouter{
		middleware: commands.NewInputMiddleware(registry),
	}
}

// ProcessMessage checks if an incoming message is a slash command.
// If the message is a command, it executes the command and returns an
// OutgoingMessage with the result. If the message is not a command,
// it returns nil — the caller should forward such messages to the LLM.
func (r *CommandRouter) ProcessMessage(msg IncomingMessage) *OutgoingMessage {
	result, handled := r.middleware.Process(msg.Text)
	if !handled {
		return nil
	}

	// Silent commands produce no visible response.
	if result.Silent {
		return nil
	}

	text := result.Output
	if result.Error != "" {
		text = result.Error
	}

	return &OutgoingMessage{
		ChannelID: msg.ChannelID,
		Text:      text,
		ReplyToID: msg.ReplyToID,
		Format:    FormatPlain,
	}
}

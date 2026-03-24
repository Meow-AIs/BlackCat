package channels

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// MessageHandler is called when a channel receives a message.
type MessageHandler func(ctx context.Context, msg IncomingMessage) (string, error)

// Gateway manages all channel adapters and routes messages.
type Gateway struct {
	mu        sync.Mutex
	adapters  map[Platform]Adapter
	handler   MessageHandler
	sessions  *SessionManager
	limiter   *RateLimiter
	pairing   *PairingManager
	cmdRouter *CommandRouter
}

// GatewayOption configures optional Gateway behaviour.
type GatewayOption func(*Gateway)

// WithCommandRouter attaches a CommandRouter so slash commands are handled
// directly on the channel without going through the LLM.
func WithCommandRouter(router *CommandRouter) GatewayOption {
	return func(g *Gateway) {
		g.cmdRouter = router
	}
}

// NewGateway creates a gateway with the given message handler.
func NewGateway(handler MessageHandler, opts ...GatewayOption) *Gateway {
	gw := &Gateway{
		adapters: make(map[Platform]Adapter),
		handler:  handler,
		sessions: NewSessionManager(),
		limiter:  NewRateLimiter(30, 60), // 30 msgs per 60 seconds
		pairing:  NewPairingManager(),
	}
	for _, opt := range opts {
		opt(gw)
	}
	return gw
}

// Register adds a channel adapter to the gateway.
func (g *Gateway) Register(adapter Adapter) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.adapters[adapter.Platform()] = adapter
}

// Start launches all registered adapters and begins routing messages.
func (g *Gateway) Start(ctx context.Context) error {
	g.mu.Lock()
	adapters := make([]Adapter, 0, len(g.adapters))
	for _, a := range g.adapters {
		adapters = append(adapters, a)
	}
	g.mu.Unlock()

	for _, adapter := range adapters {
		if err := adapter.Start(ctx); err != nil {
			return fmt.Errorf("start %s: %w", adapter.Platform(), err)
		}
		go g.routeMessages(ctx, adapter)
	}

	return nil
}

// Stop gracefully shuts down all adapters.
func (g *Gateway) Stop(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	for platform, adapter := range g.adapters {
		if err := adapter.Stop(ctx); err != nil {
			log.Printf("error stopping %s: %v", platform, err)
		}
	}
	return nil
}

// Adapters returns a list of registered platforms.
func (g *Gateway) Adapters() []Platform {
	g.mu.Lock()
	defer g.mu.Unlock()

	platforms := make([]Platform, 0, len(g.adapters))
	for p := range g.adapters {
		platforms = append(platforms, p)
	}
	return platforms
}

func (g *Gateway) routeMessages(ctx context.Context, adapter Adapter) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-adapter.Receive():
			if !ok {
				return
			}
			go g.handleMessage(ctx, adapter, msg)
		}
	}
}

func (g *Gateway) handleMessage(ctx context.Context, adapter Adapter, msg IncomingMessage) {
	userKey := fmt.Sprintf("%s:%s", msg.Platform, msg.UserID)

	// Rate limiting
	if !g.limiter.Allow(userKey) {
		adapter.Send(ctx, OutgoingMessage{
			ChannelID: msg.ChannelID,
			Text:      "Rate limited. Please wait before sending more messages.",
			ReplyToID: msg.ReplyToID,
			Format:    FormatPlain,
		})
		return
	}

	// Slash command interception — respond instantly, skip LLM.
	if g.cmdRouter != nil {
		if reply := g.cmdRouter.ProcessMessage(msg); reply != nil {
			adapter.Send(ctx, *reply)
			return
		}
	}

	// Process via handler
	if g.handler != nil {
		response, err := g.handler(ctx, msg)
		if err != nil {
			response = fmt.Sprintf("Error: %s", err.Error())
		}

		adapter.Send(ctx, OutgoingMessage{
			ChannelID: msg.ChannelID,
			Text:      response,
			ReplyToID: msg.ReplyToID,
			Format:    FormatMarkdown,
		})
	}
}

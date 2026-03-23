package plugin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/meowai/blackcat/internal/channels"
	"github.com/meowai/blackcat/internal/llm"
)

// ProviderBridge wraps a provider plugin as an llm.Provider.
type ProviderBridge struct {
	client   *PluginClient
	name     string
	manifest PluginManifest
}

// NewProviderBridge creates a bridge that translates llm.Provider calls
// into plugin JSON-RPC calls.
func NewProviderBridge(client *PluginClient, manifest PluginManifest) *ProviderBridge {
	return &ProviderBridge{
		client:   client,
		name:     manifest.Name,
		manifest: manifest,
	}
}

// Chat sends a chat request to the plugin and returns the response.
func (b *ProviderBridge) Chat(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
	params := map[string]any{
		"model":    req.Model,
		"messages": req.Messages,
	}
	if req.MaxTokens > 0 {
		params["max_tokens"] = req.MaxTokens
	}
	if req.Temperature != nil {
		params["temperature"] = *req.Temperature
	}
	if len(req.Tools) > 0 {
		params["tools"] = req.Tools
	}

	result, err := b.client.CallWithTimeout(ctx, "chat", params)
	if err != nil {
		return llm.ChatResponse{}, fmt.Errorf("plugin chat: %w", err)
	}

	return decodeProviderChatResponse(result)
}

// Stream sends a streaming request to the plugin. Currently returns a single
// chunk from a non-streaming call — full streaming support requires an
// event-based protocol extension.
func (b *ProviderBridge) Stream(ctx context.Context, req llm.ChatRequest) (<-chan llm.StreamChunk, error) {
	resp, err := b.Chat(ctx, req)
	if err != nil {
		return nil, err
	}

	ch := make(chan llm.StreamChunk, 1)
	ch <- llm.StreamChunk{
		Content:      resp.Content,
		Done:         true,
		FinishReason: resp.FinishReason,
		Usage:        &resp.Usage,
	}
	close(ch)
	return ch, nil
}

// Models returns the list of models available from the plugin provider.
func (b *ProviderBridge) Models() []llm.ModelInfo {
	result, err := b.client.Call("models", nil)
	if err != nil {
		return nil
	}
	return decodeModelInfoList(result)
}

// Name returns the provider name.
func (b *ProviderBridge) Name() string {
	return b.name
}

// decodeProviderChatResponse converts a raw plugin response into llm.ChatResponse.
func decodeProviderChatResponse(result any) (llm.ChatResponse, error) {
	data, err := json.Marshal(result)
	if err != nil {
		return llm.ChatResponse{}, fmt.Errorf("encode result: %w", err)
	}
	var resp llm.ChatResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return llm.ChatResponse{}, fmt.Errorf("decode chat response: %w", err)
	}
	return resp, nil
}

// decodeModelInfoList converts a raw plugin response into []llm.ModelInfo.
func decodeModelInfoList(result any) []llm.ModelInfo {
	data, err := json.Marshal(result)
	if err != nil {
		return nil
	}
	var models []llm.ModelInfo
	if err := json.Unmarshal(data, &models); err != nil {
		return nil
	}
	return models
}

// ChannelBridge wraps a channel plugin as a channels.Adapter.
type ChannelBridge struct {
	client   *PluginClient
	manifest PluginManifest
	incoming chan channels.IncomingMessage
}

// NewChannelBridge creates a bridge that translates channels.Adapter calls
// into plugin JSON-RPC calls.
func NewChannelBridge(client *PluginClient, manifest PluginManifest) *ChannelBridge {
	return &ChannelBridge{
		client:   client,
		manifest: manifest,
		incoming: make(chan channels.IncomingMessage, 64),
	}
}

// Start tells the plugin to begin listening on its platform.
func (b *ChannelBridge) Start(ctx context.Context) error {
	_, err := b.client.CallWithTimeout(ctx, "start", nil)
	return err
}

// Stop tells the plugin to disconnect from its platform.
func (b *ChannelBridge) Stop(ctx context.Context) error {
	_, err := b.client.CallWithTimeout(ctx, "stop", nil)
	return err
}

// Send delivers a message through the plugin.
func (b *ChannelBridge) Send(ctx context.Context, msg channels.OutgoingMessage) error {
	params := map[string]any{
		"channel_id":  msg.ChannelID,
		"text":        msg.Text,
		"reply_to_id": msg.ReplyToID,
		"format":      string(msg.Format),
	}
	_, err := b.client.CallWithTimeout(ctx, "send", params)
	return err
}

// Receive returns a channel that emits incoming messages from the plugin.
func (b *ChannelBridge) Receive() <-chan channels.IncomingMessage {
	return b.incoming
}

// Platform returns a platform identifier for this plugin channel.
func (b *ChannelBridge) Platform() channels.Platform {
	return channels.Platform("plugin:" + b.manifest.Name)
}

// DomainBridge wraps a domain plugin as domain knowledge.
type DomainBridge struct {
	client   *PluginClient
	manifest PluginManifest
}

// NewDomainBridge creates a bridge for a domain specialization plugin.
func NewDomainBridge(client *PluginClient, manifest PluginManifest) *DomainBridge {
	return &DomainBridge{
		client:   client,
		manifest: manifest,
	}
}

// SystemPrompt returns the domain-specific system prompt from the plugin.
func (b *DomainBridge) SystemPrompt() string {
	result, err := b.client.Call("system_prompt", nil)
	if err != nil {
		return ""
	}
	s, _ := result.(string)
	return s
}

// Detect asks the plugin how well it matches the given project path.
func (b *DomainBridge) Detect(projectPath string) (float64, error) {
	result, err := b.client.Call("detect", map[string]any{"path": projectPath})
	if err != nil {
		return 0, err
	}

	m, ok := result.(map[string]any)
	if !ok {
		return 0, fmt.Errorf("unexpected detect result type: %T", result)
	}

	conf, ok := m["confidence"].(float64)
	if !ok {
		return 0, fmt.Errorf("missing confidence in detect result")
	}
	return conf, nil
}

// Tools returns the domain-specific tool names from the plugin.
func (b *DomainBridge) Tools() []string {
	result, err := b.client.Call("tools", nil)
	if err != nil {
		return nil
	}

	items, ok := result.([]any)
	if !ok {
		return nil
	}

	toolNames := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok {
			toolNames = append(toolNames, s)
		}
	}
	return toolNames
}

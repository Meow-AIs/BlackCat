package plugin

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/meowai/blackcat/internal/channels"
	"github.com/meowai/blackcat/internal/llm"
)

// mockProviderPlugin simulates a provider plugin.
func mockProviderPlugin(input io.Reader, output io.Writer) {
	decoder := json.NewDecoder(input)
	encoder := json.NewEncoder(output)

	for {
		var req PluginRequest
		if err := decoder.Decode(&req); err != nil {
			return
		}

		var resp PluginResponse
		resp.ID = req.ID

		switch req.Method {
		case "ping":
			resp.Result = "pong"
		case "chat":
			resp.Result = map[string]any{
				"content":       "Hello from plugin provider",
				"model":         "plugin-model",
				"finish_reason": "stop",
				"usage": map[string]any{
					"prompt_tokens":     float64(10),
					"completion_tokens": float64(20),
					"total_tokens":      float64(30),
				},
			}
		case "models":
			resp.Result = []any{
				map[string]any{
					"id":                 "plugin-model",
					"name":               "Plugin Model",
					"max_tokens":         float64(4096),
					"input_cost_per_1m":  float64(1.0),
					"output_cost_per_1m": float64(2.0),
				},
			}
		default:
			resp.Error = "unknown method: " + req.Method
		}

		encoder.Encode(resp)
	}
}

// mockChannelPlugin simulates a channel plugin.
func mockChannelPlugin(input io.Reader, output io.Writer) {
	decoder := json.NewDecoder(input)
	encoder := json.NewEncoder(output)

	for {
		var req PluginRequest
		if err := decoder.Decode(&req); err != nil {
			return
		}

		var resp PluginResponse
		resp.ID = req.ID

		switch req.Method {
		case "ping":
			resp.Result = "pong"
		case "start":
			resp.Result = "ok"
		case "stop":
			resp.Result = "ok"
		case "send":
			resp.Result = "sent"
		default:
			resp.Error = "unknown method: " + req.Method
		}

		encoder.Encode(resp)
	}
}

// mockDomainPlugin simulates a domain plugin.
func mockDomainPlugin(input io.Reader, output io.Writer) {
	decoder := json.NewDecoder(input)
	encoder := json.NewEncoder(output)

	for {
		var req PluginRequest
		if err := decoder.Decode(&req); err != nil {
			return
		}

		var resp PluginResponse
		resp.ID = req.ID

		switch req.Method {
		case "ping":
			resp.Result = "pong"
		case "system_prompt":
			resp.Result = "You are a Kubernetes expert."
		case "detect":
			resp.Result = map[string]any{
				"confidence": float64(0.85),
			}
		case "tools":
			resp.Result = []any{"kubectl_apply", "helm_install"}
		default:
			resp.Error = "unknown method: " + req.Method
		}

		encoder.Encode(resp)
	}
}

func TestProviderBridgeChat(t *testing.T) {
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	go mockProviderPlugin(stdinR, stdoutW)

	client := NewPluginClient(stdinW, stdoutR)
	defer client.Close()

	manifest := PluginManifest{Name: "test/provider", Type: PluginProvider}
	bridge := NewProviderBridge(client, manifest)

	req := llm.ChatRequest{
		Model:    "plugin-model",
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "hello"}},
	}

	resp, err := bridge.Chat(context.Background(), req)
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}
	if resp.Content != "Hello from plugin provider" {
		t.Errorf("expected 'Hello from plugin provider', got %s", resp.Content)
	}
	if resp.Model != "plugin-model" {
		t.Errorf("expected model plugin-model, got %s", resp.Model)
	}
}

func TestProviderBridgeModels(t *testing.T) {
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	go mockProviderPlugin(stdinR, stdoutW)

	client := NewPluginClient(stdinW, stdoutR)
	defer client.Close()

	manifest := PluginManifest{Name: "test/provider", Type: PluginProvider}
	bridge := NewProviderBridge(client, manifest)

	models := bridge.Models()
	if len(models) == 0 {
		t.Fatal("expected at least one model")
	}
	if models[0].ID != "plugin-model" {
		t.Errorf("expected plugin-model, got %s", models[0].ID)
	}
}

func TestProviderBridgeName(t *testing.T) {
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	go mockProviderPlugin(stdinR, stdoutW)

	client := NewPluginClient(stdinW, stdoutR)
	defer client.Close()

	manifest := PluginManifest{Name: "test/provider", Type: PluginProvider}
	bridge := NewProviderBridge(client, manifest)

	if bridge.Name() != "test/provider" {
		t.Errorf("expected test/provider, got %s", bridge.Name())
	}
}

func TestProviderBridgeImplementsInterface(t *testing.T) {
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	go mockProviderPlugin(stdinR, stdoutW)

	client := NewPluginClient(stdinW, stdoutR)
	defer client.Close()

	manifest := PluginManifest{Name: "test/provider", Type: PluginProvider}
	bridge := NewProviderBridge(client, manifest)

	// Compile-time check that ProviderBridge implements llm.Provider
	var _ llm.Provider = bridge
}

func TestChannelBridgeSend(t *testing.T) {
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	go mockChannelPlugin(stdinR, stdoutW)

	client := NewPluginClient(stdinW, stdoutR)
	defer client.Close()

	manifest := PluginManifest{Name: "test/channel", Type: PluginChannel}
	bridge := NewChannelBridge(client, manifest)

	msg := channels.OutgoingMessage{
		ChannelID: "test-channel",
		Text:      "hello",
		Format:    channels.FormatPlain,
	}

	err := bridge.Send(context.Background(), msg)
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}
}

func TestChannelBridgeStartStop(t *testing.T) {
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	go mockChannelPlugin(stdinR, stdoutW)

	client := NewPluginClient(stdinW, stdoutR)
	defer client.Close()

	manifest := PluginManifest{Name: "test/channel", Type: PluginChannel}
	bridge := NewChannelBridge(client, manifest)

	if err := bridge.Start(context.Background()); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if err := bridge.Stop(context.Background()); err != nil {
		t.Fatalf("stop failed: %v", err)
	}
}

func TestChannelBridgeReceive(t *testing.T) {
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	go mockChannelPlugin(stdinR, stdoutW)

	client := NewPluginClient(stdinW, stdoutR)
	defer client.Close()

	manifest := PluginManifest{Name: "test/channel", Type: PluginChannel}
	bridge := NewChannelBridge(client, manifest)

	ch := bridge.Receive()
	if ch == nil {
		t.Fatal("expected non-nil receive channel")
	}
}

func TestChannelBridgePlatform(t *testing.T) {
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	go mockChannelPlugin(stdinR, stdoutW)

	client := NewPluginClient(stdinW, stdoutR)
	defer client.Close()

	manifest := PluginManifest{Name: "test/channel", Type: PluginChannel}
	bridge := NewChannelBridge(client, manifest)

	p := bridge.Platform()
	if p != channels.Platform("plugin:test/channel") {
		t.Errorf("expected platform 'plugin:test/channel', got %s", p)
	}
}

func TestChannelBridgeImplementsInterface(t *testing.T) {
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	go mockChannelPlugin(stdinR, stdoutW)

	client := NewPluginClient(stdinW, stdoutR)
	defer client.Close()

	manifest := PluginManifest{Name: "test/channel", Type: PluginChannel}
	bridge := NewChannelBridge(client, manifest)

	var _ channels.Adapter = bridge
}

func TestDomainBridgeSystemPrompt(t *testing.T) {
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	go mockDomainPlugin(stdinR, stdoutW)

	client := NewPluginClient(stdinW, stdoutR)
	defer client.Close()

	manifest := PluginManifest{Name: "test/domain", Type: PluginDomain}
	bridge := NewDomainBridge(client, manifest)

	prompt := bridge.SystemPrompt()
	if prompt != "You are a Kubernetes expert." {
		t.Errorf("expected 'You are a Kubernetes expert.', got %s", prompt)
	}
}

func TestDomainBridgeDetect(t *testing.T) {
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	go mockDomainPlugin(stdinR, stdoutW)

	client := NewPluginClient(stdinW, stdoutR)
	defer client.Close()

	manifest := PluginManifest{Name: "test/domain", Type: PluginDomain}
	bridge := NewDomainBridge(client, manifest)

	confidence, err := bridge.Detect("/some/project")
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if confidence != 0.85 {
		t.Errorf("expected confidence 0.85, got %f", confidence)
	}
}

func TestDomainBridgeTools(t *testing.T) {
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	go mockDomainPlugin(stdinR, stdoutW)

	client := NewPluginClient(stdinW, stdoutR)
	defer client.Close()

	manifest := PluginManifest{Name: "test/domain", Type: PluginDomain}
	bridge := NewDomainBridge(client, manifest)

	tools := bridge.Tools()
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
	if tools[0] != "kubectl_apply" {
		t.Errorf("expected kubectl_apply, got %s", tools[0])
	}
}

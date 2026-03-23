package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/meowai/blackcat/internal/llm"
	"github.com/meowai/blackcat/internal/memory"
	"github.com/meowai/blackcat/internal/security"
	"github.com/meowai/blackcat/internal/tools"
)

// SkillProvider returns skill context for prompt injection.
type SkillProvider interface {
	FormatSkillContext() string
}

// DomainProvider returns domain-specific context.
type DomainProvider interface {
	DetectedDomain() string
	DomainPrompt() string
}

// NudgeProvider returns behavioral nudge messages.
type NudgeProvider interface {
	CurrentNudges() []string
}

// Core is the main agent orchestrator.
type Core struct {
	provider    llm.Provider
	registry    tools.Registry
	memEngine   memory.Engine
	checker     *security.PermissionChecker
	costTracker *llm.CostTracker

	// Context providers — optional, wire these to enable full knowledge injection
	skillProvider  SkillProvider
	domainProvider DomainProvider
	nudgeProvider  NudgeProvider

	tokenBudget int // system prompt token budget (default 4000)

	mu       sync.Mutex
	sessions map[string]*sessionState
}

type sessionState struct {
	Session  Session
	Messages []llm.Message
}

// CoreConfig holds dependencies for creating a Core agent.
type CoreConfig struct {
	Provider    llm.Provider
	Registry    tools.Registry
	MemEngine   memory.Engine
	Checker     *security.PermissionChecker
	CostTracker *llm.CostTracker

	// Optional providers for full context injection
	SkillProvider  SkillProvider
	DomainProvider DomainProvider
	NudgeProvider  NudgeProvider
	TokenBudget    int // 0 = default 4000
}

// NewCore creates a new agent core.
func NewCore(cfg CoreConfig) *Core {
	budget := cfg.TokenBudget
	if budget <= 0 {
		budget = 4000
	}
	return &Core{
		provider:       cfg.Provider,
		registry:       cfg.Registry,
		memEngine:      cfg.MemEngine,
		checker:        cfg.Checker,
		costTracker:    cfg.CostTracker,
		skillProvider:  cfg.SkillProvider,
		domainProvider: cfg.DomainProvider,
		nudgeProvider:  cfg.NudgeProvider,
		tokenBudget:    budget,
		sessions:       make(map[string]*sessionState),
	}
}

func (c *Core) StartSession(_ context.Context, projectID string, userID string) (Session, error) {
	s := Session{
		ID:        fmt.Sprintf("sess-%d", time.Now().UnixNano()),
		ProjectID: projectID,
		UserID:    userID,
		State:     StateIdle,
		CreatedAt: time.Now().Unix(),
	}

	c.mu.Lock()
	c.sessions[s.ID] = &sessionState{
		Session:  s,
		Messages: []llm.Message{},
	}
	c.mu.Unlock()

	return s, nil
}

func (c *Core) ResumeSession(_ context.Context, sessionID string) (Session, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	ss, ok := c.sessions[sessionID]
	if !ok {
		return Session{}, fmt.Errorf("session %q not found", sessionID)
	}
	return ss.Session, nil
}

// Process handles user input through the agent loop:
// 1. Inject memory snapshot as system context
// 2. Send to LLM
// 3. If tool calls → execute tools → send results back to LLM → repeat
// 4. Return final text response
func (c *Core) Process(ctx context.Context, sessionID string, input string) (Response, error) {
	c.mu.Lock()
	ss, ok := c.sessions[sessionID]
	if !ok {
		c.mu.Unlock()
		return Response{}, fmt.Errorf("session %q not found", sessionID)
	}
	ss.Session.State = StateThinking
	c.mu.Unlock()

	// Add user message
	ss.Messages = append(ss.Messages, llm.Message{
		Role:    llm.RoleUser,
		Content: input,
	})

	// Build system prompt with memory snapshot
	systemPrompt := c.buildSystemPrompt(ctx, ss)

	maxIterations := 10
	var allToolUses []ToolUse

	for i := 0; i < maxIterations; i++ {
		// Build messages with system prompt
		messages := append([]llm.Message{{Role: llm.RoleSystem, Content: systemPrompt}}, ss.Messages...)

		// Build tool definitions
		var toolDefs []llm.ToolDefinition
		for _, def := range c.registry.List() {
			toolDefs = append(toolDefs, llm.ToolDefinition{
				Name:        def.Name,
				Description: def.Description,
				Parameters:  buildJSONSchema(def.Parameters),
			})
		}

		// Call LLM
		resp, err := c.provider.Chat(ctx, llm.ChatRequest{
			Model:     "", // use default
			Messages:  messages,
			Tools:     toolDefs,
			MaxTokens: 4096,
		})
		if err != nil {
			return Response{}, fmt.Errorf("LLM call failed: %w", err)
		}

		// Track cost
		if c.costTracker != nil {
			c.costTracker.Record(resp.Model, resp.Usage, nil)
		}

		// No tool calls → return text response
		if len(resp.ToolCalls) == 0 {
			ss.Messages = append(ss.Messages, llm.Message{
				Role:    llm.RoleAssistant,
				Content: resp.Content,
			})
			ss.Session.State = StateDone
			return Response{
				Text:      resp.Content,
				ToolCalls: allToolUses,
				Done:      true,
			}, nil
		}

		// Has tool calls → execute them
		ss.Messages = append(ss.Messages, llm.Message{
			Role:      llm.RoleAssistant,
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		ss.Session.State = StateExecuting

		for _, tc := range resp.ToolCalls {
			toolUse := ToolUse{Name: tc.Name}

			// Parse args
			var args map[string]any
			if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
				args = map[string]any{}
			}
			toolUse.Args = args

			// Check permissions
			if c.checker != nil {
				action := security.Action{Type: toolActionType(tc.Name), Command: tc.Arguments}
				decision := c.checker.Check(action)
				if decision.Level == security.LevelDeny {
					toolUse.Error = "permission denied"
					ss.Messages = append(ss.Messages, llm.Message{
						Role:       llm.RoleTool,
						Content:    "Error: permission denied",
						ToolCallID: tc.ID,
					})
					allToolUses = append(allToolUses, toolUse)
					continue
				}
			}

			// Execute tool
			tool := c.registry.Get(tc.Name)
			if tool == nil {
				toolUse.Error = "tool not found"
				ss.Messages = append(ss.Messages, llm.Message{
					Role:       llm.RoleTool,
					Content:    fmt.Sprintf("Error: tool %q not found", tc.Name),
					ToolCallID: tc.ID,
				})
			} else {
				result, err := tool.Execute(ctx, args)
				if err != nil {
					sanitizedErr := sanitizeForLLM(err.Error())
					toolUse.Error = sanitizedErr
					ss.Messages = append(ss.Messages, llm.Message{
						Role:       llm.RoleTool,
						Content:    fmt.Sprintf("Error: %s", sanitizedErr),
						ToolCallID: tc.ID,
					})
				} else {
					output := result.Output
					if result.Error != "" {
						output = fmt.Sprintf("Error: %s\n%s", result.Error, output)
					}
					// Sanitize tool output before adding to LLM message history
					// to prevent secrets from leaking to the LLM provider.
					output = sanitizeForLLM(output)
					toolUse.Result = output
					ss.Messages = append(ss.Messages, llm.Message{
						Role:       llm.RoleTool,
						Content:    output,
						ToolCallID: tc.ID,
					})
				}
			}
			allToolUses = append(allToolUses, toolUse)
		}

		ss.Session.State = StateThinking
	}

	return Response{
		Text:      "Maximum iterations reached",
		ToolCalls: allToolUses,
		Done:      true,
	}, nil
}

func (c *Core) SpawnSubAgent(_ context.Context, _ string, task SubAgentTask) (string, error) {
	// Placeholder — full sub-agent implementation in Phase 4
	return task.ID, nil
}

func (c *Core) ListSubAgents(_ context.Context, _ string) ([]SubAgentTask, error) {
	return nil, nil
}

func (c *Core) buildSystemPrompt(ctx context.Context, ss *sessionState) string {
	asm := NewContextAssembler(c.tokenBudget)

	// Layer 1: Persona (required, always present)
	asm.AddLayer(NewContextLayer("persona",
		"You are BlackCat, an AI coding agent by MeowAI. "+
			"You help users with software engineering tasks. "+
			"You can read/write files, run commands, search code, and manage projects. "+
			"Always explain what you're doing. Ask for permission before destructive actions.",
		100, LayerRequired))

	// Layer 2: Domain knowledge (required when detected)
	if c.domainProvider != nil {
		if domainPrompt := c.domainProvider.DomainPrompt(); domainPrompt != "" {
			asm.AddLayer(NewContextLayer("domain", domainPrompt, 95, LayerRequired))
		}
	}

	// Layer 3: ReAct reasoning instructions (important)
	asm.AddLayer(NewContextLayer("reasoning",
		"Use structured reasoning when solving complex tasks:\n"+
			"1. <thinking> — analyze the problem\n"+
			"2. <action> — decide what tool to use\n"+
			"3. <critique> — evaluate the result (confidence 0-1)\n"+
			"If confidence < 0.7, retry with a different approach (max 2 retries).\n"+
			"If stuck, step back and ask: what is the higher-level goal?",
		85, LayerImportant))

	// Layer 4: Memory snapshot (important)
	if c.memEngine != nil {
		snap, err := c.memEngine.BuildSnapshot(ctx, ss.Session.ProjectID, ss.Session.UserID)
		if err == nil && snap.Content != "" {
			asm.AddLayer(NewContextLayer("memory", snap.Content, 80, LayerImportant))
		}
	}

	// Layer 5: Skills index (important)
	if c.skillProvider != nil {
		if skillCtx := c.skillProvider.FormatSkillContext(); skillCtx != "" {
			asm.AddLayer(NewContextLayer("skills", skillCtx, 70, LayerImportant))
		}
	}

	// Layer 6: Available tools summary (important)
	toolDefs := c.registry.List()
	if len(toolDefs) > 0 {
		var toolList strings.Builder
		for _, def := range toolDefs {
			fmt.Fprintf(&toolList, "- **%s**: %s\n", def.Name, def.Description)
		}
		asm.AddLayer(NewContextLayer("tools", toolList.String(), 60, LayerImportant))
	}

	// Layer 7: Behavioral nudges (optional)
	if c.nudgeProvider != nil {
		nudges := c.nudgeProvider.CurrentNudges()
		if len(nudges) > 0 {
			asm.AddLayer(NewContextLayer("nudges", strings.Join(nudges, "\n"), 20, LayerOptional))
		}
	}

	return asm.Assemble()
}

func buildJSONSchema(params []tools.Parameter) map[string]any {
	properties := make(map[string]any)
	var required []string

	for _, p := range params {
		prop := map[string]any{
			"type":        p.Type,
			"description": p.Description,
		}
		if len(p.Enum) > 0 {
			prop["enum"] = p.Enum
		}
		properties[p.Name] = prop
		if p.Required {
			required = append(required, p.Name)
		}
	}

	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func toolActionType(name string) security.ActionType {
	switch name {
	case "read_file":
		return security.ActionReadFile
	case "write_file":
		return security.ActionWriteFile
	case "execute":
		return security.ActionShell
	case "list_dir":
		return security.ActionListDir
	case "search_files", "search_content":
		return security.ActionSearchCode
	default:
		return security.ActionType(name)
	}
}

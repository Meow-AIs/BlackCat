package llm

import (
	"context"
	"errors"
	"testing"
)

// mockProvider is a configurable test double that satisfies the Provider interface.
type mockProvider struct {
	name     string
	chatResp ChatResponse
	chatErr  error
	models   []ModelInfo
	calls    []ChatRequest
}

func (m *mockProvider) Name() string { return m.name }

func (m *mockProvider) Models() []ModelInfo {
	if m.models == nil {
		return []ModelInfo{}
	}
	return m.models
}

func (m *mockProvider) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	m.calls = append(m.calls, req)
	return m.chatResp, m.chatErr
}

func (m *mockProvider) Stream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	return nil, errors.New("mock: stream not implemented")
}

func newMock(name string) *mockProvider {
	return &mockProvider{
		name: name,
		chatResp: ChatResponse{
			Content: "mock response from " + name,
			Model:   name + "-model",
			Usage: Usage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Route() tests
// ---------------------------------------------------------------------------

func TestRouterRoute_MainTasks(t *testing.T) {
	main := newMock("main")
	aux := newMock("aux")
	local := newMock("local")
	router := NewModelRouter(main, aux, local, nil)

	mainTasks := []TaskType{TaskReasoning, TaskCodeGen, TaskVision}
	for _, tt := range mainTasks {
		got := router.Route(tt)
		if got != main {
			t.Errorf("Route(%v) = %q, want main provider", tt, got.Name())
		}
	}
}

func TestRouterRoute_AuxiliaryTasks(t *testing.T) {
	main := newMock("main")
	aux := newMock("aux")
	local := newMock("local")
	router := NewModelRouter(main, aux, local, nil)

	auxTasks := []TaskType{
		TaskSummarize,
		TaskClassify,
		TaskExtractFacts,
		TaskMemorySearch,
		TaskDangerAssess,
		TaskCompression,
	}
	for _, tt := range auxTasks {
		got := router.Route(tt)
		if got != aux {
			t.Errorf("Route(%v) = %q, want aux provider", tt, got.Name())
		}
	}
}

func TestRouterRoute_EmbedWithLocalProvider(t *testing.T) {
	main := newMock("main")
	aux := newMock("aux")
	local := newMock("local")
	router := NewModelRouter(main, aux, local, nil)

	got := router.Route(TaskEmbed)
	if got != local {
		t.Errorf("Route(TaskEmbed) = %q, want local provider", got.Name())
	}
}

func TestRouterRoute_EmbedFallsBackToAuxWhenLocalNil(t *testing.T) {
	main := newMock("main")
	aux := newMock("aux")
	router := NewModelRouter(main, aux, nil, nil)

	got := router.Route(TaskEmbed)
	if got != aux {
		t.Errorf("Route(TaskEmbed) with nil local = %q, want aux provider", got.Name())
	}
}

func TestRouterRoute_AllTaskTypesHandled(t *testing.T) {
	main := newMock("main")
	aux := newMock("aux")
	local := newMock("local")
	router := NewModelRouter(main, aux, local, nil)

	// Ensure every declared TaskType returns a non-nil provider (no panic, no nil).
	allTasks := []TaskType{
		TaskReasoning,
		TaskCodeGen,
		TaskSummarize,
		TaskClassify,
		TaskEmbed,
		TaskExtractFacts,
		TaskMemorySearch,
		TaskDangerAssess,
		TaskCompression,
		TaskVision,
	}
	for _, tt := range allTasks {
		got := router.Route(tt)
		if got == nil {
			t.Errorf("Route(%v) returned nil provider", tt)
		}
	}
}

// ---------------------------------------------------------------------------
// SetOverride() tests
// ---------------------------------------------------------------------------

func TestRouterSetOverride_TakesPrecedence(t *testing.T) {
	main := newMock("main")
	aux := newMock("aux")
	special := newMock("special")
	router := NewModelRouter(main, aux, nil, nil)

	router.SetOverride(TaskReasoning, special)

	got := router.Route(TaskReasoning)
	if got != special {
		t.Errorf("Route(TaskReasoning) after override = %q, want special provider", got.Name())
	}
}

func TestRouterSetOverride_DoesNotAffectOtherTasks(t *testing.T) {
	main := newMock("main")
	aux := newMock("aux")
	special := newMock("special")
	router := NewModelRouter(main, aux, nil, nil)

	router.SetOverride(TaskReasoning, special)

	// TaskCodeGen should still route to main, not special.
	got := router.Route(TaskCodeGen)
	if got != main {
		t.Errorf("Route(TaskCodeGen) = %q, want main (override should not affect it)", got.Name())
	}
}

func TestRouterSetOverride_AuxTaskOverridden(t *testing.T) {
	main := newMock("main")
	aux := newMock("aux")
	premium := newMock("premium")
	router := NewModelRouter(main, aux, nil, nil)

	router.SetOverride(TaskSummarize, premium)

	got := router.Route(TaskSummarize)
	if got != premium {
		t.Errorf("Route(TaskSummarize) after override = %q, want premium provider", got.Name())
	}
}

// ---------------------------------------------------------------------------
// Chat() delegation tests
// ---------------------------------------------------------------------------

func TestRouterChat_DelegatesToCorrectProvider(t *testing.T) {
	main := newMock("main")
	aux := newMock("aux")
	router := NewModelRouter(main, aux, nil, nil)

	ctx := context.Background()
	req := ChatRequest{Model: "test-model", Messages: []Message{{Role: RoleUser, Content: "hello"}}}

	_, err := router.Chat(ctx, TaskReasoning, req)
	if err != nil {
		t.Fatalf("Chat(TaskReasoning) unexpected error: %v", err)
	}

	if len(main.calls) != 1 {
		t.Errorf("main provider call count = %d, want 1", len(main.calls))
	}
	if len(aux.calls) != 0 {
		t.Errorf("aux provider call count = %d, want 0", len(aux.calls))
	}
}

func TestRouterChat_AuxTaskGoesToAux(t *testing.T) {
	main := newMock("main")
	aux := newMock("aux")
	router := NewModelRouter(main, aux, nil, nil)

	ctx := context.Background()
	req := ChatRequest{Model: "test-model", Messages: []Message{{Role: RoleUser, Content: "summarize this"}}}

	_, err := router.Chat(ctx, TaskSummarize, req)
	if err != nil {
		t.Fatalf("Chat(TaskSummarize) unexpected error: %v", err)
	}

	if len(aux.calls) != 1 {
		t.Errorf("aux provider call count = %d, want 1", len(aux.calls))
	}
	if len(main.calls) != 0 {
		t.Errorf("main provider call count = %d, want 0", len(main.calls))
	}
}

func TestRouterChat_ReturnsProviderResponse(t *testing.T) {
	main := newMock("main")
	main.chatResp = ChatResponse{Content: "deep thoughts", Model: "gpt-flagship"}
	aux := newMock("aux")
	router := NewModelRouter(main, aux, nil, nil)

	ctx := context.Background()
	req := ChatRequest{Model: "gpt-flagship", Messages: []Message{{Role: RoleUser, Content: "think"}}}

	resp, err := router.Chat(ctx, TaskReasoning, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "deep thoughts" {
		t.Errorf("resp.Content = %q, want %q", resp.Content, "deep thoughts")
	}
}

func TestRouterChat_PropagatesProviderError(t *testing.T) {
	main := newMock("main")
	main.chatErr = errors.New("network timeout")
	aux := newMock("aux")
	router := NewModelRouter(main, aux, nil, nil)

	ctx := context.Background()
	req := ChatRequest{Model: "any", Messages: []Message{{Role: RoleUser, Content: "hi"}}}

	_, err := router.Chat(ctx, TaskReasoning, req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, main.chatErr) {
		t.Errorf("error = %v, want to wrap %v", err, main.chatErr)
	}
}

// ---------------------------------------------------------------------------
// CostTracker integration tests
// ---------------------------------------------------------------------------

func TestRouterChat_RecordsCostOnSuccess(t *testing.T) {
	main := newMock("main")
	main.chatResp = ChatResponse{
		Content: "result",
		Model:   "main-model",
		Usage:   Usage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150},
	}
	aux := newMock("aux")
	cost := NewCostTracker(0, 0)
	router := NewModelRouter(main, aux, nil, cost)

	ctx := context.Background()
	req := ChatRequest{Model: "main-model", Messages: []Message{{Role: RoleUser, Content: "go"}}}

	_, err := router.Chat(ctx, TaskReasoning, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	summary := cost.Summary()
	if summary.Entries != 1 {
		t.Errorf("cost entries = %d, want 1", summary.Entries)
	}
	if summary.TotalPrompt != 100 {
		t.Errorf("total prompt tokens = %d, want 100", summary.TotalPrompt)
	}
	if summary.TotalCompletion != 50 {
		t.Errorf("total completion tokens = %d, want 50", summary.TotalCompletion)
	}
}

func TestRouterChat_DoesNotRecordCostOnError(t *testing.T) {
	main := newMock("main")
	main.chatErr = errors.New("api error")
	aux := newMock("aux")
	cost := NewCostTracker(0, 0)
	router := NewModelRouter(main, aux, nil, cost)

	ctx := context.Background()
	req := ChatRequest{Model: "any", Messages: []Message{{Role: RoleUser, Content: "hi"}}}

	router.Chat(ctx, TaskReasoning, req) //nolint:errcheck

	summary := cost.Summary()
	if summary.Entries != 0 {
		t.Errorf("cost entries = %d, want 0 (error should not be recorded)", summary.Entries)
	}
}

func TestRouterChat_NilCostTrackerDoesNotPanic(t *testing.T) {
	main := newMock("main")
	aux := newMock("aux")
	router := NewModelRouter(main, aux, nil, nil)

	ctx := context.Background()
	req := ChatRequest{Model: "any", Messages: []Message{{Role: RoleUser, Content: "hi"}}}

	// Should not panic with nil cost tracker.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panic with nil cost tracker: %v", r)
		}
	}()
	router.Chat(ctx, TaskReasoning, req) //nolint:errcheck
}

// ---------------------------------------------------------------------------
// Nil local provider edge cases
// ---------------------------------------------------------------------------

func TestRouterNilLocal_AllNonEmbedTasksStillRoute(t *testing.T) {
	main := newMock("main")
	aux := newMock("aux")
	router := NewModelRouter(main, aux, nil, nil)

	for _, tt := range []TaskType{TaskReasoning, TaskCodeGen, TaskVision, TaskSummarize, TaskClassify} {
		got := router.Route(tt)
		if got == nil {
			t.Errorf("Route(%v) with nil local returned nil", tt)
		}
	}
}

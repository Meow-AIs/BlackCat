package llm

import (
	"context"
	"fmt"
)

// TaskType identifies the category of work a model call must perform.
// The router maps each TaskType to the appropriate provider tier.
type TaskType int

const (
	TaskReasoning    TaskType = iota // -> Main (flagship / expensive)
	TaskCodeGen                      // -> Main
	TaskSummarize                    // -> Auxiliary (cheaper)
	TaskClassify                     // -> Auxiliary
	TaskEmbed                        // -> Local, falls back to Auxiliary when Local is nil
	TaskExtractFacts                 // -> Auxiliary
	TaskMemorySearch                 // -> Auxiliary
	TaskDangerAssess                 // -> Auxiliary
	TaskCompression                  // -> Auxiliary
	TaskVision                       // -> Main (requires multimodal capability)
)

// taskTier describes which provider tier handles a given TaskType by default.
type taskTier int

const (
	tierMain      taskTier = iota
	tierAuxiliary          // cheap / fast
	tierLocal              // embedding / private; falls back to auxiliary
)

// defaultTier maps every TaskType to its tier. Using a slice rather than a
// switch so the compiler catches any out-of-bounds access at runtime and we
// can enumerate all values in tests.
var defaultTier = map[TaskType]taskTier{
	TaskReasoning:    tierMain,
	TaskCodeGen:      tierMain,
	TaskVision:       tierMain,
	TaskSummarize:    tierAuxiliary,
	TaskClassify:     tierAuxiliary,
	TaskExtractFacts: tierAuxiliary,
	TaskMemorySearch: tierAuxiliary,
	TaskDangerAssess: tierAuxiliary,
	TaskCompression:  tierAuxiliary,
	TaskEmbed:        tierLocal,
}

// ModelRouter selects the right Provider for each TaskType, applying an
// optional per-task override map and recording cost via CostTracker.
type ModelRouter struct {
	Main      Provider
	Auxiliary Provider
	Local     Provider // may be nil — TaskEmbed falls back to Auxiliary
	Overrides map[TaskType]Provider
	cost      *CostTracker
}

// NewModelRouter creates a router with the supplied provider tiers and an
// optional CostTracker (nil is allowed).
func NewModelRouter(main, auxiliary, local Provider, cost *CostTracker) *ModelRouter {
	return &ModelRouter{
		Main:      main,
		Auxiliary: auxiliary,
		Local:     local,
		Overrides: make(map[TaskType]Provider),
		cost:      cost,
	}
}

// Route returns the Provider that should handle the given TaskType.
// Priority: explicit override > default tier mapping.
// For TaskEmbed the Local provider is used when available; otherwise Auxiliary.
func (r *ModelRouter) Route(taskType TaskType) Provider {
	if p, ok := r.Overrides[taskType]; ok {
		return p
	}

	tier, ok := defaultTier[taskType]
	if !ok {
		// Unknown task type — fall back to Main to be safe.
		return r.Main
	}

	switch tier {
	case tierMain:
		return r.Main
	case tierAuxiliary:
		return r.Auxiliary
	case tierLocal:
		if r.Local != nil {
			return r.Local
		}
		return r.Auxiliary
	default:
		return r.Main
	}
}

// SetOverride registers a provider override for a specific TaskType.
// The override takes precedence over the default tier routing.
func (r *ModelRouter) SetOverride(taskType TaskType, provider Provider) {
	r.Overrides[taskType] = provider
}

// Chat routes the request to the correct provider for taskType, records
// cost on success, and returns the provider's response unchanged.
func (r *ModelRouter) Chat(ctx context.Context, taskType TaskType, req ChatRequest) (ChatResponse, error) {
	provider := r.Route(taskType)

	resp, err := provider.Chat(ctx, req)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("router: %w", err)
	}

	if r.cost != nil {
		r.cost.Record(resp.Model, resp.Usage, nil)
	}

	return resp, nil
}

package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// --- FetchAnthropicModels tests ---

func TestFetchAnthropicModels_Success(t *testing.T) {
	resp := map[string]any{
		"data": []map[string]any{
			{"id": "claude-sonnet-4-6", "display_name": "Claude Sonnet 4.6", "type": "model"},
			{"id": "claude-opus-4-6", "display_name": "Claude Opus 4.6", "type": "model"},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/models" {
			t.Errorf("expected /v1/models, got %s", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("expected x-api-key header")
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("expected anthropic-version header")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	models, err := FetchAnthropicModels(context.Background(), "test-key", srv.URL, srv.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0].ID != "claude-sonnet-4-6" {
		t.Errorf("expected claude-sonnet-4-6, got %s", models[0].ID)
	}
	if models[0].Name != "Claude Sonnet 4.6" {
		t.Errorf("expected Claude Sonnet 4.6, got %s", models[0].Name)
	}
}

func TestFetchAnthropicModels_Enrichment(t *testing.T) {
	resp := map[string]any{
		"data": []map[string]any{
			{"id": "claude-opus-4-6", "display_name": "Claude Opus 4.6", "type": "model"},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	models, err := FetchAnthropicModels(context.Background(), "test-key", srv.URL, srv.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}
	// Should be enriched with known metadata
	if models[0].MaxTokens != 32000 {
		t.Errorf("expected MaxTokens 32000, got %d", models[0].MaxTokens)
	}
	if models[0].InputCost != 15.0 {
		t.Errorf("expected InputCost 15.0, got %f", models[0].InputCost)
	}
}

func TestFetchAnthropicModels_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	_, err := FetchAnthropicModels(context.Background(), "test-key", srv.URL, srv.Client())
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

// --- FetchOpenAIModels tests ---

func TestFetchOpenAIModels_Success(t *testing.T) {
	resp := map[string]any{
		"data": []map[string]any{
			{"id": "gpt-4.1", "object": "model", "owned_by": "openai"},
			{"id": "gpt-4.1-mini", "object": "model", "owned_by": "openai"},
			{"id": "gpt-4.1-nano", "object": "model", "owned_by": "openai"},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/models" {
			t.Errorf("expected /v1/models, got %s", r.URL.Path)
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-key" {
			t.Errorf("expected Bearer auth header, got %s", auth)
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	models, err := FetchOpenAIModels(context.Background(), "test-key", srv.URL, srv.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 3 {
		t.Fatalf("expected 3 models, got %d", len(models))
	}
	// gpt-4.1 should be enriched
	if models[0].MaxTokens != 1000000 {
		t.Errorf("expected MaxTokens 1000000, got %d", models[0].MaxTokens)
	}
	if models[0].InputCost != 2.0 {
		t.Errorf("expected InputCost 2.0, got %f", models[0].InputCost)
	}
}

func TestFetchOpenAIModels_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("unauthorized"))
	}))
	defer srv.Close()

	_, err := FetchOpenAIModels(context.Background(), "bad-key", srv.URL, srv.Client())
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

// --- FetchOpenRouterModels tests ---

func TestFetchOpenRouterModels_Success(t *testing.T) {
	resp := map[string]any{
		"data": []map[string]any{
			{
				"id":             "anthropic/claude-sonnet-4-6",
				"name":           "Claude Sonnet 4.6",
				"context_length": 200000,
				"pricing": map[string]any{
					"prompt":     "0.000003",
					"completion": "0.000015",
				},
			},
			{
				"id":             "openai/gpt-4.1",
				"name":           "GPT-4.1",
				"context_length": 128000,
				"pricing": map[string]any{
					"prompt":     "0.000002",
					"completion": "0.000008",
				},
			},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/models" {
			t.Errorf("expected /v1/models, got %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	models, err := FetchOpenRouterModels(context.Background(), srv.URL, srv.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0].ID != "anthropic/claude-sonnet-4-6" {
		t.Errorf("expected anthropic/claude-sonnet-4-6, got %s", models[0].ID)
	}
	if models[0].Name != "Claude Sonnet 4.6" {
		t.Errorf("expected Claude Sonnet 4.6, got %s", models[0].Name)
	}
	// Pricing: "0.000003" per token = 3.0 per 1M tokens
	if models[0].InputCost != 3.0 {
		t.Errorf("expected InputCost 3.0, got %f", models[0].InputCost)
	}
	// Pricing: "0.000015" per token = 15.0 per 1M tokens
	if models[0].OutputCost != 15.0 {
		t.Errorf("expected OutputCost 15.0, got %f", models[0].OutputCost)
	}
}

func TestFetchOpenRouterModels_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	_, err := FetchOpenRouterModels(context.Background(), srv.URL, srv.Client())
	if err == nil {
		t.Fatal("expected error for 503 response")
	}
}

// --- FetchGroqModels tests ---

func TestFetchGroqModels_Success(t *testing.T) {
	resp := map[string]any{
		"data": []map[string]any{
			{"id": "llama-3.3-70b-versatile", "object": "model", "owned_by": "meta"},
			{"id": "mixtral-8x7b-32768", "object": "model", "owned_by": "mistralai"},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer groq-key" {
			t.Errorf("expected Bearer auth")
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	models, err := FetchGroqModels(context.Background(), "groq-key", srv.URL, srv.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0].ID != "llama-3.3-70b-versatile" {
		t.Errorf("expected llama-3.3-70b-versatile, got %s", models[0].ID)
	}
}

// --- EnrichModelInfo tests ---

func TestEnrichModelInfo_KnownModel(t *testing.T) {
	m := ModelInfo{ID: "claude-opus-4-6", Name: "Claude Opus 4.6"}
	enriched := EnrichModelInfo(m)
	if enriched.MaxTokens != 32000 {
		t.Errorf("expected MaxTokens 32000, got %d", enriched.MaxTokens)
	}
	if enriched.InputCost != 15.0 {
		t.Errorf("expected InputCost 15.0, got %f", enriched.InputCost)
	}
	if enriched.OutputCost != 75.0 {
		t.Errorf("expected OutputCost 75.0, got %f", enriched.OutputCost)
	}
}

func TestEnrichModelInfo_UnknownModel(t *testing.T) {
	m := ModelInfo{ID: "unknown-model", Name: "Unknown", MaxTokens: 1000, InputCost: 0.5, OutputCost: 1.0}
	enriched := EnrichModelInfo(m)
	// Should keep original values
	if enriched.MaxTokens != 1000 {
		t.Errorf("expected MaxTokens 1000, got %d", enriched.MaxTokens)
	}
	if enriched.InputCost != 0.5 {
		t.Errorf("expected InputCost 0.5, got %f", enriched.InputCost)
	}
}

func TestEnrichModelInfo_UnknownModelNoDefaults(t *testing.T) {
	m := ModelInfo{ID: "unknown-model", Name: "Unknown"}
	enriched := EnrichModelInfo(m)
	// No known metadata and no existing values -- stays at zero
	if enriched.MaxTokens != 0 {
		t.Errorf("expected MaxTokens 0, got %d", enriched.MaxTokens)
	}
}

// --- ModelFetcher caching tests ---

func TestModelFetcher_CachesResults(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := map[string]any{
			"data": []map[string]any{
				{"id": "claude-sonnet-4-6", "display_name": "Claude Sonnet 4.6", "type": "model"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	fetcher := NewModelFetcher(1 * time.Hour)

	// First fetch
	models, err := fetcher.FetchModels(context.Background(), "anthropic", srv.URL, "key", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}
	if callCount != 1 {
		t.Errorf("expected 1 API call, got %d", callCount)
	}

	// Second fetch should use cache
	models2, err := fetcher.FetchModels(context.Background(), "anthropic", srv.URL, "key", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models2) != 1 {
		t.Fatalf("expected 1 model from cache, got %d", len(models2))
	}
	if callCount != 1 {
		t.Errorf("expected still 1 API call (cached), got %d", callCount)
	}
}

func TestModelFetcher_CacheExpiry(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := map[string]any{
			"data": []map[string]any{
				{"id": "model-v" + string(rune('0'+callCount)), "display_name": "Model", "type": "model"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	// Very short TTL
	fetcher := NewModelFetcher(1 * time.Millisecond)

	_, err := fetcher.FetchModels(context.Background(), "anthropic", srv.URL, "key", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Wait for cache to expire
	time.Sleep(5 * time.Millisecond)

	_, err = fetcher.FetchModels(context.Background(), "anthropic", srv.URL, "key", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if callCount != 2 {
		t.Errorf("expected 2 API calls after expiry, got %d", callCount)
	}
}

func TestModelFetcher_ClearCache(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := map[string]any{
			"data": []map[string]any{
				{"id": "model", "display_name": "Model", "type": "model"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	fetcher := NewModelFetcher(1 * time.Hour)

	fetcher.FetchModels(context.Background(), "anthropic", srv.URL, "key", nil)
	if callCount != 1 {
		t.Fatalf("expected 1 call, got %d", callCount)
	}

	// Clear specific provider
	fetcher.ClearCache("anthropic")

	fetcher.FetchModels(context.Background(), "anthropic", srv.URL, "key", nil)
	if callCount != 2 {
		t.Errorf("expected 2 calls after clear, got %d", callCount)
	}
}

func TestModelFetcher_ClearAllCache(t *testing.T) {
	fetcher := NewModelFetcher(1 * time.Hour)

	// Manually populate cache
	fetcher.mu.Lock()
	fetcher.cache["provider1"] = []ModelInfo{{ID: "m1"}}
	fetcher.cacheTime["provider1"] = time.Now()
	fetcher.cache["provider2"] = []ModelInfo{{ID: "m2"}}
	fetcher.cacheTime["provider2"] = time.Now()
	fetcher.mu.Unlock()

	fetcher.ClearCache("")

	if cached := fetcher.GetCached("provider1"); cached != nil {
		t.Error("expected nil after clear all")
	}
	if cached := fetcher.GetCached("provider2"); cached != nil {
		t.Error("expected nil after clear all")
	}
}

func TestModelFetcher_GetCached_Expired(t *testing.T) {
	fetcher := NewModelFetcher(1 * time.Millisecond)

	fetcher.mu.Lock()
	fetcher.cache["test"] = []ModelInfo{{ID: "m1"}}
	fetcher.cacheTime["test"] = time.Now().Add(-1 * time.Second)
	fetcher.mu.Unlock()

	if cached := fetcher.GetCached("test"); cached != nil {
		t.Error("expected nil for expired cache")
	}
}

func TestModelFetcher_GetCached_Valid(t *testing.T) {
	fetcher := NewModelFetcher(1 * time.Hour)

	fetcher.mu.Lock()
	fetcher.cache["test"] = []ModelInfo{{ID: "m1"}}
	fetcher.cacheTime["test"] = time.Now()
	fetcher.mu.Unlock()

	cached := fetcher.GetCached("test")
	if cached == nil {
		t.Fatal("expected cached models")
	}
	if len(cached) != 1 || cached[0].ID != "m1" {
		t.Error("unexpected cached result")
	}
}

// --- Default model fallback tests ---

func TestDefaultAnthropicModels(t *testing.T) {
	models := DefaultAnthropicModels()
	if len(models) == 0 {
		t.Fatal("expected non-empty default models")
	}
	found := false
	for _, m := range models {
		if m.ID == "claude-sonnet-4-6" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected claude-sonnet-4-6 in defaults")
	}
}

func TestDefaultOpenAIModels(t *testing.T) {
	models := DefaultOpenAIModels()
	if len(models) == 0 {
		t.Fatal("expected non-empty default models")
	}
}

func TestDefaultOpenRouterModels(t *testing.T) {
	models := DefaultOpenRouterModels()
	if len(models) == 0 {
		t.Fatal("expected non-empty default models")
	}
}

// --- DynamicProvider tests ---

type fetcherMockProvider struct {
	name   string
	models []ModelInfo
}

func (m *fetcherMockProvider) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	return ChatResponse{Content: "mock response"}, nil
}

func (m *fetcherMockProvider) Stream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk)
	close(ch)
	return ch, nil
}

func (m *fetcherMockProvider) Models() []ModelInfo { return m.models }
func (m *fetcherMockProvider) Name() string        { return m.name }

func TestDynamicProvider_DelegatesToInner(t *testing.T) {
	inner := &fetcherMockProvider{
		name:   "test-provider",
		models: []ModelInfo{{ID: "fallback-model"}},
	}
	fetcher := NewModelFetcher(1 * time.Hour)
	dp := NewDynamicProvider(inner, fetcher, "key", "http://invalid-url")

	// Name delegates
	if dp.Name() != "test-provider" {
		t.Errorf("expected test-provider, got %s", dp.Name())
	}

	// Chat delegates
	resp, err := dp.Chat(context.Background(), ChatRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "mock response" {
		t.Errorf("expected mock response, got %s", resp.Content)
	}

	// Stream delegates
	ch, err := dp.Stream(context.Background(), ChatRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Channel should be closed (empty)
	for range ch {
		// drain
	}
}

func TestDynamicProvider_Models_UsesFetcher(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": []map[string]any{
				{"id": "fetched-model", "display_name": "Fetched Model", "type": "model"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	inner := &fetcherMockProvider{
		name:   "anthropic",
		models: []ModelInfo{{ID: "fallback-model"}},
	}
	fetcher := NewModelFetcher(1 * time.Hour)
	dp := NewDynamicProvider(inner, fetcher, "key", srv.URL)

	models := dp.Models()
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}
	if models[0].ID != "fetched-model" {
		t.Errorf("expected fetched-model, got %s", models[0].ID)
	}
}

func TestDynamicProvider_Models_FallsBackOnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	inner := &fetcherMockProvider{
		name:   "anthropic",
		models: []ModelInfo{{ID: "fallback-model"}},
	}
	fetcher := NewModelFetcher(1 * time.Hour)
	dp := NewDynamicProvider(inner, fetcher, "key", srv.URL)

	models := dp.Models()
	if len(models) != 1 {
		t.Fatalf("expected 1 fallback model, got %d", len(models))
	}
	if models[0].ID != "fallback-model" {
		t.Errorf("expected fallback-model, got %s", models[0].ID)
	}
}

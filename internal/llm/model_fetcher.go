package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// ModelMetadata holds known pricing and limits for a model.
type ModelMetadata struct {
	MaxTokens  int
	InputCost  float64
	OutputCost float64
}

// KnownModelMetadata maps model IDs to their known metadata.
// Used to enrich API responses that lack pricing/limit info.
var KnownModelMetadata = map[string]ModelMetadata{
	// Anthropic (March 2026)
	"claude-opus-4-6":   {MaxTokens: 32000, InputCost: 15.0, OutputCost: 75.0},
	"claude-sonnet-4-6": {MaxTokens: 64000, InputCost: 3.0, OutputCost: 15.0},
	"claude-opus-4-5":   {MaxTokens: 32000, InputCost: 15.0, OutputCost: 75.0},
	"claude-sonnet-4-5": {MaxTokens: 64000, InputCost: 3.0, OutputCost: 15.0},
	"claude-haiku-4-5":  {MaxTokens: 64000, InputCost: 0.8, OutputCost: 4.0},

	// OpenAI (March 2026)
	"gpt-5.4":      {MaxTokens: 128000, InputCost: 5.0, OutputCost: 15.0},
	"gpt-4.5":      {MaxTokens: 128000, InputCost: 75.0, OutputCost: 150.0},
	"gpt-4.1":      {MaxTokens: 1000000, InputCost: 2.0, OutputCost: 8.0},
	"gpt-4.1-mini": {MaxTokens: 1000000, InputCost: 0.4, OutputCost: 1.6},
	"gpt-4.1-nano": {MaxTokens: 1000000, InputCost: 0.1, OutputCost: 0.4},
	"o4-mini":      {MaxTokens: 200000, InputCost: 1.1, OutputCost: 4.4},
	"o3":           {MaxTokens: 200000, InputCost: 2.0, OutputCost: 8.0},
	"o3-pro":       {MaxTokens: 200000, InputCost: 20.0, OutputCost: 80.0},
	"o3-mini":      {MaxTokens: 200000, InputCost: 1.1, OutputCost: 4.4},

	// xAI/Grok (March 2026)
	"grok-4-1-fast-latest": {MaxTokens: 131072, InputCost: 3.0, OutputCost: 15.0},
	"grok-4":               {MaxTokens: 131072, InputCost: 3.0, OutputCost: 15.0},
	"grok-4-heavy":         {MaxTokens: 131072, InputCost: 5.0, OutputCost: 25.0},
	"grok-code-fast-1":     {MaxTokens: 131072, InputCost: 3.0, OutputCost: 15.0},
	"grok-3":               {MaxTokens: 131072, InputCost: 3.0, OutputCost: 15.0},
	"grok-3-mini":          {MaxTokens: 131072, InputCost: 0.3, OutputCost: 0.5},

	// Z.ai/GLM (March 2026)
	"glm-5":         {MaxTokens: 200000, InputCost: 1.0, OutputCost: 3.2},
	"glm-5-turbo":   {MaxTokens: 200000, InputCost: 1.2, OutputCost: 4.0},
	"glm-4.7":       {MaxTokens: 128000, InputCost: 0.7, OutputCost: 2.2},
	"glm-4.7-flash": {MaxTokens: 128000, InputCost: 0.0, OutputCost: 0.0},
	"glm-4.5":       {MaxTokens: 128000, InputCost: 0.7, OutputCost: 2.2},

	// Kimi/Moonshot (March 2026)
	"kimi-k2.5":      {MaxTokens: 256000, InputCost: 1.0, OutputCost: 4.0},
	"kimi-k2.5-mini": {MaxTokens: 128000, InputCost: 0.3, OutputCost: 1.0},

	// DeepSeek (via OpenRouter/Groq)
	"deepseek-chat":     {MaxTokens: 65536, InputCost: 0.14, OutputCost: 0.28},
	"deepseek-reasoner": {MaxTokens: 65536, InputCost: 0.55, OutputCost: 2.19},

	// Groq-hosted models (March 2026)
	"llama-4-scout-17b-16e-instruct": {MaxTokens: 131072, InputCost: 0.11, OutputCost: 0.34},
	"llama-3.3-70b-versatile":        {MaxTokens: 131072, InputCost: 0.59, OutputCost: 0.79},
	"llama-3.1-8b-instant":           {MaxTokens: 131072, InputCost: 0.05, OutputCost: 0.08},
	"deepseek-r1-distill-llama-70b":  {MaxTokens: 131072, InputCost: 0.75, OutputCost: 0.99},

	// Google Gemini (via OpenRouter)
	"gemini-2.5-pro":   {MaxTokens: 1000000, InputCost: 1.25, OutputCost: 10.0},
	"gemini-2.5-flash": {MaxTokens: 1000000, InputCost: 0.15, OutputCost: 0.60},

	// Meta Llama (via OpenRouter)
	"meta-llama/llama-4-maverick": {MaxTokens: 1000000, InputCost: 0.2, OutputCost: 0.6},
	"meta-llama/llama-4-scout":    {MaxTokens: 10000000, InputCost: 0.11, OutputCost: 0.34},
}

// EnrichModelInfo merges API-fetched model info with known metadata.
// Only overwrites zero-value fields from the known lookup.
func EnrichModelInfo(model ModelInfo) ModelInfo {
	meta, ok := KnownModelMetadata[model.ID]
	if !ok {
		return model
	}
	enriched := ModelInfo{
		ID:         model.ID,
		Name:       model.Name,
		MaxTokens:  model.MaxTokens,
		InputCost:  model.InputCost,
		OutputCost: model.OutputCost,
	}
	if enriched.MaxTokens == 0 {
		enriched.MaxTokens = meta.MaxTokens
	}
	if enriched.InputCost == 0 {
		enriched.InputCost = meta.InputCost
	}
	if enriched.OutputCost == 0 {
		enriched.OutputCost = meta.OutputCost
	}
	return enriched
}

// --- Provider-specific API response types ---

type anthropicModelsResponse struct {
	Data []struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
		Type        string `json:"type"`
	} `json:"data"`
}

type openaiModelsResponse struct {
	Data []struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		OwnedBy string `json:"owned_by"`
	} `json:"data"`
}

type openRouterModelsResponse struct {
	Data []struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		ContextLength int    `json:"context_length"`
		Pricing       struct {
			Prompt     string `json:"prompt"`
			Completion string `json:"completion"`
		} `json:"pricing"`
	} `json:"data"`
}

// --- Fetch functions ---

// FetchAnthropicModels fetches available models from the Anthropic API.
// baseURL should be the API base (e.g. "https://api.anthropic.com").
func FetchAnthropicModels(ctx context.Context, apiKey string, baseURL string, httpClient *http.Client) ([]ModelInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/models", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var parsed anthropicModelsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	models := make([]ModelInfo, 0, len(parsed.Data))
	for _, m := range parsed.Data {
		info := EnrichModelInfo(ModelInfo{
			ID:   m.ID,
			Name: m.DisplayName,
		})
		models = append(models, info)
	}
	return models, nil
}

// FetchOpenAIModels fetches available models from an OpenAI-compatible API.
// baseURL should include /v1 prefix (e.g. "https://api.openai.com").
func FetchOpenAIModels(ctx context.Context, apiKey string, baseURL string, httpClient *http.Client) ([]ModelInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/models", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var parsed openaiModelsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	models := make([]ModelInfo, 0, len(parsed.Data))
	for _, m := range parsed.Data {
		info := EnrichModelInfo(ModelInfo{
			ID:   m.ID,
			Name: m.ID,
		})
		models = append(models, info)
	}
	return models, nil
}

// FetchOpenRouterModels fetches available models from the OpenRouter API.
// baseURL should be the API base (e.g. "https://openrouter.ai/api").
func FetchOpenRouterModels(ctx context.Context, baseURL string, httpClient *http.Client) ([]ModelInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/models", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var parsed openRouterModelsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	models := make([]ModelInfo, 0, len(parsed.Data))
	for _, m := range parsed.Data {
		inputCost := parsePerTokenToPerMillion(m.Pricing.Prompt)
		outputCost := parsePerTokenToPerMillion(m.Pricing.Completion)

		info := ModelInfo{
			ID:         m.ID,
			Name:       m.Name,
			MaxTokens:  m.ContextLength,
			InputCost:  inputCost,
			OutputCost: outputCost,
		}
		models = append(models, info)
	}
	return models, nil
}

// FetchGroqModels fetches available models from the Groq API (OpenAI-compatible).
func FetchGroqModels(ctx context.Context, apiKey string, baseURL string, httpClient *http.Client) ([]ModelInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/openai/v1/models", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var parsed openaiModelsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	models := make([]ModelInfo, 0, len(parsed.Data))
	for _, m := range parsed.Data {
		info := EnrichModelInfo(ModelInfo{
			ID:   m.ID,
			Name: m.ID,
		})
		models = append(models, info)
	}
	return models, nil
}

// parsePerTokenToPerMillion converts a per-token price string to per-1M-tokens float.
func parsePerTokenToPerMillion(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v * 1_000_000
}

// --- Hardcoded fallbacks ---

// DefaultAnthropicModels returns the hardcoded Anthropic model list.
func DefaultAnthropicModels() []ModelInfo {
	return []ModelInfo{
		{ID: "claude-sonnet-4-6", Name: "Claude Sonnet 4.6", MaxTokens: 64000, InputCost: 3.0, OutputCost: 15.0},
		{ID: "claude-opus-4-6", Name: "Claude Opus 4.6", MaxTokens: 32000, InputCost: 15.0, OutputCost: 75.0},
		{ID: "claude-haiku-4-5", Name: "Claude Haiku 4.5", MaxTokens: 64000, InputCost: 0.8, OutputCost: 4.0},
	}
}

// DefaultOpenAIModels returns the hardcoded OpenAI model list.
func DefaultOpenAIModels() []ModelInfo {
	return []ModelInfo{
		{ID: "gpt-4.1", Name: "GPT-4.1", MaxTokens: 32768, InputCost: 2.0, OutputCost: 8.0},
		{ID: "gpt-4.1-mini", Name: "GPT-4.1 Mini", MaxTokens: 32768, InputCost: 0.4, OutputCost: 1.6},
		{ID: "o4-mini", Name: "o4 Mini", MaxTokens: 100000, InputCost: 1.1, OutputCost: 4.4},
	}
}

// DefaultOpenRouterModels returns the hardcoded OpenRouter model list.
func DefaultOpenRouterModels() []ModelInfo {
	return []ModelInfo{
		{ID: "anthropic/claude-sonnet-4-6", Name: "Claude Sonnet 4.6", MaxTokens: 64000, InputCost: 3.0, OutputCost: 15.0},
		{ID: "anthropic/claude-haiku-4-5", Name: "Claude Haiku 4.5", MaxTokens: 64000, InputCost: 0.8, OutputCost: 4.0},
		{ID: "openai/gpt-4.1", Name: "GPT-4.1", MaxTokens: 32768, InputCost: 2.0, OutputCost: 8.0},
		{ID: "openai/gpt-4.1-mini", Name: "GPT-4.1 Mini", MaxTokens: 32768, InputCost: 0.4, OutputCost: 1.6},
		{ID: "deepseek/deepseek-chat", Name: "DeepSeek Chat", MaxTokens: 65536, InputCost: 0.14, OutputCost: 0.28},
		{ID: "google/gemini-2.5-pro", Name: "Gemini 2.5 Pro", MaxTokens: 65536, InputCost: 1.25, OutputCost: 10.0},
		{ID: "meta-llama/llama-4-maverick", Name: "Llama 4 Maverick", MaxTokens: 65536, InputCost: 0.2, OutputCost: 0.6},
	}
}

// --- ModelFetcher with caching ---

// ModelFetcher fetches and caches model lists from provider APIs.
type ModelFetcher struct {
	cache     map[string][]ModelInfo
	cacheTime map[string]time.Time
	cacheTTL  time.Duration
	mu        sync.RWMutex
}

// NewModelFetcher creates a new ModelFetcher with the given cache TTL.
func NewModelFetcher(cacheTTL time.Duration) *ModelFetcher {
	return &ModelFetcher{
		cache:     make(map[string][]ModelInfo),
		cacheTime: make(map[string]time.Time),
		cacheTTL:  cacheTTL,
	}
}

// GetCached returns cached models for a provider, or nil if expired/missing.
func (f *ModelFetcher) GetCached(provider string) []ModelInfo {
	f.mu.RLock()
	defer f.mu.RUnlock()

	t, ok := f.cacheTime[provider]
	if !ok {
		return nil
	}
	if time.Since(t) > f.cacheTTL {
		return nil
	}
	return f.cache[provider]
}

// ClearCache clears the cache for a provider (or all if provider is "").
func (f *ModelFetcher) ClearCache(provider string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if provider == "" {
		f.cache = make(map[string][]ModelInfo)
		f.cacheTime = make(map[string]time.Time)
		return
	}
	delete(f.cache, provider)
	delete(f.cacheTime, provider)
}

// FetchModels queries a provider's models API and returns the list.
// Uses cache if available and not expired. The provider name determines
// which fetch function to call. Supported: "anthropic", "openai", "openrouter", "groq".
func (f *ModelFetcher) FetchModels(ctx context.Context, provider string, apiURL string, apiKey string, headers map[string]string) ([]ModelInfo, error) {
	if cached := f.GetCached(provider); cached != nil {
		return cached, nil
	}

	client := &http.Client{Timeout: 10 * time.Second}

	var models []ModelInfo
	var err error

	switch provider {
	case "anthropic":
		models, err = FetchAnthropicModels(ctx, apiKey, apiURL, client)
	case "openai":
		models, err = FetchOpenAIModels(ctx, apiKey, apiURL, client)
	case "openrouter":
		models, err = FetchOpenRouterModels(ctx, apiURL, client)
	case "groq":
		models, err = FetchGroqModels(ctx, apiKey, apiURL, client)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	if err != nil {
		return nil, err
	}

	f.mu.Lock()
	f.cache[provider] = models
	f.cacheTime[provider] = time.Now()
	f.mu.Unlock()

	return models, nil
}

// --- DynamicProvider wrapper ---

// DynamicProvider wraps a Provider and overrides Models() with API-fetched models.
type DynamicProvider struct {
	inner   Provider
	fetcher *ModelFetcher
	apiKey  string
	baseURL string
}

// NewDynamicProvider creates a DynamicProvider that wraps an existing provider.
func NewDynamicProvider(inner Provider, fetcher *ModelFetcher, apiKey, baseURL string) *DynamicProvider {
	return &DynamicProvider{
		inner:   inner,
		fetcher: fetcher,
		apiKey:  apiKey,
		baseURL: baseURL,
	}
}

// Chat delegates to the inner provider.
func (d *DynamicProvider) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	return d.inner.Chat(ctx, req)
}

// Stream delegates to the inner provider.
func (d *DynamicProvider) Stream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	return d.inner.Stream(ctx, req)
}

// Name delegates to the inner provider.
func (d *DynamicProvider) Name() string {
	return d.inner.Name()
}

// Models fetches models dynamically from the provider API.
// Falls back to the inner provider's hardcoded list on error.
func (d *DynamicProvider) Models() []ModelInfo {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	models, err := d.fetcher.FetchModels(ctx, d.inner.Name(), d.baseURL, d.apiKey, nil)
	if err != nil {
		return d.inner.Models()
	}
	return models
}

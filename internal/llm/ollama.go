package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// OllamaProvider implements Provider for the Ollama local inference API.
type OllamaProvider struct {
	baseURL string
	client  *http.Client
}

// NewOllamaProvider creates an Ollama provider.
// If baseURL is empty, defaults to http://localhost:11434.
func NewOllamaProvider(baseURL string) *OllamaProvider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return &OllamaProvider{baseURL: baseURL, client: &http.Client{}}
}

func (p *OllamaProvider) Name() string { return "ollama" }

// ollamaChatRequest is the request body for POST /api/chat.
type ollamaChatRequest struct {
	Model    string            `json:"model"`
	Messages []ollamaMessage   `json:"messages"`
	Stream   bool              `json:"stream"`
	Tools    []ollamaTool      `json:"tools,omitempty"`
	Options  map[string]any    `json:"options,omitempty"`
}

type ollamaMessage struct {
	Role      string            `json:"role"`
	Content   string            `json:"content"`
	ToolCalls []ollamaToolCall  `json:"tool_calls,omitempty"`
}

type ollamaTool struct {
	Type     string         `json:"type"`
	Function ollamaFunction `json:"function"`
}

type ollamaFunction struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

type ollamaToolCall struct {
	Function struct {
		Name      string `json:"name"`
		Arguments any    `json:"arguments"`
	} `json:"function"`
}

// ollamaChatResponse is the response from POST /api/chat.
type ollamaChatResponse struct {
	Model           string        `json:"model"`
	Message         ollamaMessage `json:"message"`
	Done            bool          `json:"done"`
	PromptEvalCount int           `json:"prompt_eval_count"`
	EvalCount       int           `json:"eval_count"`
}

// ollamaTagsResponse is the response from GET /api/tags.
type ollamaTagsResponse struct {
	Models []ollamaModelEntry `json:"models"`
}

type ollamaModelEntry struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

func (p *OllamaProvider) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	body := p.buildRequest(req, false)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/chat", bytes.NewReader(jsonBody))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("send request: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return ChatResponse{}, fmt.Errorf("API error (status %d): %s", httpResp.StatusCode, string(respBody))
	}

	var oResp ollamaChatResponse
	if err := json.Unmarshal(respBody, &oResp); err != nil {
		return ChatResponse{}, fmt.Errorf("parse response: %w", err)
	}

	return p.parseResponse(oResp), nil
}

func (p *OllamaProvider) Stream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	body := p.buildRequest(req, true)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/chat", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		defer httpResp.Body.Close()
		respBody, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", httpResp.StatusCode, string(respBody))
	}

	ch := make(chan StreamChunk)
	go func() {
		defer close(ch)
		defer httpResp.Body.Close()

		scanner := bufio.NewScanner(httpResp.Body)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			var oResp ollamaChatResponse
			if err := json.Unmarshal(line, &oResp); err != nil {
				continue
			}

			chunk := StreamChunk{
				Content: oResp.Message.Content,
				Done:    oResp.Done,
			}

			if oResp.Done {
				chunk.FinishReason = "stop"
				chunk.Usage = &Usage{
					PromptTokens:     oResp.PromptEvalCount,
					CompletionTokens: oResp.EvalCount,
					TotalTokens:      oResp.PromptEvalCount + oResp.EvalCount,
				}
			}

			// Map tool calls from streaming chunk
			for _, tc := range oResp.Message.ToolCalls {
				argsJSON, _ := json.Marshal(tc.Function.Arguments)
				chunk.ToolCalls = append(chunk.ToolCalls, ToolCall{
					Name:      tc.Function.Name,
					Arguments: string(argsJSON),
				})
			}

			select {
			case ch <- chunk:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch, nil
}

func (p *OllamaProvider) Models() []ModelInfo {
	resp, err := http.Get(p.baseURL + "/api/tags")
	if err != nil {
		return []ModelInfo{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []ModelInfo{}
	}

	var tagsResp ollamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
		return []ModelInfo{}
	}

	models := make([]ModelInfo, len(tagsResp.Models))
	for i, m := range tagsResp.Models {
		models[i] = ModelInfo{
			ID:   m.Name,
			Name: m.Name,
		}
	}
	return models
}

func (p *OllamaProvider) buildRequest(req ChatRequest, stream bool) ollamaChatRequest {
	msgs := make([]ollamaMessage, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = ollamaMessage{
			Role:    string(m.Role),
			Content: m.Content,
		}
	}

	var tools []ollamaTool
	for _, td := range req.Tools {
		tools = append(tools, ollamaTool{
			Type: "function",
			Function: ollamaFunction{
				Name:        td.Name,
				Description: td.Description,
				Parameters:  td.Parameters,
			},
		})
	}

	var options map[string]any
	if req.Temperature != nil {
		options = map[string]any{"temperature": *req.Temperature}
	}
	if req.MaxTokens > 0 {
		if options == nil {
			options = map[string]any{}
		}
		options["num_predict"] = req.MaxTokens
	}

	return ollamaChatRequest{
		Model:    req.Model,
		Messages: msgs,
		Stream:   stream,
		Tools:    tools,
		Options:  options,
	}
}

func (p *OllamaProvider) parseResponse(resp ollamaChatResponse) ChatResponse {
	result := ChatResponse{
		Model: resp.Model,
		Usage: Usage{
			PromptTokens:     resp.PromptEvalCount,
			CompletionTokens: resp.EvalCount,
			TotalTokens:      resp.PromptEvalCount + resp.EvalCount,
		},
	}

	if resp.Done {
		result.FinishReason = "stop"
	}

	result.Content = resp.Message.Content

	for _, tc := range resp.Message.ToolCalls {
		argsJSON, _ := json.Marshal(tc.Function.Arguments)
		result.ToolCalls = append(result.ToolCalls, ToolCall{
			Name:      tc.Function.Name,
			Arguments: string(argsJSON),
		})
	}

	return result
}

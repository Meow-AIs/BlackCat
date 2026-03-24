package builtin

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/meowai/blackcat/internal/llm"
	"github.com/meowai/blackcat/internal/tools"
)

// VisionTool analyzes images via a vision-capable LLM.
type VisionTool struct {
	provider llm.Provider
}

// NewVisionTool creates a new VisionTool. If ANTHROPIC_API_KEY or
// OPENAI_API_KEY are set in the environment a provider will be created
// automatically; otherwise Execute falls back to returning image metadata
// without LLM analysis.
func NewVisionTool() *VisionTool {
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		return &VisionTool{provider: llm.NewAnthropicProvider(key, "")}
	}
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		return &VisionTool{provider: llm.NewOpenAIProvider(key, "https://api.openai.com/v1", "openai")}
	}
	return &VisionTool{}
}

// NewVisionToolWithProvider creates a VisionTool that uses the supplied
// llm.Provider for image analysis. Useful for testing or when the caller
// already holds a configured provider.
func NewVisionToolWithProvider(p llm.Provider) *VisionTool {
	return &VisionTool{provider: p}
}

// Info returns the tool definition for analyze_image.
func (t *VisionTool) Info() tools.Definition {
	return tools.Definition{
		Name:        "analyze_image",
		Description: "Analyze an image file or URL. Describe what you see, extract text (OCR), identify diagrams, or answer questions about the image.",
		Category:    "multimodal",
		Parameters: []tools.Parameter{
			{Name: "path", Type: "string", Description: "Path to an image file or a URL", Required: true},
			{Name: "question", Type: "string", Description: "Optional question about the image"},
		},
	}
}

// Execute reads an image and returns metadata plus base64 data so the agent
// core can forward it to a vision-capable LLM. When a provider is configured
// the image is sent directly and the LLM response is returned.
func (t *VisionTool) Execute(ctx context.Context, args map[string]any) (tools.Result, error) {
	path, err := requireStringArg(args, "path")
	if err != nil {
		return tools.Result{}, err
	}

	question := ""
	if q, ok := args["question"].(string); ok {
		question = q
	}

	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return t.handleURL(ctx, path, question)
	}
	return t.handleFile(ctx, path, question)
}

// handleURL processes an image referenced by URL.
func (t *VisionTool) handleURL(ctx context.Context, url, question string) (tools.Result, error) {
	if t.provider != nil {
		return t.callProvider(ctx, url, "", "", question, true)
	}

	output := fmt.Sprintf("Image loaded: %s (URL reference)", url)
	if question != "" {
		output += fmt.Sprintf("\nQuestion: %s", question)
	}
	output += "\nNote: Send to a vision-capable LLM for analysis."

	return tools.Result{
		Output:   output,
		ExitCode: 0,
		Metadata: map[string]any{
			"image_url": url,
		},
	}, nil
}

// handleFile processes a local image file.
func (t *VisionTool) handleFile(ctx context.Context, path, question string) (tools.Result, error) {
	info, err := os.Stat(path)
	if err != nil {
		return tools.Result{
			Error:    fmt.Sprintf("cannot access file: %s", err),
			ExitCode: 1,
		}, nil
	}

	if !llm.IsImageFile(path) {
		return tools.Result{
			Error:    fmt.Sprintf("unsupported image format: %s", path),
			ExitCode: 1,
		}, nil
	}

	mediaType := llm.DetectMediaType(path)

	// Read and base64-encode the file.
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		return tools.Result{
			Error:    fmt.Sprintf("cannot read file: %s", readErr),
			ExitCode: 1,
		}, nil
	}
	encoded := base64.StdEncoding.EncodeToString(data)

	if t.provider != nil {
		return t.callProvider(ctx, "", encoded, mediaType, question, false)
	}

	output := fmt.Sprintf("Image loaded: %s, %d bytes, %s", path, info.Size(), mediaType)
	if question != "" {
		output += fmt.Sprintf("\nQuestion: %s", question)
	}
	output += "\nNote: Send to a vision-capable LLM for analysis."

	return tools.Result{
		Output:   output,
		ExitCode: 0,
		Metadata: map[string]any{
			"image_data": encoded,
			"media_type": mediaType,
		},
	}, nil
}

// callProvider sends the image to the configured LLM provider and returns its
// analysis. For local files encoded is the base64 data and mediaType identifies
// the MIME type; for URLs imageURL carries the reference.
func (t *VisionTool) callProvider(ctx context.Context, imageURL, encoded, mediaType, question string, isURL bool) (tools.Result, error) {
	if question == "" {
		question = "Describe this image in detail."
	}

	// Build message content: include the image data followed by the question.
	var content string
	if isURL {
		content = fmt.Sprintf("[image url=%s]\n%s", imageURL, question)
	} else {
		content = fmt.Sprintf("[image media_type=%s data=%s]\n%s", mediaType, encoded, question)
	}

	req := llm.ChatRequest{
		Model: firstModel(t.provider),
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: content},
		},
		MaxTokens: 1024,
	}

	resp, err := t.provider.Chat(ctx, req)
	if err != nil {
		return tools.Result{
			Error:    fmt.Sprintf("vision analysis failed: %s", err),
			ExitCode: 1,
		}, nil
	}

	return tools.Result{Output: resp.Content, ExitCode: 0}, nil
}

// firstModel returns the ID of the first model advertised by the provider, or
// an empty string if the provider reports none.
func firstModel(p llm.Provider) string {
	models := p.Models()
	if len(models) == 0 {
		return ""
	}
	return models[0].ID
}

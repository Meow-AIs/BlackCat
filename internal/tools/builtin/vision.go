package builtin

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/meowai/blackcat/internal/llm"
	"github.com/meowai/blackcat/internal/tools"
)

// VisionTool analyzes images via a vision-capable LLM.
type VisionTool struct{}

// NewVisionTool creates a new VisionTool.
func NewVisionTool() *VisionTool {
	return &VisionTool{}
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

// Execute reads an image and returns a placeholder description.
// A real implementation would send the image to a vision-capable LLM.
func (t *VisionTool) Execute(_ context.Context, args map[string]any) (tools.Result, error) {
	path, err := requireStringArg(args, "path")
	if err != nil {
		return tools.Result{}, err
	}

	question := ""
	if q, ok := args["question"].(string); ok {
		question = q
	}

	// Handle URL references
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return visionURLResult(path, question), nil
	}

	// Handle local file
	return visionFileResult(path, question)
}

func visionURLResult(url, question string) tools.Result {
	output := fmt.Sprintf("Image loaded: %s (URL reference)", url)
	if question != "" {
		output += fmt.Sprintf("\nQuestion: %s", question)
	}
	output += "\nNote: Send to a vision-capable LLM for analysis."
	return tools.Result{Output: output, ExitCode: 0}
}

func visionFileResult(path, question string) (tools.Result, error) {
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
	output := fmt.Sprintf("Image loaded: %s, %d bytes, %s", path, info.Size(), mediaType)
	if question != "" {
		output += fmt.Sprintf("\nQuestion: %s", question)
	}
	output += "\nNote: Send to a vision-capable LLM for analysis."

	return tools.Result{Output: output, ExitCode: 0}, nil
}

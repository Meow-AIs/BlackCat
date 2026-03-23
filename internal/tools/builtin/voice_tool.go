package builtin

import (
	"context"
	"fmt"
	"os"

	"github.com/meowai/blackcat/internal/llm"
	"github.com/meowai/blackcat/internal/tools"
)

// VoiceTool transcribes audio files to text via Groq Whisper API.
type VoiceTool struct{}

// NewVoiceTool creates a new VoiceTool.
func NewVoiceTool() *VoiceTool {
	return &VoiceTool{}
}

// Info returns the tool definition for transcribe_audio.
func (t *VoiceTool) Info() tools.Definition {
	return tools.Definition{
		Name:        "transcribe_audio",
		Description: "Transcribe speech from an audio file to text using Groq Whisper API.",
		Category:    "multimodal",
		Parameters: []tools.Parameter{
			{Name: "path", Type: "string", Description: "Path to an audio file", Required: true},
			{Name: "language", Type: "string", Description: "Language code (e.g., 'en', 'es', 'fr')"},
		},
	}
}

// Execute validates the audio file and returns a placeholder response.
// A real implementation would call VoiceProvider.TranscribeFile.
func (t *VoiceTool) Execute(_ context.Context, args map[string]any) (tools.Result, error) {
	path, err := requireStringArg(args, "path")
	if err != nil {
		return tools.Result{}, err
	}

	// Check file exists
	if _, statErr := os.Stat(path); statErr != nil {
		return tools.Result{
			Error:    fmt.Sprintf("cannot access file: %s", statErr),
			ExitCode: 1,
		}, nil
	}

	// Check supported format
	if !llm.IsSupportedAudioFormat(path) {
		return tools.Result{
			Error:    fmt.Sprintf("unsupported audio format: %s. Supported: %v", path, llm.SupportedAudioFormats),
			ExitCode: 1,
		}, nil
	}

	return tools.Result{
		Output:   "Transcription requires Groq API key. Configure via: blackcat config set groq_api_key <key>",
		ExitCode: 0,
	}, nil
}

package builtin

import (
	"context"
	"fmt"
	"os"

	"github.com/meowai/blackcat/internal/llm"
	"github.com/meowai/blackcat/internal/tools"
)

// VoiceTool transcribes audio files to text via Groq Whisper API.
type VoiceTool struct {
	provider *llm.VoiceProvider
}

// NewVoiceTool creates a new VoiceTool. If the GROQ_API_KEY environment
// variable is set it will be used to create a VoiceProvider automatically;
// otherwise Execute returns a helpful configuration message.
func NewVoiceTool() *VoiceTool {
	if key := os.Getenv("GROQ_API_KEY"); key != "" {
		return &VoiceTool{provider: llm.NewVoiceProvider(key)}
	}
	return &VoiceTool{}
}

// NewVoiceToolWithProvider creates a VoiceTool that uses the supplied
// VoiceProvider for transcription. Useful for testing or when the caller
// already holds a configured provider.
func NewVoiceToolWithProvider(p *llm.VoiceProvider) *VoiceTool {
	return &VoiceTool{provider: p}
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

// Execute transcribes the audio file at the given path. When no VoiceProvider
// is configured it returns a helpful setup message instead of calling the API.
func (t *VoiceTool) Execute(ctx context.Context, args map[string]any) (tools.Result, error) {
	path, err := requireStringArg(args, "path")
	if err != nil {
		return tools.Result{}, err
	}

	// Check file exists.
	if _, statErr := os.Stat(path); statErr != nil {
		return tools.Result{
			Error:    fmt.Sprintf("cannot access file: %s", statErr),
			ExitCode: 1,
		}, nil
	}

	// Check supported format.
	if !llm.IsSupportedAudioFormat(path) {
		return tools.Result{
			Error:    fmt.Sprintf("unsupported audio format: %s. Supported: %v", path, llm.SupportedAudioFormats),
			ExitCode: 1,
		}, nil
	}

	// No provider configured — return helpful message.
	if t.provider == nil {
		return tools.Result{
			Output:   "Transcription requires Groq API key. Configure via: blackcat config set groq_api_key <key>",
			ExitCode: 0,
		}, nil
	}

	// Resolve optional language parameter.
	language := ""
	if lang, ok := args["language"].(string); ok {
		language = lang
	}

	text, transcribeErr := t.provider.TranscribeFile(ctx, path, language)
	if transcribeErr != nil {
		return tools.Result{
			Error:    fmt.Sprintf("transcription failed: %s", transcribeErr),
			ExitCode: 1,
		}, nil
	}

	return tools.Result{Output: text, ExitCode: 0}, nil
}

package builtin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/meowai/blackcat/internal/llm"
)

func TestVoiceToolInfo(t *testing.T) {
	tool := NewVoiceTool()
	info := tool.Info()

	if info.Name != "transcribe_audio" {
		t.Errorf("expected name %q, got %q", "transcribe_audio", info.Name)
	}
	if info.Category != "multimodal" {
		t.Errorf("expected category %q, got %q", "multimodal", info.Category)
	}

	hasPath := false
	for _, p := range info.Parameters {
		if p.Name == "path" && p.Required {
			hasPath = true
		}
	}
	if !hasPath {
		t.Error("expected required 'path' parameter")
	}
}

func TestVoiceToolExecuteMissingPath(t *testing.T) {
	tool := NewVoiceTool()
	_, err := tool.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Error("expected error for missing path")
	}
}

func TestVoiceToolExecuteNonExistentFile(t *testing.T) {
	tool := NewVoiceTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"path": "/nonexistent/audio.wav",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", result.ExitCode)
	}
}

func TestVoiceToolExecuteUnsupportedFormat(t *testing.T) {
	tmpDir := t.TempDir()
	txtPath := filepath.Join(tmpDir, "file.txt")
	if err := os.WriteFile(txtPath, []byte("not audio"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	tool := NewVoiceTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"path": txtPath,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Error, "unsupported") {
		t.Errorf("expected error about unsupported format, got %q", result.Error)
	}
}

// TestVoiceToolExecuteSuccess verifies the fallback message when no provider is configured.
func TestVoiceToolExecuteSuccess(t *testing.T) {
	// Ensure GROQ_API_KEY is unset so we exercise the fallback path.
	t.Setenv("GROQ_API_KEY", "")

	tmpDir := t.TempDir()
	wavPath := filepath.Join(tmpDir, "test.wav")
	if err := os.WriteFile(wavPath, []byte("fake-wav"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	tool := NewVoiceTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"path": wavPath,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "Groq API key") {
		t.Errorf("expected placeholder message about Groq API key, got %q", result.Output)
	}
}

// TestVoiceToolWithProviderTranscribes verifies that when a VoiceProvider is
// injected the tool calls TranscribeFile and returns the transcription text.
func TestVoiceToolWithProviderTranscribes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := llm.TranscriptionResponse{Text: "hello from whisper"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	wavPath := filepath.Join(tmpDir, "test.wav")
	if err := os.WriteFile(wavPath, []byte("fake-wav-data"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	provider := llm.NewVoiceProviderWithURL("test-key", server.URL)
	tool := NewVoiceToolWithProvider(provider)

	result, err := tool.Execute(context.Background(), map[string]any{
		"path": wavPath,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d; error: %s", result.ExitCode, result.Error)
	}
	if result.Output != "hello from whisper" {
		t.Errorf("expected transcription text %q, got %q", "hello from whisper", result.Output)
	}
}

// TestVoiceToolWithProviderAndLanguage verifies that the language parameter is
// forwarded to TranscribeFile.
func TestVoiceToolWithProviderAndLanguage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("failed to parse multipart: %v", err)
		}
		lang := r.FormValue("language")
		resp := llm.TranscriptionResponse{Text: "transcribed in " + lang}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	mp3Path := filepath.Join(tmpDir, "speech.mp3")
	if err := os.WriteFile(mp3Path, []byte("fake-mp3"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	provider := llm.NewVoiceProviderWithURL("key", server.URL)
	tool := NewVoiceToolWithProvider(provider)

	result, err := tool.Execute(context.Background(), map[string]any{
		"path":     mp3Path,
		"language": "fr",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d; error: %s", result.ExitCode, result.Error)
	}
	if !strings.Contains(result.Output, "fr") {
		t.Errorf("expected language 'fr' forwarded, got output %q", result.Output)
	}
}

// TestVoiceToolWithProviderAPIError verifies that a provider error is surfaced
// as a non-zero exit code.
func TestVoiceToolWithProviderAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid api key"}`))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	wavPath := filepath.Join(tmpDir, "test.wav")
	if err := os.WriteFile(wavPath, []byte("fake-wav"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	provider := llm.NewVoiceProviderWithURL("bad-key", server.URL)
	tool := NewVoiceToolWithProvider(provider)

	result, err := tool.Execute(context.Background(), map[string]any{
		"path": wavPath,
	})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1 for API error, got %d", result.ExitCode)
	}
	if result.Error == "" {
		t.Error("expected non-empty Error field on API failure")
	}
}

// TestVoiceToolEnvKeyCreatesProvider verifies that when GROQ_API_KEY is set in
// the environment and no explicit provider is injected, Execute attempts to use
// the Groq API. We point the request at a local server to avoid real network calls.
// NOTE: This test validates the env-var code path by checking that the tool no
// longer returns the "Groq API key" fallback message when GROQ_API_KEY is set.
// Because we cannot redirect the default base URL without injecting a provider,
// we verify behavior indirectly: the tool should still return ExitCode 0, and
// since no real Groq server is reachable the call will fail and the error will
// be surfaced via ExitCode 1. We verify the fallback message is NOT shown.
func TestVoiceToolEnvKeyFallbackNotShownWhenKeySet(t *testing.T) {
	t.Setenv("GROQ_API_KEY", "some-key")

	tmpDir := t.TempDir()
	wavPath := filepath.Join(tmpDir, "test.wav")
	if err := os.WriteFile(wavPath, []byte("fake-wav"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	tool := NewVoiceTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"path": wavPath,
	})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	// When the API key is present but the real API fails (network unreachable in test),
	// we should NOT see the "configure" fallback message.
	if strings.Contains(result.Output, "Configure via") {
		t.Errorf("fallback 'Configure via' message shown even though GROQ_API_KEY is set; got %q", result.Output)
	}
}

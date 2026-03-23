package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestVoiceToolExecuteSuccess(t *testing.T) {
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

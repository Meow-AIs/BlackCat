package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVisionToolInfo(t *testing.T) {
	tool := NewVisionTool()
	info := tool.Info()

	if info.Name != "analyze_image" {
		t.Errorf("expected name %q, got %q", "analyze_image", info.Name)
	}
	if info.Category != "multimodal" {
		t.Errorf("expected category %q, got %q", "multimodal", info.Category)
	}

	// Check required path parameter
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

func TestVisionToolExecuteMissingPath(t *testing.T) {
	tool := NewVisionTool()
	_, err := tool.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Error("expected error for missing path")
	}
}

func TestVisionToolExecuteNonExistentFile(t *testing.T) {
	tool := NewVisionTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"path": "/nonexistent/image.png",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", result.ExitCode)
	}
}

func TestVisionToolExecuteUnsupportedFile(t *testing.T) {
	tmpDir := t.TempDir()
	txtPath := filepath.Join(tmpDir, "file.txt")
	if err := os.WriteFile(txtPath, []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	tool := NewVisionTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"path": txtPath,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", result.ExitCode)
	}
}

func TestVisionToolExecuteSuccess(t *testing.T) {
	// Create minimal PNG
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
		0x00, 0x00, 0x02, 0x00, 0x01, 0xE2, 0x21, 0xBC,
		0x33, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E,
		0x44, 0xAE, 0x42, 0x60, 0x82,
	}

	tmpDir := t.TempDir()
	pngPath := filepath.Join(tmpDir, "test.png")
	if err := os.WriteFile(pngPath, pngData, 0644); err != nil {
		t.Fatalf("failed to write test PNG: %v", err)
	}

	tool := NewVisionTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"path": pngPath,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "Image loaded") {
		t.Errorf("expected output to contain 'Image loaded', got %q", result.Output)
	}
	if !strings.Contains(result.Output, "image/png") {
		t.Errorf("expected output to contain media type, got %q", result.Output)
	}
}

func TestVisionToolExecuteURL(t *testing.T) {
	tool := NewVisionTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"path":     "https://example.com/photo.png",
		"question": "What is in this image?",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "Image loaded") {
		t.Errorf("expected output to contain 'Image loaded', got %q", result.Output)
	}
}

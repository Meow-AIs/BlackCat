package builtin

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/meowai/blackcat/internal/llm"
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

// TestVisionToolFileEncodeBase64 verifies that when a local image file is
// processed, the output includes base64-encoded image data that round-trips
// correctly back to the original bytes.
func TestVisionToolFileEncodeBase64(t *testing.T) {
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xDE,
	}

	tmpDir := t.TempDir()
	pngPath := filepath.Join(tmpDir, "encode_test.png")
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
		t.Errorf("expected exit code 0, got %d; error: %s", result.ExitCode, result.Error)
	}

	// The result metadata should carry the base64-encoded image data.
	encoded, ok := result.Metadata["image_data"].(string)
	if !ok || encoded == "" {
		t.Fatalf("expected result.Metadata[\"image_data\"] to be a non-empty string, got %v", result.Metadata["image_data"])
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("image_data is not valid base64: %v", err)
	}
	if string(decoded) != string(pngData) {
		t.Errorf("decoded image data does not match original bytes")
	}
}

// TestVisionToolFileMetadataMediaType verifies that result.Metadata contains
// the correct media_type for the image.
func TestVisionToolFileMetadataMediaType(t *testing.T) {
	pngData := []byte{0x89, 0x50, 0x4E, 0x47}
	tmpDir := t.TempDir()
	pngPath := filepath.Join(tmpDir, "meta_test.png")
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
		t.Errorf("expected exit code 0, got %d; error: %s", result.ExitCode, result.Error)
	}

	mediaType, ok := result.Metadata["media_type"].(string)
	if !ok || mediaType != "image/png" {
		t.Errorf("expected Metadata[\"media_type\"] == \"image/png\", got %v", result.Metadata["media_type"])
	}
}

// TestVisionToolURLMetadata verifies that URL images populate metadata with the
// image URL and no base64 data.
func TestVisionToolURLMetadata(t *testing.T) {
	imageURL := "https://example.com/cat.jpg"
	tool := NewVisionTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"path": imageURL,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	imageURLMeta, ok := result.Metadata["image_url"].(string)
	if !ok || imageURLMeta != imageURL {
		t.Errorf("expected Metadata[\"image_url\"] == %q, got %v", imageURL, result.Metadata["image_url"])
	}
	// URL images should not include base64 data in metadata
	if _, hasData := result.Metadata["image_data"]; hasData {
		t.Error("URL images should not include image_data in metadata")
	}
}

// TestVisionToolWithProviderLocalFile verifies that when an llm.Provider is
// injected and a local image is given, the tool calls Chat and returns the
// provider's response.
func TestVisionToolWithProviderLocalFile(t *testing.T) {
	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	tmpDir := t.TempDir()
	pngPath := filepath.Join(tmpDir, "provider_test.png")
	if err := os.WriteFile(pngPath, pngData, 0644); err != nil {
		t.Fatalf("failed to write test PNG: %v", err)
	}

	mock := &mockVisionProvider{response: "A small PNG image with a red pixel."}
	tool := NewVisionToolWithProvider(mock)

	result, err := tool.Execute(context.Background(), map[string]any{
		"path":     pngPath,
		"question": "What do you see?",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d; error: %s", result.ExitCode, result.Error)
	}
	if result.Output != "A small PNG image with a red pixel." {
		t.Errorf("expected provider response, got %q", result.Output)
	}
	if !mock.chatCalled {
		t.Error("expected provider Chat to be called")
	}
}

// TestVisionToolWithProviderURL verifies that URL images are also forwarded to
// the provider.
func TestVisionToolWithProviderURL(t *testing.T) {
	mock := &mockVisionProvider{response: "A cat on the internet."}
	tool := NewVisionToolWithProvider(mock)

	result, err := tool.Execute(context.Background(), map[string]any{
		"path":     "https://example.com/cat.png",
		"question": "Describe the image.",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d; error: %s", result.ExitCode, result.Error)
	}
	if result.Output != "A cat on the internet." {
		t.Errorf("expected provider response, got %q", result.Output)
	}
	if !mock.chatCalled {
		t.Error("expected provider Chat to be called")
	}
}

// TestVisionToolWithProviderError verifies that a provider error is surfaced as
// a non-zero exit code.
func TestVisionToolWithProviderError(t *testing.T) {
	pngData := []byte{0x89, 0x50, 0x4E, 0x47}
	tmpDir := t.TempDir()
	pngPath := filepath.Join(tmpDir, "error_test.png")
	if err := os.WriteFile(pngPath, pngData, 0644); err != nil {
		t.Fatalf("failed to write test PNG: %v", err)
	}

	mock := &mockVisionProvider{err: context.DeadlineExceeded}
	tool := NewVisionToolWithProvider(mock)

	result, err := tool.Execute(context.Background(), map[string]any{
		"path": pngPath,
	})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1 for provider error, got %d", result.ExitCode)
	}
	if result.Error == "" {
		t.Error("expected non-empty Error field on provider failure")
	}
}

// TestVisionToolWithProviderMessageContainsBase64 verifies that when the provider
// is called for a local file, the Chat request message includes base64 image data.
func TestVisionToolWithProviderMessageContainsBase64(t *testing.T) {
	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A}
	tmpDir := t.TempDir()
	pngPath := filepath.Join(tmpDir, "b64_check.png")
	if err := os.WriteFile(pngPath, pngData, 0644); err != nil {
		t.Fatalf("failed to write test PNG: %v", err)
	}

	mock := &mockVisionProvider{response: "ok"}
	tool := NewVisionToolWithProvider(mock)

	_, err := tool.Execute(context.Background(), map[string]any{
		"path": pngPath,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.lastRequest.Messages) == 0 {
		t.Fatal("expected at least one message in Chat request")
	}
	// The message content must contain base64-encoded image data.
	expectedBase64 := base64.StdEncoding.EncodeToString(pngData)
	msgContent := mock.lastRequest.Messages[len(mock.lastRequest.Messages)-1].Content
	if !strings.Contains(msgContent, expectedBase64) {
		t.Errorf("expected Chat message to contain base64 image data %q, got %q", expectedBase64, msgContent)
	}
}

// TestVisionToolNewVisionToolAnthropicEnvKey verifies that NewVisionTool picks up
// ANTHROPIC_API_KEY and creates a provider automatically.
func TestVisionToolNewVisionToolAnthropicEnvKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	t.Setenv("OPENAI_API_KEY", "")

	tool := NewVisionTool()
	if tool.provider == nil {
		t.Error("expected provider to be set when ANTHROPIC_API_KEY is present")
	}
}

// TestVisionToolNewVisionToolOpenAIEnvKey verifies that NewVisionTool falls back
// to OPENAI_API_KEY when ANTHROPIC_API_KEY is absent.
func TestVisionToolNewVisionToolOpenAIEnvKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "openai-key")

	tool := NewVisionTool()
	if tool.provider == nil {
		t.Error("expected provider to be set when OPENAI_API_KEY is present")
	}
}

// TestVisionToolNoEnvKeysNoProvider verifies that NewVisionTool returns a tool
// with no provider when neither API key is set.
func TestVisionToolNoEnvKeysNoProvider(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")

	tool := NewVisionTool()
	if tool.provider != nil {
		t.Error("expected nil provider when no API keys are set")
	}
}

// TestVisionToolWithProviderNoModels verifies that firstModel returns "" when
// the provider reports no models, and the tool still calls Chat without crashing.
func TestVisionToolWithProviderNoModels(t *testing.T) {
	pngData := []byte{0x89, 0x50, 0x4E, 0x47}
	tmpDir := t.TempDir()
	pngPath := filepath.Join(tmpDir, "no_models.png")
	if err := os.WriteFile(pngPath, pngData, 0644); err != nil {
		t.Fatalf("failed to write test PNG: %v", err)
	}

	mock := &mockVisionProvider{response: "no model id needed", noModels: true}
	tool := NewVisionToolWithProvider(mock)

	result, err := tool.Execute(context.Background(), map[string]any{
		"path": pngPath,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d; error: %s", result.ExitCode, result.Error)
	}
}

// mockVisionProvider is a test double for llm.Provider.
type mockVisionProvider struct {
	response    string
	err         error
	chatCalled  bool
	lastRequest llm.ChatRequest
	noModels    bool
}

func (m *mockVisionProvider) Chat(_ context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
	m.chatCalled = true
	m.lastRequest = req
	if m.err != nil {
		return llm.ChatResponse{}, m.err
	}
	return llm.ChatResponse{Content: m.response}, nil
}

func (m *mockVisionProvider) Stream(_ context.Context, _ llm.ChatRequest) (<-chan llm.StreamChunk, error) {
	ch := make(chan llm.StreamChunk, 1)
	ch <- llm.StreamChunk{Content: m.response, Done: true}
	close(ch)
	return ch, nil
}

func (m *mockVisionProvider) Models() []llm.ModelInfo {
	if m.noModels {
		return nil
	}
	return []llm.ModelInfo{{ID: "mock-model", Name: "Mock"}}
}
func (m *mockVisionProvider) Name() string { return "mock" }

package llm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestContentBlockConstructors(t *testing.T) {
	t.Run("TextBlock", func(t *testing.T) {
		b := TextBlock("hello world")
		if b.Type != ContentText {
			t.Errorf("expected type %q, got %q", ContentText, b.Type)
		}
		if b.Text != "hello world" {
			t.Errorf("expected text %q, got %q", "hello world", b.Text)
		}
		if b.Data != "" || b.URL != "" || b.MediaType != "" {
			t.Error("expected empty Data, URL, MediaType for text block")
		}
	})

	t.Run("ImageBlock", func(t *testing.T) {
		b := ImageBlock("image/png", "iVBOR...")
		if b.Type != ContentImage {
			t.Errorf("expected type %q, got %q", ContentImage, b.Type)
		}
		if b.MediaType != "image/png" {
			t.Errorf("expected media_type %q, got %q", "image/png", b.MediaType)
		}
		if b.Data != "iVBOR..." {
			t.Errorf("expected data %q, got %q", "iVBOR...", b.Data)
		}
		if b.URL != "" || b.Text != "" {
			t.Error("expected empty URL and Text for image block")
		}
	})

	t.Run("ImageURLBlock", func(t *testing.T) {
		b := ImageURLBlock("https://example.com/img.png")
		if b.Type != ContentImage {
			t.Errorf("expected type %q, got %q", ContentImage, b.Type)
		}
		if b.URL != "https://example.com/img.png" {
			t.Errorf("expected url %q, got %q", "https://example.com/img.png", b.URL)
		}
		if b.Data != "" || b.Text != "" {
			t.Error("expected empty Data and Text for image URL block")
		}
	})

	t.Run("AudioBlock", func(t *testing.T) {
		b := AudioBlock("audio/wav", "AAAA...")
		if b.Type != ContentAudio {
			t.Errorf("expected type %q, got %q", ContentAudio, b.Type)
		}
		if b.MediaType != "audio/wav" {
			t.Errorf("expected media_type %q, got %q", "audio/wav", b.MediaType)
		}
		if b.Data != "AAAA..." {
			t.Errorf("expected data %q, got %q", "AAAA...", b.Data)
		}
	})
}

func TestDetectMediaType(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"photo.png", "image/png"},
		{"photo.PNG", "image/png"},
		{"photo.jpg", "image/jpeg"},
		{"photo.jpeg", "image/jpeg"},
		{"photo.gif", "image/gif"},
		{"photo.webp", "image/webp"},
		{"photo.bmp", "image/bmp"},
		{"audio.wav", "audio/wav"},
		{"audio.mp3", "audio/mpeg"},
		{"audio.ogg", "audio/ogg"},
		{"audio.flac", "audio/flac"},
		{"audio.m4a", "audio/mp4"},
		{"file.txt", ""},
		{"file", ""},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			got := DetectMediaType(tc.path)
			if got != tc.expected {
				t.Errorf("DetectMediaType(%q) = %q, want %q", tc.path, got, tc.expected)
			}
		})
	}
}

func TestIsImageFile(t *testing.T) {
	if !IsImageFile("photo.png") {
		t.Error("expected photo.png to be image")
	}
	if !IsImageFile("photo.JPG") {
		t.Error("expected photo.JPG to be image")
	}
	if IsImageFile("audio.wav") {
		t.Error("expected audio.wav not to be image")
	}
	if IsImageFile("file.txt") {
		t.Error("expected file.txt not to be image")
	}
}

func TestIsAudioFile(t *testing.T) {
	if !IsAudioFile("audio.wav") {
		t.Error("expected audio.wav to be audio")
	}
	if !IsAudioFile("audio.MP3") {
		t.Error("expected audio.MP3 to be audio")
	}
	if IsAudioFile("photo.png") {
		t.Error("expected photo.png not to be audio")
	}
	if IsAudioFile("file.txt") {
		t.Error("expected file.txt not to be audio")
	}
}

func TestToMultimodal(t *testing.T) {
	msg := Message{Role: RoleUser, Content: "hello"}
	mm := ToMultimodal(msg)

	if mm.Role != RoleUser {
		t.Errorf("expected role %q, got %q", RoleUser, mm.Role)
	}
	if len(mm.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(mm.Content))
	}
	if mm.Content[0].Type != ContentText {
		t.Errorf("expected text block, got %q", mm.Content[0].Type)
	}
	if mm.Content[0].Text != "hello" {
		t.Errorf("expected text %q, got %q", "hello", mm.Content[0].Text)
	}
}

func TestFromMultimodal(t *testing.T) {
	mm := MultimodalMessage{
		Role: RoleAssistant,
		Content: []ContentBlock{
			TextBlock("first"),
			ImageBlock("image/png", "data"),
			TextBlock(" second"),
		},
	}

	msg := FromMultimodal(mm)
	if msg.Role != RoleAssistant {
		t.Errorf("expected role %q, got %q", RoleAssistant, msg.Role)
	}
	if msg.Content != "first second" {
		t.Errorf("expected content %q, got %q", "first second", msg.Content)
	}
}

func TestEncodeImageFile(t *testing.T) {
	// Create a minimal valid PNG (1x1 pixel)
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
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

	block, err := EncodeImageFile(pngPath)
	if err != nil {
		t.Fatalf("EncodeImageFile failed: %v", err)
	}
	if block.Type != ContentImage {
		t.Errorf("expected type %q, got %q", ContentImage, block.Type)
	}
	if block.MediaType != "image/png" {
		t.Errorf("expected media_type %q, got %q", "image/png", block.MediaType)
	}
	if block.Data == "" {
		t.Error("expected non-empty base64 data")
	}

	// Test non-existent file
	_, err = EncodeImageFile("/nonexistent/file.png")
	if err == nil {
		t.Error("expected error for non-existent file")
	}

	// Test unsupported extension
	txtPath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(txtPath, []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	_, err = EncodeImageFile(txtPath)
	if err == nil {
		t.Error("expected error for unsupported file type")
	}
}

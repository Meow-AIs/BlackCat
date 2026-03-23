package llm

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ContentType represents the type of a content block in a multimodal message.
type ContentType string

const (
	// ContentText is a text content block.
	ContentText ContentType = "text"
	// ContentImage is an image content block (base64 or URL).
	ContentImage ContentType = "image"
	// ContentAudio is an audio content block (base64).
	ContentAudio ContentType = "audio"
)

// ContentBlock represents a single piece of content in a multimodal message.
type ContentBlock struct {
	Type      ContentType `json:"type"`
	Text      string      `json:"text,omitempty"`
	MediaType string      `json:"media_type,omitempty"` // "image/png", "image/jpeg", "audio/wav"
	Data      string      `json:"data,omitempty"`       // base64 encoded
	URL       string      `json:"url,omitempty"`        // or URL reference
}

// MultimodalMessage is a message that can contain mixed content types.
type MultimodalMessage struct {
	Role    Role           `json:"role"`
	Content []ContentBlock `json:"content"`
}

// TextBlock creates a text content block.
func TextBlock(text string) ContentBlock {
	return ContentBlock{
		Type: ContentText,
		Text: text,
	}
}

// ImageBlock creates an image content block from base64 data.
func ImageBlock(mediaType, base64Data string) ContentBlock {
	return ContentBlock{
		Type:      ContentImage,
		MediaType: mediaType,
		Data:      base64Data,
	}
}

// ImageURLBlock creates an image content block from a URL.
func ImageURLBlock(url string) ContentBlock {
	return ContentBlock{
		Type: ContentImage,
		URL:  url,
	}
}

// AudioBlock creates an audio content block from base64 data.
func AudioBlock(mediaType, base64Data string) ContentBlock {
	return ContentBlock{
		Type:      ContentAudio,
		MediaType: mediaType,
		Data:      base64Data,
	}
}

// ToMultimodal converts a plain Message to a MultimodalMessage.
func ToMultimodal(msg Message) MultimodalMessage {
	return MultimodalMessage{
		Role:    msg.Role,
		Content: []ContentBlock{TextBlock(msg.Content)},
	}
}

// FromMultimodal converts a MultimodalMessage to a plain Message,
// extracting only text content blocks.
func FromMultimodal(mm MultimodalMessage) Message {
	var parts []string
	for _, block := range mm.Content {
		if block.Type == ContentText && block.Text != "" {
			parts = append(parts, block.Text)
		}
	}
	return Message{
		Role:    mm.Role,
		Content: strings.Join(parts, ""),
	}
}

// imageExtensions maps file extensions to MIME types for images.
var imageExtensions = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".webp": "image/webp",
	".bmp":  "image/bmp",
	".svg":  "image/svg+xml",
	".tiff": "image/tiff",
	".tif":  "image/tiff",
}

// audioExtensions maps file extensions to MIME types for audio.
var audioExtensions = map[string]string{
	".wav":  "audio/wav",
	".mp3":  "audio/mpeg",
	".ogg":  "audio/ogg",
	".flac": "audio/flac",
	".m4a":  "audio/mp4",
	".aac":  "audio/aac",
	".wma":  "audio/x-ms-wma",
	".webm": "audio/webm",
}

// DetectMediaType returns the MIME type for a file based on its extension.
// Returns an empty string if the extension is not recognized.
func DetectMediaType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if mt, ok := imageExtensions[ext]; ok {
		return mt
	}
	if mt, ok := audioExtensions[ext]; ok {
		return mt
	}
	return ""
}

// IsImageFile returns true if the file path has a recognized image extension.
func IsImageFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	_, ok := imageExtensions[ext]
	return ok
}

// IsAudioFile returns true if the file path has a recognized audio extension.
func IsAudioFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	_, ok := audioExtensions[ext]
	return ok
}

// EncodeImageFile reads an image file from disk and returns it as a base64
// ContentBlock. Returns an error if the file cannot be read or has an
// unsupported extension.
func EncodeImageFile(path string) (ContentBlock, error) {
	if !IsImageFile(path) {
		return ContentBlock{}, fmt.Errorf("unsupported image file type: %s", filepath.Ext(path))
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return ContentBlock{}, fmt.Errorf("failed to read image file: %w", err)
	}

	mediaType := DetectMediaType(path)
	encoded := base64.StdEncoding.EncodeToString(data)

	return ImageBlock(mediaType, encoded), nil
}

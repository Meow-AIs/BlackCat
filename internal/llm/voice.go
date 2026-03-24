package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultGroqBaseURL  = "https://api.groq.com/openai/v1"
	defaultWhisperModel = "whisper-large-v3-turbo"
	voiceTimeout        = 60 * time.Second
)

// SupportedAudioFormats lists audio file extensions supported by Groq Whisper.
var SupportedAudioFormats = []string{
	".mp3", ".mp4", ".mpeg", ".mpga", ".m4a", ".wav", ".webm", ".ogg", ".flac",
}

// VoiceProvider handles speech-to-text transcription via Groq Whisper API.
type VoiceProvider struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewVoiceProvider creates a VoiceProvider with default Groq settings.
func NewVoiceProvider(apiKey string) *VoiceProvider {
	return &VoiceProvider{
		apiKey:     apiKey,
		baseURL:    defaultGroqBaseURL,
		model:      defaultWhisperModel,
		httpClient: &http.Client{Timeout: voiceTimeout},
	}
}

// NewVoiceProviderWithURL creates a VoiceProvider with a custom base URL.
func NewVoiceProviderWithURL(apiKey, baseURL string) *VoiceProvider {
	return &VoiceProvider{
		apiKey:     apiKey,
		baseURL:    baseURL,
		model:      defaultWhisperModel,
		httpClient: &http.Client{Timeout: voiceTimeout},
	}
}

// TranscriptionResponse is the JSON response from the Whisper API.
type TranscriptionResponse struct {
	Text string `json:"text"`
}

// Transcribe sends audio data to Groq Whisper API and returns the transcribed text.
func (v *VoiceProvider) Transcribe(ctx context.Context, audioData []byte, language string) (string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add audio file part
	part, err := writer.CreateFormFile("file", "audio.wav")
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write(audioData); err != nil {
		return "", fmt.Errorf("failed to write audio data: %w", err)
	}

	// Add model field
	if err := writer.WriteField("model", v.model); err != nil {
		return "", fmt.Errorf("failed to write model field: %w", err)
	}

	// Add response format
	if err := writer.WriteField("response_format", "json"); err != nil {
		return "", fmt.Errorf("failed to write response_format field: %w", err)
	}

	// Add optional language
	if language != "" {
		if err := writer.WriteField("language", language); err != nil {
			return "", fmt.Errorf("failed to write language field: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	url := v.baseURL + "/audio/transcriptions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+v.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("transcription request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("transcription API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result TranscriptionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse transcription response: %w", err)
	}

	return result.Text, nil
}

// TranscribeFile reads an audio file from disk and transcribes it.
func (v *VoiceProvider) TranscribeFile(ctx context.Context, filePath string, language string) (string, error) {
	if !IsSupportedAudioFormat(filePath) {
		return "", fmt.Errorf("unsupported audio format: %s", filepath.Ext(filePath))
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read audio file: %w", err)
	}

	return v.Transcribe(ctx, data, language)
}

// IsSupportedAudioFormat returns true if the file has a supported audio extension.
func IsSupportedAudioFormat(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, supported := range SupportedAudioFormats {
		if ext == supported {
			return true
		}
	}
	return false
}

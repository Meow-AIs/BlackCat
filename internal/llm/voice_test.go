package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNewVoiceProvider(t *testing.T) {
	vp := NewVoiceProvider("test-key")
	if vp.apiKey != "test-key" {
		t.Errorf("expected apiKey %q, got %q", "test-key", vp.apiKey)
	}
	if vp.baseURL != "https://api.groq.com/openai/v1" {
		t.Errorf("unexpected default baseURL: %s", vp.baseURL)
	}
	if vp.model != "whisper-large-v3-turbo" {
		t.Errorf("unexpected default model: %s", vp.model)
	}
}

func TestNewVoiceProviderWithURL(t *testing.T) {
	vp := NewVoiceProviderWithURL("key2", "https://custom.api.com/v1")
	if vp.baseURL != "https://custom.api.com/v1" {
		t.Errorf("expected baseURL %q, got %q", "https://custom.api.com/v1", vp.baseURL)
	}
}

func TestTranscribe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/audio/transcriptions" {
			t.Errorf("expected path /audio/transcriptions, got %s", r.URL.Path)
		}
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-key" {
			t.Errorf("expected auth header %q, got %q", "Bearer test-key", authHeader)
		}
		contentType := r.Header.Get("Content-Type")
		if len(contentType) < 19 { // "multipart/form-data" prefix
			t.Errorf("expected multipart content type, got %q", contentType)
		}

		// Parse multipart to verify fields
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("failed to parse multipart: %v", err)
		}
		if r.FormValue("model") != "whisper-large-v3-turbo" {
			t.Errorf("unexpected model: %s", r.FormValue("model"))
		}
		if r.FormValue("response_format") != "json" {
			t.Errorf("unexpected response_format: %s", r.FormValue("response_format"))
		}

		resp := TranscriptionResponse{Text: "Hello, world!"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	vp := NewVoiceProviderWithURL("test-key", server.URL)
	text, err := vp.Transcribe(context.Background(), []byte("fake-audio-data"), "")
	if err != nil {
		t.Fatalf("Transcribe failed: %v", err)
	}
	if text != "Hello, world!" {
		t.Errorf("expected %q, got %q", "Hello, world!", text)
	}
}

func TestTranscribeWithLanguage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("failed to parse multipart: %v", err)
		}
		if r.FormValue("language") != "en" {
			t.Errorf("expected language %q, got %q", "en", r.FormValue("language"))
		}

		resp := TranscriptionResponse{Text: "Hello"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	vp := NewVoiceProviderWithURL("test-key", server.URL)
	text, err := vp.Transcribe(context.Background(), []byte("audio"), "en")
	if err != nil {
		t.Fatalf("Transcribe failed: %v", err)
	}
	if text != "Hello" {
		t.Errorf("expected %q, got %q", "Hello", text)
	}
}

func TestTranscribeAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": {"message": "invalid api key"}}`))
	}))
	defer server.Close()

	vp := NewVoiceProviderWithURL("bad-key", server.URL)
	_, err := vp.Transcribe(context.Background(), []byte("audio"), "")
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

func TestTranscribeFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := TranscriptionResponse{Text: "File transcription"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	wavPath := filepath.Join(tmpDir, "test.wav")
	if err := os.WriteFile(wavPath, []byte("fake-wav"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	vp := NewVoiceProviderWithURL("test-key", server.URL)
	text, err := vp.TranscribeFile(context.Background(), wavPath, "")
	if err != nil {
		t.Fatalf("TranscribeFile failed: %v", err)
	}
	if text != "File transcription" {
		t.Errorf("expected %q, got %q", "File transcription", text)
	}
}

func TestTranscribeFileNonExistent(t *testing.T) {
	vp := NewVoiceProvider("test-key")
	_, err := vp.TranscribeFile(context.Background(), "/nonexistent/audio.wav", "")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestTranscribeFileUnsupportedFormat(t *testing.T) {
	tmpDir := t.TempDir()
	txtPath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(txtPath, []byte("not audio"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	vp := NewVoiceProvider("test-key")
	_, err := vp.TranscribeFile(context.Background(), txtPath, "")
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
}

func TestIsSupportedAudioFormat(t *testing.T) {
	supported := []string{".mp3", ".mp4", ".mpeg", ".mpga", ".m4a", ".wav", ".webm", ".ogg", ".flac"}
	for _, ext := range supported {
		if !IsSupportedAudioFormat("file" + ext) {
			t.Errorf("expected %s to be supported", ext)
		}
	}
	unsupported := []string{".txt", ".png", ".pdf", ".doc"}
	for _, ext := range unsupported {
		if IsSupportedAudioFormat("file" + ext) {
			t.Errorf("expected %s to not be supported", ext)
		}
	}
}

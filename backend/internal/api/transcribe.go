package api

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/xela-io/xelanote/internal/llm"
)

// transcribeAudio handles POST /llm/transcribe.
// Accepts multipart/form-data with an audio file and returns transcribed text.
func (s *Server) transcribeAudio(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Limit request body to 25 MB (OpenAI Whisper limit)
	r.Body = http.MaxBytesReader(w, r.Body, 25*1024*1024)

	if err := r.ParseMultipartForm(25 * 1024 * 1024); err != nil {
		respondError(w, http.StatusBadRequest, "invalid multipart form or file too large (max 25 MB)")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "audio file is required (field: file)")
		return
	}
	defer file.Close()

	// Validate content type
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		// Infer from filename extension
		name := strings.ToLower(header.Filename)
		switch {
		case strings.HasSuffix(name, ".webm"):
			contentType = "audio/webm"
		case strings.HasSuffix(name, ".ogg"):
			contentType = "audio/ogg"
		case strings.HasSuffix(name, ".wav"):
			contentType = "audio/wav"
		case strings.HasSuffix(name, ".mp3"):
			contentType = "audio/mpeg"
		case strings.HasSuffix(name, ".mp4"), strings.HasSuffix(name, ".m4a"):
			contentType = "audio/mp4"
		}
	}

	allowedTypes := map[string]bool{
		"audio/webm":           true,
		"audio/ogg":            true,
		"audio/wav":            true,
		"audio/mpeg":           true,
		"audio/mp4":            true,
		"audio/m4a":            true,
		"audio/x-m4a":          true,
		"audio/mp4a-latm":      true,
		"audio/webm;codecs=opus": true,
		"audio/ogg;codecs=opus":  true,
	}

	// Normalize: strip params for validation
	baseType := strings.Split(contentType, ";")[0]
	baseType = strings.TrimSpace(baseType)

	if !allowedTypes[contentType] && !allowedTypes[baseType] {
		respondError(w, http.StatusBadRequest, "unsupported audio format: "+contentType)
		return
	}

	audioData, err := io.ReadAll(file)
	if err != nil {
		respondError(w, http.StatusBadRequest, "failed to read audio file")
		return
	}

	if len(audioData) == 0 {
		respondError(w, http.StatusBadRequest, "audio file is empty")
		return
	}

	filename := header.Filename
	if filename == "" {
		filename = "audio.webm"
	}

	text, err := s.summarizeService.TranscribeAudio(r.Context(), userID, audioData, filename)
	if err != nil {
		s.logger().Error("transcription failed",
			"error", err,
			"user_id", userID,
			"audio_size", len(audioData),
		)

		switch {
		case errors.Is(err, llm.ErrChatGPTNotConfigured):
			respondError(w, http.StatusPreconditionFailed, "OpenAI API key required for transcription — add it in Settings → AI")
		case errors.Is(err, r.Context().Err()):
			respondError(w, http.StatusGatewayTimeout, "Transcription timed out — try a shorter recording")
		default:
			respondError(w, http.StatusInternalServerError, "Transcription failed")
		}
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"text": text,
	})
}

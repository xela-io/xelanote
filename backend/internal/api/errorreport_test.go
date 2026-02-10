package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xela-io/xelanote/internal/service"
)

func newErrorReportTestServer(svc *service.ErrorReportService) *Server {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewServer(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, svc, nil, nil, logger, []byte("test-secret-key-that-is-at-least-64-chars-long-for-testing-purposes!!"), "", nil)
}

func TestSubmitErrorReport_Disabled(t *testing.T) {
	svc := service.NewErrorReportService("", "", "", "", slog.New(slog.NewTextHandler(io.Discard, nil)))
	server := newErrorReportTestServer(svc)

	body := `{"type":"automatic","message":"test error","fingerprint":"1234567890abcdef"}`
	req := httptest.NewRequest("POST", "/api/error-reports", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.submitErrorReport(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestSubmitErrorReport_NilService(t *testing.T) {
	server := newErrorReportTestServer(nil)

	body := `{"type":"automatic","message":"test error","fingerprint":"1234567890abcdef"}`
	req := httptest.NewRequest("POST", "/api/error-reports", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.submitErrorReport(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestSubmitErrorReport_InvalidType(t *testing.T) {
	svc := service.NewErrorReportService("https://forgejo.example.com", "owner", "repo", "token",
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	server := newErrorReportTestServer(svc)

	body := `{"type":"invalid","message":"test error","fingerprint":"1234567890abcdef"}`
	req := httptest.NewRequest("POST", "/api/error-reports", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.submitErrorReport(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSubmitErrorReport_EmptyMessage(t *testing.T) {
	svc := service.NewErrorReportService("https://forgejo.example.com", "owner", "repo", "token",
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	server := newErrorReportTestServer(svc)

	body := `{"type":"automatic","message":"ab","fingerprint":"1234567890abcdef"}`
	req := httptest.NewRequest("POST", "/api/error-reports", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.submitErrorReport(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSubmitErrorReport_ManualShortMessage(t *testing.T) {
	svc := service.NewErrorReportService("https://forgejo.example.com", "owner", "repo", "token",
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	server := newErrorReportTestServer(svc)

	body := `{"type":"manual","message":"short","fingerprint":"1234567890abcdef"}`
	req := httptest.NewRequest("POST", "/api/error-reports", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.submitErrorReport(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSubmitErrorReport_InvalidFingerprint(t *testing.T) {
	svc := service.NewErrorReportService("https://forgejo.example.com", "owner", "repo", "token",
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	server := newErrorReportTestServer(svc)

	tests := []struct {
		name        string
		fingerprint string
	}{
		{"too short", `"abc"`},
		{"too long", `"1234567890abcdef1234"`},
		{"uppercase", `"1234567890ABCDEF"`},
		{"non-hex", `"123456789xabcdef"`},
		{"empty", `""`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := `{"type":"automatic","message":"test error","fingerprint":` + tt.fingerprint + `}`
			req := httptest.NewRequest("POST", "/api/error-reports", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.submitErrorReport(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", w.Code)
			}
		})
	}
}

func TestSubmitErrorReport_PayloadTooLarge(t *testing.T) {
	svc := service.NewErrorReportService("https://forgejo.example.com", "owner", "repo", "token",
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	server := newErrorReportTestServer(svc)

	// Create payload larger than 16KB
	largeMessage := strings.Repeat("x", 20000)
	body := `{"type":"automatic","message":"` + largeMessage + `","fingerprint":"1234567890abcdef"}`
	req := httptest.NewRequest("POST", "/api/error-reports", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.submitErrorReport(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", w.Code)
	}
}

func TestSubmitErrorReport_ValidLargePayload(t *testing.T) {
	svc := service.NewErrorReportService("https://forgejo.example.com", "owner", "repo", "token",
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	server := newErrorReportTestServer(svc)

	// Create valid payload under 16KB limit
	payload := map[string]string{
		"type":        "automatic",
		"message":     strings.Repeat("x", 400),
		"fingerprint": "1234567890abcdef",
		"stack":       strings.Repeat("at foo.js:42\n", 100),
		"url":         "/notes",
	}
	jsonBody, _ := json.Marshal(payload)
	if len(jsonBody) >= MaxErrorReportBodySize {
		t.Fatalf("test payload too large: %d >= %d", len(jsonBody), MaxErrorReportBodySize)
	}

	req := httptest.NewRequest("POST", "/api/error-reports", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.submitErrorReport(w, req)

	// Should be 200 OK (Forgejo is not reachable, so it will fail gracefully)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSubmitErrorReport_FieldLengthLimits(t *testing.T) {
	svc := service.NewErrorReportService("https://forgejo.example.com", "owner", "repo", "token",
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	server := newErrorReportTestServer(svc)

	tests := []struct {
		name string
		body string
	}{
		{"message too long", `{"type":"automatic","message":"` + strings.Repeat("x", 501) + `","fingerprint":"1234567890abcdef"}`},
		{"stack too long", `{"type":"automatic","message":"test error","fingerprint":"1234567890abcdef","stack":"` + strings.Repeat("x", 4001) + `"}`},
		{"description too long", `{"type":"automatic","message":"test error","fingerprint":"1234567890abcdef","description":"` + strings.Repeat("x", 2001) + `"}`},
		{"steps too long", `{"type":"automatic","message":"test error","fingerprint":"1234567890abcdef","steps_to_reproduce":"` + strings.Repeat("x", 2001) + `"}`},
		{"error_type too long", `{"type":"automatic","message":"test error","fingerprint":"1234567890abcdef","error_type":"` + strings.Repeat("x", 51) + `"}`},
		{"url too long", `{"type":"automatic","message":"test error","fingerprint":"1234567890abcdef","url":"/` + strings.Repeat("x", 500) + `"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/error-reports", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.submitErrorReport(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", w.Code)
			}
		})
	}
}

func TestSubmitErrorReport_WhitespaceOnlyMessage(t *testing.T) {
	svc := service.NewErrorReportService("https://forgejo.example.com", "owner", "repo", "token",
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	server := newErrorReportTestServer(svc)

	body := `{"type":"automatic","message":"   \n\t   ","fingerprint":"1234567890abcdef"}`
	req := httptest.NewRequest("POST", "/api/error-reports", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.submitErrorReport(w, req)

	// After trim, message is empty → should fail minimum length
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSubmitErrorReport_UserAgentFromHeader(t *testing.T) {
	// We can't easily verify the UserAgent is from the header without a real Forgejo,
	// but we can verify the handler doesn't crash and uses proper sanitization
	svc := service.NewErrorReportService("https://forgejo.example.com", "owner", "repo", "token",
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	server := newErrorReportTestServer(svc)

	body := `{"type":"automatic","message":"test error msg","fingerprint":"1234567890abcdef"}`
	req := httptest.NewRequest("POST", "/api/error-reports", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "TestBrowser/1.0")
	w := httptest.NewRecorder()

	server.submitErrorReport(w, req)

	// Will be 200 (with accepted: false because Forgejo not reachable)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestSubmitErrorReport_TrimSanitization(t *testing.T) {
	svc := service.NewErrorReportService("https://forgejo.example.com", "owner", "repo", "token",
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	server := newErrorReportTestServer(svc)

	// Message with leading/trailing whitespace — after trim should still be valid
	body := `{"type":"automatic","message":"  test error with spaces  ","fingerprint":"1234567890abcdef"}`
	req := httptest.NewRequest("POST", "/api/error-reports", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.submitErrorReport(w, req)

	// Should pass validation (trimmed message is long enough)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSubmitErrorReport_RelativeURLRequired(t *testing.T) {
	svc := service.NewErrorReportService("https://forgejo.example.com", "owner", "repo", "token",
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	server := newErrorReportTestServer(svc)

	body := `{"type":"automatic","message":"test error msg","fingerprint":"1234567890abcdef","url":"https://evil.com/steal"}`
	req := httptest.NewRequest("POST", "/api/error-reports", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.submitErrorReport(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for absolute URL, got %d", w.Code)
	}
}

func TestSubmitErrorReport_InvalidJSON(t *testing.T) {
	svc := service.NewErrorReportService("https://forgejo.example.com", "owner", "repo", "token",
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	server := newErrorReportTestServer(svc)

	req := httptest.NewRequest("POST", "/api/error-reports", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.submitErrorReport(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

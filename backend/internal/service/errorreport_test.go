package service

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestComputeFingerprint_Deterministic(t *testing.T) {
	fp1 := ComputeFingerprint("BackendError", "failed to fetch notes")
	fp2 := ComputeFingerprint("BackendError", "failed to fetch notes")
	if fp1 != fp2 {
		t.Fatalf("fingerprints not deterministic: %q != %q", fp1, fp2)
	}
	if len(fp1) != 16 {
		t.Fatalf("expected 16-char fingerprint, got %d chars: %q", len(fp1), fp1)
	}
}

func TestComputeFingerprint_DifferentInputsDiffer(t *testing.T) {
	fp1 := ComputeFingerprint("BackendError", "error A")
	fp2 := ComputeFingerprint("BackendError", "error B")
	if fp1 == fp2 {
		t.Fatal("different messages should produce different fingerprints")
	}
}

func TestComputeFingerprint_NormalizesVolatileParts(t *testing.T) {
	// Messages that differ only by numbers/UUIDs/dates should produce the same fingerprint.
	fp1 := ComputeFingerprint("BackendError", "note 42 not found")
	fp2 := ComputeFingerprint("BackendError", "note 999 not found")
	if fp1 != fp2 {
		t.Fatalf("expected same fingerprint after number normalisation: %q != %q", fp1, fp2)
	}

	fp3 := ComputeFingerprint("BackendError", "user a1b2c3d4-e5f6-7890-abcd-ef1234567890 error")
	fp4 := ComputeFingerprint("BackendError", "user 11111111-2222-3333-4444-555555555555 error")
	if fp3 != fp4 {
		t.Fatalf("expected same fingerprint after UUID normalisation: %q != %q", fp3, fp4)
	}
}

func TestNormalizeMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "numbers replaced",
			input:    "note 42 not found in list 7",
			expected: "note N not found in list N",
		},
		{
			name:     "UUID replaced",
			input:    "user a1b2c3d4-e5f6-7890-abcd-ef1234567890 deleted",
			expected: "user UUID deleted",
		},
		{
			name:     "ISO date replaced",
			input:    "expired at 2024-01-15T10:30:00Z",
			expected: "expired at DATE",
		},
		{
			name:     "mixed",
			input:    "note 5 by a1b2c3d4-e5f6-7890-abcd-ef1234567890 at 2024-01-15T10:30:00.123+02:00",
			expected: "note N by UUID at DATE",
		},
		{
			name:     "no volatile parts",
			input:    "database connection failed",
			expected: "database connection failed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeMessage(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizeMessage(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestNewErrorReportService_Disabled(t *testing.T) {
	svc := NewErrorReportService("", "", "", "", newTestLogger())
	if svc.IsEnabled() {
		t.Fatal("expected service to be disabled when params are empty")
	}
}

func TestNewErrorReportService_Enabled(t *testing.T) {
	svc := NewErrorReportService("https://forgejo.example.com", "owner", "repo", "token", newTestLogger())
	if !svc.IsEnabled() {
		t.Fatal("expected service to be enabled")
	}
}

func TestSubmitReport_Disabled(t *testing.T) {
	svc := NewErrorReportService("", "", "", "", newTestLogger())
	result, err := svc.SubmitReport(context.Background(), ErrorReport{
		Type:        "automatic",
		Message:     "test error",
		Fingerprint: "1234567890abcdef",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Accepted {
		t.Fatal("expected Accepted=false when service is disabled")
	}
}

func TestSubmitReport_NewIssue(t *testing.T) {
	var (
		labelsCreated int32
		issueCreated  int32
		createdIssue  map[string]interface{}
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/labels"):
			// Return existing auto-report, user-feedback, and backend labels
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]forgejoLabel{
				{ID: 1, Name: "auto-report", Color: "#e11d48"},
				{ID: 2, Name: "user-feedback", Color: "#2563eb"},
				{ID: 3, Name: "backend", Color: "#f59e0b"},
			})

		case r.Method == "POST" && strings.Contains(r.URL.Path, "/labels"):
			// Create fingerprint label
			atomic.AddInt32(&labelsCreated, 1)
			var payload map[string]string
			json.NewDecoder(r.Body).Decode(&payload)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(forgejoLabel{
				ID:    100,
				Name:  payload["name"],
				Color: payload["color"],
			})

		case r.Method == "GET" && strings.Contains(r.URL.Path, "/issues"):
			// No existing issues (new issue scenario)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]forgejoIssue{})

		case r.Method == "POST" && strings.Contains(r.URL.Path, "/issues") && !strings.Contains(r.URL.Path, "/comments"):
			atomic.AddInt32(&issueCreated, 1)
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &createdIssue)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{"number": 42})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	svc := NewErrorReportService(server.URL, "owner", "repo", "token", newTestLogger())
	if err := svc.EnsureLabels(context.Background()); err != nil {
		t.Fatalf("EnsureLabels failed: %v", err)
	}

	result, err := svc.SubmitReport(context.Background(), ErrorReport{
		Type:        "automatic",
		ErrorType:   "TypeError",
		Message:     "Cannot read property 'foo' of null",
		Stack:       "at foo.js:42",
		Fingerprint: "1234567890abcdef",
		URL:         "/notes/123",
		UserAgent:   "Test/1.0",
	})
	if err != nil {
		t.Fatalf("SubmitReport failed: %v", err)
	}
	if !result.Accepted {
		t.Fatal("expected Accepted=true")
	}
	if atomic.LoadInt32(&issueCreated) != 1 {
		t.Fatalf("expected 1 issue created, got %d", atomic.LoadInt32(&issueCreated))
	}
	if atomic.LoadInt32(&labelsCreated) != 1 {
		t.Fatalf("expected 1 label created (fingerprint), got %d", atomic.LoadInt32(&labelsCreated))
	}

	// Verify labels are IDs, not names
	labels, ok := createdIssue["labels"].([]interface{})
	if !ok {
		t.Fatal("expected labels to be an array")
	}
	if len(labels) != 2 {
		t.Fatalf("expected 2 labels, got %d", len(labels))
	}
	// Labels should be numeric IDs: auto-report=1, fingerprint=100
	labelID1, ok := labels[0].(float64)
	if !ok || labelID1 != 1 {
		t.Fatalf("expected first label ID=1, got %v", labels[0])
	}
	labelID2, ok := labels[1].(float64)
	if !ok || labelID2 != 100 {
		t.Fatalf("expected second label ID=100, got %v", labels[1])
	}
}

func TestSubmitReport_DuplicateAddsComment(t *testing.T) {
	var commentCreated int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/labels"):
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]forgejoLabel{
				{ID: 1, Name: "auto-report", Color: "#e11d48"},
				{ID: 2, Name: "user-feedback", Color: "#2563eb"},
				{ID: 3, Name: "backend", Color: "#f59e0b"},
			})

		case r.Method == "GET" && strings.Contains(r.URL.Path, "/issues"):
			// Return existing issue (duplicate scenario)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]forgejoIssue{
				{Number: 42, Title: "Existing issue"},
			})

		case r.Method == "POST" && strings.Contains(r.URL.Path, "/comments"):
			atomic.AddInt32(&commentCreated, 1)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{"id": 1})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	svc := NewErrorReportService(server.URL, "owner", "repo", "token", newTestLogger())
	if err := svc.EnsureLabels(context.Background()); err != nil {
		t.Fatalf("EnsureLabels failed: %v", err)
	}

	result, err := svc.SubmitReport(context.Background(), ErrorReport{
		Type:        "automatic",
		ErrorType:   "TypeError",
		Message:     "test error",
		Fingerprint: "1234567890abcdef",
	})
	if err != nil {
		t.Fatalf("SubmitReport failed: %v", err)
	}
	if !result.Accepted {
		t.Fatal("expected Accepted=true")
	}
	if atomic.LoadInt32(&commentCreated) != 1 {
		t.Fatalf("expected 1 comment created, got %d", atomic.LoadInt32(&commentCreated))
	}
}

func TestSubmitReport_ManualFeedbackAlwaysCreatesNewIssue(t *testing.T) {
	var (
		issueCreated   int32
		commentCreated int32
		labelsCreated  int32
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/labels"):
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]forgejoLabel{
				{ID: 1, Name: "auto-report", Color: "#e11d48"},
				{ID: 2, Name: "user-feedback", Color: "#2563eb"},
				{ID: 3, Name: "backend", Color: "#f59e0b"},
			})

		case r.Method == "POST" && strings.Contains(r.URL.Path, "/labels"):
			atomic.AddInt32(&labelsCreated, 1)
			var payload map[string]string
			json.NewDecoder(r.Body).Decode(&payload)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(forgejoLabel{
				ID:    100,
				Name:  payload["name"],
				Color: payload["color"],
			})

		case r.Method == "GET" && strings.Contains(r.URL.Path, "/issues"):
			// Return an existing issue — dedup would match this for automatic reports
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]forgejoIssue{
				{Number: 42, Title: "Existing issue"},
			})

		case r.Method == "POST" && strings.Contains(r.URL.Path, "/comments"):
			atomic.AddInt32(&commentCreated, 1)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{"id": 1})

		case r.Method == "POST" && strings.Contains(r.URL.Path, "/issues"):
			atomic.AddInt32(&issueCreated, 1)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{"number": 99})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	svc := NewErrorReportService(server.URL, "owner", "repo", "token", newTestLogger())
	if err := svc.EnsureLabels(context.Background()); err != nil {
		t.Fatalf("EnsureLabels failed: %v", err)
	}

	result, err := svc.SubmitReport(context.Background(), ErrorReport{
		Type:        "manual",
		ErrorType:   "UserFeedback",
		Message:     "The search doesn't work properly",
		Fingerprint: "1234567890abcdef",
	})
	if err != nil {
		t.Fatalf("SubmitReport failed: %v", err)
	}
	if !result.Accepted {
		t.Fatal("expected Accepted=true")
	}
	if atomic.LoadInt32(&issueCreated) != 1 {
		t.Fatal("expected manual feedback to create a new issue")
	}
	if atomic.LoadInt32(&commentCreated) != 0 {
		t.Fatal("expected manual feedback to NOT add a comment to an existing issue")
	}
}

func TestEnsureLabels_CreatesLabels(t *testing.T) {
	var labelsCreated int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/labels"):
			// No existing labels
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]forgejoLabel{})

		case r.Method == "POST" && strings.Contains(r.URL.Path, "/labels"):
			n := atomic.AddInt32(&labelsCreated, 1)
			var payload map[string]string
			json.NewDecoder(r.Body).Decode(&payload)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(forgejoLabel{
				ID:    int64(n),
				Name:  payload["name"],
				Color: payload["color"],
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	svc := NewErrorReportService(server.URL, "owner", "repo", "token", newTestLogger())
	if err := svc.EnsureLabels(context.Background()); err != nil {
		t.Fatalf("EnsureLabels failed: %v", err)
	}
	if atomic.LoadInt32(&labelsCreated) != 3 {
		t.Fatalf("expected 3 labels created (auto-report, user-feedback, backend), got %d", atomic.LoadInt32(&labelsCreated))
	}
}

func TestSubmitReport_ForgejoAuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/labels"):
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]forgejoLabel{
				{ID: 1, Name: "auto-report"},
				{ID: 2, Name: "user-feedback"},
				{ID: 3, Name: "backend"},
			})
		default:
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer server.Close()

	svc := NewErrorReportService(server.URL, "owner", "repo", "token", newTestLogger())
	svc.EnsureLabels(context.Background())

	result, err := svc.SubmitReport(context.Background(), ErrorReport{
		Type:        "automatic",
		Message:     "test error",
		Fingerprint: "1234567890abcdef",
	})
	if err == nil {
		t.Fatal("expected error for auth failure")
	}
	if result.Accepted {
		t.Fatal("expected Accepted=false for auth error")
	}
}

func TestSubmitReport_ForgejoServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/labels"):
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]forgejoLabel{
				{ID: 1, Name: "auto-report"},
				{ID: 2, Name: "user-feedback"},
				{ID: 3, Name: "backend"},
			})
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	svc := NewErrorReportService(server.URL, "owner", "repo", "token", newTestLogger())
	svc.EnsureLabels(context.Background())

	result, err := svc.SubmitReport(context.Background(), ErrorReport{
		Type:        "automatic",
		Message:     "test error",
		Fingerprint: "1234567890abcdef",
	})
	if err == nil {
		t.Fatal("expected error for server error")
	}
	if result.Accepted {
		t.Fatal("expected Accepted=false for server error")
	}
}

func TestBodyFallbackSearch(t *testing.T) {
	var searchQueries []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/labels") && !strings.Contains(r.URL.Path, "/issues"):
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]forgejoLabel{
				{ID: 1, Name: "auto-report"},
				{ID: 2, Name: "user-feedback"},
				{ID: 3, Name: "backend"},
			})

		case r.Method == "GET" && strings.Contains(r.URL.Path, "/issues"):
			query := r.URL.Query()
			if q := query.Get("q"); q != "" {
				searchQueries = append(searchQueries, q)
				// Body fallback finds the issue
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode([]forgejoIssue{
					{Number: 99, Title: "Found via body search"},
				})
			} else {
				// Label search finds nothing
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode([]forgejoIssue{})
			}

		case r.Method == "POST" && strings.Contains(r.URL.Path, "/comments"):
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{"id": 1})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	svc := NewErrorReportService(server.URL, "owner", "repo", "token", newTestLogger())
	svc.EnsureLabels(context.Background())

	result, err := svc.SubmitReport(context.Background(), ErrorReport{
		Type:        "automatic",
		Message:     "test error",
		Fingerprint: "abcdef1234567890",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Accepted {
		t.Fatal("expected Accepted=true")
	}
	if len(searchQueries) != 1 {
		t.Fatalf("expected 1 body fallback search, got %d", len(searchQueries))
	}
	if !strings.Contains(searchQueries[0], "abcdef1234567890") {
		t.Fatalf("expected search query to contain fingerprint, got %q", searchQueries[0])
	}
}

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
			// Return existing auto-report and user-feedback labels
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]forgejoLabel{
				{ID: 1, Name: "auto-report", Color: "#e11d48"},
				{ID: 2, Name: "user-feedback", Color: "#2563eb"},
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
	if atomic.LoadInt32(&labelsCreated) != 2 {
		t.Fatalf("expected 2 labels created, got %d", atomic.LoadInt32(&labelsCreated))
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

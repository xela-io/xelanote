package service

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

type rewriteTransport struct {
	base   http.RoundTripper
	target *url.URL
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.URL.Scheme = t.target.Scheme
	cloned.URL.Host = t.target.Host
	cloned.Host = t.target.Host
	return t.base.RoundTrip(cloned)
}

func newTurnstileTestService(t *testing.T, responseBody string) *TurnstileService {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responseBody))
	}))
	t.Cleanup(server.Close)

	targetURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}

	svc := NewTurnstileService("test-secret", "test-site", slog.New(slog.NewTextHandler(io.Discard, nil)))
	svc.httpClient = &http.Client{
		Transport: &rewriteTransport{
			base:   http.DefaultTransport,
			target: targetURL,
		},
	}

	return svc
}

func TestTurnstileVerify_HostnameAllowed(t *testing.T) {
	svc := newTurnstileTestService(t, `{"success":true,"hostname":"notes.example.com"}`)
	svc.SetAllowedOrigins([]string{"https://notes.example.com"})

	if err := svc.Verify(context.Background(), "token", "203.0.113.1"); err != nil {
		t.Fatalf("expected verification success, got error: %v", err)
	}
}

func TestTurnstileVerify_HostnameRejected(t *testing.T) {
	svc := newTurnstileTestService(t, `{"success":true,"hostname":"evil.example.com"}`)
	svc.SetAllowedOrigins([]string{"https://notes.example.com"})

	if err := svc.Verify(context.Background(), "token", "203.0.113.1"); err == nil {
		t.Fatal("expected verification to fail for disallowed hostname")
	}
}

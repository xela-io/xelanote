package llm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func newTestClient(rt roundTripFunc) *http.Client {
	return &http.Client{Transport: rt}
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestDoJSONRequest_DecodesResponseAndSetsHeaders(t *testing.T) {
	t.Parallel()

	var gotHeader string
	var gotMethod string
	var gotURL string
	var gotPayload map[string]string

	client := newTestClient(func(r *http.Request) (*http.Response, error) {
		gotHeader = r.Header.Get("X-Test")
		gotMethod = r.Method
		gotURL = r.URL.String()
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		return jsonResponse(http.StatusOK, `{"ok":true}`), nil
	})

	var resp struct {
		Ok bool `json:"ok"`
	}
	err := doJSONRequest(context.Background(), client, "POST", "https://example.com/test", map[string]string{
		"X-Test": "value",
	}, map[string]string{"hello": "world"}, &resp, nil, "test")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gotHeader != "value" {
		t.Fatalf("expected header value, got %q", gotHeader)
	}
	if gotMethod != "POST" {
		t.Fatalf("expected POST, got %q", gotMethod)
	}
	if gotURL != "https://example.com/test" {
		t.Fatalf("expected URL, got %q", gotURL)
	}
	if gotPayload["hello"] != "world" {
		t.Fatalf("expected payload, got %v", gotPayload)
	}
	if !resp.Ok {
		t.Fatalf("expected response ok=true")
	}
}

func TestDoJSONRequest_UsesParseError(t *testing.T) {
	t.Parallel()

	client := newTestClient(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusUnauthorized, `{"error":"nope"}`), nil
	})

	parseErr := errors.New("parsed error")
	err := doJSONRequest(context.Background(), client, "GET", "https://example.com", nil, nil, nil, func(_ []byte) error {
		return parseErr
	}, "test")
	if !errors.Is(err, parseErr) {
		t.Fatalf("expected parse error, got %v", err)
	}
}

func TestDoJSONRequest_FallsBackToStatusError(t *testing.T) {
	t.Parallel()

	client := newTestClient(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadRequest, `{"error":"bad"}`), nil
	})

	err := doJSONRequest(context.Background(), client, "GET", "https://example.com", nil, nil, nil, func(_ []byte) error {
		return nil
	}, "provider")
	if err == nil || !strings.Contains(err.Error(), "provider returned status 400") {
		t.Fatalf("expected status error, got %v", err)
	}
}

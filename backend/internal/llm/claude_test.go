package llm

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

func TestClaudeGenerate_EmptyPromptReturnsEmpty(t *testing.T) {
	t.Parallel()

	c := &ClaudeClient{
		apiKey: "key",
		model:  ClaudeDefaultModel,
		httpClient: newTestClient(func(r *http.Request) (*http.Response, error) {
			t.Fatalf("unexpected http call")
			return nil, errors.New("unexpected")
		}),
		maxTokens: ClaudeDefaultMaxTokens,
	}

	out, err := c.Generate(context.Background(), "", 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out != "" {
		t.Fatalf("expected empty output, got %q", out)
	}
}

func TestClaudeGenerate_ReturnsConcatenatedText(t *testing.T) {
	t.Parallel()

	client := newTestClient(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"content":[{"type":"text","text":"Hello"},{"type":"text","text":" world"}]}`), nil
	})
	c := &ClaudeClient{
		apiKey:     "key",
		model:      ClaudeDefaultModel,
		httpClient: client,
		maxTokens:  ClaudeDefaultMaxTokens,
	}

	out, err := c.Generate(context.Background(), "prompt", 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out != "Hello world" {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestClaudeGenerateWithImage_EmptyImageFails(t *testing.T) {
	t.Parallel()

	c := &ClaudeClient{apiKey: "key", model: ClaudeDefaultModel, httpClient: newTestClient(func(r *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected http call")
		return nil, errors.New("unexpected")
	})}
	_, err := c.GenerateWithImage(context.Background(), "prompt", nil, "image/png", 10)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestClaudeGenerate_ParsesErrorResponse(t *testing.T) {
	t.Parallel()

	client := newTestClient(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadRequest, `{"error":{"type":"bad_request","message":"nope"}}`), nil
	})
	c := &ClaudeClient{
		apiKey:     "key",
		model:      ClaudeDefaultModel,
		httpClient: client,
		maxTokens:  ClaudeDefaultMaxTokens,
	}

	_, err := c.Generate(context.Background(), "prompt", 10)
	if err == nil || !strings.Contains(err.Error(), "claude API error") {
		t.Fatalf("expected claude API error, got %v", err)
	}
}

func TestClaudeIsAvailable_Caches(t *testing.T) {
	t.Parallel()

	var calls int32
	client := newTestClient(func(r *http.Request) (*http.Response, error) {
		atomic.AddInt32(&calls, 1)
		return jsonResponse(http.StatusOK, `{"content":[{"type":"text","text":"ok"}]}`), nil
	})
	c := &ClaudeClient{
		apiKey:     "key",
		model:      ClaudeDefaultModel,
		httpClient: client,
		maxTokens:  ClaudeDefaultMaxTokens,
	}

	if !c.IsAvailable(context.Background()) {
		t.Fatalf("expected available")
	}
	if !c.IsAvailable(context.Background()) {
		t.Fatalf("expected cached available")
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

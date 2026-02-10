package llm

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

func TestGeminiGenerate_CleansCodeBlock(t *testing.T) {
	t.Parallel()

	client := newTestClient(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, "{\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"```json\\n{\\\"a\\\":1}\\n```\"}]}}]}"), nil
	})
	c := &GeminiClient{
		apiKey:     "key",
		model:      GeminiDefaultModel,
		httpClient: client,
		maxTokens:  GeminiDefaultMaxTokens,
	}

	out, err := c.Generate(context.Background(), "prompt", 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out != `{"a":1}` {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestGeminiGenerate_NoCandidates(t *testing.T) {
	t.Parallel()

	client := newTestClient(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"candidates":[]}`), nil
	})
	c := &GeminiClient{
		apiKey:     "key",
		model:      GeminiDefaultModel,
		httpClient: client,
		maxTokens:  GeminiDefaultMaxTokens,
	}

	_, err := c.Generate(context.Background(), "prompt", 0)
	if err == nil || !strings.Contains(err.Error(), "no candidates") {
		t.Fatalf("expected no candidates error, got %v", err)
	}
}

func TestGeminiGenerateWithImage_EmptyImageFails(t *testing.T) {
	t.Parallel()

	c := &GeminiClient{apiKey: "key", model: GeminiDefaultModel, httpClient: newTestClient(func(r *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected http call")
		return nil, errors.New("unexpected")
	})}
	_, err := c.GenerateWithImage(context.Background(), "prompt", nil, "image/png", 10)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestGeminiGenerate_ParsesErrorResponse(t *testing.T) {
	t.Parallel()

	client := newTestClient(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadRequest, `{"error":{"status":"INVALID_ARGUMENT","message":"nope"}}`), nil
	})
	c := &GeminiClient{
		apiKey:     "key",
		model:      GeminiDefaultModel,
		httpClient: client,
		maxTokens:  GeminiDefaultMaxTokens,
	}

	_, err := c.Generate(context.Background(), "prompt", 10)
	if err == nil || !strings.Contains(err.Error(), "gemini API error") {
		t.Fatalf("expected gemini API error, got %v", err)
	}
}

func TestGeminiIsAvailable_Caches(t *testing.T) {
	t.Parallel()

	var calls int32
	client := newTestClient(func(r *http.Request) (*http.Response, error) {
		atomic.AddInt32(&calls, 1)
		return jsonResponse(http.StatusOK, `{}`), nil
	})
	c := &GeminiClient{
		apiKey:     "key",
		model:      GeminiDefaultModel,
		httpClient: client,
		maxTokens:  GeminiDefaultMaxTokens,
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

package llm

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

func TestChatGPTGenerate_CleansCodeBlock(t *testing.T) {
	t.Parallel()

	client := newTestClient(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, "{\"choices\":[{\"message\":{\"content\":\"```json\\n{\\\"a\\\":1}\\n```\"}}]}"), nil
	})
	c := &ChatGPTClient{
		apiKey:     "key",
		model:      ChatGPTDefaultModel,
		httpClient: client,
		maxTokens:  ChatGPTDefaultMaxTokens,
	}

	out, err := c.Generate(context.Background(), "prompt", 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out != `{"a":1}` {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestChatGPTGenerate_NoChoices(t *testing.T) {
	t.Parallel()

	client := newTestClient(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"choices":[]}`), nil
	})
	c := &ChatGPTClient{
		apiKey:     "key",
		model:      ChatGPTDefaultModel,
		httpClient: client,
		maxTokens:  ChatGPTDefaultMaxTokens,
	}

	_, err := c.Generate(context.Background(), "prompt", 0)
	if err == nil || !strings.Contains(err.Error(), "no choices") {
		t.Fatalf("expected no choices error, got %v", err)
	}
}

func TestChatGPTGenerateWithImage_EmptyImageFails(t *testing.T) {
	t.Parallel()

	c := &ChatGPTClient{apiKey: "key", model: ChatGPTDefaultModel, httpClient: newTestClient(func(r *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected http call")
		return nil, errors.New("unexpected")
	})}
	_, err := c.GenerateWithImage(context.Background(), "prompt", nil, "image/png", 10)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestChatGPTGenerate_ParsesErrorResponse(t *testing.T) {
	t.Parallel()

	client := newTestClient(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadRequest, `{"error":{"type":"invalid_request_error","message":"nope"}}`), nil
	})
	c := &ChatGPTClient{
		apiKey:     "key",
		model:      ChatGPTDefaultModel,
		httpClient: client,
		maxTokens:  ChatGPTDefaultMaxTokens,
	}

	_, err := c.Generate(context.Background(), "prompt", 10)
	if err == nil || !strings.Contains(err.Error(), "openai API error") {
		t.Fatalf("expected openai API error, got %v", err)
	}
}

func TestChatGPTIsAvailable_Caches(t *testing.T) {
	t.Parallel()

	var calls int32
	client := newTestClient(func(r *http.Request) (*http.Response, error) {
		atomic.AddInt32(&calls, 1)
		return jsonResponse(http.StatusOK, `{"choices":[{"message":{"content":"ok"}}]}`), nil
	})
	c := &ChatGPTClient{
		apiKey:     "key",
		model:      ChatGPTDefaultModel,
		httpClient: client,
		maxTokens:  ChatGPTDefaultMaxTokens,
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

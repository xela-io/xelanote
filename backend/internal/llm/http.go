package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	// maxErrorResponseSize limits how much of an error response body we read (1 MB).
	maxErrorResponseSize = 1 << 20
	// maxResponseSize limits the size of a successful JSON response body (4 MB).
	maxResponseSize = 4 << 20
)

func doJSONRequest(ctx context.Context, client *http.Client, method, url string, headers map[string]string, reqBody any, respBody any, parseError func([]byte) error, provider string) error {
	var body io.Reader
	if reqBody != nil {
		payload, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	for key, value := range headers {
		httpReq.Header.Set(key, value)
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorResponseSize))
		if err != nil {
			return fmt.Errorf("failed to read error response: %w", err)
		}
		if parseError != nil {
			if parsed := parseError(bodyBytes); parsed != nil {
				return parsed
			}
		}
		return fmt.Errorf("%s returned status %d: %s", provider, resp.StatusCode, string(bodyBytes))
	}

	if respBody == nil {
		return nil
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseSize)).Decode(respBody); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	return nil
}

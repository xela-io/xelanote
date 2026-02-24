package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// ForgejoClient handles HTTP communication with the Forgejo API.
type ForgejoClient struct {
	baseURL    string
	repoOwner  string
	repoName   string
	apiToken   string
	httpClient *http.Client
	log        *slog.Logger
}

// NewForgejoClient creates a new ForgejoClient.
func NewForgejoClient(baseURL, repoOwner, repoName, apiToken string, log *slog.Logger) *ForgejoClient {
	return &ForgejoClient{
		baseURL:   baseURL,
		repoOwner: repoOwner,
		repoName:  repoName,
		apiToken:  apiToken,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		log: log,
	}
}

func (c *ForgejoClient) repoAPIURL(path string) string {
	return fmt.Sprintf("%s/api/v1/repos/%s/%s%s", c.baseURL, c.repoOwner, c.repoName, path)
}

func (c *ForgejoClient) doJSON(ctx context.Context, method, apiURL string, body any) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, apiURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "token "+c.apiToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	return c.httpClient.Do(req)
}

// FetchIssues fetches issues from Forgejo matching the given query parameters.
func (c *ForgejoClient) FetchIssues(ctx context.Context, apiURL string) ([]forgejoIssue, error) {
	resp, err := c.doJSON(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch issues: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		c.log.Error("forgejo auth error during issue search", "status", resp.StatusCode)
		return nil, fmt.Errorf("forgejo auth error: %d", resp.StatusCode)
	}
	if resp.StatusCode == http.StatusNotFound {
		c.log.Error("forgejo repo not found", "repo", c.repoOwner+"/"+c.repoName)
		return nil, fmt.Errorf("forgejo repo not found: 404")
	}
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("forgejo server error: %d", resp.StatusCode)
	}

	var issues []forgejoIssue
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return nil, fmt.Errorf("decode issues response: %w", err)
	}
	return issues, nil
}

// CreateIssue creates a new issue on Forgejo.
func (c *ForgejoClient) CreateIssue(ctx context.Context, title, body string, labelIDs []int64) error {
	payload := map[string]interface{}{
		"title":  title,
		"body":   body,
		"labels": labelIDs,
	}

	apiURL := c.repoAPIURL("/issues")
	resp, err := c.doJSON(ctx, http.MethodPost, apiURL, payload)
	if err != nil {
		return fmt.Errorf("create issue: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("forgejo auth error: %d", resp.StatusCode)
	}
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusUnprocessableEntity {
		respBody, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("forgejo create issue error: %d (read body: %w)", resp.StatusCode, readErr)
		}
		return fmt.Errorf("forgejo create issue error: %d: %s", resp.StatusCode, string(respBody))
	}
	if resp.StatusCode >= 500 {
		return fmt.Errorf("forgejo server error: %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status creating issue: %d", resp.StatusCode)
	}

	return nil
}

// AddComment adds a comment to an existing Forgejo issue.
func (c *ForgejoClient) AddComment(ctx context.Context, issueNumber int64, body string) error {
	payload := map[string]string{"body": body}

	apiURL := fmt.Sprintf("%s/issues/%d/comments", c.repoAPIURL(""), issueNumber)
	resp, err := c.doJSON(ctx, http.MethodPost, apiURL, payload)
	if err != nil {
		return fmt.Errorf("add comment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("forgejo auth error: %d", resp.StatusCode)
	}
	if resp.StatusCode >= 500 {
		return fmt.Errorf("forgejo server error: %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status adding comment: %d", resp.StatusCode)
	}

	return nil
}

// FetchLabels retrieves all labels from the repository.
func (c *ForgejoClient) FetchLabels(ctx context.Context) ([]forgejoLabel, error) {
	apiURL := c.repoAPIURL("/labels?page=1&limit=50")
	resp, err := c.doJSON(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch labels: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch labels: status %d", resp.StatusCode)
	}

	var labels []forgejoLabel
	if err := json.NewDecoder(resp.Body).Decode(&labels); err != nil {
		return nil, fmt.Errorf("decode labels: %w", err)
	}
	return labels, nil
}

// CreateLabel creates a new label on Forgejo.
func (c *ForgejoClient) CreateLabel(ctx context.Context, name, color string) (*forgejoLabel, error) {
	payload := map[string]string{
		"name":  name,
		"color": color,
	}

	apiURL := c.repoAPIURL("/labels")
	resp, err := c.doJSON(ctx, http.MethodPost, apiURL, payload)
	if err != nil {
		return nil, fmt.Errorf("create label: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnprocessableEntity {
		return nil, nil // Label might already exist (concurrent creation)
	}
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("create label: status %d", resp.StatusCode)
	}

	var label forgejoLabel
	if err := json.NewDecoder(resp.Body).Decode(&label); err != nil {
		return nil, fmt.Errorf("decode created label: %w", err)
	}
	return &label, nil
}

// SearchIssuesByLabel searches for open issues with a specific label.
func (c *ForgejoClient) SearchIssuesByLabel(ctx context.Context, labelName string) ([]forgejoIssue, error) {
	apiURL := fmt.Sprintf("%s?state=open&labels=%s&page=1&limit=1",
		c.repoAPIURL("/issues"), url.QueryEscape(labelName))
	return c.FetchIssues(ctx, apiURL)
}

// SearchIssuesByQuery searches for open issues matching a query string.
func (c *ForgejoClient) SearchIssuesByQuery(ctx context.Context, query string) ([]forgejoIssue, error) {
	apiURL := fmt.Sprintf("%s?state=open&q=%s&page=1&limit=1",
		c.repoAPIURL("/issues"), url.QueryEscape(query))
	return c.FetchIssues(ctx, apiURL)
}

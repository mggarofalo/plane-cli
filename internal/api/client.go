package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client is the Plane API HTTP client.
type Client struct {
	BaseURL    string
	Workspace  string
	HTTPClient *http.Client
	Verbose    bool
}

// NewClient creates a new API client.
func NewClient(baseURL, token, workspace string, verbose bool, debugWriter io.Writer) *Client {
	baseURL = strings.TrimRight(baseURL, "/")

	transport := &AuthTransport{
		Token:   token,
		Verbose: verbose,
		Writer:  debugWriter,
	}

	return &Client{
		BaseURL:   baseURL,
		Workspace: workspace,
		HTTPClient: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
		Verbose: verbose,
	}
}

// URL builds a full API URL from path segments.
// Example: client.URL("v1", "workspaces", slug, "projects")
func (c *Client) URL(segments ...string) string {
	path := strings.Join(segments, "/")
	return fmt.Sprintf("%s/api/%s/", c.BaseURL, path)
}

// WorkspaceURL builds a URL scoped to the current workspace.
// Example: client.WorkspaceURL("v1", "workspaces", "projects")
// becomes: {baseURL}/api/v1/workspaces/{workspace}/projects/
func (c *Client) WorkspaceURL(segments ...string) string {
	// Insert workspace into the path after "workspaces"
	parts := []string{c.BaseURL, "api"}
	parts = append(parts, segments...)
	return strings.Join(parts, "/") + "/"
}

// Get performs a GET request and returns the response body.
func (c *Client) Get(ctx context.Context, url string) ([]byte, error) {
	return c.do(ctx, http.MethodGet, url, nil)
}

// Post performs a POST request with a JSON body.
func (c *Client) Post(ctx context.Context, url string, body any) ([]byte, error) {
	return c.do(ctx, http.MethodPost, url, body)
}

// Patch performs a PATCH request with a JSON body.
func (c *Client) Patch(ctx context.Context, url string, body any) ([]byte, error) {
	return c.do(ctx, http.MethodPatch, url, body)
}

// Put performs a PUT request with a JSON body.
func (c *Client) Put(ctx context.Context, url string, body any) ([]byte, error) {
	return c.do(ctx, http.MethodPut, url, body)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, reqURL string) error {
	_, err := c.do(ctx, http.MethodDelete, reqURL, nil)
	return err
}

// GetPaginated performs a GET with pagination parameters.
func (c *Client) GetPaginated(ctx context.Context, rawURL string, params PaginationParams) ([]byte, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	params.Apply(u)
	return c.Get(ctx, u.String())
}

func (c *Client) do(ctx context.Context, method, reqURL string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       string(respBody),
			URL:        reqURL,
		}
	}

	// 204 No Content
	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	return respBody, nil
}

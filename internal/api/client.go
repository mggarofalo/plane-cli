package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// MaxBackoff is the upper bound for exponential backoff wait time.
const MaxBackoff = 60 * time.Second

// RetryConfig controls automatic retry behaviour for rate-limited requests.
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts (0 = no retry).
	MaxRetries int
	// Quiet suppresses retry log messages to stderr when true.
	Quiet bool
	// LogWriter is the destination for retry log messages (typically os.Stderr).
	LogWriter io.Writer
}

// Client is the Plane API HTTP client.
type Client struct {
	BaseURL    string
	Workspace  string
	HTTPClient *http.Client
	Verbose    bool
	Retry      RetryConfig

	// sleepFn is used to wait during retry backoff. Defaults to time.Sleep.
	// Replaced in tests for deterministic behaviour.
	sleepFn func(time.Duration)
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
		sleepFn: time.Sleep,
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
	// Marshal the body once so it can be replayed on retries.
	var bodyData []byte
	if body != nil {
		var err error
		bodyData, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
	}

	maxAttempts := 1 + c.Retry.MaxRetries
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	sleep := c.sleepFn
	if sleep == nil {
		sleep = time.Sleep
	}

	var lastErr error
	for attempt := range maxAttempts {
		var bodyReader io.Reader
		if bodyData != nil {
			bodyReader = bytes.NewReader(bodyData)
		}

		req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("executing request: %w", err)
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("reading response: %w", err)
		}

		// Not rate-limited — return normally.
		if resp.StatusCode != http.StatusTooManyRequests {
			if resp.StatusCode >= 400 {
				return nil, &APIError{
					StatusCode: resp.StatusCode,
					Status:     resp.Status,
					Body:       string(respBody),
					URL:        reqURL,
				}
			}
			if resp.StatusCode == http.StatusNoContent {
				return nil, nil
			}
			return respBody, nil
		}

		// 429 Too Many Requests — retry if attempts remain.
		lastErr = &APIError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       string(respBody),
			URL:        reqURL,
		}

		if attempt >= maxAttempts-1 {
			// No more retries; fall through to return lastErr.
			break
		}

		wait := retryDelay(resp.Header, attempt)

		if !c.Retry.Quiet && c.Retry.LogWriter != nil {
			fmt.Fprintf(c.Retry.LogWriter,
				"Rate limited (429). Retry %d/%d in %s...\n",
				attempt+1, c.Retry.MaxRetries, wait.Round(time.Millisecond))
		}

		// Check for context cancellation before sleeping.
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		sleep(wait)

		// Check again after sleeping in case context was cancelled.
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}

	return nil, lastErr
}

// retryDelay determines how long to wait before the next retry attempt.
// It uses the Retry-After header if present; otherwise falls back to
// exponential backoff: min(2^attempt seconds, MaxBackoff).
func retryDelay(header http.Header, attempt int) time.Duration {
	if ra := header.Get("Retry-After"); ra != "" {
		// Try parsing as seconds (integer).
		if secs, err := strconv.Atoi(ra); err == nil && secs > 0 {
			d := time.Duration(secs) * time.Second
			if d > MaxBackoff {
				d = MaxBackoff
			}
			return d
		}
		// Try parsing as HTTP-date.
		if t, err := http.ParseTime(ra); err == nil {
			d := time.Until(t)
			if d < 0 {
				d = 0
			}
			if d > MaxBackoff {
				d = MaxBackoff
			}
			return d
		}
	}
	// Exponential backoff: 1s, 2s, 4s, 8s, ... capped at MaxBackoff.
	d := time.Duration(math.Pow(2, float64(attempt))) * time.Second
	if d > MaxBackoff {
		d = MaxBackoff
	}
	return d
}

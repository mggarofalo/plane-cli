package api

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// SessionTransport injects session cookies for internal API endpoints.
type SessionTransport struct {
	SessionCookie string
	Base          http.RoundTripper
	Verbose       bool
	Writer        io.Writer
}

func (t *SessionTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Cookie", "session_id="+t.SessionCookie)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if t.Verbose && t.Writer != nil {
		fmt.Fprintf(t.Writer, "> %s %s\n", req.Method, req.URL.String())
		for key, vals := range req.Header {
			for _, val := range vals {
				printVal := val
				if strings.EqualFold(key, "cookie") {
					printVal = "REDACTED"
				}
				fmt.Fprintf(t.Writer, "> %s: %s\n", key, printVal)
			}
		}
	}

	start := time.Now()
	resp, err := t.base().RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if t.Verbose && t.Writer != nil {
		fmt.Fprintf(t.Writer, "< %s %s (%s)\n", resp.Status, req.URL.Path, time.Since(start).Round(time.Millisecond))
	}

	return resp, nil
}

func (t *SessionTransport) base() http.RoundTripper {
	if t.Base != nil {
		return t.Base
	}
	return http.DefaultTransport
}

// NewSessionClient creates an API client that uses session cookie authentication.
func NewSessionClient(baseURL, sessionCookie, workspace string, verbose bool, debugWriter io.Writer) *Client {
	baseURL = strings.TrimRight(baseURL, "/")

	transport := &SessionTransport{
		SessionCookie: sessionCookie,
		Verbose:       verbose,
		Writer:        debugWriter,
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

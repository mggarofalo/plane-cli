package api

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AuthTransport injects the API token into every request.
type AuthTransport struct {
	Token   string
	Base    http.RoundTripper
	Verbose bool
	Writer  io.Writer // for debug output (stderr)
}

func (t *AuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("X-API-Key", t.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if t.Verbose && t.Writer != nil {
		sanitized := sanitizeURL(req.URL.String(), t.Token)
		fmt.Fprintf(t.Writer, "> %s %s\n", req.Method, sanitized)
		for key, vals := range req.Header {
			for _, val := range vals {
				fmt.Fprintf(t.Writer, "> %s: %s\n", key, sanitizeValue(key, val, t.Token))
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

func (t *AuthTransport) base() http.RoundTripper {
	if t.Base != nil {
		return t.Base
	}
	return http.DefaultTransport
}

// sanitizeURL redacts any token that might appear in the URL.
func sanitizeURL(url, token string) string {
	if token != "" {
		return strings.ReplaceAll(url, token, "REDACTED")
	}
	return url
}

// sanitizeValue redacts authorization header values.
func sanitizeValue(header, value, token string) string {
	lower := strings.ToLower(header)
	if lower == "x-api-key" {
		return "REDACTED"
	}
	if token != "" {
		return strings.ReplaceAll(value, token, "REDACTED")
	}
	return value
}

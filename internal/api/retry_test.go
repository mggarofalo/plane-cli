package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRetryOn429_NoRetryByDefault(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, "rate limited")
	}))
	defer srv.Close()

	c := &Client{
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
		sleepFn:    func(time.Duration) {},
	}

	_, err := c.Get(context.Background(), srv.URL+"/test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 429 {
		t.Errorf("expected status 429, got %d", apiErr.StatusCode)
	}
	if calls != 1 {
		t.Errorf("expected 1 call with no retries, got %d", calls)
	}
}

func TestRetryOn429_RetriesAndSucceeds(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprint(w, "rate limited")
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	var logBuf bytes.Buffer
	c := &Client{
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
		Retry: RetryConfig{
			MaxRetries: 3,
			LogWriter:  &logBuf,
		},
		sleepFn: func(time.Duration) {},
	}

	body, err := c.Get(context.Background(), srv.URL+"/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Errorf("unexpected body: %s", body)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls (1 original + 2 retries), got %d", calls)
	}

	// Verify log messages.
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "Retry 1/3") {
		t.Errorf("expected retry 1/3 log message, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "Retry 2/3") {
		t.Errorf("expected retry 2/3 log message, got: %s", logOutput)
	}
}

func TestRetryOn429_ExhaustsRetries(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, "rate limited")
	}))
	defer srv.Close()

	c := &Client{
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
		Retry: RetryConfig{
			MaxRetries: 2,
			Quiet:      true,
		},
		sleepFn: func(time.Duration) {},
	}

	_, err := c.Get(context.Background(), srv.URL+"/test")
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 429 {
		t.Errorf("expected status 429, got %d", apiErr.StatusCode)
	}
	// 1 original + 2 retries = 3 total calls.
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestRetryOn429_QuietSuppressesLogs(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprint(w, "rate limited")
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	var logBuf bytes.Buffer
	c := &Client{
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
		Retry: RetryConfig{
			MaxRetries: 1,
			Quiet:      true,
			LogWriter:  &logBuf,
		},
		sleepFn: func(time.Duration) {},
	}

	_, err := c.Get(context.Background(), srv.URL+"/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if logBuf.Len() != 0 {
		t.Errorf("expected no log output with --quiet, got: %s", logBuf.String())
	}
}

func TestRetryOn429_RetryAfterHeaderSeconds(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("Retry-After", "5")
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprint(w, "rate limited")
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	var logBuf bytes.Buffer
	c := &Client{
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
		Retry: RetryConfig{
			MaxRetries: 1,
			LogWriter:  &logBuf,
		},
		sleepFn: func(time.Duration) {},
	}

	_, err := c.Get(context.Background(), srv.URL+"/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check the log message contains 5s.
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "5s") {
		t.Errorf("expected log to mention 5s delay, got: %s", logOutput)
	}
}

func TestRetryOn429_Non429ErrorsAreNotRetried(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "server error")
	}))
	defer srv.Close()

	c := &Client{
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
		Retry: RetryConfig{
			MaxRetries: 3,
		},
		sleepFn: func(time.Duration) {},
	}

	_, err := c.Get(context.Background(), srv.URL+"/test")
	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", apiErr.StatusCode)
	}
	if calls != 1 {
		t.Errorf("expected 1 call (no retries for 500), got %d", calls)
	}
}

func TestRetryOn429_ContextCancellation(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, "rate limited")
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := &Client{
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
		Retry: RetryConfig{
			MaxRetries: 5,
			Quiet:      true,
		},
		// Cancel the context during the first sleep to simulate user interrupt.
		sleepFn: func(d time.Duration) { cancel() },
	}

	_, err := c.Get(ctx, srv.URL+"/test")
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
	// Only 1 call should happen: the original request gets 429,
	// then sleepFn cancels the context, and the post-sleep check returns.
	if calls != 1 {
		t.Errorf("expected 1 call before cancellation, got %d", calls)
	}
}

func TestRetryOn429_PostRetriesWithBody(t *testing.T) {
	calls := 0
	var lastBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		bodyBytes, _ := io.ReadAll(r.Body)
		lastBody = string(bodyBytes)
		if calls == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprint(w, "rate limited")
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"created":true}`)
	}))
	defer srv.Close()

	c := &Client{
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
		Retry: RetryConfig{
			MaxRetries: 1,
			Quiet:      true,
		},
		sleepFn: func(time.Duration) {},
	}

	payload := map[string]string{"name": "test"}
	body, err := c.Post(context.Background(), srv.URL+"/test", payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != `{"created":true}` {
		t.Errorf("unexpected response: %s", body)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
	// Verify the body was replayed on retry.
	if !strings.Contains(lastBody, `"name":"test"`) {
		t.Errorf("expected body to contain name:test on retry, got: %s", lastBody)
	}
}

func TestRetryOn429_DeleteRetries(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := &Client{
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
		Retry: RetryConfig{
			MaxRetries: 1,
			Quiet:      true,
		},
		sleepFn: func(time.Duration) {},
	}

	err := c.Delete(context.Background(), srv.URL+"/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
}

func TestRetryDelay_ExponentialBackoff(t *testing.T) {
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{0, 1 * time.Second},
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{3, 8 * time.Second},
		{4, 16 * time.Second},
		{5, 32 * time.Second},
		{6, MaxBackoff}, // 64s capped to 60s
		{10, MaxBackoff},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt_%d", tt.attempt), func(t *testing.T) {
			got := retryDelay(http.Header{}, tt.attempt)
			if got != tt.want {
				t.Errorf("retryDelay(attempt=%d) = %v, want %v", tt.attempt, got, tt.want)
			}
		})
	}
}

func TestRetryDelay_RetryAfterHeaderSeconds(t *testing.T) {
	h := http.Header{}
	h.Set("Retry-After", "10")
	got := retryDelay(h, 0)
	if got != 10*time.Second {
		t.Errorf("got %v, want 10s", got)
	}
}

func TestRetryDelay_RetryAfterHeaderCapped(t *testing.T) {
	h := http.Header{}
	h.Set("Retry-After", "120")
	got := retryDelay(h, 0)
	if got != MaxBackoff {
		t.Errorf("got %v, want %v (capped)", got, MaxBackoff)
	}
}

func TestRetryDelay_RetryAfterHeaderDate(t *testing.T) {
	future := time.Now().Add(5 * time.Second)
	h := http.Header{}
	h.Set("Retry-After", future.UTC().Format(http.TimeFormat))
	got := retryDelay(h, 0)
	// Allow some tolerance since time passes between Now() calls.
	if got < 3*time.Second || got > 6*time.Second {
		t.Errorf("got %v, want ~5s", got)
	}
}

func TestRetryDelay_RetryAfterHeaderInvalid(t *testing.T) {
	h := http.Header{}
	h.Set("Retry-After", "not-a-number")
	got := retryDelay(h, 2)
	// Should fall back to exponential: 2^2 = 4s.
	if got != 4*time.Second {
		t.Errorf("got %v, want 4s (fallback to exponential)", got)
	}
}


package api

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestFormatErrorJSON_APIError(t *testing.T) {
	tests := []struct {
		name     string
		err      *APIError
		wantCode int
		wantExit int
		wantURL  string
		wantMsg  string
	}{
		{
			name:     "400 bad request with body",
			err:      &APIError{StatusCode: 400, Status: "400 Bad Request", Body: `{"detail":"invalid field"}`, URL: "https://api.plane.so/api/v1/issues/"},
			wantCode: 400,
			wantExit: ExitValidation,
			wantURL:  "https://api.plane.so/api/v1/issues/",
			wantMsg:  `{"detail":"invalid field"}`,
		},
		{
			name:     "401 unauthorized no body",
			err:      &APIError{StatusCode: 401, Status: "401 Unauthorized", Body: "", URL: "https://api.plane.so/api/v1/me/"},
			wantCode: 401,
			wantExit: ExitAuthError,
			wantURL:  "https://api.plane.so/api/v1/me/",
			wantMsg:  "401 Unauthorized",
		},
		{
			name:     "404 not found",
			err:      &APIError{StatusCode: 404, Status: "404 Not Found", Body: "Not Found", URL: "https://api.plane.so/api/v1/issues/abc/"},
			wantCode: 404,
			wantExit: ExitNotFound,
			wantURL:  "https://api.plane.so/api/v1/issues/abc/",
			wantMsg:  "Not Found",
		},
		{
			name:     "429 rate limited",
			err:      &APIError{StatusCode: 429, Status: "429 Too Many Requests", Body: "rate limited", URL: "https://api.plane.so/api/v1/issues/"},
			wantCode: 429,
			wantExit: ExitRateLimited,
			wantURL:  "https://api.plane.so/api/v1/issues/",
			wantMsg:  "rate limited",
		},
		{
			name:     "500 server error",
			err:      &APIError{StatusCode: 500, Status: "500 Internal Server Error", Body: "", URL: "https://api.plane.so/api/v1/issues/"},
			wantCode: 500,
			wantExit: ExitGeneralError,
			wantURL:  "https://api.plane.so/api/v1/issues/",
			wantMsg:  "500 Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := FormatErrorJSON(tt.err)
			if data == nil {
				t.Fatal("expected non-nil JSON, got nil")
			}

			var env JSONErrorEnvelope
			if err := json.Unmarshal(data, &env); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}

			if !env.Error {
				t.Error("expected error=true")
			}
			if env.Code != tt.wantCode {
				t.Errorf("code: got %d, want %d", env.Code, tt.wantCode)
			}
			if env.ExitCode != tt.wantExit {
				t.Errorf("exit_code: got %d, want %d", env.ExitCode, tt.wantExit)
			}
			if env.URL != tt.wantURL {
				t.Errorf("url: got %q, want %q", env.URL, tt.wantURL)
			}
			if env.Message != tt.wantMsg {
				t.Errorf("message: got %q, want %q", env.Message, tt.wantMsg)
			}
		})
	}
}

func TestFormatErrorJSON_NonAPIError(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantMsg string
	}{
		{
			name:    "auth missing error",
			err:     errors.New("no API URL configured. Run 'plane auth login' or set PLANE_URL"),
			wantMsg: "no API URL configured. Run 'plane auth login' or set PLANE_URL",
		},
		{
			name:    "project required error",
			err:     errors.New("project is required. Use --project flag"),
			wantMsg: "project is required. Use --project flag",
		},
		{
			name:    "workspace required error",
			err:     errors.New("workspace is required"),
			wantMsg: "workspace is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := FormatErrorJSON(tt.err)
			if data == nil {
				t.Fatal("expected non-nil JSON, got nil")
			}

			var env JSONErrorEnvelope
			if err := json.Unmarshal(data, &env); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}

			if !env.Error {
				t.Error("expected error=true")
			}
			if env.Code != 0 {
				t.Errorf("code: got %d, want 0 for non-API error", env.Code)
			}
			if env.ExitCode != ExitGeneralError {
				t.Errorf("exit_code: got %d, want %d", env.ExitCode, ExitGeneralError)
			}
			if env.URL != "" {
				t.Errorf("url: got %q, want empty for non-API error", env.URL)
			}
			if env.Message != tt.wantMsg {
				t.Errorf("message: got %q, want %q", env.Message, tt.wantMsg)
			}
		})
	}
}

func TestFormatErrorJSON_NilError(t *testing.T) {
	data := FormatErrorJSON(nil)
	if data != nil {
		t.Errorf("expected nil for nil error, got %s", data)
	}
}

func TestFormatErrorJSON_ValidJSON(t *testing.T) {
	// Verify the output is valid, compact JSON (no pretty printing)
	err := &APIError{StatusCode: 400, Body: "bad request", URL: "https://example.com/api"}
	data := FormatErrorJSON(err)

	// Should not contain newlines (compact format)
	for _, b := range data {
		if b == '\n' {
			t.Error("expected compact JSON without newlines")
			break
		}
	}

	// Should be valid JSON
	if !json.Valid(data) {
		t.Errorf("output is not valid JSON: %s", data)
	}
}

func TestFormatErrorJSON_URLOmittedWhenEmpty(t *testing.T) {
	// Non-API errors should omit the url field entirely (omitempty)
	err := errors.New("some error")
	data := FormatErrorJSON(err)

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if _, ok := raw["url"]; ok {
		t.Error("url field should be omitted for non-API errors")
	}
}

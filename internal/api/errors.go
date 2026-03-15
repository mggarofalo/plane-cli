package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Exit codes
const (
	ExitSuccess       = 0
	ExitGeneralError  = 1
	ExitAuthError     = 2
	ExitNotFound      = 3
	ExitValidation    = 4
	ExitRateLimited   = 5
)

// APIError represents an error response from the Plane API.
type APIError struct {
	StatusCode int
	Status     string
	Body       string
	URL        string
}

func (e *APIError) Error() string {
	if e.Body != "" {
		return fmt.Sprintf("API error %d (%s): %s", e.StatusCode, e.URL, e.Body)
	}
	return fmt.Sprintf("API error %d (%s): %s", e.StatusCode, e.URL, e.Status)
}

// ExitCode returns the appropriate exit code for this error.
func (e *APIError) ExitCode() int {
	switch e.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return ExitAuthError
	case http.StatusNotFound:
		return ExitNotFound
	case http.StatusUnprocessableEntity, http.StatusBadRequest:
		return ExitValidation
	case http.StatusTooManyRequests:
		return ExitRateLimited
	default:
		return ExitGeneralError
	}
}

// IsNotFound returns true if the error is a 404.
func IsNotFound(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}

// IsUnauthorized returns true if the error is a 401 or 403.
func IsUnauthorized(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden
	}
	return false
}

// ExitCoder is implemented by errors that have a specific exit code.
type ExitCoder interface {
	ExitCode() int
}

// ExitCodeFromError returns the appropriate exit code for any error.
func ExitCodeFromError(err error) int {
	if err == nil {
		return ExitSuccess
	}
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.ExitCode()
	}
	// Allow other error types (e.g., ResolutionError) to specify their exit code.
	if ec, ok := err.(ExitCoder); ok {
		return ec.ExitCode()
	}
	return ExitGeneralError
}

// JSONErrorEnvelope is the structured JSON error format emitted to stderr
// when --output json is active. It provides machine-parseable error details
// so that agents and scripts do not need to regex-parse plain text errors.
type JSONErrorEnvelope struct {
	Error    bool   `json:"error"`
	Code     int    `json:"code"`
	ExitCode int    `json:"exit_code"`
	Message  string `json:"message"`
	URL      string `json:"url,omitempty"`
}

// FormatErrorJSON returns a compact JSON byte slice representing the error.
// For APIError, it includes the HTTP status code and URL.
// For other errors, code is 0 and url is omitted.
func FormatErrorJSON(err error) []byte {
	if err == nil {
		return nil
	}

	env := JSONErrorEnvelope{Error: true}

	if apiErr, ok := err.(*APIError); ok {
		env.Code = apiErr.StatusCode
		env.ExitCode = apiErr.ExitCode()
		env.URL = apiErr.URL
		if apiErr.Body != "" {
			env.Message = apiErr.Body
		} else {
			env.Message = apiErr.Status
		}
	} else if ec, ok := err.(ExitCoder); ok {
		env.Code = 0
		env.ExitCode = ec.ExitCode()
		env.Message = err.Error()
	} else {
		env.Code = 0
		env.ExitCode = ExitGeneralError
		env.Message = err.Error()
	}

	data, _ := json.Marshal(env)
	return data
}

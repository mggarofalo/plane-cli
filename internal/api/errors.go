package api

import (
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

// ExitCodeFromError returns the appropriate exit code for any error.
func ExitCodeFromError(err error) int {
	if err == nil {
		return ExitSuccess
	}
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.ExitCode()
	}
	return ExitGeneralError
}

package auth

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

const (
	EnvAPIKey    = "PLANE_API_KEY"
	EnvURL       = "PLANE_URL"
	EnvWorkspace = "PLANE_WORKSPACE"
	EnvProfile   = "PLANE_PROFILE"
	EnvDocsURL   = "PLANE_DOCS_URL"
)

// EnvSource reads credentials from environment variables.
type EnvSource struct {
	Quiet bool
}

// Resolve returns a Credential from the PLANE_API_KEY env var, or empty if unset.
// Warns on stderr if running in a TTY (interactive) session.
func (e *EnvSource) Resolve() (Credential, error) {
	key := os.Getenv(EnvAPIKey)
	if key == "" {
		return Credential{}, nil
	}

	if err := ValidateTokenFormat(key); err != nil {
		return Credential{}, fmt.Errorf("invalid %s: %w", EnvAPIKey, err)
	}

	// Warn if running interactively — env vars are visible in process listings
	if !e.Quiet && term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Fprintf(os.Stderr, "Warning: using %s environment variable. Consider 'plane auth login' for secure storage.\n", EnvAPIKey)
	}

	return NewCredential(key), nil
}

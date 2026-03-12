package auth

import (
	"fmt"
	"os"
)

// Resolver determines the credential to use, following the resolution chain:
// CLI flag → env var → keyring → error.
type Resolver struct {
	FlagToken string
	Env       *EnvSource
	Store     SecretStore
	Config    *Config
}

// ResolvedCredential contains the credential and its source for status display.
type ResolvedCredential struct {
	Credential Credential
	Source     string // "flag", "env", "keyring"
}

// Resolve returns the first available credential from the resolution chain.
func (r *Resolver) Resolve() (*ResolvedCredential, error) {
	// 1. CLI flag (highest priority)
	if r.FlagToken != "" {
		if err := ValidateTokenFormat(r.FlagToken); err != nil {
			return nil, fmt.Errorf("invalid --api-key: %w", err)
		}
		return &ResolvedCredential{
			Credential: NewCredential(r.FlagToken),
			Source:     "flag",
		}, nil
	}

	// 2. Environment variable
	if r.Env != nil {
		cred, err := r.Env.Resolve()
		if err != nil {
			return nil, err
		}
		if !cred.IsEmpty() {
			return &ResolvedCredential{
				Credential: cred,
				Source:     "env",
			}, nil
		}
	}

	// 3. Keyring (active profile)
	if r.Store != nil && r.Config != nil {
		key := r.Config.ActiveProfile + "/api-key"
		token, err := r.Store.Get(key)
		if err == nil && token != "" {
			return &ResolvedCredential{
				Credential: NewCredential(token),
				Source:     "keyring",
			}, nil
		}
	}

	return nil, fmt.Errorf("no credentials found. Run 'plane auth login' or set %s", EnvAPIKey)
}

// ResolveAPIURL returns the API base URL from flag, env, or config.
func (r *Resolver) ResolveAPIURL(flagURL string) string {
	if flagURL != "" {
		return flagURL
	}
	if v := os.Getenv(EnvURL); v != "" {
		return v
	}
	if r.Config != nil {
		if p := r.Config.ActiveProfileConfig(); p.APIURL != "" {
			return p.APIURL
		}
	}
	return ""
}

// ResolveDocsURL returns the docs base URL from flag, env, profile, or default.
func (r *Resolver) ResolveDocsURL(flagDocsURL string) string {
	if flagDocsURL != "" {
		return flagDocsURL
	}
	if v := os.Getenv(EnvDocsURL); v != "" {
		return v
	}
	if r.Config != nil {
		if p := r.Config.ActiveProfileConfig(); p.DocsURL != "" {
			return p.DocsURL
		}
	}
	return ""
}

// ResolveWorkspace returns the workspace slug from flag, env, or config.
func (r *Resolver) ResolveWorkspace(flagWorkspace string) string {
	if flagWorkspace != "" {
		return flagWorkspace
	}
	if v := os.Getenv(EnvWorkspace); v != "" {
		return v
	}
	if r.Config != nil {
		if p := r.Config.ActiveProfileConfig(); p.Workspace != "" {
			return p.Workspace
		}
	}
	return ""
}

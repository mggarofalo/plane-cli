package auth

import (
	"fmt"
	"strings"
)

// Credential holds an API token as a byte slice so it can be zeroed after use.
type Credential struct {
	token []byte
}

// NewCredential creates a Credential from a string token.
func NewCredential(token string) Credential {
	b := make([]byte, len(token))
	copy(b, token)
	return Credential{token: b}
}

// Token returns the token as a string. The caller should defer cred.Clear().
func (c *Credential) Token() string {
	return string(c.token)
}

// Clear zeroes the token bytes in memory.
func (c *Credential) Clear() {
	for i := range c.token {
		c.token[i] = 0
	}
	c.token = nil
}

// IsEmpty returns true if the credential has no token.
func (c *Credential) IsEmpty() bool {
	return len(c.token) == 0
}

// Masked returns a redacted version of the token for display.
// Example: "plane_api_abc123def456" → "plane_***f456"
func (c *Credential) Masked() string {
	s := string(c.token)
	if len(s) <= 4 {
		return "***"
	}
	suffix := s[len(s)-4:]
	// Try to preserve the prefix before the first underscore-delimited secret part
	if idx := strings.LastIndex(s[:len(s)-4], "_"); idx >= 0 && idx < 10 {
		return s[:idx] + "_***" + suffix
	}
	return "***" + suffix
}

// ValidateTokenFormat checks that the token looks like a Plane API key.
func ValidateTokenFormat(token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("token is empty")
	}
	if len(token) < 10 {
		return fmt.Errorf("token is too short (minimum 10 characters)")
	}
	return nil
}

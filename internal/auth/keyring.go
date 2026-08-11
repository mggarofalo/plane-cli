package auth

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/zalando/go-keyring"
)

const serviceName = "plane-cli"

// SecretStore abstracts credential storage.
type SecretStore interface {
	Get(key string) (string, error)
	Set(key, value string) error
	Delete(key string) error
}

// KeyringStore implements SecretStore using the OS keyring: macOS Keychain,
// Windows Credential Manager, or the freedesktop Secret Service on Linux.
// All three backends are reachable without cgo, so they survive the
// CGO_ENABLED=0 cross-compilation used for release builds.
type KeyringStore struct{}

// NewKeyringStore creates a KeyringStore. The OS keyring is opened lazily on
// each operation, so construction never fails; the error return is retained so
// callers can keep treating the store as best-effort.
func NewKeyringStore() (*KeyringStore, error) {
	return &KeyringStore{}, nil
}

// Get retrieves a secret from the keyring.
func (k *KeyringStore) Get(key string) (string, error) {
	value, err := keyring.Get(serviceName, key)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			noteLegacyKeyring(os.Stderr, key)
			return "", fmt.Errorf("no credential found for %q: %w", key, err)
		}
		return "", fmt.Errorf("failed to get credential: %w", err)
	}
	return value, nil
}

// Set stores a secret in the keyring.
func (k *KeyringStore) Set(key, value string) error {
	if err := keyring.Set(serviceName, key, value); err != nil {
		return fmt.Errorf("failed to store credential: %w; if this machine has no OS keyring, set %s instead", err, EnvAPIKey)
	}
	return nil
}

// Delete removes a secret from the keyring.
func (k *KeyringStore) Delete(key string) error {
	if err := keyring.Delete(serviceName, key); err != nil {
		return fmt.Errorf("failed to delete credential: %w", err)
	}
	return nil
}

var legacyNoticeOnce sync.Once

// legacyKeyringDir is where plane-cli v0.6.1 and earlier stored credentials
// when it fell back to the encrypted file backend. That blob is protected by a
// passphrase we no longer have any way to prompt for, so it cannot be migrated.
func legacyKeyringDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "plane-cli", "keyring")
}

// noteLegacyKeyring tells the user to re-authenticate if a leftover file-backend
// keyring from an older version is on disk. Printed at most once per process, and
// only for the API key — a missing session token is routine and needs no nudge.
// Users on PLANE_API_KEY are already authenticated, so the stale file is moot.
func noteLegacyKeyring(w io.Writer, key string) {
	if !strings.HasSuffix(key, "/api-key") || os.Getenv(EnvAPIKey) != "" {
		return
	}
	legacyNoticeOnce.Do(func() {
		dir := legacyKeyringDir()
		if dir == "" {
			return
		}
		if _, err := os.Stat(dir); err != nil {
			return
		}
		fmt.Fprintf(w, "Note: credentials in %s are from an older version and can no longer be read — run 'plane auth login' to store them in the OS keyring.\n", dir)
	})
}

package auth

import (
	"fmt"

	"github.com/99designs/keyring"
)

const serviceName = "plane-cli"

// SecretStore abstracts credential storage.
type SecretStore interface {
	Get(key string) (string, error)
	Set(key, value string) error
	Delete(key string) error
}

// KeyringStore implements SecretStore using the OS keyring.
type KeyringStore struct {
	ring keyring.Keyring
}

// NewKeyringStore creates a KeyringStore with the given backend preference.
// Pass "" for backend to use the default OS keyring.
func NewKeyringStore(backend string) (*KeyringStore, error) {
	cfg := keyring.Config{
		ServiceName:      serviceName,
		FileDir:          "~/.config/plane-cli/keyring",
		FilePasswordFunc: keyring.TerminalPrompt,
	}

	if backend == "file" {
		cfg.AllowedBackends = []keyring.BackendType{keyring.FileBackend}
	}

	ring, err := keyring.Open(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to open keyring: %w", err)
	}

	return &KeyringStore{ring: ring}, nil
}

// Get retrieves a secret from the keyring.
func (k *KeyringStore) Get(key string) (string, error) {
	item, err := k.ring.Get(key)
	if err != nil {
		if err == keyring.ErrKeyNotFound {
			return "", fmt.Errorf("no credential found for %q: %w", key, err)
		}
		return "", fmt.Errorf("failed to get credential: %w", err)
	}
	return string(item.Data), nil
}

// Set stores a secret in the keyring.
func (k *KeyringStore) Set(key, value string) error {
	err := k.ring.Set(keyring.Item{
		Key:  key,
		Data: []byte(value),
	})
	if err != nil {
		return fmt.Errorf("failed to store credential: %w", err)
	}
	return nil
}

// Delete removes a secret from the keyring.
func (k *KeyringStore) Delete(key string) error {
	err := k.ring.Remove(key)
	if err != nil {
		return fmt.Errorf("failed to delete credential: %w", err)
	}
	return nil
}

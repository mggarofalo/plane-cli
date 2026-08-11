package auth

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestKeyringStoreRoundTrip(t *testing.T) {
	keyring.MockInit()

	store, err := NewKeyringStore()
	if err != nil {
		t.Fatalf("NewKeyringStore: %v", err)
	}

	const key = "default/api-key"
	if err := store.Set(key, "plane_api_secret"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := store.Get(key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "plane_api_secret" {
		t.Fatalf("Get = %q, want %q", got, "plane_api_secret")
	}

	if err := store.Delete(key); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := store.Get(key); !errors.Is(err, keyring.ErrNotFound) {
		t.Fatalf("Get after Delete = %v, want ErrNotFound", err)
	}
}

// The "no credential found" wording distinguishes a miss from a broken keyring;
// cmd/auth and the not-authenticated path rely on that distinction.
func TestKeyringStoreGetNotFoundMessage(t *testing.T) {
	keyring.MockInit()

	store, _ := NewKeyringStore()
	_, err := store.Get("default/api-key")
	if err == nil {
		t.Fatal("Get on empty keyring returned no error")
	}
	if !strings.Contains(err.Error(), `no credential found for "default/api-key"`) {
		t.Fatalf("Get error = %q, want it to mention no credential found", err)
	}
	if !errors.Is(err, keyring.ErrNotFound) {
		t.Fatalf("Get error does not wrap ErrNotFound: %v", err)
	}
}

func TestKeyringStoreSetErrorMentionsEnvFallback(t *testing.T) {
	keyring.MockInitWithError(errors.New("no dbus session"))
	t.Cleanup(keyring.MockInit)

	store, _ := NewKeyringStore()
	err := store.Set("default/api-key", "secret")
	if err == nil {
		t.Fatal("Set with a broken keyring returned no error")
	}
	if !strings.Contains(err.Error(), EnvAPIKey) {
		t.Fatalf("Set error = %q, want it to point at %s", err, EnvAPIKey)
	}
}

func TestNoteLegacyKeyring(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(EnvAPIKey, "")
	t.Cleanup(func() { legacyNoticeOnce = sync.Once{} })

	var buf bytes.Buffer
	legacyNoticeOnce = sync.Once{}
	noteLegacyKeyring(&buf, "default/api-key")
	if buf.Len() != 0 {
		t.Fatalf("notice printed with no legacy directory: %q", buf.String())
	}

	dir := filepath.Join(home, ".config", "plane-cli", "keyring")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}

	// A missing session token is routine and must not nudge.
	legacyNoticeOnce = sync.Once{}
	noteLegacyKeyring(&buf, "default/session-token")
	if buf.Len() != 0 {
		t.Fatalf("notice printed for a session-token miss: %q", buf.String())
	}

	legacyNoticeOnce = sync.Once{}
	noteLegacyKeyring(&buf, "default/api-key")
	if !strings.Contains(buf.String(), "plane auth login") {
		t.Fatalf("notice = %q, want it to suggest re-authenticating", buf.String())
	}

	// Only once per process, so it cannot spam a per-command CLI.
	buf.Reset()
	noteLegacyKeyring(&buf, "default/api-key")
	if buf.Len() != 0 {
		t.Fatalf("notice printed twice: %q", buf.String())
	}
}

// Users on PLANE_API_KEY are already authenticated; the stale file is moot.
func TestNoteLegacyKeyringSilentWithEnvKey(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(EnvAPIKey, "plane_api_00000000000000000000000000000000")
	t.Cleanup(func() { legacyNoticeOnce = sync.Once{} })

	if err := os.MkdirAll(filepath.Join(home, ".config", "plane-cli", "keyring"), 0o700); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	legacyNoticeOnce = sync.Once{}
	noteLegacyKeyring(&buf, "default/api-key")
	if buf.Len() != 0 {
		t.Fatalf("notice printed while %s is set: %q", EnvAPIKey, buf.String())
	}
}

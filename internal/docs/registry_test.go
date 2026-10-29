package docs

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// errTransport fails every request, so Load can never reach the network.
type errTransport struct{}

func (errTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("network disabled in tests")
}

// newHermeticRegistry returns a registry whose cache lives in a per-test
// temp dir and whose HTTP client is the given one (or an always-failing one
// when nil), so Load never touches the user's cache or the live docs site.
func newHermeticRegistry(t *testing.T, baseURL string, client *http.Client) *DocsRegistry {
	t.Helper()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	if client == nil {
		client = &http.Client{Transport: errTransport{}}
	}
	r := NewRegistry("test", baseURL)
	r.Quiet = true
	r.HTTPClient = client
	return r
}

func TestLoad_FallbackUsesCustomBaseURL(t *testing.T) {
	custom := "https://custom.example.com"
	r := newHermeticRegistry(t, custom, nil)

	if err := r.Load(context.Background()); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	topics := r.Topics()
	if len(topics) == 0 {
		t.Fatal("Load() returned no topics")
	}

	for _, topic := range topics {
		for _, entry := range topic.Entries {
			if strings.Contains(entry.URL, DefaultBaseURL) {
				t.Errorf("entry URL still points to DefaultBaseURL: %s", entry.URL)
			}
			if !strings.HasPrefix(entry.URL, custom) {
				t.Errorf("entry URL doesn't start with custom base %q: %s", custom, entry.URL)
			}
		}
	}
}

func TestLoad_FallbackDefaultBaseURL(t *testing.T) {
	// When using DefaultBaseURL and the fetch fails, Load should fall back
	// to the unmodified DefaultTopics.
	r := newHermeticRegistry(t, DefaultBaseURL, nil)

	if err := r.Load(context.Background()); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	topics := r.Topics()
	if len(topics) == 0 {
		t.Fatal("Load() returned no topics")
	}

	for _, topic := range topics {
		for _, entry := range topic.Entries {
			if !strings.HasPrefix(entry.URL, DefaultBaseURL) {
				t.Errorf("entry URL doesn't start with DefaultBaseURL: %s", entry.URL)
			}
		}
	}
}

// remoteIndexFixture mirrors the shape of the live llms.txt, including
// cross-references to pages outside the docs base URL.
const remoteIndexFixture = `# Plane Developer Documentation

> Developer docs.

## API Reference
- [Create Work Item](BASE/api-reference/issue/add-issue): Create a work item

## Related documentation
- [Plane product documentation](https://docs.plane.so/llms.txt): Product docs
- [Plane on GitHub](https://github.com/makeplane/plane): Source code
`

func TestLoad_RemoteIndexKeepsOffBaseEntriesAndCaches(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/llms.txt" {
			http.NotFound(w, req)
			return
		}
		_, _ = w.Write([]byte(strings.ReplaceAll(remoteIndexFixture, "BASE", "http://"+req.Host)))
	}))
	defer srv.Close()

	r := newHermeticRegistry(t, srv.URL, srv.Client())
	if err := r.Load(context.Background()); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	related := r.Lookup("related-documentation")
	if related == nil {
		t.Fatal("remote topic \"related-documentation\" not loaded")
	}
	want := []string{"https://docs.plane.so/llms.txt", "https://github.com/makeplane/plane"}
	if len(related.Entries) != len(want) {
		t.Fatalf("got %d related entries, want %d", len(related.Entries), len(want))
	}
	for i, e := range related.Entries {
		if e.URL != want[i] {
			t.Errorf("related entry %d URL = %q, want %q (off-base links are kept verbatim)", i, e.URL, want[i])
		}
	}

	// A second registry with the network disabled must be served from the
	// cache written by the first Load, not from DefaultTopics.
	cached := NewRegistry("test", srv.URL)
	cached.Quiet = true
	cached.HTTPClient = &http.Client{Transport: errTransport{}}
	if err := cached.Load(context.Background()); err != nil {
		t.Fatalf("cached Load() error: %v", err)
	}
	if cached.Lookup("related-documentation") == nil {
		t.Error("second Load did not use the cached remote index")
	}
}

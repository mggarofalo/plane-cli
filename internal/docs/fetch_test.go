package docs

import (
	"strings"
	"testing"
)

func TestHTMLToMarkdown_ListItemStartsOnItsOwnLine(t *testing.T) {
	// Plane renders a parameter's valid values as a <ul> that can follow an
	// inline element with no block break between them. Without a leading
	// newline on each <li>, the first value is glued onto the preceding line
	// and lost to the parser — which silently dropped the first value of
	// every bulleted enum in the docs (urgent, backlog, DRAFT, -2, ...).
	html := `<code class="param-name">priority</code><span>:</span><span>optional</span><span>string</span>` +
		`<div class="param-description"><ul>` +
		`<li><code>urgent</code> - Urgent</li>` +
		`<li><code>high</code> - High</li>` +
		`</ul></div>`

	got := htmlToMarkdown(html)
	lines := strings.Split(got, "\n")

	var sawParam, sawUrgent bool
	for _, line := range lines {
		switch strings.TrimSpace(line) {
		case "`priority`:optionalstring":
			sawParam = true
		case "- `urgent` - Urgent":
			sawUrgent = true
		}
	}

	if !sawParam {
		t.Errorf("param row is not on its own line.\nGot:\n%s", got)
	}
	if !sawUrgent {
		t.Errorf("first list item is not on its own line.\nGot:\n%s", got)
	}
}

func TestHTMLToMarkdown_CollapsesBlankLines(t *testing.T) {
	// Each <li> now contributes newlines on both sides, so consecutive items
	// would otherwise stack up blank lines.
	html := `<ul><li>one</li><li>two</li><li>three</li></ul>`

	got := htmlToMarkdown(html)

	if strings.Contains(got, "\n\n\n") {
		t.Errorf("expected runs of blank lines to be collapsed, got:\n%q", got)
	}
	for _, want := range []string{"- one", "- two", "- three"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing list item %q in:\n%s", want, got)
		}
	}
}

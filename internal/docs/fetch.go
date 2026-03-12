package docs

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Fetch retrieves a doc page and returns cleaned markdown content.
func Fetch(ctx context.Context, url string) (string, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "text/html")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetching %s: HTTP %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	return htmlToMarkdown(string(body)), nil
}

// htmlToMarkdown does a best-effort conversion of the HTML doc page to readable terminal text.
// This is intentionally simple — we extract the main content and strip HTML tags.
func htmlToMarkdown(html string) string {
	// Try to extract the main content area
	content := html
	if idx := strings.Index(html, `<main`); idx >= 0 {
		content = html[idx:]
		if end := strings.Index(content, `</main>`); end >= 0 {
			content = content[:end]
		}
	} else if idx := strings.Index(html, `class="vp-doc"`); idx >= 0 {
		// VitePress content area
		start := strings.LastIndex(html[:idx], "<")
		if start >= 0 {
			content = html[start:]
		}
	}

	// Convert common HTML elements to markdown
	replacements := []struct {
		pattern *regexp.Regexp
		replace string
	}{
		// Code blocks
		{regexp.MustCompile(`(?s)<pre[^>]*><code[^>]*>(.*?)</code></pre>`), "\n```\n$1\n```\n"},
		// Inline code
		{regexp.MustCompile(`<code[^>]*>(.*?)</code>`), "`$1`"},
		// Headers
		{regexp.MustCompile(`<h1[^>]*>(.*?)</h1>`), "\n# $1\n"},
		{regexp.MustCompile(`<h2[^>]*>(.*?)</h2>`), "\n## $1\n"},
		{regexp.MustCompile(`<h3[^>]*>(.*?)</h3>`), "\n### $1\n"},
		{regexp.MustCompile(`<h4[^>]*>(.*?)</h4>`), "\n#### $1\n"},
		// Table elements
		{regexp.MustCompile(`</tr>`), "\n"},
		{regexp.MustCompile(`<th[^>]*>(.*?)</th>`), "| $1 "},
		{regexp.MustCompile(`<td[^>]*>(.*?)</td>`), "| $1 "},
		// Lists
		{regexp.MustCompile(`<li[^>]*>(.*?)</li>`), "- $1\n"},
		// Links
		{regexp.MustCompile(`<a[^>]*href="([^"]*)"[^>]*>(.*?)</a>`), "$2 ($1)"},
		// Bold / emphasis
		{regexp.MustCompile(`<strong>(.*?)</strong>`), "**$1**"},
		{regexp.MustCompile(`<b>(.*?)</b>`), "**$1**"},
		{regexp.MustCompile(`<em>(.*?)</em>`), "*$1*"},
		// Paragraphs / line breaks
		{regexp.MustCompile(`<br\s*/?>`), "\n"},
		{regexp.MustCompile(`<p[^>]*>`), "\n"},
		{regexp.MustCompile(`</p>`), "\n"},
		// Horizontal rules
		{regexp.MustCompile(`<hr[^>]*/?>`), "\n---\n"},
	}

	for _, r := range replacements {
		content = r.pattern.ReplaceAllString(content, r.replace)
	}

	// Strip remaining HTML tags
	tagRe := regexp.MustCompile(`<[^>]+>`)
	content = tagRe.ReplaceAllString(content, "")

	// Decode common HTML entities
	content = strings.ReplaceAll(content, "&amp;", "&")
	content = strings.ReplaceAll(content, "&lt;", "<")
	content = strings.ReplaceAll(content, "&gt;", ">")
	content = strings.ReplaceAll(content, "&quot;", `"`)
	content = strings.ReplaceAll(content, "&#39;", "'")
	content = strings.ReplaceAll(content, "&nbsp;", " ")

	// Clean up excessive whitespace
	multiNewline := regexp.MustCompile(`\n{3,}`)
	content = multiNewline.ReplaceAllString(content, "\n\n")
	content = strings.TrimSpace(content)

	return content
}

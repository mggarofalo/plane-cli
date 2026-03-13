package markdown

import (
	"bytes"
	"strings"

	htmltomd "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/yuin/goldmark"
)

// ToHTML converts CommonMark markdown to HTML.
func ToHTML(md string) (string, error) {
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(md), &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// FromHTML converts HTML to markdown.
func FromHTML(html string) (string, error) {
	result, err := htmltomd.ConvertString(html)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result), nil
}

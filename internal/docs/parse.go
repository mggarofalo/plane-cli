package docs

import (
	"strings"
)

// ParseLLMSTxt parses an llms.txt file and returns the topics it contains.
// The expected format uses ## headings for sections and markdown links for entries:
//
//	## Section Name
//	- [Title](URL): description
func ParseLLMSTxt(content string) ([]Topic, error) {
	var topics []Topic
	var current *Topic

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)

		// Skip preamble / metadata lines
		if line == "" || strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "> ") ||
			strings.HasPrefix(line, "Docs:") || strings.HasPrefix(line, "Full LLM-friendly content:") {
			continue
		}

		// Section heading
		if strings.HasPrefix(line, "## ") {
			name := strings.TrimPrefix(line, "## ")
			name = strings.ToLower(strings.TrimSpace(name))
			name = strings.ReplaceAll(name, " ", "-")
			topics = append(topics, Topic{Name: name})
			current = &topics[len(topics)-1]
			continue
		}

		// Entry line: - [Title](URL): description
		if current != nil && strings.HasPrefix(line, "- [") {
			entry, ok := parseEntryLine(line)
			if ok {
				current.Entries = append(current.Entries, entry)
			}
		}
	}

	return topics, nil
}

// parseEntryLine parses a line like: - [Title](URL): description
func parseEntryLine(line string) (Entry, bool) {
	// Strip leading "- "
	line = strings.TrimPrefix(line, "- ")

	// Extract [Title]
	if !strings.HasPrefix(line, "[") {
		return Entry{}, false
	}
	closeBracket := strings.Index(line, "](")
	if closeBracket < 0 {
		return Entry{}, false
	}
	title := line[1:closeBracket]

	// Extract (URL)
	rest := line[closeBracket+2:]
	closeParen := strings.Index(rest, ")")
	if closeParen < 0 {
		return Entry{}, false
	}
	url := rest[:closeParen]

	// Extract optional description after ": "
	var desc string
	after := rest[closeParen+1:]
	if strings.HasPrefix(after, ": ") {
		desc = strings.TrimPrefix(after, ": ")
	}

	return Entry{
		Title:       title,
		URL:         url,
		Description: desc,
	}, true
}

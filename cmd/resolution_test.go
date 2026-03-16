package cmd

import (
	"strings"
	"testing"
)

func TestResolutionHelpText_ContainsKeyTopics(t *testing.T) {
	text := ResolutionHelpText

	required := []string{
		"state",
		"label",
		"cycle",
		"module",
		"member",
		"project",
		"Sequence IDs",
		"PROJ-42",
		"--no-resolve",
		"--strict",
		"Soft TTL",
		"Hard TTL",
		"1 hour",
		"7 days",
		"Cache behavior",
	}

	for _, s := range required {
		if !strings.Contains(text, s) {
			t.Errorf("ResolutionHelpText missing required content: %q", s)
		}
	}
}

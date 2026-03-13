package cmdgen

import (
	"strings"
)

// DeriveSubcommandName converts an entry title to a CLI subcommand name.
// Examples:
//
//	"Create Work Item"       → "create"
//	"List Work Items"        → "list"
//	"Get Work Item Detail"   → "get"
//	"Add Cycle Work Items"   → "add-work-items"
//	"List Archived Cycles"   → "list-archived"
//	"Get by Sequence ID"     → "get-by-sequence-id"
//	"Transfer Work Items"    → "transfer-work-items"
//	"Overview"               → "" (skip)
func DeriveSubcommandName(entryTitle, topicName string) string {
	if strings.EqualFold(entryTitle, "Overview") {
		return ""
	}
	if strings.EqualFold(entryTitle, "API Introduction") {
		return ""
	}

	lower := strings.ToLower(entryTitle)
	words := strings.Fields(lower)
	if len(words) == 0 {
		return ""
	}

	// The first word is typically the action verb
	action := words[0]
	rest := words[1:]

	// Remove noise words that just describe the topic (resource name)
	rest = removeTopicWords(rest, topicName)

	if len(rest) == 0 {
		return action
	}

	// Common patterns: strip trailing "detail" since it's noise
	if rest[len(rest)-1] == "detail" || rest[len(rest)-1] == "details" {
		rest = rest[:len(rest)-1]
	}

	if len(rest) == 0 {
		return action
	}

	return action + "-" + strings.Join(rest, "-")
}

// removeTopicWords strips words from the phrase that are just the topic name or
// its plural/singular variants.
func removeTopicWords(words []string, topicName string) []string {
	topicLower := strings.ToLower(topicName)
	topicWords := topicAliases(topicLower)

	var result []string
	for _, w := range words {
		if topicWords[w] {
			continue
		}
		result = append(result, w)
	}
	return result
}

// topicAliases returns a set of words to strip when they just refer to the topic.
func topicAliases(topicName string) map[string]bool {
	aliases := map[string]bool{
		topicName:      true,
		topicName + "s": true,
	}

	// Handle known irregular plurals and name variants
	switch topicName {
	case "issue":
		aliases["issues"] = true
		aliases["work"] = true
		aliases["item"] = true
		aliases["items"] = true
	case "activity":
		aliases["activities"] = true
		aliases["issue-activity"] = true
	case "comment":
		aliases["comments"] = true
		aliases["issue-comment"] = true
	case "sticky":
		aliases["stickies"] = true
	case "cycle":
		aliases["cycles"] = true
	case "module":
		aliases["modules"] = true
	case "page":
		aliases["pages"] = true
	case "label":
		aliases["labels"] = true
	case "state":
		aliases["states"] = true
	case "member":
		aliases["members"] = true
	case "intake":
		aliases["intakes"] = true
		aliases["issue"] = true
		aliases["issues"] = true
	case "link":
		aliases["links"] = true
	case "project":
		aliases["projects"] = true
	case "attachment":
		aliases["attachments"] = true
	case "customer":
		aliases["customers"] = true
	case "teamspace":
		aliases["teamspaces"] = true
	case "epic":
		aliases["epics"] = true
	case "initiative":
		aliases["initiatives"] = true
	case "worklog":
		aliases["worklogs"] = true
	case "user":
		aliases["users"] = true
		aliases["current"] = true
	}

	return aliases
}

// ParamToFlagName converts a parameter name to a CLI flag name.
// Examples: "name" → "name", "state_id" → "state-id", "description_html" → "description-html"
func ParamToFlagName(paramName string) string {
	return strings.ReplaceAll(paramName, "_", "-")
}

// IsHTMLParam returns true if the API parameter name has an _html suffix.
func IsHTMLParam(apiParamName string) bool {
	return strings.HasSuffix(apiParamName, "_html")
}

// MarkdownFlagName returns the markdown flag name for an _html param.
// For example, "description_html" → "description".
func MarkdownFlagName(apiParamName string) string {
	return ParamToFlagName(strings.TrimSuffix(apiParamName, "_html"))
}

// IsAPIReferenceURL returns true if the URL contains /api-reference/.
func IsAPIReferenceURL(url string) bool {
	return strings.Contains(url, "/api-reference/")
}

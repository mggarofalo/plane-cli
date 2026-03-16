package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mggarofalo/plane-cli/internal/docs"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolEntry pairs a built MCP tool with its handler.
type ToolEntry struct {
	Tool    *mcp.Tool
	Handler mcp.ToolHandler
}

// BuildTool converts an EndpointSpec into an MCP tool definition and handler.
// The tool name follows the pattern {topic}_{action} with underscores.
func BuildTool(topicName string, spec *docs.EndpointSpec, cfg *Config) *ToolEntry {
	actionName := deriveActionName(spec.EntryTitle, topicName)
	if actionName == "" {
		return nil
	}

	toolName := topicName + "_" + actionName
	description := fmt.Sprintf("%s %s -- %s", spec.Method, spec.PathTemplate, spec.EntryTitle)

	tool := &mcp.Tool{
		Name:        toolName,
		Description: description,
		InputSchema: BuildInputSchema(spec),
		Annotations: AnnotationsFromMethod(spec.Method),
	}

	handler := makeHandler(spec, cfg)

	return &ToolEntry{
		Tool:    tool,
		Handler: handler,
	}
}

// makeHandler creates a ToolHandler that executes the endpoint spec.
// Tool call errors are returned as CallToolResult with IsError=true,
// never as Go errors (which would crash the server).
func makeHandler(spec *docs.EndpointSpec, cfg *Config) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Parse arguments from raw JSON
		args, err := parseArgs(req.Params.Arguments)
		if err != nil {
			return errorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
		}

		// Execute the endpoint
		respBody, err := ExecuteEndpoint(ctx, spec, args, cfg)
		if err != nil {
			return errorResult(err.Error()), nil
		}

		// Return JSON response as text content
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(respBody)},
			},
		}, nil
	}
}

// parseArgs unmarshals the raw JSON arguments into a map.
func parseArgs(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	var args map[string]any
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	return args, nil
}

// errorResult creates a CallToolResult with IsError=true.
func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
		IsError: true,
	}
}

// deriveActionName converts an entry title to an MCP tool action name.
// Uses underscore-separated words instead of hyphens (MCP convention).
// Examples:
//
//	"Create Work Item" (topic=issue) → "create"
//	"List Work Items" (topic=issue) → "list"
//	"Add Cycle Work Items" (topic=cycle) → "add_work_items"
//	"Get by Sequence ID" (topic=issue) → "get_by_sequence_id"
//	"Overview" → "" (skip)
func deriveActionName(entryTitle, topicName string) string {
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

	action := words[0]
	rest := words[1:]

	// Remove topic name words (same logic as CLI naming)
	rest = removeTopicWords(rest, topicName)

	// Strip trailing "detail"/"details"
	if len(rest) > 0 && (rest[len(rest)-1] == "detail" || rest[len(rest)-1] == "details") {
		rest = rest[:len(rest)-1]
	}

	if len(rest) == 0 {
		return action
	}

	return action + "_" + strings.Join(rest, "_")
}

// removeTopicWords strips words from the phrase that are just the topic name
// or its plural/singular variants. Mirrors cmdgen.removeTopicWords logic.
func removeTopicWords(words []string, topicName string) []string {
	topicLower := strings.ToLower(topicName)
	aliases := topicAliases(topicLower)

	var result []string
	for _, w := range words {
		if aliases[w] {
			continue
		}
		result = append(result, w)
	}
	return result
}

// topicAliases returns a set of words to strip when they refer to the topic.
// Mirrors cmdgen.topicAliases.
func topicAliases(topicName string) map[string]bool {
	aliases := map[string]bool{
		topicName:       true,
		topicName + "s": true,
	}

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

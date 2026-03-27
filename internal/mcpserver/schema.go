package mcpserver

import (
	"encoding/json"

	"github.com/mggarofalo/plane-cli/internal/docs"
)

// jsonSchema is a minimal JSON Schema representation used for MCP tool input
// schemas. We build these manually because the schemas are dynamic (derived
// from endpoint specs at runtime), not inferred from Go structs.
type jsonSchema struct {
	Type        string                `json:"type"`
	Properties  map[string]jsonSchema `json:"properties,omitempty"`
	Required    []string              `json:"required,omitempty"`
	Description string                `json:"description,omitempty"`
	Items       *jsonSchema           `json:"items,omitempty"`
}

// globalParams that are handled by server-level context rather than being
// passed to the API directly. These are excluded from the per-tool schema.
// The Plane API inconsistently names the workspace param: some endpoints use
// "workspace_slug", others use "slug". Both map to the --workspace flag.
var hiddenParams = map[string]bool{
	"workspace_slug": true,
	"slug":           true,
	"project_id":     true,
}

// BuildInputSchema converts ParamSpec[] to a JSON-encodable schema object
// suitable for mcp.Tool.InputSchema. It adds optional workspace and project
// override parameters for endpoints that require them.
func BuildInputSchema(spec *docs.EndpointSpec) json.RawMessage {
	schema := jsonSchema{
		Type:       "object",
		Properties: make(map[string]jsonSchema),
	}

	for _, p := range spec.Params {
		if hiddenParams[p.Name] {
			continue
		}

		name := p.Name
		// Expose _html params as their markdown counterpart for agent ergonomics.
		// Agents produce markdown natively; we convert to HTML before sending.
		if isHTMLParam(name) {
			name = markdownParamName(name)
		}

		prop := paramToSchema(p)
		schema.Properties[name] = prop

		if p.Required {
			schema.Required = append(schema.Required, name)
		}
	}

	// Add workspace/project override params for endpoints that need them.
	if spec.RequiresWorkspace() {
		schema.Properties["workspace"] = jsonSchema{
			Type:        "string",
			Description: "Workspace slug (overrides server default)",
		}
	}
	if spec.RequiresProject() {
		schema.Properties["project"] = jsonSchema{
			Type:        "string",
			Description: "Project ID or identifier (overrides server default)",
		}
	}

	// Add pagination params for GET endpoints.
	if spec.Method == "GET" {
		schema.Properties["page_size"] = jsonSchema{
			Type:        "number",
			Description: "Items per page (max 100)",
		}
		schema.Properties["cursor"] = jsonSchema{
			Type:        "string",
			Description: "Pagination cursor from previous response",
		}
		schema.Properties["all"] = jsonSchema{
			Type:        "boolean",
			Description: "Auto-paginate and return all results",
		}
	}

	data, _ := json.Marshal(schema)
	return json.RawMessage(data)
}

// paramToSchema converts a single ParamSpec to a jsonSchema property.
func paramToSchema(p docs.ParamSpec) jsonSchema {
	s := jsonSchema{
		Description: p.Description,
	}

	switch p.Type {
	case "number":
		s.Type = "number"
	case "boolean":
		s.Type = "boolean"
	case "string[]":
		s.Type = "array"
		s.Items = &jsonSchema{Type: "string"}
	default:
		s.Type = "string"
	}

	return s
}

// isHTMLParam returns true if the API parameter name has an _html suffix.
func isHTMLParam(name string) bool {
	return len(name) > 5 && name[len(name)-5:] == "_html"
}

// markdownParamName returns the markdown-friendly name for an _html param.
// For example, "description_html" becomes "description".
func markdownParamName(name string) string {
	return name[:len(name)-5]
}

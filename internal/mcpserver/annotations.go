package mcpserver

import "github.com/modelcontextprotocol/go-sdk/mcp"

// boolPtr returns a pointer to a bool value. Used for ToolAnnotations fields
// that default to true in the MCP spec (DestructiveHint, OpenWorldHint) and
// therefore must use pointers to distinguish "false" from "unset".
func boolPtr(b bool) *bool {
	return &b
}

// AnnotationsFromMethod infers MCP ToolAnnotations from an HTTP method.
//
// | Method | ReadOnly | Destructive | Idempotent |
// |--------|----------|-------------|------------|
// | GET    | true     | false       | true       |
// | POST   | false    | false       | false      |
// | PUT    | false    | false       | true       |
// | PATCH  | false    | false       | false      |
// | DELETE | false    | true        | true       |
func AnnotationsFromMethod(method string) *mcp.ToolAnnotations {
	switch method {
	case "GET":
		return &mcp.ToolAnnotations{
			ReadOnlyHint:    true,
			DestructiveHint: boolPtr(false),
			IdempotentHint:  true,
			OpenWorldHint:   boolPtr(false),
		}
	case "POST":
		return &mcp.ToolAnnotations{
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		}
	case "PUT":
		return &mcp.ToolAnnotations{
			DestructiveHint: boolPtr(false),
			IdempotentHint:  true,
			OpenWorldHint:   boolPtr(false),
		}
	case "PATCH":
		return &mcp.ToolAnnotations{
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		}
	case "DELETE":
		return &mcp.ToolAnnotations{
			DestructiveHint: boolPtr(true),
			IdempotentHint:  true,
			OpenWorldHint:   boolPtr(false),
		}
	default:
		return &mcp.ToolAnnotations{
			OpenWorldHint: boolPtr(false),
		}
	}
}

// Package weburl constructs Plane web UI URLs from API path templates
// and response data. It maps API paths to their corresponding web routes
// so the CLI can include a clickable web_url in single-resource responses.
package weburl

import (
	"encoding/json"
	"regexp"
	"strings"
)

// pathSegmentRe matches a single path segment in curly-brace template syntax,
// e.g. {workspace_slug} or {project_id}.
var pathSegmentRe = regexp.MustCompile(`\{([^}]+)\}`)

// route maps an API path pattern to a web URL path builder.
// The key is the "resource segment" from the API path (e.g. "work-items",
// "cycles", "modules") and the value describes how to construct the web path.
type route struct {
	// webSegment is the path segment used in the web UI (e.g. "issues" for
	// API segment "work-items").
	webSegment string
	// projectScoped indicates the resource lives under a project.
	projectScoped bool
}

// resourceRoutes maps the final resource segment of the API path to the
// corresponding web URL route. Only resources that have meaningful web UI
// pages are included.
var resourceRoutes = map[string]route{
	"work-items":   {webSegment: "issues", projectScoped: true},
	"cycles":       {webSegment: "cycles", projectScoped: true},
	"modules":      {webSegment: "modules", projectScoped: true},
	"pages":        {webSegment: "pages", projectScoped: true},
	"labels":       {webSegment: "settings/labels", projectScoped: true},
	"states":       {webSegment: "settings/states", projectScoped: true},
	"projects":     {webSegment: "projects", projectScoped: false},
	"initiatives":  {webSegment: "initiatives", projectScoped: false},
}

// Inject adds a "web_url" field to a single-resource JSON response body when:
//   - The response is a single JSON object (not an array or paginated envelope)
//   - The object has an "id" field
//   - The API path template matches a known resource type
//   - The response doesn't already have a "web_url" field
//
// It returns the (possibly modified) response body. If injection is not
// applicable, the original body is returned unchanged.
func Inject(respBody []byte, apiBaseURL, workspace, projectID, pathTemplate string) []byte {
	// Quick guard: empty or array responses are never single-resource
	trimmed := strings.TrimSpace(string(respBody))
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return respBody
	}

	// Parse as a generic map
	var obj map[string]any
	if err := json.Unmarshal(respBody, &obj); err != nil {
		return respBody
	}

	// Skip paginated envelopes (they have a "results" key with an array)
	if _, hasPaginatedResults := obj["results"]; hasPaginatedResults {
		return respBody
	}

	// Skip if web_url already present
	if _, hasWebURL := obj["web_url"]; hasWebURL {
		return respBody
	}

	// Need an id field
	id, ok := obj["id"].(string)
	if !ok || id == "" {
		return respBody
	}

	webURL := Build(apiBaseURL, workspace, projectID, pathTemplate, id)
	if webURL == "" {
		return respBody
	}

	// Inject the web_url
	obj["web_url"] = webURL

	// Re-marshal; on failure, return the original
	out, err := json.Marshal(obj)
	if err != nil {
		return respBody
	}
	return out
}

// Build constructs a Plane web UI URL for a single resource.
// Returns empty string if the path template doesn't match a known resource.
//
// The function works for both detail paths (GET/PATCH with {resource_id}) and
// collection paths (POST to list URL that returns a single created resource).
// As long as we can identify the resource type from the path template and have
// a resource ID from the response, we can construct the web URL.
//
// Parameters:
//   - apiBaseURL: the API base URL (e.g. "https://plane.example.com")
//   - workspace: the workspace slug
//   - projectID: the resolved project UUID (may be empty for non-project resources)
//   - pathTemplate: the API path template (e.g. "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/{work_item_id}/")
//   - resourceID: the resource UUID from the response
func Build(apiBaseURL, workspace, projectID, pathTemplate, resourceID string) string {
	if workspace == "" || resourceID == "" {
		return ""
	}

	// Strip the base URL to get just the web origin
	webBase := toWebBase(apiBaseURL)
	if webBase == "" {
		return ""
	}

	// Find the resource segment from the path template
	seg := resourceSegment(pathTemplate)
	if seg == "" {
		return ""
	}

	r, ok := resourceRoutes[seg]
	if !ok {
		return ""
	}

	if r.projectScoped {
		if projectID == "" {
			return ""
		}
		return webBase + "/" + workspace + "/projects/" + projectID + "/" + r.webSegment + "/" + resourceID
	}

	// Workspace-scoped resources
	if seg == "projects" {
		// Projects link to their issues view
		return webBase + "/" + workspace + "/projects/" + resourceID + "/issues/"
	}
	return webBase + "/" + workspace + "/" + r.webSegment + "/" + resourceID
}

// toWebBase derives the web UI base URL from the API base URL.
// For self-hosted Plane, the API and web UI share the same origin.
// For cloud Plane (api.plane.so), the web UI is at app.plane.so.
func toWebBase(apiBaseURL string) string {
	if apiBaseURL == "" {
		return ""
	}

	// Trim trailing slashes
	base := strings.TrimRight(apiBaseURL, "/")

	// Handle cloud Plane: api.plane.so -> app.plane.so
	if strings.Contains(base, "://api.plane.so") {
		return strings.Replace(base, "://api.plane.so", "://app.plane.so", 1)
	}

	// Self-hosted: same origin
	return base
}

// resourceSegment extracts the resource type segment from an API path template.
// For example, given "/api/v1/workspaces/{ws}/projects/{proj}/work-items/{id}/",
// it returns "work-items".
//
// It looks for the last named segment before a {placeholder} at the end of the path.
func resourceSegment(pathTemplate string) string {
	// Normalize: strip trailing slash, split
	path := strings.TrimRight(pathTemplate, "/")
	parts := strings.Split(path, "/")

	// Walk from the end to find the resource segment.
	// Pattern: .../{resource-type}/{resource_id} or .../{resource-type}/
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		if pathSegmentRe.MatchString(part) {
			// This is a placeholder; the segment before it is the resource type
			continue
		}
		// Skip known non-resource segments
		if part == "api" || part == "v1" || part == "" {
			continue
		}
		return part
	}
	return ""
}

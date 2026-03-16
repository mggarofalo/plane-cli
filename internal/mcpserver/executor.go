package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/mggarofalo/plane-cli/internal/api"
	"github.com/mggarofalo/plane-cli/internal/cmdgen"
	"github.com/mggarofalo/plane-cli/internal/docs"
	"github.com/mggarofalo/plane-cli/internal/markdown"
)

// ExecuteEndpoint runs an API call for the given EndpointSpec using the
// provided arguments and returns the raw JSON response body.
// It handles URL building, body construction, name resolution, pagination,
// and relation attachment (module/cycle) -- mirroring the CLI executor
// but without Cobra dependency.
func ExecuteEndpoint(ctx context.Context, spec *docs.EndpointSpec, args map[string]any, cfg *Config) ([]byte, error) {
	workspace, projectID, err := resolveContext(spec, args, cfg)
	if err != nil {
		return nil, err
	}

	client, err := cfg.NewClient(workspace)
	if err != nil {
		return nil, err
	}

	// Build URL from path template
	reqURL, err := buildURLFromMap(client, spec, args, projectID, cfg)
	if err != nil {
		return nil, err
	}

	// Collect body params
	body, err := collectBodyFromMap(ctx, spec, args, cfg)
	if err != nil {
		return nil, err
	}
	body = cmdgen.InjectGlobalBodyParams(body, spec, workspace, projectID)

	// Extract relation params for POST
	var relations map[string]string
	if spec.Method == "POST" {
		relations = cmdgen.ExtractRelationParams(body)
	}

	// Execute the request
	var respBody []byte
	switch spec.Method {
	case "GET":
		if allVal, ok := args["all"]; ok {
			if b, isBool := allVal.(bool); isBool && b {
				return executeAutoPageinate(ctx, client, reqURL, spec, args)
			}
		}
		respBody, err = client.GetPaginated(ctx, reqURL, paginationFromArgs(args))
	case "POST":
		if body == nil {
			body = map[string]any{}
		}
		respBody, err = client.Post(ctx, reqURL, body)
	case "PATCH":
		if body == nil {
			body = map[string]any{}
		}
		respBody, err = client.Patch(ctx, reqURL, body)
	case "PUT":
		if body == nil {
			body = map[string]any{}
		}
		respBody, err = client.Put(ctx, reqURL, body)
	case "DELETE":
		if err := client.Delete(ctx, reqURL); err != nil {
			return nil, err
		}
		return []byte(`{"status":"deleted"}`), nil
	default:
		return nil, fmt.Errorf("unsupported method: %s", spec.Method)
	}

	if err != nil {
		return nil, err
	}

	// Handle post-creation actions (module/cycle attach)
	if len(relations) > 0 && len(respBody) > 0 {
		postCreateActionsRaw(ctx, relations, respBody, client, projectID)
	}

	if len(respBody) == 0 {
		return []byte(`{}`), nil
	}

	return respBody, nil
}

// resolveContext determines workspace and project from per-call args or
// server-level defaults.
func resolveContext(spec *docs.EndpointSpec, args map[string]any, cfg *Config) (workspace, projectID string, err error) {
	workspace = stringArg(args, "workspace")
	if workspace == "" {
		workspace = cfg.Workspace
	}

	if spec.RequiresWorkspace() && workspace == "" {
		return "", "", fmt.Errorf("workspace is required but not configured; pass 'workspace' param or configure via CLI")
	}

	projectRaw := stringArg(args, "project")
	if projectRaw == "" {
		projectRaw = cfg.Project
	}

	if spec.RequiresProject() {
		if projectRaw == "" {
			return "", "", fmt.Errorf("project is required but not configured; pass 'project' param or configure via CLI")
		}
		// Resolve project identifier to UUID if needed
		if !isUUID(projectRaw) {
			resolved, resolveErr := resolveProjectIdentifier(context.Background(), projectRaw, workspace, cfg)
			if resolveErr != nil {
				return "", "", fmt.Errorf("resolving project %q: %w", projectRaw, resolveErr)
			}
			projectRaw = resolved
		}
		projectID = projectRaw
	}

	return workspace, projectID, nil
}

// resolveProjectIdentifier resolves a short project identifier to a UUID.
func resolveProjectIdentifier(ctx context.Context, identifier, workspace string, cfg *Config) (string, error) {
	client, err := cfg.NewClient(workspace)
	if err != nil {
		return "", err
	}

	svc := api.NewProjectsService(client)
	resp, err := svc.List(ctx, api.PaginationParams{PerPage: 100})
	if err != nil {
		return "", err
	}

	upper := strings.ToUpper(identifier)
	for _, p := range resp.Results {
		if strings.ToUpper(p.Identifier) == upper {
			return p.ID, nil
		}
	}

	return "", fmt.Errorf("project identifier %q not found", identifier)
}

// buildURLFromMap constructs the API URL by substituting path params and
// appending query params from the args map.
func buildURLFromMap(client *api.Client, spec *docs.EndpointSpec, args map[string]any, projectID string, cfg *Config) (string, error) {
	path := spec.PathTemplate

	workspace := stringArg(args, "workspace")
	if workspace == "" {
		workspace = cfg.Workspace
	}
	path = strings.ReplaceAll(path, "{workspace_slug}", workspace)
	if projectID != "" {
		path = strings.ReplaceAll(path, "{project_id}", projectID)
	}

	// Substitute remaining path params
	for _, p := range spec.Params {
		if p.Location != docs.ParamPath {
			continue
		}
		placeholder := "{" + p.Name + "}"
		if !strings.Contains(path, placeholder) {
			continue
		}
		val := stringArg(args, p.Name)
		if val == "" {
			return "", fmt.Errorf("required path parameter %q not provided", p.Name)
		}
		// Resolve name to UUID if needed
		resolved, resolveErr := resolveValue(context.Background(), val, p.Name, client, projectID, cfg)
		if resolveErr != nil {
			return "", resolveErr
		}
		path = strings.ReplaceAll(path, placeholder, resolved)
	}

	reqURL := client.BaseURL + path

	// Add query params
	u, err := url.Parse(reqURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	for _, p := range spec.Params {
		if p.Location != docs.ParamQuery {
			continue
		}
		val := stringArg(args, p.Name)
		if val != "" {
			q.Set(p.Name, val)
		}
	}
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// collectBodyFromMap builds the request body from the args map.
func collectBodyFromMap(ctx context.Context, spec *docs.EndpointSpec, args map[string]any, cfg *Config) (map[string]any, error) {
	if spec.Method == "GET" || spec.Method == "DELETE" {
		return nil, nil
	}

	body := map[string]any{}
	for _, p := range spec.Params {
		if p.Location != docs.ParamBody {
			continue
		}

		// Handle _html params: accept markdown, convert to HTML
		if isHTMLParam(p.Name) {
			mdName := markdownParamName(p.Name)
			if val := stringArg(args, mdName); val != "" {
				html, err := markdown.ToHTML(val)
				if err == nil {
					body[p.Name] = html
				}
				continue
			}
			// Also accept raw HTML via the original param name
			if val := stringArg(args, p.Name); val != "" {
				body[p.Name] = val
			}
			continue
		}

		val, exists := args[p.Name]
		if !exists {
			continue
		}

		switch p.Type {
		case "string[]":
			if slice, ok := toStringSlice(val); ok && len(slice) > 0 {
				body[p.Name] = slice
			}
		case "number":
			if n, ok := toNumber(val); ok {
				body[p.Name] = n
			}
		case "boolean":
			if b, ok := toBool(val); ok {
				body[p.Name] = b
			}
		default:
			if s, ok := val.(string); ok && s != "" {
				resolved, err := resolveValue(ctx, s, p.Name, nil, "", cfg)
				if err != nil {
					return nil, err
				}
				body[p.Name] = resolved
			}
		}
	}

	if len(body) == 0 {
		return nil, nil
	}
	return body, nil
}

// resolveValue resolves a value to UUID if needed, similar to cmdgen.resolveIfNeeded
// but without Cobra dependency.
func resolveValue(ctx context.Context, value, paramName string, client *api.Client, projectID string, cfg *Config) (string, error) {
	if isUUID(value) {
		return value, nil
	}

	// Sequence ID resolution for issue-reference params
	if cmdgen.IsSequenceID(value) {
		if isIssueRefParam(paramName) {
			deps := cfg.BuildDeps()
			resolved, err := cmdgen.ResolveSequenceID(ctx, value, deps)
			if err == nil {
				return resolved, nil
			}
			// Could not resolve; return literal
			return value, nil
		}
	}

	// Standard _id suffix params
	if strings.HasSuffix(paramName, "_id") {
		deps := cfg.BuildDeps()
		resolved, err := cmdgen.ResolveNameToUUID(ctx, value, paramName, deps)
		if err != nil {
			return value, nil // best-effort: return literal
		}
		return resolved, nil
	}

	// Known params that accept UUIDs without _id suffix
	resolvableParams := map[string]string{
		"state": "state", "module": "module",
		"cycle": "cycle", "label": "label",
	}
	if resourceName, ok := resolvableParams[paramName]; ok {
		deps := cfg.BuildDeps()
		resolved, err := cmdgen.ResolveNameToUUID(ctx, value, resourceName+"_id", deps)
		if err != nil {
			return value, nil
		}
		return resolved, nil
	}

	return value, nil
}

// isIssueRefParam returns true if the param accepts issue references.
func isIssueRefParam(name string) bool {
	return name == "work_item_id" || name == "parent" || name == "issues"
}

// postCreateActionsRaw performs module/cycle attach after issue creation.
func postCreateActionsRaw(ctx context.Context, relations map[string]string, respBody []byte, client *api.Client, projectID string) {
	issueID, err := cmdgen.ExtractCreatedID(respBody)
	if err != nil {
		return
	}

	payload := map[string]any{"issues": []string{issueID}}

	if moduleID, ok := relations["module"]; ok {
		moduleURL := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/modules/%s/module-issues/",
			client.BaseURL, client.Workspace, projectID, moduleID)
		_, _ = client.Post(ctx, moduleURL, payload)
	}

	if cycleID, ok := relations["cycle"]; ok {
		cycleURL := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/cycles/%s/cycle-issues/",
			client.BaseURL, client.Workspace, projectID, cycleID)
		_, _ = client.Post(ctx, cycleURL, payload)
	}
}

// executeAutoPageinate fetches all pages and returns combined results.
func executeAutoPageinate(ctx context.Context, client *api.Client, baseURL string, spec *docs.EndpointSpec, args map[string]any) ([]byte, error) {
	var allResults []json.RawMessage
	cursor := ""
	perPage := 100
	if ps, ok := toNumber(args["page_size"]); ok && ps > 0 && ps <= 100 {
		perPage = int(ps)
	}

	for {
		u, err := url.Parse(baseURL)
		if err != nil {
			return nil, err
		}
		q := u.Query()
		q.Set("per_page", fmt.Sprintf("%d", perPage))
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		u.RawQuery = q.Encode()

		respBody, err := client.Get(ctx, u.String())
		if err != nil {
			return nil, err
		}

		var raw api.RawPaginatedResponse
		if err := json.Unmarshal(respBody, &raw); err != nil {
			return respBody, nil
		}

		if raw.Results != nil {
			var page []json.RawMessage
			if err := json.Unmarshal(raw.Results, &page); err != nil {
				return respBody, nil
			}
			allResults = append(allResults, page...)
		}

		if !raw.NextPageResults || raw.NextCursor == "" {
			break
		}
		cursor = raw.NextCursor
	}

	envelope := map[string]any{
		"results":     allResults,
		"total_count": len(allResults),
	}
	return json.Marshal(envelope)
}

// paginationFromArgs extracts pagination params from the args map.
func paginationFromArgs(args map[string]any) api.PaginationParams {
	p := api.PaginationParams{PerPage: 100}
	if ps, ok := toNumber(args["page_size"]); ok && ps > 0 {
		p.PerPage = int(ps)
		if p.PerPage > 100 {
			p.PerPage = 100
		}
	}
	if c, ok := args["cursor"].(string); ok && c != "" {
		p.Cursor = c
	}
	return p
}

// Helper functions for type conversion from JSON-unmarshaled values.

func stringArg(args map[string]any, key string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func toStringSlice(v any) ([]string, bool) {
	switch val := v.(type) {
	case []string:
		return val, true
	case []any:
		result := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result, len(result) > 0
	}
	return nil, false
}

func toNumber(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}

func toBool(v any) (bool, bool) {
	if b, ok := v.(bool); ok {
		return b, true
	}
	return false, false
}

func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		}
	}
	return true
}

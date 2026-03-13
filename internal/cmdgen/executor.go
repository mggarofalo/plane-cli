package cmdgen

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/mggarofalo/plane-cli/internal/api"
	"github.com/mggarofalo/plane-cli/internal/docs"
	"github.com/mggarofalo/plane-cli/internal/markdown"
	"github.com/mggarofalo/plane-cli/internal/output"
	"github.com/spf13/cobra"
)

// ClientFactory creates an API client. Injected from cmd package.
type ClientFactory func() (*api.Client, error)

// WorkspaceRequirer validates workspace is set. Injected from cmd package.
type WorkspaceRequirer func(client *api.Client) error

// ProjectRequirer resolves the project ID. Injected from cmd package.
type ProjectRequirer func() (string, error)

// PaginationParamsFactory returns pagination params from flags. Injected from cmd package.
type PaginationParamsFactory func() api.PaginationParams

// FormatterFactory returns the output formatter. Injected from cmd package.
type FormatterFactory func() output.Formatter

// UUIDChecker checks if a string is a UUID. Injected from cmd package.
type UUIDChecker func(string) bool

// Deps holds injected dependencies from the cmd package.
type Deps struct {
	NewClient        ClientFactory
	RequireWorkspace WorkspaceRequirer
	RequireProject   ProjectRequirer
	PaginationParams PaginationParamsFactory
	Formatter        FormatterFactory
	IsUUID           UUIDChecker
	FlagAll          *bool
	FlagPerPage      *int
	Profile          string
	BaseURL          string
}

// ExecuteSpec runs an API call based on the endpoint spec and cobra flags.
func ExecuteSpec(ctx context.Context, cmd *cobra.Command, spec *docs.EndpointSpec, deps *Deps) error {
	client, err := deps.NewClient()
	if err != nil {
		return err
	}

	if spec.RequiresWorkspace() {
		if err := deps.RequireWorkspace(client); err != nil {
			return err
		}
	}

	var projectID string
	if spec.RequiresProject() {
		projectID, err = deps.RequireProject()
		if err != nil {
			return err
		}
	}

	// Build URL from path template
	reqURL, err := buildURL(client, spec, cmd, projectID, deps)
	if err != nil {
		return err
	}

	// Collect body params
	body := collectBodyParams(cmd, spec, deps)
	// Inject global path params (project_id, workspace_slug) when spec has them as body params
	body = injectGlobalBodyParams(body, spec, client.Workspace, projectID)

	// Extract many-to-many relation params before sending POST requests.
	// Only extract for POST — PATCH/PUT may legitimately include these fields.
	var relations map[string]string
	if spec.Method == "POST" {
		relations = extractRelationParams(body)
	}

	// Execute the request
	var respBody []byte
	switch spec.Method {
	case "GET":
		if deps.FlagAll != nil && *deps.FlagAll {
			return executeAutoPageinate(ctx, client, reqURL, spec, deps)
		}
		respBody, err = client.GetPaginated(ctx, reqURL, deps.PaginationParams())
	case "POST":
		if body != nil {
			respBody, err = client.Post(ctx, reqURL, body)
		} else {
			respBody, err = client.Post(ctx, reqURL, map[string]any{})
		}
	case "PATCH":
		if body != nil {
			respBody, err = client.Patch(ctx, reqURL, body)
		} else {
			respBody, err = client.Patch(ctx, reqURL, map[string]any{})
		}
	case "PUT":
		if body != nil {
			respBody, err = client.Put(ctx, reqURL, body)
		} else {
			respBody, err = client.Put(ctx, reqURL, map[string]any{})
		}
	case "DELETE":
		if err := client.Delete(ctx, reqURL); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "Deleted.")
		return nil
	default:
		return fmt.Errorf("unsupported method: %s", spec.Method)
	}

	if err != nil {
		return err
	}

	if len(respBody) == 0 {
		return nil
	}

	// Handle post-creation actions for many-to-many relations
	if len(relations) > 0 {
		if err := postCreateActions(ctx, relations, respBody, client, projectID); err != nil {
			return err
		}
	}

	return formatResponse(respBody, deps)
}

// ExecuteSpecFromArgs runs an API call using manually parsed args (Mode B).
func ExecuteSpecFromArgs(ctx context.Context, spec *docs.EndpointSpec, parsed *ParsedArgs, deps *Deps) error {
	client, err := deps.NewClient()
	if err != nil {
		return err
	}

	if spec.RequiresWorkspace() {
		if err := deps.RequireWorkspace(client); err != nil {
			return err
		}
	}

	var projectID string
	if spec.RequiresProject() {
		projectID, err = deps.RequireProject()
		if err != nil {
			return err
		}
	}

	// Build URL from path template
	reqURL, err := buildURLFromArgs(client, spec, parsed, projectID, deps)
	if err != nil {
		return err
	}

	// Collect body params from parsed args
	body := collectBodyParamsFromArgs(spec, parsed, deps)
	body = injectGlobalBodyParams(body, spec, client.Workspace, projectID)

	// Extract many-to-many relation params before sending POST requests.
	// Only extract for POST — PATCH/PUT may legitimately include these fields.
	var relations map[string]string
	if spec.Method == "POST" {
		relations = extractRelationParams(body)
	}

	// Execute
	var respBody []byte
	switch spec.Method {
	case "GET":
		respBody, err = client.GetPaginated(ctx, reqURL, deps.PaginationParams())
	case "POST":
		if body != nil {
			respBody, err = client.Post(ctx, reqURL, body)
		} else {
			respBody, err = client.Post(ctx, reqURL, map[string]any{})
		}
	case "PATCH":
		if body != nil {
			respBody, err = client.Patch(ctx, reqURL, body)
		} else {
			respBody, err = client.Patch(ctx, reqURL, map[string]any{})
		}
	case "PUT":
		if body != nil {
			respBody, err = client.Put(ctx, reqURL, body)
		} else {
			respBody, err = client.Put(ctx, reqURL, map[string]any{})
		}
	case "DELETE":
		if err := client.Delete(ctx, reqURL); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "Deleted.")
		return nil
	default:
		return fmt.Errorf("unsupported method: %s", spec.Method)
	}

	if err != nil {
		return err
	}

	if len(respBody) == 0 {
		return nil
	}

	// Handle post-creation actions for many-to-many relations
	if len(relations) > 0 {
		if err := postCreateActions(ctx, relations, respBody, client, projectID); err != nil {
			return err
		}
	}

	return formatResponse(respBody, deps)
}

func buildURL(client *api.Client, spec *docs.EndpointSpec, cmd *cobra.Command, projectID string, deps *Deps) (string, error) {
	path := spec.PathTemplate

	// Substitute known variables
	path = strings.ReplaceAll(path, "{workspace_slug}", client.Workspace)
	if projectID != "" {
		path = strings.ReplaceAll(path, "{project_id}", projectID)
	}

	// Substitute remaining path params from flags
	for _, p := range spec.Params {
		if p.Location != docs.ParamPath {
			continue
		}
		placeholder := "{" + p.Name + "}"
		if !strings.Contains(path, placeholder) {
			continue
		}
		flagName := ParamToFlagName(p.Name)
		val, _ := cmd.Flags().GetString(flagName)
		if val == "" {
			return "", fmt.Errorf("required path parameter --%s not provided", flagName)
		}
		// Resolve name to UUID if needed
		val = resolveIfNeeded(cmd.Context(), val, p.Name, client, projectID, deps)
		path = strings.ReplaceAll(path, placeholder, val)
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
		flagName := ParamToFlagName(p.Name)
		val, _ := cmd.Flags().GetString(flagName)
		if val != "" {
			q.Set(p.Name, val)
		}
	}
	u.RawQuery = q.Encode()

	return u.String(), nil
}

func buildURLFromArgs(client *api.Client, spec *docs.EndpointSpec, parsed *ParsedArgs, projectID string, deps *Deps) (string, error) {
	path := spec.PathTemplate

	path = strings.ReplaceAll(path, "{workspace_slug}", client.Workspace)
	if projectID != "" {
		path = strings.ReplaceAll(path, "{project_id}", projectID)
	}

	for _, p := range spec.Params {
		if p.Location != docs.ParamPath {
			continue
		}
		placeholder := "{" + p.Name + "}"
		if !strings.Contains(path, placeholder) {
			continue
		}
		flagName := ParamToFlagName(p.Name)
		val := parsed.Get(flagName)
		if val == "" {
			return "", fmt.Errorf("required path parameter --%s not provided", flagName)
		}
		val = resolveIfNeeded(context.Background(), val, p.Name, client, projectID, deps)
		path = strings.ReplaceAll(path, placeholder, val)
	}

	reqURL := client.BaseURL + path

	u, err := url.Parse(reqURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	for _, p := range spec.Params {
		if p.Location != docs.ParamQuery {
			continue
		}
		flagName := ParamToFlagName(p.Name)
		val := parsed.Get(flagName)
		if val != "" {
			q.Set(p.Name, val)
		}
	}
	u.RawQuery = q.Encode()

	return u.String(), nil
}

func collectBodyParams(cmd *cobra.Command, spec *docs.EndpointSpec, deps *Deps) map[string]any {
	if spec.Method == "GET" || spec.Method == "DELETE" {
		return nil
	}

	body := map[string]any{}
	for _, p := range spec.Params {
		if p.Location != docs.ParamBody {
			continue
		}

		// For _html params, check the markdown flag first, then the raw HTML flag.
		if IsHTMLParam(p.Name) {
			mdFlag := MarkdownFlagName(p.Name)
			htmlFlag := ParamToFlagName(p.Name)
			if cmd.Flags().Changed(mdFlag) {
				val, _ := cmd.Flags().GetString(mdFlag)
				if val != "" {
					html, err := markdown.ToHTML(val)
					if err == nil {
						body[p.Name] = html
					}
				}
			} else if cmd.Flags().Changed(htmlFlag) {
				val, _ := cmd.Flags().GetString(htmlFlag)
				if val != "" {
					body[p.Name] = val
				}
			}
			continue
		}

		flagName := ParamToFlagName(p.Name)
		if !cmd.Flags().Changed(flagName) {
			continue
		}

		switch p.Type {
		case "string[]":
			val, _ := cmd.Flags().GetStringSlice(flagName)
			if len(val) > 0 {
				if issueRefParams[p.Name] {
					val = resolveSliceIfNeeded(cmd.Context(), val, p.Name, deps)
				}
				body[p.Name] = val
			}
		case "number":
			val, _ := cmd.Flags().GetInt(flagName)
			body[p.Name] = val
		case "boolean":
			val, _ := cmd.Flags().GetBool(flagName)
			body[p.Name] = val
		default:
			val, _ := cmd.Flags().GetString(flagName)
			if val != "" {
				// Resolve name to UUID for _id fields
				val = resolveIfNeeded(cmd.Context(), val, p.Name, nil, "", deps)
				body[p.Name] = val
			}
		}
	}

	if len(body) == 0 {
		return nil
	}
	return body
}

func collectBodyParamsFromArgs(spec *docs.EndpointSpec, parsed *ParsedArgs, deps *Deps) map[string]any {
	if spec.Method == "GET" || spec.Method == "DELETE" {
		return nil
	}

	body := map[string]any{}
	for _, p := range spec.Params {
		if p.Location != docs.ParamBody {
			continue
		}

		// For _html params, check the markdown flag first, then the raw HTML flag.
		if IsHTMLParam(p.Name) {
			mdFlag := MarkdownFlagName(p.Name)
			htmlFlag := ParamToFlagName(p.Name)
			if parsed.Has(mdFlag) {
				val := parsed.Get(mdFlag)
				if val != "" {
					html, err := markdown.ToHTML(val)
					if err == nil {
						body[p.Name] = html
					}
				}
			} else if parsed.Has(htmlFlag) {
				val := parsed.Get(htmlFlag)
				if val != "" {
					body[p.Name] = val
				}
			}
			continue
		}

		flagName := ParamToFlagName(p.Name)
		if !parsed.Has(flagName) {
			continue
		}

		switch p.Type {
		case "string[]":
			val := parsed.GetSlice(flagName)
			if len(val) > 0 {
				if issueRefParams[p.Name] {
					val = resolveSliceIfNeeded(context.Background(), val, p.Name, deps)
				}
				body[p.Name] = val
			}
		case "number":
			val := parsed.Get(flagName)
			// Parse as int
			var n int
			_, _ = fmt.Sscanf(val, "%d", &n)
			body[p.Name] = n
		case "boolean":
			val := parsed.Get(flagName)
			body[p.Name] = val == "true" || val == "1" || val == "yes"
		default:
			val := parsed.Get(flagName)
			if val != "" {
				val = resolveIfNeeded(context.Background(), val, p.Name, nil, "", deps)
				body[p.Name] = val
			}
		}
	}

	if len(body) == 0 {
		return nil
	}
	return body
}

// injectGlobalBodyParams adds project_id and workspace_slug to the body when
// the spec lists them as body params. Some Plane endpoints (e.g., cycle create)
// require these in the body even though they're also in the URL path.
func injectGlobalBodyParams(body map[string]any, spec *docs.EndpointSpec, workspace, projectID string) map[string]any {
	for _, p := range spec.Params {
		if p.Location != docs.ParamBody {
			continue
		}
		switch p.Name {
		case "project_id", "project":
			if projectID != "" {
				if body == nil {
					body = map[string]any{}
				}
				if _, exists := body[p.Name]; !exists {
					body[p.Name] = projectID
				}
			}
		case "workspace_slug", "workspace":
			if workspace != "" {
				if body == nil {
					body = map[string]any{}
				}
				if _, exists := body[p.Name]; !exists {
					body[p.Name] = workspace
				}
			}
		}
	}
	return body
}

func formatResponse(respBody []byte, deps *Deps) error {
	f := deps.Formatter()

	// Try to format as dynamic table if table output
	if _, ok := f.(*output.TableFormatter); ok {
		return output.FormatDynamicTable(os.Stdout, respBody)
	}

	// JSON output: pass through as raw message
	var raw json.RawMessage = respBody
	return f.Format(os.Stdout, raw)
}

func executeAutoPageinate(ctx context.Context, client *api.Client, baseURL string, spec *docs.EndpointSpec, deps *Deps) error {
	var allResults []json.RawMessage
	cursor := ""
	perPage := 100
	if deps.FlagPerPage != nil && *deps.FlagPerPage > 0 {
		perPage = *deps.FlagPerPage
	}
	if perPage > 100 {
		perPage = 100
	}

	for {
		u, err := url.Parse(baseURL)
		if err != nil {
			return err
		}
		q := u.Query()
		q.Set("per_page", fmt.Sprintf("%d", perPage))
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		u.RawQuery = q.Encode()

		respBody, err := client.Get(ctx, u.String())
		if err != nil {
			return err
		}

		var raw api.RawPaginatedResponse
		if err := json.Unmarshal(respBody, &raw); err != nil {
			// Not paginated — just return as-is
			return formatResponse(respBody, deps)
		}

		if raw.Results != nil {
			var page []json.RawMessage
			if err := json.Unmarshal(raw.Results, &page); err != nil {
				return formatResponse(respBody, deps)
			}
			allResults = append(allResults, page...)
		}

		if !raw.NextPageResults || raw.NextCursor == "" {
			break
		}
		cursor = raw.NextCursor
	}

	// Wrap results in an envelope
	envelope := map[string]any{
		"results":     allResults,
		"total_count": len(allResults),
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	return formatResponse(data, deps)
}

// issueRefParams are parameter names that accept work-item references
// (UUIDs or sequence IDs like "PROJ-42").
var issueRefParams = map[string]bool{
	"work_item_id": true,
	"parent":       true,
	"issues":       true,
}

// resolvableParams are body parameter names (without _id suffix) that accept
// UUIDs but whose API field name doesn't end with _id. When the user passes a
// human-readable name for one of these, we resolve it to a UUID.
var resolvableParams = map[string]string{
	"state":  "state",
	"module": "module",
	"parent": "issue",
	"cycle":  "cycle",
	"label":  "label",
}

// resolveIfNeeded resolves a value to UUID if the param expects an ID and the value is not a UUID.
func resolveIfNeeded(ctx context.Context, value, paramName string, client *api.Client, projectID string, deps *Deps) string {
	if deps.IsUUID == nil || deps.IsUUID(value) {
		return value
	}

	// Sequence ID resolution for issue-reference params (e.g. "PROJ-42")
	if issueRefParams[paramName] && IsSequenceID(value) {
		resolved, err := ResolveSequenceID(ctx, value, deps)
		if err == nil {
			return resolved
		}
		// Fall through to other resolution strategies
	}

	// Standard _id suffix params (e.g., state_id, label_id)
	if strings.HasSuffix(paramName, "_id") {
		resolved, err := ResolveNameToUUID(ctx, value, paramName, deps)
		if err != nil {
			return value
		}
		return resolved
	}

	// Known params that accept UUIDs without _id suffix
	if resourceName, ok := resolvableParams[paramName]; ok {
		resolved, err := ResolveNameToUUID(ctx, value, resourceName+"_id", deps)
		if err != nil {
			return value
		}
		return resolved
	}

	return value
}

// resolveSliceIfNeeded resolves each element in a string slice using resolveIfNeeded.
func resolveSliceIfNeeded(ctx context.Context, values []string, paramName string, deps *Deps) []string {
	resolved := make([]string, len(values))
	for i, v := range values {
		resolved[i] = resolveIfNeeded(ctx, v, paramName, nil, "", deps)
	}
	return resolved
}

// relationParams are body parameter names for many-to-many relationships that
// the Plane API silently ignores in the issue creation body. These must be
// handled via separate API calls after the issue is created.
var relationParams = map[string]struct{}{
	"module": {},
	"cycle":  {},
}

// extractRelationParams removes many-to-many relation fields (module, cycle)
// from the body map and returns them separately. These fields are silently
// ignored by the Plane API on issue creation and must be handled via follow-up
// API calls.
func extractRelationParams(body map[string]any) map[string]string {
	if body == nil {
		return nil
	}

	relations := map[string]string{}
	for key := range relationParams {
		val, ok := body[key]
		if !ok {
			continue
		}
		strVal, ok := val.(string)
		if !ok || strVal == "" {
			continue
		}
		relations[key] = strVal
		delete(body, key)
	}

	if len(relations) == 0 {
		return nil
	}
	return relations
}

// extractCreatedID parses the API response JSON and returns the "id" field.
func extractCreatedID(respBody []byte) (string, error) {
	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parsing response to extract issue ID: %w", err)
	}
	id, ok := result["id"].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("response missing 'id' field")
	}
	return id, nil
}

// postCreateActions performs follow-up API calls to attach a newly created issue
// to modules and/or cycles. The Plane API requires separate endpoints for these
// many-to-many relationships.
func postCreateActions(ctx context.Context, relations map[string]string, respBody []byte, client *api.Client, projectID string) error {
	issueID, err := extractCreatedID(respBody)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"issues": []string{issueID},
	}

	if moduleID, ok := relations["module"]; ok {
		moduleURL := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/modules/%s/module-issues/",
			client.BaseURL, client.Workspace, projectID, moduleID)
		if _, err := client.Post(ctx, moduleURL, payload); err != nil {
			return fmt.Errorf("adding issue to module: %w", err)
		}
		fmt.Fprintln(os.Stderr, "Added to module.")
	}

	if cycleID, ok := relations["cycle"]; ok {
		cycleURL := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/cycles/%s/cycle-issues/",
			client.BaseURL, client.Workspace, projectID, cycleID)
		if _, err := client.Post(ctx, cycleURL, payload); err != nil {
			return fmt.Errorf("adding issue to cycle: %w", err)
		}
		fmt.Fprintln(os.Stderr, "Added to cycle.")
	}

	return nil
}

// GenerateHelp prints help text for a spec.
func GenerateHelp(w io.Writer, topicName, cmdName string, spec *docs.EndpointSpec) {
	fmt.Fprintf(w, "Usage: plane %s %s [flags]\n\n", topicName, cmdName)
	fmt.Fprintf(w, "%s %s\n", spec.Method, spec.PathTemplate)
	fmt.Fprintf(w, "Source: %s\n\n", spec.SourceURL)

	if len(spec.Params) == 0 {
		return
	}

	fmt.Fprintln(w, "Flags:")
	for _, p := range spec.Params {
		if p.Name == "workspace_slug" || p.Name == "project_id" {
			continue
		}
		desc := p.Description
		if desc == "" {
			desc = p.Type
		}
		req := ""
		if p.Required {
			req = " (required)"
		}

		if IsHTMLParam(p.Name) {
			mdFlag := MarkdownFlagName(p.Name)
			htmlFlag := ParamToFlagName(p.Name)
			mdDesc := desc
			if mdDesc == p.Name {
				mdDesc = mdFlag
			}
			fmt.Fprintf(w, "  --%s\t%s (markdown)%s\n", mdFlag, mdDesc, req)
			fmt.Fprintf(w, "  --%s\t%s (raw HTML)%s\n", htmlFlag, mdDesc, req)
			continue
		}

		flagName := ParamToFlagName(p.Name)
		fmt.Fprintf(w, "  --%s\t%s%s\n", flagName, desc, req)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Global flags:")
	fmt.Fprintln(w, "  -w, --workspace\tWorkspace slug")
	fmt.Fprintln(w, "  -p, --project\t\tProject ID or identifier")
	fmt.Fprintln(w, "  -o, --output\t\tOutput format: json, table")
	fmt.Fprintln(w, "      --all\t\tAuto-paginate and return all results")
}

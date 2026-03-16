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
	"github.com/mggarofalo/plane-cli/internal/cache"
	"github.com/mggarofalo/plane-cli/internal/docs"
	"github.com/mggarofalo/plane-cli/internal/markdown"
	"github.com/mggarofalo/plane-cli/internal/output"
	"github.com/mggarofalo/plane-cli/internal/weburl"
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
	CacheStore       *cache.Store
	FlagAll          *bool
	FlagPerPage      *int
	FlagDryRun       *bool
	FlagField        *string
	FlagFields       *string
	FlagQuiet        *bool
	FlagStrict       *bool
	FlagNoResolve    *bool
	FlagIDOnly       *bool
	FlagBatch        *bool
	FlagStdin        *bool
	Profile          string
	BaseURL          string
}

// isDryRun returns true when the dry-run flag is set. Nil-safe.
func isDryRun(deps *Deps) bool {
	return deps.FlagDryRun != nil && *deps.FlagDryRun
}

// IsQuiet returns true when the quiet flag is set. Nil-safe.
func IsQuiet(deps *Deps) bool {
	return deps != nil && deps.FlagQuiet != nil && *deps.FlagQuiet
}

// IsStrict returns true when the strict flag is set. Nil-safe.
func IsStrict(deps *Deps) bool {
	return deps != nil && deps.FlagStrict != nil && *deps.FlagStrict
}

// IsNoResolve returns true when the no-resolve flag is set. Nil-safe.
func IsNoResolve(deps *Deps) bool {
	return deps != nil && deps.FlagNoResolve != nil && *deps.FlagNoResolve
}
// isIDOnly returns true when the id-only flag is set. Nil-safe.
func isIDOnly(deps *Deps) bool {
	return deps != nil && deps.FlagIDOnly != nil && *deps.FlagIDOnly
}

// Infof writes an informational message to stderr unless quiet mode is active.
// Use for status messages like "Deleted.", "Added to module.", hints, etc.
// Errors should still go directly to stderr (not through this function).
func Infof(deps *Deps, format string, args ...any) {
	if IsQuiet(deps) {
		return
	}
	fmt.Fprintf(os.Stderr, format, args...)
}

// Warnf writes a warning message to stderr. Unlike Infof, warnings are emitted
// even in quiet mode because they indicate potential problems. Only --quiet
// suppresses informational messages, not warnings.
func Warnf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
}

// ResolutionError indicates a name-to-UUID resolution failure when --strict is
// active. It produces exit code 4 (validation) via the standard error handling.
type ResolutionError struct {
	Msg string
}

func (e *ResolutionError) Error() string    { return e.Msg }
func (e *ResolutionError) ExitCode() int    { return api.ExitValidation }

// snapshotBody returns a shallow copy of body so the original is preserved
// before extractRelationParams mutates it.
func snapshotBody(body map[string]any) map[string]any {
	if body == nil {
		return nil
	}
	cp := make(map[string]any, len(body))
	for k, v := range body {
		cp[k] = v
	}
	return cp
}

// printDryRun writes the would-be request details to stderr.
func printDryRun(method, reqURL string, body map[string]any, relations map[string]string, workspace, projectID string, deps *Deps) {
	Infof(deps, "DRY RUN: %s %s\n", method, reqURL)

	if body != nil && method != "DELETE" {
		data, err := json.MarshalIndent(body, "", "  ")
		if err == nil {
			Infof(deps, "Body: %s\n", data)
		}
	}

	for kind, id := range relations {
		var endpoint string
		switch kind {
		case "module":
			endpoint = fmt.Sprintf("/api/v1/workspaces/%s/projects/%s/modules/%s/module-issues/", workspace, projectID, id)
		case "cycle":
			endpoint = fmt.Sprintf("/api/v1/workspaces/%s/projects/%s/cycles/%s/cycle-issues/", workspace, projectID, id)
		}
		if endpoint != "" {
			Infof(deps, "(would also call: POST %s with {\"issues\":[\"<new-issue-id>\"]})\n", endpoint)
		}
	}
}

// ExecuteSpec runs an API call based on the endpoint spec and cobra flags.
func ExecuteSpec(ctx context.Context, cmd *cobra.Command, spec *docs.EndpointSpec, deps *Deps) error {
	// Batch mode: read JSONL from stdin instead of using flags
	if isBatch(deps) {
		return ExecuteBatch(ctx, spec, deps)
	}

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
	body, err := collectBodyParams(cmd, spec, deps)
	if err != nil {
		return err
	}

	// Merge stdin JSON if --stdin flag is set (for POST/PATCH/PUT only)
	if isStdin(deps) && spec.Method != "GET" && spec.Method != "DELETE" {
		stdinBody, stdinErr := ReadStdinJSON()
		if stdinErr != nil {
			return stdinErr
		}
		// Track which keys came from stdin (not overridden by flags) for resolution
		stdinKeys := StdinKeys(stdinBody, body)
		body = MergeStdinWithFlags(stdinBody, body)
		// Apply name resolution to stdin-originated fields
		if resolveErr := ResolveStdinBody(ctx, body, stdinKeys, deps); resolveErr != nil {
			return resolveErr
		}
	}

	// Inject global path params (project_id, workspace_slug) when spec has them as body params
	body = InjectGlobalBodyParams(body, spec, client.Workspace, projectID)

	// Snapshot body before relation extraction mutates it (for dry-run output)
	var snapshot map[string]any
	if isDryRun(deps) && spec.Method != "GET" {
		snapshot = snapshotBody(body)
	}

	// Extract many-to-many relation params before sending POST requests.
	// Only extract for POST — PATCH/PUT may legitimately include these fields.
	var relations map[string]string
	if spec.Method == "POST" {
		relations = ExtractRelationParams(body)
	}

	// Dry-run: print what would be sent and return early
	if isDryRun(deps) && spec.Method != "GET" {
		printDryRun(spec.Method, reqURL, snapshot, relations, client.Workspace, projectID, deps)
		return nil
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
		Infof(deps, "Deleted.\n")
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

	// Inject web_url for single-resource responses
	respBody = weburl.Inject(respBody, client.BaseURL, client.Workspace, projectID, spec.PathTemplate)

	// Always show the created resource, even if post-creation attach fails.
	if err := formatResponse(respBody, deps); err != nil {
		return err
	}

	// Handle post-creation actions for many-to-many relations
	if len(relations) > 0 {
		PostCreateActions(ctx, relations, respBody, client, projectID, deps)
	}

	return nil
}

// ExecuteSpecFromArgs runs an API call using manually parsed args (Mode B).
func ExecuteSpecFromArgs(ctx context.Context, spec *docs.EndpointSpec, parsed *ParsedArgs, deps *Deps) error {
	// Batch mode: read JSONL from stdin instead of using parsed args
	if isBatch(deps) {
		return ExecuteBatch(ctx, spec, deps)
	}

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
	body, err := collectBodyParamsFromArgs(ctx, spec, parsed, deps)
	if err != nil {
		return err
	}

	// Merge stdin JSON if --stdin flag is set (for POST/PATCH/PUT only)
	if isStdin(deps) && spec.Method != "GET" && spec.Method != "DELETE" {
		stdinBody, stdinErr := ReadStdinJSON()
		if stdinErr != nil {
			return stdinErr
		}
		// Track which keys came from stdin (not overridden by flags) for resolution
		stdinKeys := StdinKeys(stdinBody, body)
		body = MergeStdinWithFlags(stdinBody, body)
		// Apply name resolution to stdin-originated fields
		if resolveErr := ResolveStdinBody(ctx, body, stdinKeys, deps); resolveErr != nil {
			return resolveErr
		}
	}

	body = InjectGlobalBodyParams(body, spec, client.Workspace, projectID)

	// Snapshot body before relation extraction mutates it (for dry-run output)
	var snapshot map[string]any
	if isDryRun(deps) && spec.Method != "GET" {
		snapshot = snapshotBody(body)
	}

	// Extract many-to-many relation params before sending POST requests.
	// Only extract for POST — PATCH/PUT may legitimately include these fields.
	var relations map[string]string
	if spec.Method == "POST" {
		relations = ExtractRelationParams(body)
	}

	// Dry-run: print what would be sent and return early
	if isDryRun(deps) && spec.Method != "GET" {
		printDryRun(spec.Method, reqURL, snapshot, relations, client.Workspace, projectID, deps)
		return nil
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
		Infof(deps, "Deleted.\n")
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

	// Inject web_url for single-resource responses
	respBody = weburl.Inject(respBody, client.BaseURL, client.Workspace, projectID, spec.PathTemplate)

	// Always show the created resource, even if post-creation attach fails.
	if err := formatResponse(respBody, deps); err != nil {
		return err
	}

	// Handle post-creation actions for many-to-many relations
	if len(relations) > 0 {
		PostCreateActions(ctx, relations, respBody, client, projectID, deps)
	}

	return nil
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
		resolved, resolveErr := resolveIfNeeded(cmd.Context(), val, p.Name, client, projectID, deps)
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
		resolved, resolveErr := resolveIfNeeded(context.Background(), val, p.Name, client, projectID, deps)
		if resolveErr != nil {
			return "", resolveErr
		}
		path = strings.ReplaceAll(path, placeholder, resolved)
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

func collectBodyParams(cmd *cobra.Command, spec *docs.EndpointSpec, deps *Deps) (map[string]any, error) {
	if spec.Method == "GET" || spec.Method == "DELETE" {
		return nil, nil
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
					resolved, err := resolveSliceIfNeeded(cmd.Context(), val, p.Name, deps)
					if err != nil {
						return nil, err
					}
					val = resolved
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
				resolved, err := resolveIfNeeded(cmd.Context(), val, p.Name, nil, "", deps)
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

func collectBodyParamsFromArgs(ctx context.Context, spec *docs.EndpointSpec, parsed *ParsedArgs, deps *Deps) (map[string]any, error) {
	if spec.Method == "GET" || spec.Method == "DELETE" {
		return nil, nil
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
					resolved, err := resolveSliceIfNeeded(ctx, val, p.Name, deps)
					if err != nil {
						return nil, err
					}
					val = resolved
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
				resolved, err := resolveIfNeeded(ctx, val, p.Name, nil, "", deps)
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

// InjectGlobalBodyParams adds project_id and workspace_slug to the body when
// the spec lists them as body params. Some Plane endpoints (e.g., cycle create)
// require these in the body even though they're also in the URL path.
func InjectGlobalBodyParams(body map[string]any, spec *docs.EndpointSpec, workspace, projectID string) map[string]any {
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
	// --id-only: extract id and print raw UUID without newline
	if isIDOnly(deps) {
		return output.ExtractID(os.Stdout, respBody)
	}

	// --field: extract a single field, output raw value
	if deps.FlagField != nil && *deps.FlagField != "" {
		return output.ExtractField(os.Stdout, respBody, *deps.FlagField)
	}

	// --fields: extract multiple fields, output TSV
	if deps.FlagFields != nil && *deps.FlagFields != "" {
		var paths []string
		for _, p := range strings.Split(*deps.FlagFields, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				paths = append(paths, p)
			}
		}
		if len(paths) > 0 {
			return output.ExtractFields(os.Stdout, respBody, paths)
		}
	}

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
	"cycle":  "cycle",
	"label":  "label",
}

// resolveIfNeeded resolves a value to UUID if the param expects an ID and the
// value is not a UUID. When resolution fails, it emits a warning to stderr and
// passes the literal value through. If --strict is active, resolution failure
// returns a ResolutionError instead. If --no-resolve is active, the value is
// returned as-is without any resolution attempts.
func resolveIfNeeded(ctx context.Context, value, paramName string, client *api.Client, projectID string, deps *Deps) (string, error) {
	if IsNoResolve(deps) {
		return value, nil
	}
	if deps.IsUUID == nil || deps.IsUUID(value) {
		return value, nil
	}

	// Sequence ID resolution for issue-reference params (e.g. "PROJ-42")
	if issueRefParams[paramName] && IsSequenceID(value) {
		resolved, err := ResolveSequenceID(ctx, value, deps)
		if err == nil {
			return resolved, nil
		}
		// Sequence ID looked right but could not be resolved
		return warnOrFailResolution(value, paramName, deps)
	}

	// Standard _id suffix params (e.g., state_id, label_id)
	if strings.HasSuffix(paramName, "_id") {
		resolved, err := ResolveNameToUUID(ctx, value, paramName, deps)
		if err != nil {
			return warnOrFailResolution(value, paramName, deps)
		}
		return resolved, nil
	}

	// Known params that accept UUIDs without _id suffix
	if resourceName, ok := resolvableParams[paramName]; ok {
		resolved, err := ResolveNameToUUID(ctx, value, resourceName+"_id", deps)
		if err != nil {
			return warnOrFailResolution(value, paramName, deps)
		}
		return resolved, nil
	}

	return value, nil
}

// warnOrFailResolution emits a warning about a failed name resolution. When
// --strict is active, it returns a ResolutionError; otherwise it passes the
// literal value through.
func warnOrFailResolution(value, paramName string, deps *Deps) (string, error) {
	if IsStrict(deps) {
		return "", &ResolutionError{
			Msg: fmt.Sprintf("could not resolve %q for %s; passing literal value (use --strict=false to allow)", value, paramName),
		}
	}
	Warnf("Warning: could not resolve %q for %s; passing literal value\n", value, paramName)
	return value, nil
}

// resolveSliceIfNeeded resolves each element in a string slice using resolveIfNeeded.
// If --no-resolve is active, the original slice is returned as-is.
func resolveSliceIfNeeded(ctx context.Context, values []string, paramName string, deps *Deps) ([]string, error) {
	if IsNoResolve(deps) {
		return values, nil
	}
	resolved := make([]string, len(values))
	for i, v := range values {
		r, err := resolveIfNeeded(ctx, v, paramName, nil, "", deps)
		if err != nil {
			return nil, err
		}
		resolved[i] = r
	}
	return resolved, nil
}

// relationParams are body parameter names for many-to-many relationships that
// the Plane API silently ignores in the issue creation body. These must be
// handled via separate API calls after the issue is created.
var relationParams = map[string]struct{}{
	"module": {},
	"cycle":  {},
}

// ExtractRelationParams removes many-to-many relation fields (module, cycle)
// from the body map and returns them separately. These fields are silently
// ignored by the Plane API on issue creation and must be handled via follow-up
// API calls.
func ExtractRelationParams(body map[string]any) map[string]string {
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

// ExtractCreatedID parses the API response JSON and returns the "id" field.
func ExtractCreatedID(respBody []byte) (string, error) {
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

// PostCreateActions performs follow-up API calls to attach a newly created issue
// to modules and/or cycles. The Plane API requires separate endpoints for these
// many-to-many relationships. Failures are printed as warnings to stderr; the
// created issue is never rolled back.
func PostCreateActions(ctx context.Context, relations map[string]string, respBody []byte, client *api.Client, projectID string, deps *Deps) {
	issueID, err := ExtractCreatedID(respBody)
	if err != nil {
		Infof(deps, "Warning: issue created but could not extract ID for relation attach: %v\n", err)
		return
	}

	payload := map[string]any{
		"issues": []string{issueID},
	}

	if moduleID, ok := relations["module"]; ok {
		moduleURL := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/modules/%s/module-issues/",
			client.BaseURL, client.Workspace, projectID, moduleID)
		if _, err := client.Post(ctx, moduleURL, payload); err != nil {
			Infof(deps, "Warning: issue created but failed to add to module: %v\n", err)
		} else {
			Infof(deps, "Added to module.\n")
		}
	}

	if cycleID, ok := relations["cycle"]; ok {
		cycleURL := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/cycles/%s/cycle-issues/",
			client.BaseURL, client.Workspace, projectID, cycleID)
		if _, err := client.Post(ctx, cycleURL, payload); err != nil {
			Infof(deps, "Warning: issue created but failed to add to cycle: %v\n", err)
		} else {
			Infof(deps, "Added to cycle.\n")
		}
	}
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
	fmt.Fprintln(w, "  -n, --dry-run\t\tPrint request details without executing")
	fmt.Fprintln(w, "      --stdin\t\tRead JSON body from stdin (POST/PATCH/PUT)")
	fmt.Fprintln(w, "      --field\t\tExtract a single field (supports dotted paths)")
	fmt.Fprintln(w, "      --fields\t\tExtract multiple fields as TSV (comma-separated)")
	fmt.Fprintln(w, "      --id-only\t\tPrint only the resource ID (raw UUID, no newline)")
	fmt.Fprintln(w, "  -q, --quiet\t\tSuppress informational stderr messages")
	fmt.Fprintln(w, "      --strict\t\tTreat name-resolution failures as hard errors")
	fmt.Fprintln(w, "      --no-resolve\tSkip name-to-UUID resolution")
	fmt.Fprintln(w, "      --batch\t\tRead JSONL from stdin (POST/PATCH/PUT only)")
}

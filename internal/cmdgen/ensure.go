package cmdgen

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/mggarofalo/plane-cli/internal/api"
	"github.com/mggarofalo/plane-cli/internal/docs"
	"github.com/mggarofalo/plane-cli/internal/markdown"
	"github.com/spf13/cobra"
)

// ensureMatchField is the flag name for specifying which field to match on.
const ensureMatchField = "match-field"

// findSpecByMethod returns the first cached spec matching the given HTTP method.
func findSpecByMethod(specs []docs.CachedSpec, method string) *docs.EndpointSpec {
	for i := range specs {
		if specs[i].Spec.Method == method {
			return &specs[i].Spec
		}
	}
	return nil
}

// ensureSpecs groups the specs needed by an ensure command.
type ensureSpecs struct {
	create *docs.EndpointSpec
	update *docs.EndpointSpec
	list   *docs.EndpointSpec
}

// findEnsureSpecs locates the create, update, and list specs from cached specs.
// Returns nil if the minimum required specs (create + list) are not found.
func findEnsureSpecs(topicName string, cachedSpecs []docs.CachedSpec) *ensureSpecs {
	// For list, find a GET endpoint that doesn't require a resource-specific
	// path param (i.e., the collection endpoint, not a detail/get-by-id endpoint).
	listSpec := findListSpec(cachedSpecs)

	// For create, find the POST endpoint at the topic level.
	createSpec := findSpecByMethod(cachedSpecs, "POST")

	// For update, find PATCH or PUT.
	updateSpec := findSpecByMethod(cachedSpecs, "PATCH")
	if updateSpec == nil {
		updateSpec = findSpecByMethod(cachedSpecs, "PUT")
	}

	if createSpec == nil || listSpec == nil {
		return nil
	}

	return &ensureSpecs{
		create: createSpec,
		update: updateSpec,
		list:   listSpec,
	}
}

// findListSpec returns the GET spec that serves as the collection list endpoint.
// It prefers GET endpoints without resource-specific path params (i.e., endpoints
// that list all resources rather than getting a single one by ID).
func findListSpec(specs []docs.CachedSpec) *docs.EndpointSpec {
	var fallback *docs.EndpointSpec
	for i := range specs {
		if specs[i].Spec.Method != "GET" {
			continue
		}
		if fallback == nil {
			fallback = &specs[i].Spec
		}
		// A list endpoint has no resource-specific path params
		// (only workspace_slug and project_id).
		if !hasResourcePathParam(&specs[i].Spec) {
			return &specs[i].Spec
		}
	}
	return fallback
}

// hasResourcePathParam returns true if the spec has path params beyond
// workspace_slug and project_id (indicating it's a detail/single-resource endpoint).
func hasResourcePathParam(spec *docs.EndpointSpec) bool {
	for _, p := range spec.Params {
		if p.Location != docs.ParamPath {
			continue
		}
		if p.Name == "workspace_slug" || p.Name == "project_id" {
			continue
		}
		return true
	}
	return false
}

// BuildEnsureCommand creates the "ensure" subcommand for a topic.
// It uses the create spec's params for flags and adds --match-field.
func BuildEnsureCommand(topicName string, specs *ensureSpecs, deps *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ensure",
		Short: fmt.Sprintf("Create or update a %s (idempotent upsert)", topicName),
		Long: fmt.Sprintf(`Idempotent create-or-update for %s resources.

Matches an existing resource by the --match-field value (default: name).
If a match is found, updates it with the provided fields.
If no match is found, creates a new resource.
Returns the resource either way (same output as create/get).`, topicName),
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeEnsure(cmd.Context(), cmd, topicName, specs, deps)
		},
	}

	// Add --match-field flag
	cmd.Flags().String(ensureMatchField, "name", "Field to match existing resources on")

	// Register flags from the create spec's params
	for _, p := range specs.create.Params {
		if p.Name == "workspace_slug" || p.Name == "project_id" {
			continue
		}
		flagName := ParamToFlagName(p.Name)
		if globalFlagNames[flagName] {
			continue
		}
		desc := p.Description
		if desc == "" {
			desc = p.Name
		}

		if IsHTMLParam(p.Name) {
			mdFlag := MarkdownFlagName(p.Name)
			mdDesc := desc
			if mdDesc == p.Name {
				mdDesc = mdFlag
			}
			cmd.Flags().String(mdFlag, "", mdDesc+" (markdown)")
			cmd.Flags().String(flagName, "", mdDesc+" (raw HTML)")
			cmd.MarkFlagsMutuallyExclusive(mdFlag, flagName)
			continue
		}

		switch p.Type {
		case "string[]":
			cmd.Flags().StringSlice(flagName, nil, desc)
		case "number":
			cmd.Flags().Int(flagName, 0, desc)
		case "boolean":
			cmd.Flags().Bool(flagName, false, desc)
		default:
			cmd.Flags().String(flagName, "", desc)
		}
	}

	return cmd
}

// executeEnsure implements the ensure logic: list, match, create-or-update.
func executeEnsure(ctx context.Context, cmd *cobra.Command, topicName string, specs *ensureSpecs, deps *Deps) error {
	client, err := deps.NewClient()
	if err != nil {
		return err
	}

	if specs.create.RequiresWorkspace() {
		if err := deps.RequireWorkspace(client); err != nil {
			return err
		}
	}

	var projectID string
	if specs.create.RequiresProject() {
		projectID, err = deps.RequireProject()
		if err != nil {
			return err
		}
	}

	matchField, _ := cmd.Flags().GetString(ensureMatchField)
	if matchField == "" {
		matchField = "name"
	}

	// Collect body params from the create spec
	body := collectEnsureBodyParams(cmd, specs.create, deps)
	body = InjectGlobalBodyParams(body, specs.create, client.Workspace, projectID)

	// Ensure body is non-nil before field lookups
	if body == nil {
		body = map[string]any{}
	}

	// Determine the match value from the body
	matchValue, ok := body[matchField]
	if !ok {
		// Try with _id suffix and other common patterns
		matchValue, ok = body[matchField+"_id"]
		if ok {
			matchField = matchField + "_id"
		}
	}
	if !ok {
		return fmt.Errorf("match field %q not provided; set --%s or use --match-field to specify a different field",
			matchField, ParamToFlagName(matchField))
	}

	matchStr, isStr := matchValue.(string)
	if !isStr {
		return fmt.Errorf("match field %q value must be a string, got %T", matchField, matchValue)
	}

	// List existing resources to find a match
	listURL, err := buildEnsureListURL(client, specs.list, projectID)
	if err != nil {
		return fmt.Errorf("building list URL: %w", err)
	}

	// Dry-run: show what would be done
	if isDryRun(deps) {
		Infof(deps, "DRY RUN: ensure %s (match-field=%s, match-value=%s)\n", topicName, matchField, matchStr)
		Infof(deps, "  Would list: GET %s\n", listURL)
		Infof(deps, "  If match found: PATCH existing resource\n")
		Infof(deps, "  If no match: POST new resource\n")
		if body != nil {
			data, err := json.MarshalIndent(body, "  ", "  ")
			if err == nil {
				Infof(deps, "  Body: %s\n", data)
			}
		}
		return nil
	}

	existingID, err := findMatchInList(ctx, client, listURL, matchField, matchStr, deps)
	if err != nil {
		// If listing fails, fall through to create
		Infof(deps, "Warning: could not list existing %s resources: %v; attempting create\n", topicName, err)
		existingID = ""
	}

	if existingID != "" {
		// Update path
		return executeEnsureUpdate(ctx, cmd, client, topicName, existingID, body, specs, projectID, deps)
	}

	// Create path
	return executeEnsureCreate(ctx, cmd, client, topicName, body, specs, projectID, deps)
}

// findMatchInList fetches all pages from the list endpoint and returns the ID
// of the first resource whose matchField equals matchValue (case-insensitive).
func findMatchInList(ctx context.Context, client *api.Client, listURL, matchField, matchValue string, deps *Deps) (string, error) {
	lowerMatch := strings.ToLower(matchValue)
	cursor := ""

	for {
		u, err := url.Parse(listURL)
		if err != nil {
			return "", err
		}
		q := u.Query()
		q.Set("per_page", "100")
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		u.RawQuery = q.Encode()

		respBody, err := client.Get(ctx, u.String())
		if err != nil {
			return "", err
		}

		id, nextCursor, err := searchPageForMatch(respBody, matchField, lowerMatch)
		if err == nil && id != "" {
			return id, nil
		}

		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	return "", nil
}

// searchPageForMatch searches a single API response page for a matching resource.
// Returns the matched resource ID, the next cursor (if any), and an error.
func searchPageForMatch(respBody []byte, matchField, lowerMatch string) (id, nextCursor string, err error) {
	// Try as paginated response
	var paginated struct {
		Results         json.RawMessage `json:"results"`
		NextPageResults bool            `json:"next_page_results"`
		NextCursor      string          `json:"next_cursor"`
	}
	if err := json.Unmarshal(respBody, &paginated); err == nil && len(paginated.Results) > 0 {
		var items []json.RawMessage
		if err := json.Unmarshal(paginated.Results, &items); err == nil {
			for _, raw := range items {
				if id := matchItem(raw, matchField, lowerMatch); id != "" {
					return id, "", nil
				}
			}
		}
		if paginated.NextPageResults {
			return "", paginated.NextCursor, nil
		}
		return "", "", nil
	}

	// Try as plain array
	var items []json.RawMessage
	if err := json.Unmarshal(respBody, &items); err == nil {
		for _, raw := range items {
			if id := matchItem(raw, matchField, lowerMatch); id != "" {
				return id, "", nil
			}
		}
	}

	return "", "", nil
}

// matchItem checks if a single JSON object's matchField equals the lowerMatch value.
func matchItem(raw json.RawMessage, matchField, lowerMatch string) string {
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return ""
	}

	id, _ := obj["id"].(string)
	if id == "" {
		return ""
	}

	val, ok := obj[matchField]
	if !ok {
		return ""
	}

	strVal, ok := val.(string)
	if !ok {
		return ""
	}

	if strings.ToLower(strVal) == lowerMatch {
		return id
	}

	return ""
}

// executeEnsureCreate runs the create path of ensure.
func executeEnsureCreate(ctx context.Context, _ *cobra.Command, client *api.Client, topicName string, body map[string]any, specs *ensureSpecs, projectID string, deps *Deps) error {
	createURL, err := buildEnsureURL(client, specs.create, projectID)
	if err != nil {
		return fmt.Errorf("building create URL: %w", err)
	}

	// Snapshot body before relation extraction
	snapshot := snapshotBody(body)
	_ = snapshot // keep for potential dry-run logging

	// Extract relation params for POST
	relations := ExtractRelationParams(body)

	if body == nil {
		body = map[string]any{}
	}

	respBody, err := client.Post(ctx, createURL, body)
	if err != nil {
		return err
	}

	Infof(deps, "Created.\n")

	if len(respBody) == 0 {
		return nil
	}

	if err := formatResponse(respBody, deps); err != nil {
		return err
	}

	// Handle post-creation actions for many-to-many relations
	if len(relations) > 0 {
		PostCreateActions(ctx, relations, respBody, client, projectID, deps)
	}

	return nil
}

// executeEnsureUpdate runs the update path of ensure.
func executeEnsureUpdate(ctx context.Context, _ *cobra.Command, client *api.Client, topicName, existingID string, body map[string]any, specs *ensureSpecs, projectID string, deps *Deps) error {
	if specs.update == nil {
		// No update spec available; return the existing resource via GET
		Infof(deps, "Found existing %s (%s) but no update endpoint available; returning as-is.\n", topicName, existingID)
		return ensureGetExisting(ctx, client, specs.list, existingID, projectID, deps)
	}

	updateURL, err := buildEnsureUpdateURL(client, specs.update, existingID, projectID)
	if err != nil {
		return fmt.Errorf("building update URL: %w", err)
	}

	// For update, only include fields that were explicitly set by the user.
	// The body already contains only changed fields from collectEnsureBodyParams.
	// Remove the match field value if it's the only thing to avoid a no-op update.
	if body == nil {
		body = map[string]any{}
	}

	var respBody []byte
	if specs.update.Method == "PUT" {
		respBody, err = client.Put(ctx, updateURL, body)
	} else {
		respBody, err = client.Patch(ctx, updateURL, body)
	}
	if err != nil {
		return err
	}

	Infof(deps, "Updated.\n")

	if len(respBody) == 0 {
		return nil
	}

	return formatResponse(respBody, deps)
}

// ensureGetExisting fetches an existing resource by ID when no update endpoint
// is available. It uses the list spec's path template to build a GET URL.
func ensureGetExisting(ctx context.Context, client *api.Client, listSpec *docs.EndpointSpec, resourceID, projectID string, deps *Deps) error {
	// Construct get URL from list URL + resourceID
	getURL, err := buildEnsureURL(client, listSpec, projectID)
	if err != nil {
		return err
	}
	// Append resource ID to the list URL to get the detail endpoint
	getURL = strings.TrimRight(getURL, "/") + "/" + resourceID + "/"

	respBody, err := client.Get(ctx, getURL)
	if err != nil {
		return err
	}

	if len(respBody) == 0 {
		return nil
	}

	return formatResponse(respBody, deps)
}

// buildEnsureURL constructs a URL from a spec's path template, substituting
// workspace and project IDs.
func buildEnsureURL(client *api.Client, spec *docs.EndpointSpec, projectID string) (string, error) {
	path := spec.PathTemplate
	path = strings.ReplaceAll(path, "{workspace_slug}", client.Workspace)
	if projectID != "" {
		path = strings.ReplaceAll(path, "{project_id}", projectID)
	}
	return client.BaseURL + path, nil
}

// buildEnsureListURL constructs the list URL for the ensure command.
func buildEnsureListURL(client *api.Client, listSpec *docs.EndpointSpec, projectID string) (string, error) {
	return buildEnsureURL(client, listSpec, projectID)
}

// buildEnsureUpdateURL constructs the update URL, substituting the resource ID
// into the path template.
func buildEnsureUpdateURL(client *api.Client, updateSpec *docs.EndpointSpec, resourceID, projectID string) (string, error) {
	path := updateSpec.PathTemplate
	path = strings.ReplaceAll(path, "{workspace_slug}", client.Workspace)
	if projectID != "" {
		path = strings.ReplaceAll(path, "{project_id}", projectID)
	}

	// Replace any remaining path param placeholders with the resource ID.
	// Update specs typically have one path param beyond workspace/project (e.g., {work_item_id}, {state_id}).
	for _, p := range updateSpec.Params {
		if p.Location != docs.ParamPath {
			continue
		}
		if p.Name == "workspace_slug" || p.Name == "project_id" {
			continue
		}
		placeholder := "{" + p.Name + "}"
		if strings.Contains(path, placeholder) {
			path = strings.ReplaceAll(path, placeholder, resourceID)
		}
	}

	return client.BaseURL + path, nil
}

// collectEnsureBodyParams collects body parameters from cobra flags for the ensure
// command, using the create spec's params as the schema.
func collectEnsureBodyParams(cmd *cobra.Command, createSpec *docs.EndpointSpec, deps *Deps) map[string]any {
	body := map[string]any{}
	for _, p := range createSpec.Params {
		if p.Location != docs.ParamBody {
			continue
		}

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
					val, _ = resolveSliceIfNeeded(cmd.Context(), val, p.Name, deps)
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
				val, _ = resolveIfNeeded(cmd.Context(), val, p.Name, nil, "", deps)
				body[p.Name] = val
			}
		}
	}

	if len(body) == 0 {
		return nil
	}
	return body
}

// shouldAddEnsure checks if a topic should get an ensure subcommand.
// Topics that don't have meaningful upsert semantics are excluded.
var ensureExcludedTopics = map[string]bool{
	"activity":   true, // read-only
	"comment":    true, // identified by content, not name
	"attachment": true, // binary uploads
	"link":       true, // identified by URL, not name
	"worklog":    true, // time entries, no unique name
	"epic":       true, // read-only in API
	"page":       true, // complex structure
	"member":     true, // read-only listing
}

// TopicSupportsEnsure returns true if the topic should have an ensure subcommand.
func TopicSupportsEnsure(topicName string) bool {
	return !ensureExcludedTopics[topicName]
}

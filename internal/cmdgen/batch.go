package cmdgen

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mggarofalo/plane-cli/internal/api"
	"github.com/mggarofalo/plane-cli/internal/docs"
)

// isBatch returns true when the batch flag is set. Nil-safe.
func isBatch(deps *Deps) bool {
	return deps != nil && deps.FlagBatch != nil && *deps.FlagBatch
}

// batchResult represents the outcome of a single batch item.
type batchResult struct {
	// Success holds the API response for successful requests.
	Success json.RawMessage `json:"result,omitempty"`
	// Error holds the error details for failed requests.
	Error *batchError `json:"error,omitempty"`
}

// batchError is the error portion of a batch result line.
type batchError struct {
	Message string `json:"message"`
	Code    int    `json:"code,omitempty"`
}

// ExecuteBatch reads JSONL from stdin, executes each line as a separate API
// request using the given spec, and writes one JSON response per line to stdout.
// Errors for individual items are reported inline (never abort the batch).
// A summary is written to stderr at the end.
func ExecuteBatch(ctx context.Context, spec *docs.EndpointSpec, deps *Deps) error {
	if spec.Method == "GET" || spec.Method == "DELETE" {
		return fmt.Errorf("--batch is only supported for POST, PATCH, and PUT methods (got %s)", spec.Method)
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

	// Build the URL template once (path params from global context only)
	reqURL, err := buildBatchURL(client, spec, projectID)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(os.Stdin)
	// Allow up to 1MB per line
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	succeeded := 0
	failed := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		result := executeBatchLine(ctx, client, spec, reqURL, line, projectID, deps)
		if result.Error != nil {
			failed++
		} else {
			succeeded++
		}

		// Write one JSON object per line to stdout
		out, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			// This should never happen, but handle gracefully.
			// Adjust counts: the result was already counted above,
			// so only fix if it was originally a success.
			if result.Error == nil {
				succeeded--
				failed++
			}
			fmt.Fprintf(os.Stdout, `{"error":{"message":"marshal error: %s"}}%s`, marshalErr.Error(), "\n")
			continue
		}
		fmt.Fprintf(os.Stdout, "%s\n", out)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}

	// Summary to stderr. When failed > 0, the returned batchSummaryError is
	// also printed by the root command's error handler, so only emit the
	// summary here for the all-success case (where no error is returned).
	if failed > 0 {
		return &batchSummaryError{succeeded: succeeded, failed: failed}
	}
	fmt.Fprintf(os.Stderr, "%d succeeded, %d failed\n", succeeded, failed)
	return nil
}

// executeBatchLine processes a single JSONL line and returns the result.
func executeBatchLine(ctx context.Context, client *api.Client, spec *docs.EndpointSpec, reqURL, line string, projectID string, deps *Deps) batchResult {
	// Parse the JSON line into a body map
	var body map[string]any
	if err := json.Unmarshal([]byte(line), &body); err != nil {
		return batchResult{Error: &batchError{Message: fmt.Sprintf("invalid JSON: %s", err.Error())}}
	}

	// Inject global body params
	body = InjectGlobalBodyParams(body, spec, client.Workspace, projectID)

	// Execute the request
	var respBody []byte
	var err error
	switch spec.Method {
	case "POST":
		respBody, err = client.Post(ctx, reqURL, body)
	case "PATCH":
		respBody, err = client.Patch(ctx, reqURL, body)
	case "PUT":
		respBody, err = client.Put(ctx, reqURL, body)
	default:
		return batchResult{Error: &batchError{Message: fmt.Sprintf("unsupported method: %s", spec.Method)}}
	}

	if err != nil {
		be := &batchError{Message: err.Error()}
		if apiErr, ok := err.(*api.APIError); ok {
			be.Code = apiErr.StatusCode
		}
		return batchResult{Error: be}
	}

	if len(respBody) == 0 {
		// Success with no body (e.g., 204)
		return batchResult{Success: json.RawMessage(`{}`)}
	}

	return batchResult{Success: json.RawMessage(respBody)}
}

// buildBatchURL constructs the request URL for batch mode. Unlike the normal
// buildURL, it only substitutes global path params (workspace_slug, project_id)
// and does not handle per-request path params. Batch mode requires that the URL
// be fully determined from global context.
func buildBatchURL(client *api.Client, spec *docs.EndpointSpec, projectID string) (string, error) {
	path := spec.PathTemplate
	path = strings.ReplaceAll(path, "{workspace_slug}", client.Workspace)
	if projectID != "" {
		path = strings.ReplaceAll(path, "{project_id}", projectID)
	}

	// Check for unresolved path params — batch mode does not support per-line path params
	if strings.Contains(path, "{") {
		return "", fmt.Errorf("--batch requires all path parameters to be provided via global flags; unresolved: %s", path)
	}

	return client.BaseURL + path, nil
}

// batchSummaryError is returned when some batch items failed. It allows the
// caller to distinguish partial failures from total failures.
type batchSummaryError struct {
	succeeded int
	failed    int
}

func (e *batchSummaryError) Error() string {
	return fmt.Sprintf("%d succeeded, %d failed", e.succeeded, e.failed)
}

// ExitCode returns exit code 1 for partial batch failures.
func (e *batchSummaryError) ExitCode() int {
	return api.ExitGeneralError
}

// IsBatchCompatibleMethod returns true if the HTTP method supports batch mode.
func IsBatchCompatibleMethod(method string) bool {
	switch method {
	case "POST", "PATCH", "PUT":
		return true
	default:
		return false
	}
}

// WriteBatchHelp appends batch mode documentation to the help writer.
func WriteBatchHelp(w io.Writer) {
	fmt.Fprintln(w, "Batch mode (--batch):")
	fmt.Fprintln(w, "  Read JSONL from stdin, one JSON object per line.")
	fmt.Fprintln(w, "  Each line is sent as a separate request body.")
	fmt.Fprintln(w, "  Output: one JSON result per line (JSONL).")
	fmt.Fprintln(w, "  Errors are reported inline; the batch is never aborted.")
	fmt.Fprintln(w, "  Summary (N succeeded, M failed) is written to stderr.")
	fmt.Fprintln(w)
}

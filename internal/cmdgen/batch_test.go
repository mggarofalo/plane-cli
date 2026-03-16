package cmdgen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/mggarofalo/plane-cli/internal/api"
	"github.com/mggarofalo/plane-cli/internal/docs"
	"github.com/mggarofalo/plane-cli/internal/output"
)

// setupBatchTest creates a test server, deps, and spec for batch tests.
func setupBatchTest(handler http.HandlerFunc) (*httptest.Server, *Deps, *docs.EndpointSpec) {
	srv := httptest.NewServer(handler)

	batch := true
	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		RequireProject:   func() (string, error) { return "proj-uuid", nil },
		PaginationParams: func() api.PaginationParams { return api.PaginationParams{PerPage: 100} },
		Formatter:        func() output.Formatter { return output.New("json") },
		IsUUID:           func(s string) bool { return len(s) == 36 && s[8] == '-' },
		FlagBatch:        &batch,
	}

	spec := &docs.EndpointSpec{
		Method:       "POST",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/",
		EntryTitle:   "Create Work Item",
		Params: []docs.ParamSpec{
			{Name: "name", Location: docs.ParamBody, Type: "string"},
		},
	}

	return srv, deps, spec
}

// withStdin replaces os.Stdin with a reader containing the given input,
// runs the function, and restores os.Stdin.
func withStdin(input string, fn func()) {
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	go func() {
		_, _ = io.WriteString(w, input)
		_ = w.Close()
	}()
	defer func() { os.Stdin = oldStdin }()
	fn()
}

// captureStdout captures stdout output during fn execution.
func captureStdout(fn func()) string {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

func TestBatchMode_AllSucceed(t *testing.T) {
	var requestCount int32
	srv, deps, spec := setupBatchTest(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&requestCount, 1)
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, `{"id": "uuid-%d", "name": %q}`, n, body["name"])
	})
	defer srv.Close()

	input := `{"name": "Issue One"}
{"name": "Issue Two"}
{"name": "Issue Three"}
`

	var stdoutOutput string
	withStdin(input, func() {
		stdoutOutput = captureStdout(func() {
			err := ExecuteBatch(context.Background(), spec, deps)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})

	// Verify 3 requests were made
	if atomic.LoadInt32(&requestCount) != 3 {
		t.Errorf("expected 3 requests, got %d", requestCount)
	}

	// Verify JSONL output
	lines := strings.Split(strings.TrimSpace(stdoutOutput), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 output lines, got %d: %q", len(lines), stdoutOutput)
	}

	for i, line := range lines {
		var result batchResult
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			t.Fatalf("line %d: invalid JSON: %v", i+1, err)
		}
		if result.Error != nil {
			t.Errorf("line %d: unexpected error: %s", i+1, result.Error.Message)
		}
		if result.Success == nil {
			t.Errorf("line %d: expected success result", i+1)
		}
	}
}

func TestBatchMode_MixedSuccessAndFailure(t *testing.T) {
	srv, deps, spec := setupBatchTest(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		name, _ := body["name"].(string)
		if name == "Bad Issue" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"detail": "name is invalid"}`)
			return
		}
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, `{"id": "new-uuid", "name": %q}`, name)
	})
	defer srv.Close()

	input := `{"name": "Good Issue"}
{"name": "Bad Issue"}
{"name": "Another Good"}
`

	var stdoutOutput string
	var batchErr error
	withStdin(input, func() {
		stdoutOutput = captureStdout(func() {
			batchErr = ExecuteBatch(context.Background(), spec, deps)
		})
	})

	// Should return a batchSummaryError since 1 failed
	if batchErr == nil {
		t.Fatal("expected error for partial failure, got nil")
	}
	summaryErr, ok := batchErr.(*batchSummaryError)
	if !ok {
		t.Fatalf("expected *batchSummaryError, got %T: %v", batchErr, batchErr)
	}
	if summaryErr.succeeded != 2 {
		t.Errorf("expected 2 succeeded, got %d", summaryErr.succeeded)
	}
	if summaryErr.failed != 1 {
		t.Errorf("expected 1 failed, got %d", summaryErr.failed)
	}

	// Verify output lines
	lines := strings.Split(strings.TrimSpace(stdoutOutput), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 output lines, got %d", len(lines))
	}

	// Line 1: success
	var r1 batchResult
	_ = json.Unmarshal([]byte(lines[0]), &r1)
	if r1.Error != nil {
		t.Errorf("line 1: expected success, got error: %s", r1.Error.Message)
	}

	// Line 2: error
	var r2 batchResult
	_ = json.Unmarshal([]byte(lines[1]), &r2)
	if r2.Error == nil {
		t.Error("line 2: expected error, got success")
	} else if r2.Error.Code != 400 {
		t.Errorf("line 2: expected code 400, got %d", r2.Error.Code)
	}

	// Line 3: success
	var r3 batchResult
	_ = json.Unmarshal([]byte(lines[2]), &r3)
	if r3.Error != nil {
		t.Errorf("line 3: expected success, got error: %s", r3.Error.Message)
	}
}

func TestBatchMode_InvalidJSON(t *testing.T) {
	srv, deps, spec := setupBatchTest(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not make request for invalid JSON line")
	})
	defer srv.Close()

	input := `not valid json
`

	var stdoutOutput string
	var batchErr error
	withStdin(input, func() {
		stdoutOutput = captureStdout(func() {
			batchErr = ExecuteBatch(context.Background(), spec, deps)
		})
	})

	// Should return error (1 failed)
	if batchErr == nil {
		t.Fatal("expected error, got nil")
	}

	lines := strings.Split(strings.TrimSpace(stdoutOutput), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 output line, got %d", len(lines))
	}

	var result batchResult
	_ = json.Unmarshal([]byte(lines[0]), &result)
	if result.Error == nil {
		t.Fatal("expected error result")
	}
	if !strings.Contains(result.Error.Message, "invalid JSON") {
		t.Errorf("expected 'invalid JSON' in message, got: %s", result.Error.Message)
	}
}

func TestBatchMode_EmptyLines(t *testing.T) {
	var requestCount int32
	srv, deps, spec := setupBatchTest(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "uuid-1", "name": "test"}`)
	})
	defer srv.Close()

	// Empty lines and whitespace-only lines should be skipped
	input := `
{"name": "Issue One"}


{"name": "Issue Two"}

`

	var stdoutOutput string
	withStdin(input, func() {
		stdoutOutput = captureStdout(func() {
			err := ExecuteBatch(context.Background(), spec, deps)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})

	if atomic.LoadInt32(&requestCount) != 2 {
		t.Errorf("expected 2 requests (empty lines skipped), got %d", requestCount)
	}

	lines := strings.Split(strings.TrimSpace(stdoutOutput), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 output lines, got %d", len(lines))
	}
}

func TestBatchMode_EmptyInput(t *testing.T) {
	srv, deps, spec := setupBatchTest(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not make any requests for empty input")
	})
	defer srv.Close()

	input := ""

	withStdin(input, func() {
		_ = captureStdout(func() {
			err := ExecuteBatch(context.Background(), spec, deps)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})
}

func TestBatchMode_RejectsGET(t *testing.T) {
	srv, deps, _ := setupBatchTest(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not make any requests")
	})
	defer srv.Close()

	spec := &docs.EndpointSpec{
		Method:       "GET",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/",
		EntryTitle:   "List Work Items",
	}

	err := ExecuteBatch(context.Background(), spec, deps)
	if err == nil {
		t.Fatal("expected error for GET method, got nil")
	}
	if !strings.Contains(err.Error(), "only supported for POST, PATCH, and PUT") {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestBatchMode_RejectsDELETE(t *testing.T) {
	srv, deps, _ := setupBatchTest(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not make any requests")
	})
	defer srv.Close()

	spec := &docs.EndpointSpec{
		Method:       "DELETE",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/{work_item_id}/",
		EntryTitle:   "Delete Work Item",
	}

	err := ExecuteBatch(context.Background(), spec, deps)
	if err == nil {
		t.Fatal("expected error for DELETE method, got nil")
	}
}

func TestBatchMode_PATCH(t *testing.T) {
	var methodReceived string
	srv, deps, _ := setupBatchTest(func(w http.ResponseWriter, r *http.Request) {
		methodReceived = r.Method
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"id": "uuid-1", "name": "updated"}`)
	})
	defer srv.Close()

	spec := &docs.EndpointSpec{
		Method:       "PATCH",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/",
		EntryTitle:   "Update Work Item",
	}

	input := `{"name": "updated"}
`

	withStdin(input, func() {
		_ = captureStdout(func() {
			err := ExecuteBatch(context.Background(), spec, deps)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})

	if methodReceived != "PATCH" {
		t.Errorf("expected PATCH method, got %s", methodReceived)
	}
}

func TestBatchMode_PUT(t *testing.T) {
	var methodReceived string
	srv, deps, _ := setupBatchTest(func(w http.ResponseWriter, r *http.Request) {
		methodReceived = r.Method
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"id": "uuid-1", "name": "replaced"}`)
	})
	defer srv.Close()

	spec := &docs.EndpointSpec{
		Method:       "PUT",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/",
		EntryTitle:   "Replace Work Item",
	}

	input := `{"name": "replaced"}
`

	withStdin(input, func() {
		_ = captureStdout(func() {
			err := ExecuteBatch(context.Background(), spec, deps)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})

	if methodReceived != "PUT" {
		t.Errorf("expected PUT method, got %s", methodReceived)
	}
}

func TestBatchMode_UnresolvedPathParams(t *testing.T) {
	srv, deps, _ := setupBatchTest(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not make any requests")
	})
	defer srv.Close()

	spec := &docs.EndpointSpec{
		Method:       "PATCH",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/{work_item_id}/",
		EntryTitle:   "Update Work Item",
		Params: []docs.ParamSpec{
			{Name: "work_item_id", Location: docs.ParamPath, Type: "string", Required: true},
		},
	}

	input := `{"name": "test"}
`

	var batchErr error
	withStdin(input, func() {
		_ = captureStdout(func() {
			batchErr = ExecuteBatch(context.Background(), spec, deps)
		})
	})

	if batchErr == nil {
		t.Fatal("expected error for unresolved path params, got nil")
	}
	if !strings.Contains(batchErr.Error(), "unresolved") {
		t.Errorf("expected 'unresolved' in error message, got: %s", batchErr.Error())
	}
}

func TestBatchMode_InjectsGlobalBodyParams(t *testing.T) {
	var receivedBody map[string]any
	srv, deps, _ := setupBatchTest(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "uuid-1"}`)
	})
	defer srv.Close()

	spec := &docs.EndpointSpec{
		Method:       "POST",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/",
		EntryTitle:   "Create Work Item",
		Params: []docs.ParamSpec{
			{Name: "name", Location: docs.ParamBody, Type: "string"},
			{Name: "project_id", Location: docs.ParamBody, Type: "string"},
		},
	}

	input := `{"name": "Test Issue"}
`

	withStdin(input, func() {
		_ = captureStdout(func() {
			err := ExecuteBatch(context.Background(), spec, deps)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})

	// project_id should be injected into the body
	if receivedBody["project_id"] != "proj-uuid" {
		t.Errorf("expected project_id=proj-uuid, got %v", receivedBody["project_id"])
	}
}

func TestBatchMode_SummaryErrorExitCode(t *testing.T) {
	err := &batchSummaryError{succeeded: 5, failed: 2}
	if err.ExitCode() != api.ExitGeneralError {
		t.Errorf("expected exit code %d, got %d", api.ExitGeneralError, err.ExitCode())
	}
	want := "5 succeeded, 2 failed"
	if err.Error() != want {
		t.Errorf("expected %q, got %q", want, err.Error())
	}
}

func TestIsBatch(t *testing.T) {
	t.Run("returns false when nil", func(t *testing.T) {
		deps := &Deps{}
		if isBatch(deps) {
			t.Error("expected false for nil FlagBatch")
		}
	})

	t.Run("returns false when false", func(t *testing.T) {
		f := false
		deps := &Deps{FlagBatch: &f}
		if isBatch(deps) {
			t.Error("expected false")
		}
	})

	t.Run("returns true when true", func(t *testing.T) {
		f := true
		deps := &Deps{FlagBatch: &f}
		if !isBatch(deps) {
			t.Error("expected true")
		}
	})

	t.Run("returns false when deps is nil", func(t *testing.T) {
		if isBatch(nil) {
			t.Error("expected false for nil deps")
		}
	})
}

func TestIsBatchCompatibleMethod(t *testing.T) {
	tests := []struct {
		method string
		want   bool
	}{
		{"POST", true},
		{"PATCH", true},
		{"PUT", true},
		{"GET", false},
		{"DELETE", false},
	}

	for _, tc := range tests {
		got := IsBatchCompatibleMethod(tc.method)
		if got != tc.want {
			t.Errorf("IsBatchCompatibleMethod(%q) = %v, want %v", tc.method, got, tc.want)
		}
	}
}

func TestBatchMode_DoesNotAbortOnError(t *testing.T) {
	// Verify that a failure on line 2 does not prevent line 3 from executing
	var requestCount int32
	srv, deps, spec := setupBatchTest(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&requestCount, 1)
		if n == 2 {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"detail": "server error"}`)
			return
		}
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, `{"id": "uuid-%d"}`, n)
	})
	defer srv.Close()

	input := `{"name": "First"}
{"name": "Second"}
{"name": "Third"}
`

	var stdoutOutput string
	withStdin(input, func() {
		stdoutOutput = captureStdout(func() {
			_ = ExecuteBatch(context.Background(), spec, deps)
		})
	})

	// All 3 requests should have been made
	if atomic.LoadInt32(&requestCount) != 3 {
		t.Errorf("expected 3 requests, got %d", requestCount)
	}

	lines := strings.Split(strings.TrimSpace(stdoutOutput), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 output lines, got %d", len(lines))
	}

	// Line 1: success
	var r1 batchResult
	_ = json.Unmarshal([]byte(lines[0]), &r1)
	if r1.Error != nil {
		t.Errorf("line 1: expected success")
	}

	// Line 2: error
	var r2 batchResult
	_ = json.Unmarshal([]byte(lines[1]), &r2)
	if r2.Error == nil {
		t.Error("line 2: expected error")
	}

	// Line 3: success (proving batch was not aborted)
	var r3 batchResult
	_ = json.Unmarshal([]byte(lines[2]), &r3)
	if r3.Error != nil {
		t.Errorf("line 3: expected success (batch should not abort on error)")
	}
}

func TestBatchMode_ExecuteSpecDispatch(t *testing.T) {
	// Verify that ExecuteSpecFromArgs dispatches to batch mode when flag is set
	var requestCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "test-uuid"}`)
	}))
	defer srv.Close()

	batch := true
	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		RequireProject:   func() (string, error) { return "proj-uuid", nil },
		PaginationParams: func() api.PaginationParams { return api.PaginationParams{PerPage: 100} },
		Formatter:        func() output.Formatter { return output.New("json") },
		IsUUID:           func(s string) bool { return len(s) == 36 && s[8] == '-' },
		FlagBatch:        &batch,
	}

	spec := &docs.EndpointSpec{
		Method:       "POST",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/",
		EntryTitle:   "Create Work Item",
	}

	parsed := &ParsedArgs{
		Values: map[string]string{},
		Slices: map[string][]string{},
	}

	input := `{"name": "From Args"}
`

	withStdin(input, func() {
		_ = captureStdout(func() {
			err := ExecuteSpecFromArgs(context.Background(), spec, parsed, deps)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})

	if atomic.LoadInt32(&requestCount) != 1 {
		t.Errorf("expected 1 request via batch dispatch, got %d", requestCount)
	}
}

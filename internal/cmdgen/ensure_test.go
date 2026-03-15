package cmdgen

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mggarofalo/plane-cli/internal/api"
	"github.com/mggarofalo/plane-cli/internal/docs"
	"github.com/mggarofalo/plane-cli/internal/output"
)

func TestFindEnsureSpecs(t *testing.T) {
	t.Run("returns specs when create and list exist", func(t *testing.T) {
		cached := []docs.CachedSpec{
			{Spec: docs.EndpointSpec{Method: "GET", PathTemplate: "/api/v1/states/", EntryTitle: "List States"}},
			{Spec: docs.EndpointSpec{Method: "POST", PathTemplate: "/api/v1/states/", EntryTitle: "Create State"}},
			{Spec: docs.EndpointSpec{Method: "PATCH", PathTemplate: "/api/v1/states/{state_id}/", EntryTitle: "Update State"}},
		}

		specs := findEnsureSpecs("state", cached)
		if specs == nil {
			t.Fatal("expected specs, got nil")
		}
		if specs.create == nil {
			t.Error("expected create spec")
		}
		if specs.list == nil {
			t.Error("expected list spec")
		}
		if specs.update == nil {
			t.Error("expected update spec")
		}
	})

	t.Run("returns nil when no create spec", func(t *testing.T) {
		cached := []docs.CachedSpec{
			{Spec: docs.EndpointSpec{Method: "GET", PathTemplate: "/api/v1/states/"}},
		}

		specs := findEnsureSpecs("state", cached)
		if specs != nil {
			t.Error("expected nil when no create spec")
		}
	})

	t.Run("returns nil when no list spec", func(t *testing.T) {
		cached := []docs.CachedSpec{
			{Spec: docs.EndpointSpec{Method: "POST", PathTemplate: "/api/v1/states/"}},
		}

		specs := findEnsureSpecs("state", cached)
		if specs != nil {
			t.Error("expected nil when no list spec")
		}
	})

	t.Run("returns specs without update (update is optional)", func(t *testing.T) {
		cached := []docs.CachedSpec{
			{Spec: docs.EndpointSpec{Method: "GET", PathTemplate: "/api/v1/states/"}},
			{Spec: docs.EndpointSpec{Method: "POST", PathTemplate: "/api/v1/states/"}},
		}

		specs := findEnsureSpecs("state", cached)
		if specs == nil {
			t.Fatal("expected specs, got nil")
		}
		if specs.update != nil {
			t.Error("expected nil update spec")
		}
	})

	t.Run("prefers PATCH over PUT for update", func(t *testing.T) {
		cached := []docs.CachedSpec{
			{Spec: docs.EndpointSpec{Method: "GET", PathTemplate: "/api/v1/states/"}},
			{Spec: docs.EndpointSpec{Method: "POST", PathTemplate: "/api/v1/states/"}},
			{Spec: docs.EndpointSpec{Method: "PATCH", PathTemplate: "/api/v1/states/{state_id}/", EntryTitle: "Update via PATCH"}},
			{Spec: docs.EndpointSpec{Method: "PUT", PathTemplate: "/api/v1/states/{state_id}/", EntryTitle: "Update via PUT"}},
		}

		specs := findEnsureSpecs("state", cached)
		if specs == nil {
			t.Fatal("expected specs")
		}
		if specs.update.Method != "PATCH" {
			t.Errorf("expected PATCH, got %s", specs.update.Method)
		}
	})
}

func TestSearchPageForMatch(t *testing.T) {
	t.Run("matches by name in paginated response", func(t *testing.T) {
		resp := `{
			"results": [
				{"id": "aaa", "name": "Alpha"},
				{"id": "bbb", "name": "Beta"},
				{"id": "ccc", "name": "Gamma"}
			],
			"next_page_results": false
		}`

		id, cursor, err := searchPageForMatch([]byte(resp), "name", "beta")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "bbb" {
			t.Errorf("expected bbb, got %s", id)
		}
		if cursor != "" {
			t.Errorf("expected empty cursor, got %s", cursor)
		}
	})

	t.Run("returns next cursor when no match and more pages", func(t *testing.T) {
		resp := `{
			"results": [
				{"id": "aaa", "name": "Alpha"}
			],
			"next_page_results": true,
			"next_cursor": "100:1:1"
		}`

		id, cursor, err := searchPageForMatch([]byte(resp), "name", "zeta")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "" {
			t.Errorf("expected empty id, got %s", id)
		}
		if cursor != "100:1:1" {
			t.Errorf("expected cursor 100:1:1, got %s", cursor)
		}
	})

	t.Run("matches in plain array", func(t *testing.T) {
		resp := `[
			{"id": "x1", "name": "Foo"},
			{"id": "x2", "name": "Bar"}
		]`

		id, _, err := searchPageForMatch([]byte(resp), "name", "bar")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "x2" {
			t.Errorf("expected x2, got %s", id)
		}
	})

	t.Run("matches by custom field", func(t *testing.T) {
		resp := `{
			"results": [
				{"id": "s1", "name": "Backlog", "group": "backlog"},
				{"id": "s2", "name": "In Progress", "group": "started"}
			]
		}`

		id, _, err := searchPageForMatch([]byte(resp), "group", "started")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "s2" {
			t.Errorf("expected s2, got %s", id)
		}
	})

	t.Run("no match returns empty", func(t *testing.T) {
		resp := `{"results": [{"id": "a1", "name": "Alpha"}]}`

		id, _, _ := searchPageForMatch([]byte(resp), "name", "nonexistent")
		if id != "" {
			t.Errorf("expected empty, got %s", id)
		}
	})
}

func TestMatchItem(t *testing.T) {
	t.Run("matches case-insensitively", func(t *testing.T) {
		raw := json.RawMessage(`{"id": "123", "name": "Backlog"}`)
		id := matchItem(raw, "name", "backlog")
		if id != "123" {
			t.Errorf("expected 123, got %s", id)
		}
	})

	t.Run("returns empty for no match", func(t *testing.T) {
		raw := json.RawMessage(`{"id": "123", "name": "Backlog"}`)
		id := matchItem(raw, "name", "done")
		if id != "" {
			t.Errorf("expected empty, got %s", id)
		}
	})

	t.Run("returns empty for missing field", func(t *testing.T) {
		raw := json.RawMessage(`{"id": "123", "name": "Backlog"}`)
		id := matchItem(raw, "color", "red")
		if id != "" {
			t.Errorf("expected empty, got %s", id)
		}
	})

	t.Run("returns empty for missing id", func(t *testing.T) {
		raw := json.RawMessage(`{"name": "Backlog"}`)
		id := matchItem(raw, "name", "backlog")
		if id != "" {
			t.Errorf("expected empty, got %s", id)
		}
	})

	t.Run("returns empty for invalid JSON", func(t *testing.T) {
		raw := json.RawMessage(`not json`)
		id := matchItem(raw, "name", "test")
		if id != "" {
			t.Errorf("expected empty, got %s", id)
		}
	})

	t.Run("returns empty for non-string match field", func(t *testing.T) {
		raw := json.RawMessage(`{"id": "123", "count": 42}`)
		id := matchItem(raw, "count", "42")
		if id != "" {
			t.Errorf("expected empty, got %s", id)
		}
	})
}

func TestTopicSupportsEnsure(t *testing.T) {
	supported := []string{"issue", "label", "state", "cycle", "module", "project",
		"customer", "teamspace", "sticky", "initiative", "intake"}
	for _, topic := range supported {
		t.Run("supports "+topic, func(t *testing.T) {
			if !TopicSupportsEnsure(topic) {
				t.Errorf("expected %s to support ensure", topic)
			}
		})
	}

	excluded := []string{"activity", "comment", "attachment", "link", "worklog", "epic", "page", "member"}
	for _, topic := range excluded {
		t.Run("excludes "+topic, func(t *testing.T) {
			if TopicSupportsEnsure(topic) {
				t.Errorf("expected %s to be excluded from ensure", topic)
			}
		})
	}
}

func TestBuildEnsureUpdateURL(t *testing.T) {
	client := api.NewClient("https://example.com", "token", "my-ws", false, nil)

	t.Run("substitutes resource ID in path param", func(t *testing.T) {
		spec := &docs.EndpointSpec{
			Method:       "PATCH",
			PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/states/{state_id}/",
			Params: []docs.ParamSpec{
				{Name: "workspace_slug", Location: docs.ParamPath},
				{Name: "project_id", Location: docs.ParamPath},
				{Name: "state_id", Location: docs.ParamPath},
			},
		}

		url, err := buildEnsureUpdateURL(client, spec, "resource-uuid-123", "proj-uuid")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "https://example.com/api/v1/workspaces/my-ws/projects/proj-uuid/states/resource-uuid-123/"
		if url != expected {
			t.Errorf("expected %s, got %s", expected, url)
		}
	})
}

func TestBuildEnsureURL(t *testing.T) {
	client := api.NewClient("https://example.com", "token", "my-ws", false, nil)

	t.Run("substitutes workspace and project", func(t *testing.T) {
		spec := &docs.EndpointSpec{
			PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/states/",
		}

		url, err := buildEnsureURL(client, spec, "proj-uuid")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "https://example.com/api/v1/workspaces/my-ws/projects/proj-uuid/states/"
		if url != expected {
			t.Errorf("expected %s, got %s", expected, url)
		}
	})
}

func TestEnsureCreate(t *testing.T) {
	createCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/v1/workspaces/test-ws/projects/proj-uuid/states/":
			// List returns empty results (no match)
			fmt.Fprint(w, `{"results": [], "next_page_results": false}`)
		case r.Method == "POST" && r.URL.Path == "/api/v1/workspaces/test-ws/projects/proj-uuid/states/":
			createCalled = true
			fmt.Fprint(w, `{"id": "new-state-uuid", "name": "NewState", "color": "#FF0000"}`)
		default:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"detail": "not found: %s %s"}`, r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	quiet := true
	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		RequireProject:   func() (string, error) { return "proj-uuid", nil },
		PaginationParams: func() api.PaginationParams { return api.PaginationParams{PerPage: 100} },
		Formatter:        func() output.Formatter { return output.New("json") },
		IsUUID:           func(s string) bool { return len(s) == 36 && s[8] == '-' },
		FlagQuiet:        &quiet,
	}

	specs := &ensureSpecs{
		create: &docs.EndpointSpec{
			Method:       "POST",
			PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/states/",
			Params: []docs.ParamSpec{
				{Name: "name", Location: docs.ParamBody, Type: "string"},
				{Name: "color", Location: docs.ParamBody, Type: "string"},
			},
		},
		list: &docs.EndpointSpec{
			Method:       "GET",
			PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/states/",
		},
		update: &docs.EndpointSpec{
			Method:       "PATCH",
			PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/states/{state_id}/",
			Params: []docs.ParamSpec{
				{Name: "state_id", Location: docs.ParamPath, Type: "string"},
			},
		},
	}

	cmd := BuildEnsureCommand("state", specs, deps)
	cmd.SetArgs([]string{"--name", "NewState", "--color", "#FF0000"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !createCalled {
		t.Error("expected create to be called")
	}
}

func TestEnsureUpdate(t *testing.T) {
	updateCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/v1/workspaces/test-ws/projects/proj-uuid/states/":
			// List returns a matching state
			fmt.Fprint(w, `{"results": [{"id": "existing-uuid", "name": "ExistingState", "color": "#00FF00"}], "next_page_results": false}`)
		case r.Method == "PATCH" && r.URL.Path == "/api/v1/workspaces/test-ws/projects/proj-uuid/states/existing-uuid/":
			updateCalled = true
			fmt.Fprint(w, `{"id": "existing-uuid", "name": "ExistingState", "color": "#FF0000"}`)
		default:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"detail": "not found: %s %s"}`, r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	quiet := true
	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		RequireProject:   func() (string, error) { return "proj-uuid", nil },
		PaginationParams: func() api.PaginationParams { return api.PaginationParams{PerPage: 100} },
		Formatter:        func() output.Formatter { return output.New("json") },
		IsUUID:           func(s string) bool { return len(s) == 36 && s[8] == '-' },
		FlagQuiet:        &quiet,
	}

	specs := &ensureSpecs{
		create: &docs.EndpointSpec{
			Method:       "POST",
			PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/states/",
			Params: []docs.ParamSpec{
				{Name: "name", Location: docs.ParamBody, Type: "string"},
				{Name: "color", Location: docs.ParamBody, Type: "string"},
			},
		},
		list: &docs.EndpointSpec{
			Method:       "GET",
			PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/states/",
		},
		update: &docs.EndpointSpec{
			Method:       "PATCH",
			PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/states/{state_id}/",
			Params: []docs.ParamSpec{
				{Name: "state_id", Location: docs.ParamPath, Type: "string"},
			},
		},
	}

	cmd := BuildEnsureCommand("state", specs, deps)
	cmd.SetArgs([]string{"--name", "ExistingState", "--color", "#FF0000"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updateCalled {
		t.Error("expected update to be called")
	}
}

func TestEnsureCustomMatchField(t *testing.T) {
	updateCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET":
			fmt.Fprint(w, `{"results": [{"id": "id1", "name": "State1", "color": "#FF0000"}, {"id": "id2", "name": "State2", "color": "#00FF00"}], "next_page_results": false}`)
		case r.Method == "PATCH":
			updateCalled = true
			fmt.Fprint(w, `{"id": "id2", "name": "State2", "color": "#0000FF"}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	quiet := true
	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		RequireProject:   func() (string, error) { return "proj-uuid", nil },
		PaginationParams: func() api.PaginationParams { return api.PaginationParams{PerPage: 100} },
		Formatter:        func() output.Formatter { return output.New("json") },
		IsUUID:           func(s string) bool { return len(s) == 36 && s[8] == '-' },
		FlagQuiet:        &quiet,
	}

	specs := &ensureSpecs{
		create: &docs.EndpointSpec{
			Method:       "POST",
			PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/states/",
			Params: []docs.ParamSpec{
				{Name: "name", Location: docs.ParamBody, Type: "string"},
				{Name: "color", Location: docs.ParamBody, Type: "string"},
			},
		},
		list: &docs.EndpointSpec{
			Method:       "GET",
			PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/states/",
		},
		update: &docs.EndpointSpec{
			Method:       "PATCH",
			PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/states/{state_id}/",
			Params: []docs.ParamSpec{
				{Name: "state_id", Location: docs.ParamPath, Type: "string"},
			},
		},
	}

	cmd := BuildEnsureCommand("state", specs, deps)
	cmd.SetArgs([]string{"--match-field", "color", "--color", "#00FF00", "--name", "RenamedState"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updateCalled {
		t.Error("expected update to be called when matching by color")
	}
}

func TestEnsureDryRun(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("HTTP request should not be made in dry-run mode")
	}))
	defer srv.Close()

	dryRun := true
	quiet := true
	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		RequireProject:   func() (string, error) { return "proj-uuid", nil },
		PaginationParams: func() api.PaginationParams { return api.PaginationParams{PerPage: 100} },
		Formatter:        func() output.Formatter { return output.New("json") },
		IsUUID:           func(s string) bool { return len(s) == 36 && s[8] == '-' },
		FlagDryRun:       &dryRun,
		FlagQuiet:        &quiet,
	}

	specs := &ensureSpecs{
		create: &docs.EndpointSpec{
			Method:       "POST",
			PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/states/",
			Params: []docs.ParamSpec{
				{Name: "name", Location: docs.ParamBody, Type: "string"},
			},
		},
		list: &docs.EndpointSpec{
			Method:       "GET",
			PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/states/",
		},
	}

	cmd := BuildEnsureCommand("state", specs, deps)
	cmd.SetArgs([]string{"--name", "TestState"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnsureMissingMatchField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not make requests when match field is missing")
	}))
	defer srv.Close()

	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		RequireProject:   func() (string, error) { return "proj-uuid", nil },
		PaginationParams: func() api.PaginationParams { return api.PaginationParams{PerPage: 100} },
		Formatter:        func() output.Formatter { return output.New("json") },
		IsUUID:           func(s string) bool { return len(s) == 36 && s[8] == '-' },
	}

	specs := &ensureSpecs{
		create: &docs.EndpointSpec{
			Method:       "POST",
			PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/states/",
			Params: []docs.ParamSpec{
				{Name: "name", Location: docs.ParamBody, Type: "string"},
				{Name: "color", Location: docs.ParamBody, Type: "string"},
			},
		},
		list: &docs.EndpointSpec{
			Method:       "GET",
			PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/states/",
		},
	}

	// Don't provide --name (default match field)
	cmd := BuildEnsureCommand("state", specs, deps)
	cmd.SetArgs([]string{"--color", "#FF0000"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing match field, got nil")
	}
	if !contains(err.Error(), "match field") {
		t.Errorf("expected error about match field, got: %v", err)
	}
}

func TestEnsureWithRelations(t *testing.T) {
	createCalled := false
	moduleCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/v1/workspaces/test-ws/projects/proj-uuid/work-items/":
			fmt.Fprint(w, `{"results": [], "next_page_results": false}`)
		case r.Method == "POST" && r.URL.Path == "/api/v1/workspaces/test-ws/projects/proj-uuid/work-items/":
			createCalled = true
			fmt.Fprint(w, `{"id": "new-issue-uuid", "name": "Test Issue"}`)
		case r.Method == "POST" && r.URL.Path == "/api/v1/workspaces/test-ws/projects/proj-uuid/modules/mod-uuid/module-issues/":
			moduleCalled = true
			fmt.Fprint(w, `{}`)
		default:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"detail": "not found: %s %s"}`, r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	quiet := true
	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		RequireProject:   func() (string, error) { return "proj-uuid", nil },
		PaginationParams: func() api.PaginationParams { return api.PaginationParams{PerPage: 100} },
		Formatter:        func() output.Formatter { return output.New("json") },
		IsUUID:           func(s string) bool { return len(s) == 36 && s[8] == '-' },
		FlagQuiet:        &quiet,
	}

	specs := &ensureSpecs{
		create: &docs.EndpointSpec{
			Method:       "POST",
			PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/",
			Params: []docs.ParamSpec{
				{Name: "name", Location: docs.ParamBody, Type: "string"},
				{Name: "module", Location: docs.ParamBody, Type: "string"},
			},
		},
		list: &docs.EndpointSpec{
			Method:       "GET",
			PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/",
		},
		update: &docs.EndpointSpec{
			Method:       "PATCH",
			PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/{work_item_id}/",
			Params: []docs.ParamSpec{
				{Name: "work_item_id", Location: docs.ParamPath, Type: "string"},
			},
		},
	}

	cmd := BuildEnsureCommand("issue", specs, deps)
	cmd.SetArgs([]string{"--name", "Test Issue", "--module", "mod-uuid"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !createCalled {
		t.Error("expected create to be called")
	}
	if !moduleCalled {
		t.Error("expected module attach to be called")
	}
}

func TestEnsureMultiPageMatch(t *testing.T) {
	page := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		page++
		switch page {
		case 1:
			fmt.Fprint(w, `{"results": [{"id": "a1", "name": "Alpha"}], "next_page_results": true, "next_cursor": "100:1:1"}`)
		case 2:
			fmt.Fprint(w, `{"results": [{"id": "b2", "name": "Target"}], "next_page_results": false}`)
		}
	}))
	defer srv.Close()

	client := api.NewClient(srv.URL, "test-token", "test-ws", false, nil)
	listURL := srv.URL + "/api/v1/workspaces/test-ws/projects/proj/items/"

	id, err := findMatchInList(context.Background(), client, listURL, "name", "Target", &Deps{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "b2" {
		t.Errorf("expected b2, got %s", id)
	}
}

func TestFindSpecByMethod(t *testing.T) {
	specs := []docs.CachedSpec{
		{Spec: docs.EndpointSpec{Method: "GET", EntryTitle: "List"}},
		{Spec: docs.EndpointSpec{Method: "POST", EntryTitle: "Create"}},
		{Spec: docs.EndpointSpec{Method: "PATCH", EntryTitle: "Update"}},
	}

	t.Run("finds GET", func(t *testing.T) {
		s := findSpecByMethod(specs, "GET")
		if s == nil || s.EntryTitle != "List" {
			t.Error("expected List spec")
		}
	})

	t.Run("finds POST", func(t *testing.T) {
		s := findSpecByMethod(specs, "POST")
		if s == nil || s.EntryTitle != "Create" {
			t.Error("expected Create spec")
		}
	})

	t.Run("returns nil for missing method", func(t *testing.T) {
		s := findSpecByMethod(specs, "DELETE")
		if s != nil {
			t.Error("expected nil for DELETE")
		}
	})
}

// contains is a helper for checking substrings in error messages.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

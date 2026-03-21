package cmdgen

import (
	"testing"

	"github.com/mggarofalo/plane-cli/internal/docs"
)

func TestBuildEndpointCommand_DuplicateHTMLParams(t *testing.T) {
	// Regression test: specs with both "description" and "description_html"
	// caused a panic from pflag's duplicate flag detection. The _html handler
	// registers --description (markdown shorthand) and --description-html
	// (raw HTML), then the plain "description" param tried to register
	// --description again.
	spec := &docs.EndpointSpec{
		EntryTitle:   "Create Skill",
		Method:       "POST",
		PathTemplate: "/api/v1/skills/",
		SourceURL:    "https://example.com/docs",
		Params: []docs.ParamSpec{
			{Name: "name", Location: docs.ParamBody, Type: "string"},
			{Name: "description", Location: docs.ParamBody, Type: "string"},
			{Name: "description_html", Location: docs.ParamBody, Type: "string"},
		},
	}

	// Must not panic
	cmd := BuildEndpointCommand("skill", "create", spec, &Deps{})

	if cmd.Flags().Lookup("description") == nil {
		t.Error("expected --description flag (markdown shorthand)")
	}
	if cmd.Flags().Lookup("description-html") == nil {
		t.Error("expected --description-html flag (raw HTML)")
	}
	if cmd.Flags().Lookup("name") == nil {
		t.Error("expected --name flag")
	}
}

func TestBuildEndpointCommand_HTMLBeforePlain(t *testing.T) {
	// Verify no panic regardless of param ordering: _html before plain.
	spec := &docs.EndpointSpec{
		EntryTitle:   "Create Item",
		Method:       "POST",
		PathTemplate: "/api/v1/items/",
		SourceURL:    "https://example.com/docs",
		Params: []docs.ParamSpec{
			{Name: "description_html", Location: docs.ParamBody, Type: "string"},
			{Name: "description", Location: docs.ParamBody, Type: "string"},
		},
	}

	cmd := BuildEndpointCommand("item", "create", spec, &Deps{})

	if cmd.Flags().Lookup("description") == nil {
		t.Error("expected --description flag")
	}
	if cmd.Flags().Lookup("description-html") == nil {
		t.Error("expected --description-html flag")
	}
}

func TestBuildEndpointCommand_MultipleHTMLPairs(t *testing.T) {
	// Multiple _html pairs in the same spec should all register safely.
	spec := &docs.EndpointSpec{
		EntryTitle:   "Create Page",
		Method:       "POST",
		PathTemplate: "/api/v1/pages/",
		SourceURL:    "https://example.com/docs",
		Params: []docs.ParamSpec{
			{Name: "name", Location: docs.ParamBody, Type: "string"},
			{Name: "description", Location: docs.ParamBody, Type: "string"},
			{Name: "description_html", Location: docs.ParamBody, Type: "string"},
			{Name: "content", Location: docs.ParamBody, Type: "string"},
			{Name: "content_html", Location: docs.ParamBody, Type: "string"},
		},
	}

	cmd := BuildEndpointCommand("page", "create", spec, &Deps{})

	for _, flag := range []string{"name", "description", "description-html", "content", "content-html"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag", flag)
		}
	}
}

func TestBuildEndpointCommand_DuplicateParamNames(t *testing.T) {
	// If a spec somehow has two params that map to the same flag name,
	// the second should be silently skipped (not panic).
	spec := &docs.EndpointSpec{
		EntryTitle:   "Create Thing",
		Method:       "POST",
		PathTemplate: "/api/v1/things/",
		SourceURL:    "https://example.com/docs",
		Params: []docs.ParamSpec{
			{Name: "label", Location: docs.ParamBody, Type: "string"},
			{Name: "label", Location: docs.ParamBody, Type: "string"}, // duplicate
		},
	}

	// Must not panic
	cmd := BuildEndpointCommand("thing", "create", spec, &Deps{})

	if cmd.Flags().Lookup("label") == nil {
		t.Error("expected --label flag")
	}
}

func TestBuildEndpointCommand_SkipsGlobalFlags(t *testing.T) {
	spec := &docs.EndpointSpec{
		EntryTitle:   "List Items",
		Method:       "GET",
		PathTemplate: "/api/v1/items/",
		SourceURL:    "https://example.com/docs",
		Params: []docs.ParamSpec{
			{Name: "workspace_slug", Location: docs.ParamPath, Type: "string"},
			{Name: "project_id", Location: docs.ParamPath, Type: "string"},
			{Name: "name", Location: docs.ParamBody, Type: "string"},
		},
	}

	cmd := BuildEndpointCommand("item", "list", spec, &Deps{})

	// workspace and project are global — should not be registered as local flags
	if cmd.Flags().Lookup("workspace-slug") != nil {
		t.Error("workspace-slug should be skipped (handled globally)")
	}
	if cmd.Flags().Lookup("name") == nil {
		t.Error("expected --name flag")
	}
}

func TestBuildEnsureCommand_DuplicateHTMLParams(t *testing.T) {
	// Same regression test for BuildEnsureCommand.
	specs := &ensureSpecs{
		create: &docs.EndpointSpec{
			Method:       "POST",
			PathTemplate: "/api/v1/skills/",
			Params: []docs.ParamSpec{
				{Name: "name", Location: docs.ParamBody, Type: "string"},
				{Name: "description", Location: docs.ParamBody, Type: "string"},
				{Name: "description_html", Location: docs.ParamBody, Type: "string"},
			},
		},
		list: &docs.EndpointSpec{
			Method:       "GET",
			PathTemplate: "/api/v1/skills/",
		},
	}

	// Must not panic
	cmd := BuildEnsureCommand("skill", specs, &Deps{})

	if cmd.Flags().Lookup("description") == nil {
		t.Error("expected --description flag")
	}
	if cmd.Flags().Lookup("description-html") == nil {
		t.Error("expected --description-html flag")
	}
}

func TestBuildTopicCommand_NoPanicOnBadSpec(t *testing.T) {
	// Even with a nil Deps (which would panic during execution),
	// command building itself should not panic.
	topic := &docs.Topic{
		Name: "test",
		Entries: []docs.Entry{
			{Title: "Create Test", URL: "https://example.com/api-reference/test/create"},
		},
	}
	cachedSpecs := []docs.CachedSpec{
		{Spec: docs.EndpointSpec{
			EntryTitle:   "Create Test",
			Method:       "POST",
			PathTemplate: "/api/v1/tests/",
			SourceURL:    "https://example.com/docs",
			Params: []docs.ParamSpec{
				{Name: "name", Location: docs.ParamBody, Type: "string"},
				{Name: "description", Location: docs.ParamBody, Type: "string"},
				{Name: "description_html", Location: docs.ParamBody, Type: "string"},
			},
		}},
	}

	// Must not panic
	cmd := BuildTopicCommand("test", topic, cachedSpecs, &Deps{})

	if cmd.Use != "test" {
		t.Errorf("expected Use=test, got %s", cmd.Use)
	}
	if !cmd.HasSubCommands() {
		t.Error("expected subcommands")
	}
}

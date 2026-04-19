package skillgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mggarofalo/plane-cli/internal/docs"
)

func TestGenerate_WithSpecs(t *testing.T) {
	// Set up a temporary spec cache with a known spec
	profile := "test-skill"
	topicName := "label"
	spec := &docs.EndpointSpec{
		TopicName:    topicName,
		EntryTitle:   "Create Label",
		SourceURL:    "https://example.com/api-reference/label/add-label",
		Method:       "POST",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/labels/",
		Params: []docs.ParamSpec{
			{Name: "workspace_slug", Type: "string", Required: true, Location: docs.ParamPath},
			{Name: "project_id", Type: "string", Required: true, Location: docs.ParamPath},
			{Name: "name", Type: "string", Required: true, Location: docs.ParamBody},
			{Name: "color", Type: "string", Required: false, Location: docs.ParamBody},
			{Name: "description", Type: "string", Required: false, Location: docs.ParamBody},
		},
	}

	if err := docs.WriteSpec(profile, "https://example.com", spec); err != nil {
		t.Fatalf("WriteSpec: %v", err)
	}
	t.Cleanup(func() {
		dir, _ := docs.SpecCacheDir(profile)
		_ = os.RemoveAll(dir)
	})

	outputDir := t.TempDir()
	if err := Generate(profile, outputDir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Verify SKILL.md exists and has expected content
	skillPath := filepath.Join(outputDir, "SKILL.md")
	skillData, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("reading SKILL.md: %v", err)
	}
	skill := string(skillData)

	checks := []string{
		"name: plane",
		"user_invocable: false",
		"Resource Topics",
		"| label |",
		"| issue |",
		"Name Resolution",
		"Ensure",
		"Gotchas",
		"references/gotchas.md",
		"module add-work-items",
		"Modules are attached separately",
	}
	for _, check := range checks {
		if !strings.Contains(skill, check) {
			t.Errorf("SKILL.md missing %q", check)
		}
	}

	// Verify resources.md exists and has spec-derived content
	resPath := filepath.Join(outputDir, "references", "resources.md")
	resData, err := os.ReadFile(resPath)
	if err != nil {
		t.Fatalf("reading resources.md: %v", err)
	}
	res := string(resData)

	resChecks := []string{
		"## Label",
		"`plane label create`",
		"label_create",
		"POST",
		"| `--name` | string | yes |",
		"| `--color` | string | no |",
	}
	for _, check := range resChecks {
		if !strings.Contains(res, check) {
			t.Errorf("resources.md missing %q", check)
		}
	}

	// Hidden params should NOT appear
	if strings.Contains(res, "--workspace-slug") {
		t.Error("resources.md should not contain --workspace-slug (hidden param)")
	}
	if strings.Contains(res, "--project-id") {
		t.Error("resources.md should not contain --project-id (hidden param)")
	}

	// Verify gotchas.md exists and has expected content
	gotchasPath := filepath.Join(outputDir, "references", "gotchas.md")
	gotchasData, err := os.ReadFile(gotchasPath)
	if err != nil {
		t.Fatalf("reading gotchas.md: %v", err)
	}
	gotchas := string(gotchasData)

	gotchasChecks := []string{
		"Plane CLI Gotchas",
		"workspace slug",
		"MCP tool names have NO prefix",
		"JSONL",
		"--pk",
		"Modules and Other Associations",
		"module add-work-items",
	}
	for _, check := range gotchasChecks {
		if !strings.Contains(gotchas, check) {
			t.Errorf("gotchas.md missing %q", check)
		}
	}
}

func TestGenerate_WithoutSpecs(t *testing.T) {
	// Use a profile with no cached specs
	profile := "empty-skill-profile"
	outputDir := t.TempDir()

	if err := Generate(profile, outputDir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// SKILL.md should still list topics
	skillData, err := os.ReadFile(filepath.Join(outputDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("reading SKILL.md: %v", err)
	}
	skill := string(skillData)

	if !strings.Contains(skill, "| issue |") {
		t.Error("SKILL.md should list issue topic even without specs")
	}
	if !strings.Contains(skill, "| cycle |") {
		t.Error("SKILL.md should list cycle topic even without specs")
	}

	// resources.md should indicate no cached spec
	resData, err := os.ReadFile(filepath.Join(outputDir, "references", "resources.md"))
	if err != nil {
		t.Fatalf("reading resources.md: %v", err)
	}
	res := string(resData)

	if !strings.Contains(res, "No cached spec available") {
		t.Error("resources.md should indicate missing specs")
	}

	// gotchas.md should still be generated
	gotchasPath := filepath.Join(outputDir, "references", "gotchas.md")
	if _, err := os.Stat(gotchasPath); os.IsNotExist(err) {
		t.Error("gotchas.md should be generated even without specs")
	}
}

func TestFilterAndSortParams(t *testing.T) {
	params := []docs.ParamSpec{
		{Name: "workspace_slug", Type: "string", Required: true},
		{Name: "project_id", Type: "string", Required: true},
		{Name: "slug", Type: "string", Required: true},
		{Name: "name", Type: "string", Required: true},
		{Name: "color", Type: "string", Required: false},
		{Name: "description_html", Type: "string", Required: false},
		{Name: "alpha", Type: "string", Required: false},
	}

	result := filterAndSortParams(params)

	// Hidden params excluded
	for _, p := range result {
		if p.FlagName == "workspace-slug" || p.FlagName == "project-id" || p.FlagName == "slug" {
			t.Errorf("hidden param %q should be excluded", p.FlagName)
		}
	}

	// Required first
	if len(result) < 2 {
		t.Fatalf("expected at least 2 params, got %d", len(result))
	}
	if !result[0].Required {
		t.Error("first param should be required")
	}

	// HTML param converted to markdown name
	found := false
	for _, p := range result {
		if p.FlagName == "description" {
			found = true
		}
		if p.FlagName == "description-html" {
			t.Error("should use markdown name 'description', not 'description-html'")
		}
	}
	if !found {
		t.Error("description_html should become 'description' flag")
	}

	// Alphabetical within non-required group
	var nonReq []string
	for _, p := range result {
		if !p.Required {
			nonReq = append(nonReq, p.FlagName)
		}
	}
	for i := 1; i < len(nonReq); i++ {
		if nonReq[i] < nonReq[i-1] {
			t.Errorf("non-required params not sorted: %v", nonReq)
			break
		}
	}
}

func TestParamAnnotations(t *testing.T) {
	params := []docs.ParamSpec{
		{Name: "state_id", Type: "string"},
		{Name: "work_item_id", Type: "string"},
		{Name: "pk", Type: "string"},
		{Name: "name", Type: "string"},
	}

	result := filterAndSortParams(params)

	annotations := make(map[string]ParamData)
	for _, p := range result {
		annotations[p.FlagName] = p
	}

	if p, ok := annotations["state-id"]; !ok || !p.Resolvable {
		t.Error("state-id should be marked resolvable")
	}
	if p, ok := annotations["work-item-id"]; !ok || !p.IssueRef {
		t.Error("work-item-id should be marked as issue ref")
	}
	if p, ok := annotations["pk"]; !ok || !p.IsGenericPK {
		t.Error("pk should be marked as generic PK")
	}
	if p, ok := annotations["name"]; !ok || p.Resolvable || p.IssueRef || p.IsGenericPK {
		t.Error("name should not be marked as resolvable, issue ref, or generic PK")
	}
}

func TestSlugHiddenInSkillDocs(t *testing.T) {
	// Verify that specs using {slug} instead of {workspace_slug} still hide the param
	profile := "test-slug-hidden"
	spec := &docs.EndpointSpec{
		TopicName:    "project",
		EntryTitle:   "List Projects",
		SourceURL:    "https://example.com/api-reference/project/list-projects",
		Method:       "GET",
		PathTemplate: "/api/v1/workspaces/{slug}/projects/",
		Params: []docs.ParamSpec{
			{Name: "slug", Type: "string", Required: true, Location: docs.ParamPath},
			{Name: "cursor", Type: "string", Required: false, Location: docs.ParamQuery},
		},
	}

	if err := docs.WriteSpec(profile, "https://example.com", spec); err != nil {
		t.Fatalf("WriteSpec: %v", err)
	}
	t.Cleanup(func() {
		dir, _ := docs.SpecCacheDir(profile)
		_ = os.RemoveAll(dir)
	})

	outputDir := t.TempDir()
	if err := Generate(profile, outputDir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	resData, err := os.ReadFile(filepath.Join(outputDir, "references", "resources.md"))
	if err != nil {
		t.Fatalf("reading resources.md: %v", err)
	}
	res := string(resData)

	if strings.Contains(res, "| `--slug`") {
		t.Error("resources.md should not contain --slug (hidden param, alias for workspace_slug)")
	}
	// cursor should still appear
	if !strings.Contains(res, "| `--cursor`") {
		t.Error("resources.md should contain --cursor (not a hidden param)")
	}
}

func TestPKAnnotationInResources(t *testing.T) {
	profile := "test-pk-annotation"
	spec := &docs.EndpointSpec{
		TopicName:    "project",
		EntryTitle:   "Get Project",
		SourceURL:    "https://example.com/api-reference/project/get-project",
		Method:       "GET",
		PathTemplate: "/api/v1/workspaces/{slug}/projects/{pk}/",
		Params: []docs.ParamSpec{
			{Name: "slug", Type: "string", Required: true, Location: docs.ParamPath},
			{Name: "pk", Type: "string", Required: true, Location: docs.ParamPath},
		},
	}

	if err := docs.WriteSpec(profile, "https://example.com", spec); err != nil {
		t.Fatalf("WriteSpec: %v", err)
	}
	t.Cleanup(func() {
		dir, _ := docs.SpecCacheDir(profile)
		_ = os.RemoveAll(dir)
	})

	outputDir := t.TempDir()
	if err := Generate(profile, outputDir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	resData, err := os.ReadFile(filepath.Join(outputDir, "references", "resources.md"))
	if err != nil {
		t.Fatalf("reading resources.md: %v", err)
	}
	res := string(resData)

	// pk should appear with the "resource UUID" annotation
	if !strings.Contains(res, "resource UUID") {
		t.Error("resources.md should annotate --pk with 'resource UUID' hint")
	}
}
